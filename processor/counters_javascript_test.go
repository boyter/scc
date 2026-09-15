// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withoutRegexLiterals runs fn with the one deliberate disagreement of the
// JavaScript counter turned off, so everything else can be held to exact
// agreement with the generic loop.
func withoutRegexLiterals(t *testing.T, fn func()) {
	t.Helper()

	previous := ecmaRegexLiterals
	ecmaRegexLiterals = false
	defer func() { ecmaRegexLiterals = previous }()

	fn()
}

// The shapes a JavaScript file is written in, each of which the two counters
// have to read the same way. The regular expression literal is the one thing
// they must not, and it has its own test below.
func TestJavaScriptCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	withoutRegexLiterals(t, func() {
		for _, test := range []struct {
			name    string
			content string
		}{
			{"empty", ""},
			{"one newline", "\n"},
			{"no trailing newline", "let x = 1;"},
			{"blank lines", "\n\n\nlet a = 1;\n\n"},
			{"line comment", "// a comment\nlet x = 1;\n"},
			{"line comment holding a block opener", "// /* not opened\nlet x = 1;\n"},
			{"block comment over lines", "/*\n a comment\n*/\nlet x = 1;\n"},
			{"block comment closed then code", "/* a */ let x = 1;\n"},
			{"code then block comment", "let x = 1; /* a\n b */\nlet y = 2;\n"},
			{"block comment does not nest", "/* outer /* inner */ still code */\nlet x = 1;\n"},
			{"string holding a comment", "let s = \"// not a comment\";\nlet x = 1;\n"},
			{"single quoted string", "let s = 'a string';\n// a comment\n"},
			{"escaped quote in a string", "let s = \"a \\\" b\";\n// a comment\n"},
			{"escaped backslash then quote", "let s = \"a \\\\\";\n// a comment\n"},
			{"template literal", "let s = `a template`;\n// a comment\n"},
			{"template literal over lines", "let s = `line one\nline two\nline three`;\nlet x = 1;\n"},
			{"template literal holding a comment", "let s = `// not a comment`;\nlet x = 1;\n"},
			{"template literal holding a block opener", "let s = `/* not a comment */`;\nlet x = 1;\n"},
			{"template literal holding interpolation", "let s = `a ${b + c} d`;\nlet x = 1;\n"},
			{"template literal holding other quotes", "let s = `a \" b ' c`;\nlet x = 1;\n"},
			{"template literal holding a blank line", "let s = `one\n\nthree`;\nlet x = 1;\n"},
			{"escaped backtick", "let s = `a \\` b`;\n// a comment\n"},
			{"unterminated template literal", "let s = `never closed\nlet x = 1;\n"},
			{"unterminated string", "let s = \"never closed\nlet x = 1;\nlet y = 2;\n"},
			{"unterminated block comment", "/* never closed\nlet x = 1;\n"},
			{"complexity tokens", "if (a) { for (;;) { while (x) {} } }\nelse { switch (y) { case 1: break; } }\n"},
			{"complexity with brackets", "if(a) { for(;;) {} }\ncase(b)\n"},
			{"complexity inside a word", "let retry = 1;\nlet iffy = 2;\nlet switcher = 3;\nlet elsewhere = 4;\nlet cases = 5;\n"},
			{"complexity operators", "let b = a || c && d != e == f;\n"},
			{"complexity in a comment", "// if for while && || case\nlet x = 1;\n"},
			{"complexity in a string", "let s = \"if for while && || case\";\n"},
			{"optional chaining", "let v = foo?.bar;\n"},
			{"optional call and index", "fn?.();\nobj?.[\"k\"];\n"},
			{"nullish coalescing", "let v = a ?? b;\nlet w = c??d;\n"},
			{"nullish assignment", "a ??= b;\nc??=d;\n"},
			{"postfix at the first byte", "?.\n"},
			{"postfix after only whitespace", "   ??\nlet x = 1;\n"},
			{"postfix opening a line", "let x = 1;\n?.b;\n"},
			{"overlapping postfix", "a??.b;\n"},
			{"ternary is not a check", "let v = a ? b : c;\n"},
			{"else is not written with a brace", "if (a) {}\nelse{}\nelse {}\n"},
			{"while and switch want a space", "while(a) {}\nswitch(b) {}\nwhile (c) {}\nswitch (d) {}\n"},
			{"equality run", "a === b;\na !== b;\na == b;\na != b;\n"},
			{"pipes and ampersands run", "a || b;\na ||| b;\na && b;\na &&& b;\n"},
			{"crlf line endings", "let a = 1;\r\n// a comment\r\nlet b = 2;\r\n"},
			{"tabs and spaces", "\t\t// indented comment\n\t\tlet x = 1;\n"},
			{"quote at end of file", "let s = \""},
			{"backtick at end of file", "let s = `"},
			{"slash at end of file", "let x = 1; /"},
			{"star slash outside a comment", "let x = a */ b;\n"},
			{"division", "let r = a / b / c;\n"},
			{"division by a bracketed term", "let r = (a + b) / (c + d);\n"},
		} {
			fast, generic := countBothWays(t, "JavaScript", []byte(test.content))
			compareCounts(t, "JavaScript", test.name, fast, generic)
		}
	})
}

