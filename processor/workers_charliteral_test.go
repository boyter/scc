// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Follow-up to https://github.com/boyter/scc/issues/175 (dwmcrobb's comment):
// a character/rune literal containing a double quote, e.g. '"', was not modelled
// as a string, so the embedded " opened a phantom string that swallowed the
// comment which followed. Languages where ' is unambiguously a char/rune literal
// (no ' digit separator, no lifetime/quote meaning) get a plain ' -> ' quote.
// C and C++ are deliberately excluded for now because ' is also a digit
// separator there (1'000'000) and needs disambiguation.

func TestCountStatsCharLiteralGo(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go"}
	fileJob.SetContent("package main\n" +
		"func main() {\n" +
		"\tc := '\"'\n" +
		"\t/* block\n" +
		"\t   comment */\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}

func TestCountStatsCharLiteralJava(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Java"}
	fileJob.SetContent("class T {\n" +
		"  void m() {\n" +
		"    char c = '\"';\n" +
		"    /* block\n" +
		"       comment */\n" +
		"  }\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}

func TestCountStatsCharLiteralCSharp(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "C#"}
	fileJob.SetContent("class T {\n" +
		"  void M() {\n" +
		"    char c = '\"';\n" +
		"    /* block\n" +
		"       comment */\n" +
		"  }\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}

func TestCountStatsByteCharLiteralRust(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Rust"}
	fileJob.SetContent("fn is_quote(byte: u8) -> bool {\n" +
		"    matches!(byte, b'\"' | b'\\\\')\n" +
		"}\n" +
		"\n" +
		"// This comment must remain visible.\n" +
		"fn classify(value: bool) {\n" +
		"    let _ = b'\\'';\n" +
		"    if value {\n" +
		"        println!(\"yes\");\n" +
		"    }\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 9 {
		t.Errorf("Expected 9 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
	if fileJob.Complexity != 1 {
		t.Errorf("Expected complexity 1 got %d", fileJob.Complexity)
	}
}

func TestCountStatsCharLiteralKotlin(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Kotlin"}
	fileJob.SetContent("fun isQuote(c: Char): Boolean {\n" +
		"    if (c == '\"') {\n" +
		"        return true\n" +
		"    }\n" +
		"    // a comment\n" +
		"    return c == '\\''\n" +
		"}\n" +
		"\n" +
		"/* block\n" +
		"   comment */")

	CountStats(&fileJob)

	if fileJob.Code != 6 {
		t.Errorf("Expected 6 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// Kotlin names a function in backticks to give it spaces, so ' must not open a
// character literal inside one.
func TestCountStatsBacktickNameKotlin(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Kotlin"}
	fileJob.SetContent("class FooTest {\n" +
		"    @Test\n" +
		"    fun `it doesn't crash`() {\n" +
		"        // a comment\n" +
		"\n" +
		"        assertTrue(true)\n" +
		"    }\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 6 {
		t.Errorf("Expected 6 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// Elixir writes a character's code point as ?c, so ?" and ?' are code. Their
// quote opened a string that ran on to the next one and swallowed the
// comments in between.
func TestCountStatsCharLiteralQuoteElixir(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Elixir"}
	fileJob.SetContent("def quote?(c), do: c == ?\" or c == ?\\\"\n" +
		"# a comment\n" +
		"def tick?(c), do: c == ?'\n" +
		"# another\n" +
		"def s, do: \"#{1}\"\n")

	CountStats(&fileJob)

	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}
