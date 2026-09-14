// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes a Scala file is written in, each of which the two counters have to
// read the same way. The four bracket checks are the new machinery and the
// arrow of a lambda is the case worth reading twice.
func TestScalaCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "val x = 1"},
		{"blank lines", "\n\n\nobject A\n\n"},
		{"line comment", "// a comment\nval x = 1\n"},
		{"block comment over lines", "/*\n a comment\n*/\nval x = 1\n"},
		{"code then block comment", "val x = 1 /* a\n b */\nval y = 2\n"},

		{"nested on one line", "/* outer /* inner */ still comment */\nval x = 1\n"},
		{"nested over lines", "/* outer\n /* inner\n */\n still comment\n*/\nval x = 1\n"},
		{"nested three deep", "/* a /* b /* c */ d */ e */\nval x = 1\n"},
		{"nested never closed", "/* a /* b\nval x = 1\n"},
		{"opener touching closer", "/*/\nval x = 1\n"},

		{"string holding a comment", "val s = \"// not a comment\"\nval x = 1\n"},
		{"escaped quote in a string", "val s = \"a \\\" b\"\n// a comment\n"},
		{"unterminated string", "val s = \"never closed\nval x = 1\n"},
		{"unterminated block comment", "/* never closed\nval x = 1\n"},

		// The four checks that are not words. Each is spelled with a trailing
		// space, so the tight forms score nothing.
		{"bracket checks spaced", "if (a > b) { }\nif (c < d) { }\n"},
		{"bracket checks with equals", "if (a >= b) { }\nif (c <= d) { }\n"},
		{"bracket checks tight", "if (a>b) { }\nif (c<d) { }\n"},
		{"bracket check after an identifier", "val x = a> b\nval y = c< d\n"},
		{"bracket check opening a line", "> a\n< b\n>= c\n<= d\n"},

		// A lambda arrow has an = in front of its >, which carries no word on,
		// and a space behind it, so the generic loop counts it and so must this.
		{"lambda arrow", "val f = (x: Int) => x + 1\n"},
		{"arrow with no space", "val f = (x: Int) =>x\n"},
		{"for comprehension arrow", "for (x <- xs) yield x\n"},

		{"every keyword", "if (a) { for (b <- c) { } }\nelse { while (d) { } }\n"},
		{"switch is in the table", "switch (a) { }\n"},
		{"complexity inside a word", "val retry = 1\nval iffy = 2\nval elsewhere = 3\n"},
		{"complexity operators", "val b = a || c && d != e == f\n"},
		{"complexity in a comment", "// if for while && || > <\nval x = 1\n"},
		{"complexity in a string", "val s = \"if for while && || > <\"\n"},
		{"keyword opening a line", "for (a <- b) { }\nwhile (c) { }\n"},

		{"crlf line endings", "object A {\r\n// a comment\r\nval x = 1\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tval x = 1\n"},
		{"quote at end of file", "val s = \""},
		{"slash at end of file", "val x = 1 /"},
		{"bracket at end of file", "val x = a >"},
		{"star slash outside a comment", "val x = a */ b\n"},
	} {
		fast, generic := countBothWays(t, "Scala", []byte(test.content))
		compareCounts(t, "Scala", test.name, fast, generic)
	}
}

// Every Scala file of a real tree read both ways. Point SCC_DIFF_SCALA_CORPUS
// at a checkout of something large.
func TestScalaCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Scala", "SCC_DIFF_SCALA_CORPUS", ".scala")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestScalaCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestScalaCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsScalaCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Scala", "SCC_DIFF_SCALA_CORPUS", ".scala", false)
}

func BenchmarkCountStatsScalaCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Scala", "SCC_DIFF_SCALA_CORPUS", ".scala", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsScalaCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Scala", "SCC_DIFF_SCALA_CORPUS", ".scala", true)
}
