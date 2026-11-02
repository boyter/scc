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

// Dart has multi-line strings, opened by three single or three double quotes
// and closed by the same three again. Read as one quote at a time, an
// apostrophe inside one closed it, and the closing quotes opened a string that
// turned the comment under it into code.
func TestCountStatsDartMultiLineString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("const usage = '''\n" +
		"Usage: fmt [options] <files>\n" +
		"Don't pass /* globs */ unquoted.\n" +
		"''';\n" +
		"\n" +
		"// Prints the usage text.\n" +
		"void printUsage() => print(usage);")

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

func TestCountStatsDartMultiLineDoubleQuotedString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("const badPath = \"\"\"\n" +
		"A path can't hold a \" character.\n" +
		"\"\"\";\n" +
		"\n" +
		"// Shown when the path is rejected.\n" +
		"void reject() => print(badPath);")

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

// An r in front of the quote makes a raw string, where a backslash is just a
// backslash. Read as an escape, the one at the end of r'C:\' kept the string
// open on to the next quote in the file.
func TestCountStatsDartRawString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("const windowsRoot = r'C:\\';\n" +
		"const separator = r\"\\\";\n" +
		"final quoted = RegExp(r'''^\\s*'(.*)'\\s*$''');\n" +
		"\n" +
		"// Whether the path starts at the root of a drive.\n" +
		"bool isRoot(String path) => path.startsWith(windowsRoot);\n" +
		"\n" +
		"String unquote(String s) => quoted.firstMatch(s)?.group(1) ?? s;")

	CountStats(&fileJob)

	if fileJob.Lines != 8 {
		t.Errorf("Expected 8 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blank got %d", fileJob.Blank)
	}
}

// Empty strings, adjacent strings and a name ending in r in front of a quote
// are read the same as before.
func TestCountStatsDartPlainStringsStillCount(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Dart"}
	fileJob.SetContent("String greet(String user) {\n" +
		"  if (user == '' || user == \"\") return 'Hello, ' 'world';\n" +
		"  /* the name */\n" +
		"  return 'Hello, $user'; // with a name\n" +
		"}")

	CountStats(&fileJob)

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 4 {
		t.Errorf("Expected complexity 4 got %d", fileJob.Complexity)
	}
}
