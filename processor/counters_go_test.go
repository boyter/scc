// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes a Go file is written in, each of which the two counters have to
// read the same way. The raw string is the one worth the most attention: it has
// no escape mechanism, so a backslash in front of its closer does not carry it
// on, and it runs over a newline where the other two do not.
func TestGoCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "x := 1"},
		{"blank lines", "\n\n\nfunc a() {}\n\n"},
		{"line comment", "// a comment\nx := 1\n"},
		{"line comment holding a block opener", "// /* not opened\nx := 1\n"},
		{"block comment over lines", "/*\n a comment\n*/\nx := 1\n"},
		{"block comment closed then code", "/* a */ x := 1\n"},
		{"code then block comment", "x := 1 /* a\n b */\ny := 2\n"},
		{"block comment does not nest", "/* outer /* inner */ still code */\nx := 1\n"},
		{"close touching reopen", "/* one *//* two */\nx := 1\n"},

		{"plain string", "s := \"a\"\n// a comment\n"},
		{"string holding a comment", "s := \"// not a comment\"\nx := 1\n"},
		{"string holding a block opener", "s := \"/* not a comment\"\nx := 1\n"},
		{"escaped quote in a string", "s := \"a \\\" b\"\n// a comment\n"},
		{"escaped backslash then quote", "s := \"a \\\\\"\n// a comment\n"},
		{"rune literal", "r := 'x'\n// a comment\n"},
		{"rune literal holding a quote", "r := '\"'\n// a comment\n"},
		{"rune literal holding an escape", "r := '\\\\'\n// a comment\n"},
		{"unterminated string", "s := \"never closed\nx := 1\n"},
		{"unterminated block comment", "/* never closed\nx := 1\n"},

		// The raw string. One byte to open and close, no escape mechanism, and
		// it runs over a newline.
		{"raw string", "s := `a`\n// a comment\n"},
		{"raw holding a backslash", "s := `C:\\path`\n// a comment\n"},
		{"raw ending in a backslash", "s := `C:\\`\nx := 1\n"},
		{"raw holding a line comment", "s := `// not a comment`\nx := 1\n"},
		{"raw holding a block opener", "s := `/* not a comment`\nx := 1\n"},
		{"raw holding a quote", "s := `a \" b`\n// a comment\n"},
		{"raw over lines", "s := `line1\nline2`\nx := 1\n"},
		{"raw over lines holding a comment", "s := `line1\n// still the string\nline3`\nx := 1\n"},
		{"raw over many lines", "s := `a\n\n\nb`\nx := 1\n"},
		{"raw unterminated", "s := `never closed\nx := 1\n"},
		{"raw empty", "s := ``\n// a comment\n"},
		{"raw opening a line", "`a`\nx := 1\n"},
		{"backtick at end of file", "s := `"},
		{"escaped backtick opens nothing", "s := \\`\nx := 1\n"},
		{"backtick inside a line comment", "// ` not a string\nx := 1\n"},
		{"backtick inside a string", "s := \"` still the string\"\nx := 1\n"},
		{"raw holding a backtick is impossible", "s := `a`+\"`\"+`b`\nx := 1\n"},

		// go and select are Go's own, and select shares the l of else.
		{"go statement", "go f()\nx := 1\n"},
		{"go inside a word", "gopher := 1\nlogo := 2\n"},
		{"select", "select {\ncase <-c:\n}\n"},
		{"select with a brace", "select{}\n"},
		{"select inside a word", "selected := 1\nreselect := 2\n"},
		{"else and select together", "if a {\n} else {\n}\nselect {\n}\n"},
		{"else with a brace", "} else{\n}\n"},
		{"every keyword", "if a { for i := range b { } }\nswitch c {\n}\ngo d()\nselect {\n}\n"},
		{"brace forms", "for{}\nswitch{}\nelse{}\nselect{}\n"},
		{"bracket forms", "for(a)\nif(b)\nswitch(c)\n"},
		{"complexity inside a word", "retry := 1\niffy := 2\nelsewhere := 3\nselector := 4\n"},
		{"complexity operators", "b := a || c && d != e == f\n"},
		{"complexity in a comment", "// if for switch && || go select\nx := 1\n"},
		{"complexity in a string", "s := \"if for switch && || go select\"\n"},
		{"complexity in a raw string", "s := `if for switch && || go select`\n"},
		{"keyword opening a line", "for i := range b { }\ngo f()\nselect {\n}\n"},

		{"crlf line endings", "func a() {\r\n// a comment\r\nx := 1\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tx := 1\n"},
		{"quote at end of file", "s := \""},
		{"slash at end of file", "x := 1 /"},
		{"star slash outside a comment", "x := a */ b\n"},
	} {
		fast, generic := countBothWays(t, "Go", []byte(test.content))
		compareCounts(t, "Go", test.name, fast, generic)
	}
}

// Every Go file of a real tree read both ways. Point SCC_DIFF_GO_CORPUS at a
// checkout of something large.
func TestGoCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Go", "SCC_DIFF_GO_CORPUS", ".go")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestGoCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestGoCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsGoCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Go", "SCC_DIFF_GO_CORPUS", ".go", false)
}

func BenchmarkCountStatsGoCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Go", "SCC_DIFF_GO_CORPUS", ".go", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsGoCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Go", "SCC_DIFF_GO_CORPUS", ".go", true)
}
