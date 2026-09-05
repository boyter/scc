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

// The shapes a C file is written in, with the line splices that C has and Java
// does not, each of which the two counters have to read the same way.
func TestCCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"C", "C Header"} {
		for _, test := range []struct {
			name    string
			content string
		}{
			{"empty", ""},
			{"no trailing newline", "int x = 1;"},
			{"line comment", "// a comment\nint x = 1;\n"},
			{"block comment over lines", "/*\n a comment\n*/\nint x = 1;\n"},
			{"block comment does not nest", "/* outer /* inner */ still code */\nint x = 1;\n"},
			{"string holding a comment", "char *s = \"// not a comment\";\nint x = 1;\n"},
			{"escaped quote", "char *s = \"a \\\" b\";\n// a comment\n"},
			{"char literal holding a quote", "if (*str == '\"')\n// a comment\nint x = 1;\n"},
			{"splice in a line comment", "// continues \\\n   onto the next line\nint x = 1;\n"},
			{"splice with no words", "int a = 1;\n// ---- \\\nint b = 2;\nint c = 3;\n"},
			{"splice after a closed block", "int a = 1;\n/* block */ // trailing \\\nint x = 1;\nint y = 2;\n"},
			{"code then a spliced comment", "int a = 1; // comment \\\n   joined\nint x = 1;\n"},
			{"blank line inside a spliced comment", "// carries on \\\n\nint x = 1;\n"},
			{"string carried only by a splice", "char *s = \"abc \\\n\n// ended\";\nint x = 1;\n"},
			{"macro continued over lines", "#define X(a) \\\n\tdo { a; } while (0)\nint x = 1;\n"},
			{"unterminated string", "char *s = \"never closed\nint x = 1;\n"},
			{"unterminated block comment", "/* never closed\nint x = 1;\n"},
			{"complexity tokens", "if (a) { for (;;) { while (x) {} } }\nelse { switch (y) { case 1: break; } }\n"},
			{"complexity inside a word", "int retry = 1;\nint iffy = 2;\nint elsewhere = 3;\n"},
			{"complexity operators", "int b = a || c && d != e == f;\n"},
			{"crlf", "int a = 1;\r\n// a comment\r\nint b = 2;\r\n"},
			{"crlf with a splice", "// carries \\\r\n   on\r\nint x = 1;\r\n"},
		} {
			fast, generic := countBothWays(t, language, []byte(test.content))
			compareCounts(t, language+" "+test.name, fast, generic)
		}
	}
}

// Every C and C Header file of a real tree read both ways, the counts having to
// be identical. Point SCC_DIFF_C_CORPUS at a checkout of something large.
func TestCCounterAgreesOnTheCorpus(t *testing.T) {
	ProcessConstants()

	if testing.Short() {
		t.Skip("walks a whole source tree")
	}

	corpus := os.Getenv("SCC_DIFF_C_CORPUS")
	if corpus == "" {
		t.Skip("set SCC_DIFF_C_CORPUS to a tree of real C")
	}

	defer debug.SetGCPercent(debug.SetGCPercent(1600))

	limit := 0
	if v := os.Getenv("SCC_DIFF_LIMIT"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	checked := 0
	disagreed := 0
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		language := ""
		switch {
		case strings.HasSuffix(path, ".c"):
			language = "C"
		case strings.HasSuffix(path, ".h"):
			language = "C Header"
		default:
			return nil
		}

		if limit != 0 && checked >= limit {
			return filepath.SkipAll
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fast, generic := countBothWays(t, language, content)
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
		t.Skip("no C found in the corpus")
	}

	t.Logf("checked %d files, %d disagreed", checked, disagreed)
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestCCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestCCounterAgreesOnTheCorpus(t)
}

func benchmarkCCorpus(b *testing.B, specialised bool) {
	b.Helper()
	ProcessConstants()

	corpus := os.Getenv("SCC_DIFF_C_CORPUS")
	if corpus == "" {
		b.Skip("set SCC_DIFF_C_CORPUS to a tree of real C")
	}

	var files [][]byte
	var total int64
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".c") || len(files) >= 400 {
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
		b.Skip("no C found in the corpus")
	}

	SpecialisedCounters = specialised
	defer func() { SpecialisedCounters = false }()

	b.SetBytes(total)
	b.ResetTimer()

	for b.Loop() {
		for _, content := range files {
			fileJob := FileJob{Language: "C", Content: content, Bytes: int64(len(content))}
			CountStats(&fileJob)
		}
	}
}

func BenchmarkCountStatsCCorpusGeneric(b *testing.B)     { benchmarkCCorpus(b, false) }
func BenchmarkCountStatsCCorpusSpecialised(b *testing.B) { benchmarkCCorpus(b, true) }

// The same corpus scanned with complexity turned off, which is the ceiling the
// counting loop could reach if complexity cost nothing at all.
func BenchmarkCountStatsCCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCCorpus(b, true)
}
