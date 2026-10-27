// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Pkl strings work like Swift's. A string opened by one or more # before the
// quote is closed by a quote followed by the same number of #, and a backslash
// on its own is an ordinary character in it. languages.json had these the
// wrong way round, opening on "# and closing on #", so a raw string ran on
// past a backslash or a lone quote in it. It also had """ ignore escapes,
// when Pkl's multi-line strings keep them.

func TestCountStatsPklRawStringEndingInBackslash(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("windowsRoot = #\"C:\\\"#\n" +
		"\n" +
		"// where the app is installed\n" +
		"installDir = #\"C:\\Program Files\\App\"#\n" +
		"\n" +
		"/* used by the tests */\n" +
		"fixtures = \"Fixtures\"")

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blanks got %d", fileJob.Blank)
	}
}

func TestCountStatsPklRawStringHoldingQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("quotedField = Regex(#\"\"[^\"]*\"\"#)\n" +
		"\n" +
		"// fields are separated by commas\n" +
		"separator = \",\"")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

func TestCountStatsPklRawStringWithTwoPounds(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("greeting = ##\"say \"#hi\"# \\\"##\n" +
		"\n" +
		"// shown on start\n" +
		"name = \"pkl\"")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// A multi-line string keeps the backslash escape, so \""" inside one doesn't
// close it.
func TestCountStatsPklMultiLineStringEscapedQuotes(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("hint = \"\"\"\n" +
		"  Wrap it in \\\"\"\" quotes.\n" +
		"  \"\"\"\n" +
		"\n" +
		"// the name shown in the banner\n" +
		"name = \"pkl\"")

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// In a multi-line raw string only """# closes it, so a plain """ in it is
// text.
func TestCountStatsPklMultiLineRawString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("hint = #\"\"\"\n" +
		"  A multi-line string ends with \"\"\" on its own line.\n" +
		"  \"\"\"#\n" +
		"\n" +
		"// the name shown in the banner\n" +
		"name = \"pkl\"")

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

func TestCountStatsPklStringEscapesStillWork(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Pkl"}
	fileJob.SetContent("message = \"say \\\"hi\\\" // not a comment\"\n" +
		"\n" +
		"// a real comment\n" +
		"count = 1")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}
