// SPDX-License-Identifier: MIT

package processor

import "testing"

// The shapes a TypeScript file is written in. TypeScript is JavaScript with two
// more equality checks, so the rows that matter most here are the ones holding
// === and !== , which overlap == and != and are the one thing this counter does
// differently from the JavaScript one.
func TestTypeScriptCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	withoutRegexLiterals(t, func() {
		for _, test := range []struct {
			name    string
			content string
		}{
			{"empty", ""},
			{"one newline", "\n"},
			{"no trailing newline", "let x: number = 1;"},
			{"line comment", "// a comment\nlet x = 1;\n"},
			{"block comment over lines", "/*\n a comment\n*/\nlet x = 1;\n"},
			{"block comment does not nest", "/* outer /* inner */ still code */\nlet x = 1;\n"},
			{"template literal over lines", "const s = `a\nb`;\nlet x = 1;\n"},
			{"template literal holding a comment", "const s = `// not a comment`;\nlet x = 1;\n"},
			{"template literal holding interpolation", "const s = `a ${b + 1} c`;\nlet x = 1;\n"},
			{"string holding a comment", "const s = \"// not a comment\";\nlet x = 1;\n"},
			{"escaped quote", "const s = \"a \\\" b\";\n// a comment\n"},
			{"unterminated string", "const s = \"never closed\nlet x = 1;\n"},
			{"unterminated block comment", "/* never closed\nlet x = 1;\n"},
			{"strict equality", "if (a === b) {}\n"},
			{"strict inequality", "if (a !== b) {}\n"},
			{"loose equality", "if (a == b) {}\n"},
			{"loose inequality", "if (a != b) {}\n"},
			{"both equality forms on a line", "const v = a === b && c !== d;\n"},
			{"equality with no space in front", "const v = a=== b;\n"},
			{"inequality with no space in front", "const v = a!== b;\n"},
			{"a run of four equals", "const v = a ==== b;\n"},
			{"a run of five equals", "const v = a ===== b;\n"},
			{"equality at the start of a line", "const v = a\n=== b;\n"},
			{"strict equality at the start of a line", "const v = a\n!== b;\n"},
			{"equality at the end of the file", "const v = a === "},
			{"equality with nothing after it", "a ==="},
			{"assignment is not a check", "let x = 1;\nlet y = 2;\n"},
			{"arrow is not a check", "const f = () => 1;\n"},
			{"type annotations", "function f(a: string, b?: number): void {}\n"},
			{"generics look like comparison", "const m = new Map<string, number>();\n"},
			{"optional chaining", "const v = a?.b;\n"},
			{"nullish coalescing", "const v = a ?? b;\n"},
			{"nullish assignment", "a ??= b;\n"},
			{"non null assertion", "const v = a!.b;\n"},
			{"complexity tokens", "if (a) { for (;;) { while (x) {} } }\nelse { switch (y) { case 1: break; } }\n"},
			{"complexity inside a word", "let retry = 1;\nlet iffy = 2;\nlet switcher = 3;\n"},
			{"complexity in a comment", "// if for while && ||\nlet x = 1;\n"},
			{"crlf", "let a = 1;\r\n// a comment\r\nlet b = 2;\r\n"},
			{"tabs and spaces", "\t\t// indented\n\t\tlet x = 1;\n"},
			{"quote at end of file", "const s = \""},
			{"slash at end of file", "let x = 1; /"},
			{"decorator", "@Component({})\nclass A {}\n"},
		} {
			fast, generic := countBothWays(t, "TypeScript", []byte(test.content))
			compareCounts(t, "TypeScript", test.name, fast, generic)
		}
	})
}

// Every TypeScript file of a real tree read both ways, the counts having to be
// identical once the regex fix is taken out. Point SCC_DIFF_TS_CORPUS at a
// checkout of something large.
func TestTypeScriptCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpusRegex(t, "TypeScript", "SCC_DIFF_TS_CORPUS", ".ts")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestTypeScriptCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestTypeScriptCounterAgreesOnTheCorpus(t)
}

// --no-complexity sets the global, which the generic loop reads by leaving the
// checks out of its trie. The counter has to stop counting them too, the
// postfix three and the equality family included.
func TestTypeScriptCounterAgreesWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	content := []byte("if (a === b) { for (;;) {} }\nwhile (c !== d) { switch (e) { case 1: break; } }\nconst v = x?.y ?? z;\n")
	fast, generic := countBothWays(t, "TypeScript", content)
	compareCounts(t, "TypeScript", "complexity off", fast, generic)

	if fast.Complexity != 0 {
		t.Errorf("expected no complexity counted, got %d", fast.Complexity)
	}
}

func BenchmarkCountStatsTypeScriptCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "TypeScript", "SCC_DIFF_TS_CORPUS", ".ts", false)
}

func BenchmarkCountStatsTypeScriptCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "TypeScript", "SCC_DIFF_TS_CORPUS", ".ts", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsTypeScriptCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "TypeScript", "SCC_DIFF_TS_CORPUS", ".ts", true)
}
