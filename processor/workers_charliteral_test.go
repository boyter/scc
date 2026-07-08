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
