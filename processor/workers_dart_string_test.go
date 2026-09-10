// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Dart strings are delimited by either quote character, and Dart has no
// character literal, so ' is only ever a string delimiter. languages.json
// modelled the double quote alone, which left the contents of every single
// quoted string open to the scanner: a /* in one opened a block comment, an
// odd " in one opened a string, and either ran to the end of the file.

func TestCountStatsDartSingleQuotedStringHidesBlockCommentOpener(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("const iconGlob = 'assets/*/icon.png';\n" +
		"\n" +
		"// Returns the icon path for a theme.\n" +
		"String iconFor(String theme) {\n" +
		"  return 'assets/$theme/icon.png';\n" +
		"}")

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

func TestCountStatsDartSingleQuotedStringHidesDoubleQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("const doubleQuote = '\"';\n" +
		"\n" +
		"// Fields that contain a comma are wrapped in double quotes.\n" +
		"String wrap(String value) {\n" +
		"  return doubleQuote + value + doubleQuote;\n" +
		"}")

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

func TestCountStatsDartSingleQuotedStringIsNotComplexity(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("void main() {\n" +
		"  const label = 'if you switch it while it is on';\n" +
		"  print(label);\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("Expected complexity 0 got %d", fileJob.Complexity)
	}
}

func TestCountStatsDartCommentsStillCount(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("void main() {\n" +
		"  /* block\n" +
		"     comment */\n" +
		"  // line comment\n" +
		"  print(\"hello\");\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}
