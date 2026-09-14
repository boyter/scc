// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The shapes a Kotlin file is written in, each of which the two counters have
// to read the same way. The nested block comment is the one worth the most
// attention: no corpus checked out for this work holds a single one, so these
// are the only cover M8 has.
func TestKotlinCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
	}{
		{"empty", ""},
		{"one newline", "\n"},
		{"no trailing newline", "val x = 1"},
		{"blank lines", "\n\n\nclass A\n\n"},
		{"line comment", "// a comment\nval x = 1\n"},
		{"line comment holding a block opener", "// /* not opened\nval x = 1\n"},
		{"block comment over lines", "/*\n a comment\n*/\nval x = 1\n"},
		{"block comment closed then code", "/* a */ val x = 1\n"},
		{"code then block comment", "val x = 1 /* a\n b */\nval y = 2\n"},

		// Kotlin nests, which is the whole of M8. Without it the first closer
		// ends the comment and the tail of the line is read as code.
		{"nested on one line", "/* outer /* inner */ still comment */\nval x = 1\n"},
		{"nested over lines", "/* outer\n /* inner\n */\n still comment\n*/\nval x = 1\n"},
		{"nested three deep", "/* a /* b /* c */ d */ e */\nval x = 1\n"},
		{"nested after code", "val x = 1 /* a /* b */ c */ ; val y = 2\n"},
		{"nested never closed", "/* a /* b\nval x = 1\n"},
		{"opener touching closer", "/*/\nval x = 1\n"},
		{"close touching reopen", "/* one *//* two */\nval x = 1\n"},

		{"string holding a comment", "val s = \"// not a comment\"\nval x = 1\n"},
		{"string holding a block opener", "val s = \"/* not a comment\"\nval x = 1\n"},
		{"escaped quote in a string", "val s = \"a \\\" b\"\n// a comment\n"},
		{"escaped backslash then quote", "val s = \"a \\\\\"\n// a comment\n"},
		{"unterminated string", "val s = \"never closed\nval x = 1\n"},
		{"unterminated block comment", "/* never closed\nval x = 1\n"},

		// when is Kotlin's multi-way branch, where Java writes switch, and it
		// shares its first two bytes with while.
		{"when and while", "when (x) { else -> 1 }\nwhile (y) { }\n"},
		{"when with a bracket", "when(x) { }\nwhile(y) { }\n"},
		{"when inside a word", "val whenever = 1\nval whiled = 2\n"},
		{"every keyword", "if (a) { for (b in c) { } }\nelse { try { } catch (e: E) { } finally { } }\n"},
		{"brace forms", "else{ }\ntry{ }\nfinally{ }\n"},
		{"complexity inside a word", "val retry = 1\nval iffy = 2\nval elsewhere = 3\nval finallyx = 4\n"},
		{"complexity operators", "val b = a || c && d != e == f\n"},
		{"complexity in a comment", "// if for while && ||\nval x = 1\n"},
		{"complexity in a string", "val s = \"if for while && ||\"\n"},
		{"keyword opening a line", "for (a in b) { }\nwhile (c) { }\nfinally { }\n"},

		{"crlf line endings", "class A {\r\n// a comment\r\nval x = 1\r\n}\r\n"},
		{"tabs and spaces", "\t\t// indented comment\n\t\tval x = 1\n"},
		{"quote at end of file", "val s = \""},
		{"slash at end of file", "val x = 1 /"},
		{"star slash outside a comment", "val x = a */ b\n"},
	} {
		fast, generic := countBothWays(t, "Kotlin", []byte(test.content))
		compareCounts(t, "Kotlin", test.name, fast, generic)
	}
}

// Every Kotlin file of a real tree read both ways. Point SCC_DIFF_KOTLIN_CORPUS
// at a checkout of something large.
func TestKotlinCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "Kotlin", "SCC_DIFF_KOTLIN_CORPUS", ".kt")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestKotlinCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestKotlinCounterAgreesOnTheCorpus(t)
}

func BenchmarkCountStatsKotlinCorpusGeneric(b *testing.B) {
	benchmarkCorpus(b, "Kotlin", "SCC_DIFF_KOTLIN_CORPUS", ".kt", false)
}

func BenchmarkCountStatsKotlinCorpusSpecialised(b *testing.B) {
	benchmarkCorpus(b, "Kotlin", "SCC_DIFF_KOTLIN_CORPUS", ".kt", true)
}

// The same corpus with complexity off, which is the ceiling the counting loop
// could reach if complexity cost nothing at all.
func BenchmarkCountStatsKotlinCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCorpus(b, "Kotlin", "SCC_DIFF_KOTLIN_CORPUS", ".kt", true)
}
