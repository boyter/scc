// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes a C# file is written in, each of which the two counters have to
// read the same way. The verbatim string is the one worth the most attention:
// it opens with two bytes and has no escape at all, which is the first time
// either has been true of a counter here.
func TestCsharpCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "int x = 1;"},
		{"blank lines", "\n\n\nclass A {}\n\n"},
		{"line comment", "// a comment\nint x = 1;\n"},
		{"documentation comment", "/// <summary>a</summary>\nint x = 1;\n"},
		{"line comment holding a block opener", "// /* not opened\nint x = 1;\n"},
		{"block comment over lines", "/*\n a comment\n*/\nint x = 1;\n"},
		{"block comment closed then code", "/* a */ int x = 1;\n"},
		{"code then block comment", "int x = 1; /* a\n b */\nint y = 2;\n"},
		{"block comment does not nest", "/* outer /* inner */ still code */\nint x = 1;\n"},
		{"close touching reopen", "/* one *//* two */\nint x = 1;\n"},

		{"plain string", "var s = \"a\";\n// a comment\n"},
		{"string holding a comment", "var s = \"// not a comment\";\nint x = 1;\n"},
		{"string holding a block opener", "var s = \"/* not a comment\";\nint x = 1;\n"},
		{"escaped quote in a string", "var s = \"a \\\" b\";\n// a comment\n"},
		{"escaped backslash then quote", "var s = \"a \\\\\";\n// a comment\n"},
		{"char literal", "char c = 'x';\n// a comment\n"},
		{"char literal holding a quote", "char c = '\"';\n// a comment\n"},
		{"char literal holding an escape", "char c = '\\\\';\n// a comment\n"},
		{"unterminated string", "var s = \"never closed\nint x = 1;\n"},
		{"unterminated block comment", "/* never closed\nint x = 1;\n"},

		// The verbatim string. Two bytes to open, no escape mechanism, and it
		// runs over a newline.
		{"verbatim string", "var s = @\"a\";\n// a comment\n"},
		{"verbatim holding a backslash", "var s = @\"C:\\path\";\n// a comment\n"},
		{"verbatim ending in a backslash", "var s = @\"C:\\\";\nint x = 1;\n"},
		{"verbatim holding a line comment", "var s = @\"// not a comment\";\nint x = 1;\n"},
		{"verbatim holding a block opener", "var s = @\"/* not a comment\";\nint x = 1;\n"},
		{"verbatim over lines", "var s = @\"line1\nline2\";\nint x = 1;\n"},
		{"verbatim over lines holding a comment", "var s = @\"line1\n// still the string\nline3\";\nint x = 1;\n"},
		// The generic loop closes at the first quote of a doubled pair and the
		// second one opens a new string, so the two happen to balance. It does
		// not understand the doubling, and neither does the counter.
		{"verbatim with a doubled quote", "var s = @\"a\"\"b\";\n// a comment\n"},
		{"verbatim with three doubled quotes", "var s = @\"a\"\"b\"\"c\"\"d\";\nint x = 1;\n"},
		{"verbatim unterminated", "var s = @\"never closed\nint x = 1;\n"},
		{"verbatim empty", "var s = @\"\";\n// a comment\n"},
		{"verbatim opening a line", "@\"a\";\nint x = 1;\n"},
		{"at with no quote", "var @class = 1;\n// a comment\n"},
		{"at at end of file", "var s = @"},
		{"at quote at end of file", "var s = @\""},
		{"interpolated verbatim", "var s = $@\"a{b}c\";\n// a comment\n"},
		{"verbatim after an identifier", "var s = x@\"a\";\n// a comment\n"},
		{"at inside a line comment", "// @\" not a string\nint x = 1;\n"},
		{"at inside a string", "var s = \"@\\\" still the string\";\nint x = 1;\n"},

		// C# spells switch, while and else with a space and nothing else, where
		// it spells for, if and foreach twice.
		{"every keyword", "if (a) { for (;;) { while (x) {} } }\nelse { switch (y) {} }\n"},
		{"foreach", "foreach (var a in b) { }\nforeach(var c in d) { }\n"},
		{"bracket forms", "if(a) { for(;;) {} }\n"},
		{"switch with a bracket is not a check", "switch(y) {}\n"},
		{"while with a bracket is not a check", "while(y) {}\n"},
		{"else with a brace is not a check", "else{}\n"},
		{"complexity inside a word", "int retry = 1;\nint iffy = 2;\nint elsewhere = 3;\nint foreachx = 4;\n"},
		{"for is not foreach", "int forever = 1;\n"},
		{"complexity operators", "bool b = a || c && d != e == f;\n"},
		{"complexity in a comment", "// if for while && ||\nint x = 1;\n"},
		{"complexity in a string", "var s = \"if for while && ||\";\n"},
		{"complexity in a verbatim string", "var s = @\"if for while && ||\";\n"},
		{"keyword opening a line", "for (;;) { }\nwhile (c) { }\nforeach (var a in b) { }\n"},

		{"crlf line endings", "class A {\r\n// a comment\r\nint x = 1;\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tint x = 1;\n"},
		{"quote at end of file", "var s = \""},
		{"slash at end of file", "int x = 1; /"},
		{"star slash outside a comment", "int x = a */ b;\n"},
	} {
		fast, generic := countBothWays(t, "C#", []byte(test.content))
		compareCounts(t, "C#", test.name, fast, generic)
	}
}

// Every C# file of a real tree read both ways. Point SCC_DIFF_CSHARP_CORPUS at
// a checkout of something large.
func TestCsharpCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "C#", "SCC_DIFF_CSHARP_CORPUS", ".cs")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestCsharpCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestCsharpCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsCsharpCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "C#", "SCC_DIFF_CSHARP_CORPUS", ".cs", false)
}

func BenchmarkCountStatsCsharpCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "C#", "SCC_DIFF_CSHARP_CORPUS", ".cs", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsCsharpCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "C#", "SCC_DIFF_CSHARP_CORPUS", ".cs", true)
}
