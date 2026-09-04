// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

// countBothWays runs the same content through the counter written for the
// language and through the generic loop, and hands back the two results.
func countBothWays(t *testing.T, language string, content []byte) (FileJob, FileJob) {
	t.Helper()

	fast := FileJob{Language: language}
	fast.SetContent(string(content))
	DisableSpecialisedCounters = false
	CountStats(&fast)

	generic := FileJob{Language: language}
	generic.SetContent(string(content))
	DisableSpecialisedCounters = true
	CountStats(&generic)
	DisableSpecialisedCounters = false

	return fast, generic
}

func compareCounts(t *testing.T, name string, fast, generic FileJob) {
	t.Helper()

	if fast.Lines != generic.Lines || fast.Code != generic.Code ||
		fast.Comment != generic.Comment || fast.Blank != generic.Blank ||
		fast.Complexity != generic.Complexity {
		t.Errorf("%s disagrees\n  java   : lines=%d code=%d comment=%d blank=%d complexity=%d\n  generic: lines=%d code=%d comment=%d blank=%d complexity=%d",
			name,
			fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity,
			generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Complexity)
	}
}

// The shapes a Java file is written in, each of which the two counters have to
// read the same way.
func TestJavaCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "int x = 1;"},
		{"blank lines", "\n\n\nclass A {}\n\n"},
		{"line comment", "// a comment\nclass A {}\n"},
		{"line comment holding a block opener", "// /* not opened\nclass A {}\n"},
		{"block comment over lines", "/*\n a comment\n*/\nclass A {}\n"},
		{"block comment closed then code", "/* a */ class A {}\n"},
		{"code then block comment", "class A {} /* a\n b */\nint x = 1;\n"},
		{"block comment does not nest", "/* outer /* inner */ still code */\nint x = 1;\n"},
		{"close touching reopen", "/* one *//* two */\nint x = 1;\n"},
		{"string holding a comment", "String s = \"// not a comment\";\nint x = 1;\n"},
		{"string holding a block opener", "String s = \"/* not a comment\";\nint x = 1;\n"},
		{"escaped quote in a string", "String s = \"a \\\" b\";\n// a comment\n"},
		{"escaped backslash then quote", "String s = \"a \\\\\";\n// a comment\n"},
		{"char literal", "char c = 'x';\n// a comment\n"},
		{"char literal holding a quote", "char c = '\"';\n// a comment\n"},
		{"char literal holding an escape", "char c = '\\\\';\n// a comment\n"},
		{"unterminated string", "String s = \"never closed\nint x = 1;\nint y = 2;\n"},
		{"unterminated block comment", "/* never closed\nint x = 1;\n"},
		{"complexity tokens", "if (a) { for (;;) { while (x) {} } }\nelse { try {} catch (E e) {} finally {} }\n"},
		{"complexity inside a word", "int retry = 1;\nint iffy = 2;\nint switcher = 3;\n"},
		{"complexity operators", "boolean b = a || c && d != e == f;\n"},
		{"complexity in a comment", "// if for while && ||\nint x = 1;\n"},
		{"complexity in a string", "String s = \"if for while && ||\";\n"},
		{"crlf line endings", "class A {\r\n// a comment\r\nint x = 1;\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tint x = 1;\n"},
		{"quote at end of file", "String s = \""},
		{"slash at end of file", "int x = 1; /"},
		{"star slash outside a comment", "int x = a */ b;\n"},
	} {
		fast, generic := countBothWays(t, "Java", []byte(test.content))
		compareCounts(t, test.name, fast, generic)
	}
}

// The real check. Every Java file of the performance corpus is read both ways
// and the counts have to be identical, which is the whole warrant for having a
// second implementation at all.
func TestJavaCounterAgreesOnTheCorpus(t *testing.T) {
	ProcessConstants()

	if testing.Short() {
		t.Skip("walks the whole performance corpus")
	}

	// The live heap here is the trie built for every one of the languages, which
	// is large and all pointers, so the default collector rescans it on every
	// cycle and the walk below spends its time in the garbage collector rather
	// than in either counter. Nothing here holds on to a file once it is counted.
	defer debug.SetGCPercent(debug.SetGCPercent(1600))

	// The corpus in examples/performance_tests is one file written out many
	// thousands of times, which measures throughput and tells us nothing about
	// whether the two counters agree, so point this at real code instead.
	corpus := os.Getenv("SCC_DIFF_CORPUS")
	if corpus == "" {
		corpus = filepath.Join("..", "examples", "performance_tests")
	}
	if _, err := os.Stat(corpus); err != nil {
		t.Skip("no performance corpus checked out")
	}

	limit := 0
	if v := os.Getenv("SCC_DIFF_LIMIT"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	checked := 0
	disagreed := 0
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}

		if limit != 0 && checked >= limit {
			return filepath.SkipAll
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fast, generic := countBothWays(t, "Java", content)
		checked++
		if fast.Lines != generic.Lines || fast.Code != generic.Code ||
			fast.Comment != generic.Comment || fast.Blank != generic.Blank ||
			fast.Complexity != generic.Complexity {
			disagreed++
			if disagreed <= 5 {
				compareCounts(t, path, fast, generic)
			}
		}

		return nil
	})

	if checked == 0 {
		t.Skip("no java files in the corpus")
	}

	t.Logf("checked %d files, %d disagreed", checked, disagreed)
}

