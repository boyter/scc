// SPDX-License-Identifier: MIT

package processor

import "testing"

// The shapes a C++ file is written in. The raw string is the whole of what C++
// adds over C, and it is the one string form in any counted language that both
// names its own closer and survives a newline with no splice behind it.
func TestCppCounterAgreesOnHandWrittenFiles(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"C++", "C++ Header"} {
		for _, test := range []struct {
			name    string
			content string
		}{
			{"empty", ""},
			{"no trailing newline", "int x = 1;"},
			{"line comment", "// a comment\nint x = 1;\n"},
			{"block comment over lines", "/*\n a comment\n*/\nint x = 1;\n"},
			{"block comment does not nest", "/* outer /* inner */ still code */\nint x = 1;\n"},
			{"plain string", "const char *s = \"abc\";\nint x = 1;\n"},
			{"escaped quote", "const char *s = \"a \\\" b\";\n// a comment\n"},
			{"escaped backslash then quote", "const char *s = \"a \\\\\";\n// a comment\n"},
			{"unterminated plain string ends at the newline", "const char *s = \"abc\nint x = 1;\n// c\n"},
			{"plain string carried by a splice", "const char *s = \"abc\\\ndef\";\n// c\n"},

			// M6, the delimited raw string.
			{"raw string, empty delimiter", "const char *s = R\"(abc)\";\n// c\nint x = 1;\n"},
			{"raw string with a delimiter", "const char *s = R\"tag(abc)tag\";\n// c\n"},
			{"raw string holding a quote", "const char *s = R\"(a \"quote\" here)\";\n// c\nint x = 1;\n"},
			{"raw string holding its own closer", "const char *s = R\"tag(a )\" not end)tag\";\n// c\nint x = 1;\n"},
			{"raw string holding a line comment", "const char *s = R\"(// not a comment)\";\nint x = 1;\n"},
			{"raw string holding a block opener", "const char *s = R\"(/* not a block)\";\nint x = 1;\n"},
			{"raw string over lines", "const char *s = R\"(line1\nline2)\";\n// c\n"},
			{"raw string over three lines", "const char *s = R\"(a\nb\nc)\";\n// c\n"},
			{"raw string wrapping blank lines", "R\"(\n\n\n)\";\nint x = 1;\n"},
			{"raw string holding a backslash at a line end", "const char *s = R\"(a\\\nb)\";\n// c\n"},
			{"raw string holding complexity tokens", "const char *s = R\"(if (a) { for (;;) {} })\";\nint x = 1;\n"},

			// Every encoding prefix, since each is its own quote in the database.
			{"u8R prefix", "const char *s = u8R\"(x)\";\nint y = 1;\n"},
			{"uR prefix", "const char *s = uR\"(x)\";\nint y = 1;\n"},
			{"UR prefix", "const char *s = UR\"(x)\";\nint y = 1;\n"},
			{"LR prefix", "const char *s = LR\"(x)\";\nint y = 1;\n"},
			{"prefix behind an identifier", "const char *s = fooR\"(x)\";\nint y = 1;\n"},

			// The delimiter is read out of the file, so its bounds are the one
			// place a malformed input can reach.
			{"delimiter at the sixteen byte limit", "const char *s = R\"aaaaaaaaaaaaaaaa(x)aaaaaaaaaaaaaaaa\";\nint x = 1;\n"},
			{"delimiter one past the limit", "const char *s = R\"aaaaaaaaaaaaaaaaa(x)aaaaaaaaaaaaaaaaa\";\nint x = 1;\n"},
			{"delimiter holding a space is no delimiter", "const char *s = R\" bad;\nint x = 1;\n// c\n"},
			{"delimiter holding a bracket is no delimiter", "const char *s = R\")\";\n// c\n"},
			{"raw opener at the end of the file", "const char *s = R\""},
			{"raw opener and bracket at the end of the file", "const char *s = R\"("},
			{"bare R and quote", "R\"\n"},
			{"raw string never closed", "const char *s = R\"(abc\nint x = 1;\n"},

			// The splice, which C++ has and which the raw string must survive.
			{"splice in a line comment", "// continues \\\n   onto the next line\nint x = 1;\n"},
			{"macro continued over lines", "#define X(a) \\\n\tdo { a; } while (0)\nint x = 1;\n"},
			{"splice after a raw string", "const char *s = R\"(x)\"; // c \\\nstill comment\nint y = 1;\n"},

			// The checks C++ has that C does not.
			{"try and catch", "try { f(); } catch (E e) {}\nint x = 1;\n"},
			{"complexity tokens", "if (a) { for (;;) { while (x) {} } }\nelse { switch (y) {} }\n"},
			{"complexity inside a word", "int retry = 1;\nint iffy = 2;\nint elsewhere = 3;\nint dispatch = 4;\n"},
			{"complexity operators", "int b = a || c && d != e == f;\n"},
			{"crlf", "int a = 1;\r\n// a comment\r\nint b = 2;\r\n"},
			{"crlf with a splice", "// carries \\\r\n   on\r\nint x = 1;\r\n"},
			{"tabs and spaces", "\t\t// indented comment\n\t\tint x = 1;\n"},
		} {
			fast, generic := countBothWays(t, language, []byte(test.content))
			compareCounts(t, language, test.name, fast, generic)
		}
	}
}

// Every C++ and C++ Header file of a real tree read both ways, the counts having
// to be identical.
func TestCppCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "C++", "SCC_DIFF_CPP_CORPUS", ".cc")
}

func TestCppCounterAgreesOnTheCorpusCpp(t *testing.T) {
	diffCorpus(t, "C++", "SCC_DIFF_CPP_CORPUS", ".cpp")
}

func TestCppHeaderCounterAgreesOnTheCorpus(t *testing.T) {
	diffCorpus(t, "C++ Header", "SCC_DIFF_CPP_CORPUS", ".hpp")
}

// The counter scans with a smaller table when complexity is off, which is a
// second path over the same files.
func TestCppCounterAgreesOnTheCorpusWithComplexityOff(t *testing.T) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	TestCppCounterAgreesOnTheCorpus(t)
	TestCppCounterAgreesOnTheCorpusCpp(t)
	TestCppHeaderCounterAgreesOnTheCorpus(t)
}

func benchmarkCppCorpus(b *testing.B, specialised bool) {
	benchmarkCorpus(b, "C++", "SCC_DIFF_CPP_CORPUS", ".cc", specialised)
}

func BenchmarkCountStatsCppCorpusGeneric(b *testing.B)     { benchmarkCppCorpus(b, false) }
func BenchmarkCountStatsCppCorpusSpecialised(b *testing.B) { benchmarkCppCorpus(b, true) }

// The same corpus with complexity turned off, which is the ceiling the counting
// loop could reach if complexity cost nothing at all.
func BenchmarkCountStatsCppCorpusNoComplexity(b *testing.B) {
	Complexity = true
	ProcessConstants()
	defer func() {
		Complexity = false
		ProcessConstants()
	}()

	benchmarkCppCorpus(b, true)
}