// The regular expression literal, which is the one place the counter is meant
// to disagree with the generic loop. Both answers are pinned, so the divergence
// is asserted rather than merely tolerated.
func TestJavaScriptRegexLiterals(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
		// want is what the counter must answer.
		lines, code, comment, blank, complexity int64
		// generic is what the generic loop answers, which for a divergence is
		// something else and for everything else is the same.
		genericCode, genericComment int64
	}{
		{
			name:    "a quote inside a pattern opens no string",
			content: "const re = /[\"']/;\n// a comment\nconst x = 1;\n",
			lines:   3, code: 2, comment: 1, blank: 0, complexity: 0,
			genericCode: 3, genericComment: 0,
		},
		{
			name:    "a comment opener inside a pattern opens no comment",
			content: "const re = /[/*]/;\nconst y = 2;\nconst z = 3;\n",
			lines:   3, code: 3, comment: 0, blank: 0, complexity: 0,
			genericCode: 1, genericComment: 2,
		},
		{
			// A pattern may not cross a line, and a line terminator is not a
			// thing a backslash can escape, so the slash opens no pattern at
			// all and the line behind the backslash is still a line. Stepping
			// over the newline as an escaped byte lost it, and the counter
			// answered two lines where the generic loop answered three.
			name:    "a backslash at the end of a line opens no pattern",
			content: "var re = /foo\\\nbar/;\nif (x) { y(); }\n",
			lines:   3, code: 3, comment: 0, blank: 0, complexity: 1,
			genericCode: 3, genericComment: 0,
		},
		{
			name:    "an escaped slash pair inside a pattern",
			content: "const re = /^https?:\\/\\//i;\nconst x = 1;\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
		{
			name:    "a complexity token inside a pattern is not a branch",
			content: "s.replace(/\\?.*/g, '');\n",
			lines:   1, code: 1, comment: 0, blank: 0, complexity: 0,
			genericCode: 1, genericComment: 0,
		},
		{
			name:    "a pattern opening a line",
			content: "const x = 1;\n/[\"']/.test(s);\nconst y = 2;\n",
			lines:   3, code: 3, comment: 0, blank: 0, complexity: 0,
			genericCode: 3, genericComment: 0,
		},
		{
			// A slash behind something that ends an expression divides, and a
			// division is never a pattern however the rest of the line reads.
			name:    "division is not a pattern",
			content: "const r = a / b;\nconst s = \"x\";\nconst t = 2;\n",
			lines:   3, code: 3, comment: 0, blank: 0, complexity: 0,
			genericCode: 3, genericComment: 0,
		},
		{
			name:    "division by a bracketed term is not a pattern",
			content: "const r = (a + b) / (c - d);\nconst s = 'y';\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
		{
			// The slash sits where a pattern could open but nothing closes one on
			// the line, so it was never a pattern and it reads the way the
			// generic loop reads it.
			name:    "an unclosed slash is not a pattern",
			content: "let x = 1 + / b;\nlet s = \"y\";\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
		{
			// An equality check is still counted on a line holding a division.
			name:    "a division does not hide a complexity check",
			content: "const r = a === 1 / b;\nconst s = \"y\";\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 1,
			genericCode: 2, genericComment: 0,
		},
		{
			name:    "a line comment still wins over a pattern",
			content: "const x = 1; // a comment with a \" in it\nconst y = 2;\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
		{
			name:    "a block comment still wins over a pattern",
			content: "const x = 1; /* a comment\nover lines */\nconst y = 2;\n",
			lines:   3, code: 2, comment: 1, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 1,
		},
		{
			name:    "a pattern after return",
			content: "function f() { return /[\"']/; }\nconst x = 1;\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
		{
			name:    "a character class holding a slash",
			content: "const re = /[a/b]/;\nconst x = 1;\n",
			lines:   2, code: 2, comment: 0, blank: 0, complexity: 0,
			genericCode: 2, genericComment: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fast, generic := countBothWays(t, "JavaScript", []byte(test.content))

			if fast.Lines != test.lines || fast.Code != test.code ||
				fast.Comment != test.comment || fast.Blank != test.blank ||
				fast.Complexity != test.complexity {
				t.Errorf("counter got lines=%d code=%d comment=%d blank=%d complexity=%d, want %d/%d/%d/%d/%d",
					fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity,
					test.lines, test.code, test.comment, test.blank, test.complexity)
			}

			if generic.Code != test.genericCode || generic.Comment != test.genericComment {
				t.Errorf("the generic loop got code=%d comment=%d, want %d/%d. "+
					"A divergence is pinned both ways, so the generic loop moving is as much a change as the counter moving",
					generic.Code, generic.Comment, test.genericCode, test.genericComment)
			}
		})
	}
}

// Every divergence on the list has a fixture, and the fixture asserts both
// answers. A divergence the counter no longer produces is a divergence that
// should have been deleted from the list.
func TestCounterDivergenceFixturesExist(t *testing.T) {
	ProcessConstants()

	for _, divergence := range counterDivergences() {
		t.Run(divergence.Case, func(t *testing.T) {
			path := filepath.Join("..", divergence.Fixture)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("no fixture for the divergence: %v", err)
			}

			if divergence.Reason == "" {
				t.Error("a divergence has to say what the counter does instead")
			}

			fast, generic := countBothWays(t, divergence.Language, content)
			if !countsDiffer(fast, generic) {
				t.Errorf("%s no longer diverges, so it should come off the list", divergence.Case)
			}
		})
	}
}

// Every JavaScript file of a real tree read both ways, the counts having to be
// identical once the one deliberate difference is taken out. Point
// SCC_DIFF_JS_CORPUS at a checkout of something large.
func TestJavaScriptCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpusRegex(t, "JavaScript", "SCC_DIFF_JS_CORPUS", ".js")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestJavaScriptCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestJavaScriptCounterAgreesOnTheCorpus(t)
}

// --no-complexity sets the global, which the generic loop reads by leaving the
// checks out of its trie. The counter has to stop counting them too, the
// postfix three included.
func TestJavaScriptCounterAgreesWithComplexityOff(t *testing.T) {
	ProcessConstants()

	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	content := []byte("if (a) { for (;;) {} }\nwhile (b) { switch (c) { case 1: break; } }\nlet v = x?.y ?? z;\n")
	fast, generic := countBothWays(t, "JavaScript", content)
	compareCounts(t, "JavaScript", "complexity off", fast, generic)

	if fast.Complexity != 0 {
		t.Errorf("expected no complexity counted, got %d", fast.Complexity)
	}
}

func benchmarkJavaScriptCorpus(b *testing.B, specialised bool) {
	b.Helper()
	ProcessConstants()

	corpus := os.Getenv("SCC_DIFF_JS_CORPUS")
	if corpus == "" {
		b.Skip("set SCC_DIFF_JS_CORPUS to a tree of real JavaScript")
	}

	var files [][]byte
	var total int64
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".js") || len(files) >= 400 {
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
		b.Skip("no JavaScript found in the corpus")
	}

	SpecialisedCounters = specialised
	defer func() { SpecialisedCounters = false }()

	b.SetBytes(total)
	b.ResetTimer()

	for b.Loop() {
		for _, content := range files {
			fileJob := FileJob{Language: "JavaScript", Content: content, Bytes: int64(len(content))}
			CountStats(&fileJob)
		}
	}
}

func BenchmarkCountStatsJavaScriptCorpusGeneric(b *testing.B) {
	benchmarkJavaScriptCorpus(b, false)
}

func BenchmarkCountStatsJavaScriptCorpusSpecialised(b *testing.B) {
	benchmarkJavaScriptCorpus(b, true)
}

// The same corpus scanned with complexity turned off, which is the ceiling the
// counting loop could reach if complexity cost nothing at all.
func BenchmarkCountStatsJavaScriptCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkJavaScriptCorpus(b, true)
}