// The counter scans with a smaller table when complexity is off, so that is a
// second path over the same files and wants the same warrant.
func TestJavaCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestJavaCounterAgreesOnTheCorpus(t)
}

// --no-complexity sets the global, which the generic loop reads by leaving the
// checks out of its trie. The counter has to stop counting them too.
func TestJavaCounterAgreesWithComplexityOff(t *testing.T) {
	ProcessConstants()

	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	content := []byte("if (a) { for (;;) {} }\nwhile (b) { try {} catch (E e) {} }\n")
	fast, generic := countBothWays(t, "Java", content)
	compareCounts(t, "complexity off", fast, generic)

	if fast.Complexity != 0 {
		t.Errorf("expected no complexity counted, got %d", fast.Complexity)
	}
}

func benchmarkJavaCounter(b *testing.B, specialised bool) {
	b.Helper()
	ProcessConstants()

	content, err := os.ReadFile(filepath.Join("..", "examples", "performance_tests", "1", "0.java"))
	if err != nil {
		b.Skip("no performance corpus checked out")
	}

	DisableSpecialisedCounters = !specialised
	defer func() { DisableSpecialisedCounters = false }()

	b.SetBytes(int64(len(content)))
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		fileJob := FileJob{Language: "Java", Content: content, Bytes: int64(len(content))}
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsJavaGeneric(b *testing.B)     { benchmarkJavaCounter(b, false) }
func BenchmarkCountStatsJavaSpecialised(b *testing.B) { benchmarkJavaCounter(b, true) }

// benchmarkJavaCorpus counts real Java rather than the one file that
// examples/performance_tests is made of, whose mix of code, comment and string
// is its own and not that of Java at large. Point SCC_DIFF_CORPUS at a tree of
// real code.
func benchmarkJavaCorpus(b *testing.B, specialised bool) {
	b.Helper()
	ProcessConstants()

	corpus := os.Getenv("SCC_DIFF_CORPUS")
	if corpus == "" {
		b.Skip("set SCC_DIFF_CORPUS to a tree of real java")
	}

	var files [][]byte
	var total int64
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".java") || len(files) >= 400 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, content)
		total += int64(len(content))

		return nil
	})

	if len(files) == 0 {
		b.Skip("no java found in the corpus")
	}

	DisableSpecialisedCounters = !specialised
	defer func() { DisableSpecialisedCounters = false }()

	b.SetBytes(total)
	b.ResetTimer()

	for b.Loop() {
		for _, content := range files {
			fileJob := FileJob{Language: "Java", Content: content, Bytes: int64(len(content))}
			CountStats(&fileJob)
		}
	}
}

func BenchmarkCountStatsJavaCorpusGeneric(b *testing.B)     { benchmarkJavaCorpus(b, false) }
func BenchmarkCountStatsJavaCorpusSpecialised(b *testing.B) { benchmarkJavaCorpus(b, true) }

// Complexity is what stops the scan being able to skip: the checks begin with
// f, i, s, w, e, t and c, which are better than a third of the bytes of a Java
// file, so every one of them has to be looked at. With complexity off the scan
// cares about five bytes in the whole alphabet, which is the shape a vector
// instruction can answer. These two measure that ceiling.
func benchmarkJavaCorpusNoComplexity(b *testing.B, specialised bool) {
	b.Helper()

	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkJavaCorpus(b, specialised)
}

func BenchmarkCountStatsJavaCorpusNoComplexityGeneric(b *testing.B) {
	benchmarkJavaCorpusNoComplexity(b, false)
}

func BenchmarkCountStatsJavaCorpusNoComplexitySpecialised(b *testing.B) {
	benchmarkJavaCorpusNoComplexity(b, true)
}
