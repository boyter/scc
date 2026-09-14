// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes a Swift file is written in, each of which the two counters have to
// read the same way. The single byte ? is the new machinery, and the spellings
// that score nothing matter as much as the ones that score.
func TestSwiftCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "let x = 1"},
		{"blank lines", "\n\n\nstruct A { }\n\n"},
		{"line comment", "// a comment\nlet x = 1\n"},
		{"doc comment", "/// a doc comment\nlet x = 1\n"},
		{"block comment over lines", "/*\n a comment\n*/\nlet x = 1\n"},
		{"code then block comment", "let x = 1 /* a\n b */\nlet y = 2\n"},

		{"nested on one line", "/* outer /* inner */ still comment */\nlet x = 1\n"},
		{"nested over lines", "/* outer\n /* inner\n */\n still comment\n*/\nlet x = 1\n"},
		{"nested three deep", "/* a /* b /* c */ d */ e */\nlet x = 1\n"},
		{"nested never closed", "/* a /* b\nlet x = 1\n"},
		{"opener touching closer", "/*/\nlet x = 1\n"},

		{"string holding a comment", "let s = \"// not a comment\"\nlet x = 1\n"},
		{"escaped quote in a string", "let s = \"a \\\" b\"\n// a comment\n"},
		{"unterminated string", "let s = \"never closed\nlet x = 1\n"},
		{"unterminated block comment", "/* never closed\nlet x = 1\n"},

		// The ? is a prefix check of one byte, so it counts only where the byte
		// in front of it does not carry a word on. Every common spelling of an
		// optional has a letter in front of it and scores nothing.
		{"optional after an identifier", "let a: Int?\nlet b = foo?.bar\n"},
		{"optional after a keyword", "let a = try? f()\nlet b = x as? Y\n"},
		{"nil coalescing", "let a = x ?? y\n"},
		{"nil coalescing tight", "let a = x??y\n"},
		{"question after a bracket", "let a = f()?.b\nlet c = d[0]?.e\n"},
		{"question opening a line", "?\n? a\n"},
		{"question in a comment", "// a ?? b\nlet x = 1\n"},
		{"question in a string", "let s = \"a ?? b\"\n"},

		// Swift writes most of its keywords with a space and no bracket form.
		{"space forms", "switch a { }\nwhile b { }\nelse { }\nguard c else { }\n"},
		{"bracket forms score nothing", "switch(a) { }\nwhile(b) { }\n"},
		{"for and if carry both", "for a in b { }\nfor(a) { }\nif a { }\nif(a) { }\n"},
		{"catch and guard", "do { } catch { }\nguard let a = b else { return }\n"},
		{"complexity inside a word", "let iffy = 1\nlet elsewhere = 2\nlet guarded = 3\nlet catcher = 4\n"},
		{"complexity operators", "let b = a || c && d != e == f\n"},
		{"complexity in a comment", "// if for while && || guard catch\nlet x = 1\n"},
		{"complexity in a string", "let s = \"if for while && || guard catch\"\n"},
		{"keyword opening a line", "for a in b { }\nwhile c { }\nguard d else { return }\n"},

		{"crlf line endings", "struct A {\r\n// a comment\r\nlet x = 1\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tlet x = 1\n"},
		{"quote at end of file", "let s = \""},
		{"slash at end of file", "let x = 1 /"},
		{"question at end of file", "let x = a?"},
		{"star slash outside a comment", "let x = a */ b\n"},
	} {
		fast, generic := countBothWays(t, "Swift", []byte(test.content))
		compareCounts(t, "Swift", test.name, fast, generic)
	}
}

// Every Swift file of a real tree read both ways. Point SCC_DIFF_SWIFT_CORPUS
// at a checkout of something large.
func TestSwiftCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Swift", "SCC_DIFF_SWIFT_CORPUS", ".swift")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestSwiftCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestSwiftCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsSwiftCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Swift", "SCC_DIFF_SWIFT_CORPUS", ".swift", false)
}

func BenchmarkCountStatsSwiftCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Swift", "SCC_DIFF_SWIFT_CORPUS", ".swift", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsSwiftCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Swift", "SCC_DIFF_SWIFT_CORPUS", ".swift", true)
}
