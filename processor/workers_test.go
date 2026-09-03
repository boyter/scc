// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

func (job *FileJob) SetContent(content string) {
	job.Content = []byte(content)
	job.Bytes = int64(len(job.Content))
}

func TestIsWhitespace(t *testing.T) {
	if !isWhitespace(' ') {
		t.Errorf("Expected to be true")
	}
}

func TestIsBinaryTrue(t *testing.T) {
	DisableCheckBinary = false

	if !isBinary(0, 0) {
		t.Errorf("Expected to be true")
	}
}

func TestIsBinaryDisableCheck(t *testing.T) {
	DisableCheckBinary = true

	if isBinary(0, 0) {
		t.Errorf("Expected to be false")
	}
}

func TestCountStatsLines(t *testing.T) {
	Trace = false
	Debug = false
	Verbose = false

	fileJob := FileJob{
		Content: []byte(""),
		Lines:   0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Lines != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Lines)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Lines = 0
	fileJob.SetContent("a")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.SetContent("a\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	// tokei counts this as 1 because it's still on a single line unless something follows
	// the newline it's still 1 line
	fileJob.Lines = 0
	fileJob.SetContent("1\n")
	CountStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.SetContent("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Lines != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Lines)
	}

	fileJob.Lines = 0
	fileJob.SetContent("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Lines != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Lines)
	}

	content := ""
	for i := range 5000 {
		content += "a\n"
		fileJob.Lines = 0
		fileJob.SetContent(content)
		CountStats(&fileJob)
		if fileJob.Lines != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Lines)
		}
	}
}

func TestCountStatsCode(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Code:    0,
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	CountStats(&fileJob)
	if fileJob.Code != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Code)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Code = 0
	fileJob.SetContent("a")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.SetContent("i++ # comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.SetContent("i++ // comment")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.SetContent("a\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	// tokei counts this as 1 because it's still on a single line unless something follows
	// the newline it's still 1 line
	fileJob.Code = 0
	fileJob.SetContent("1\n")
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.SetContent("1\n2\n")
	CountStats(&fileJob)
	if fileJob.Code != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Code)
	}

	fileJob.Code = 0
	fileJob.SetContent("1\n2\n3")
	CountStats(&fileJob)
	if fileJob.Code != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Code)
	}

	content := ""
	for i := range 100 {
		content += "a\n"
		fileJob.Code = 0
		fileJob.SetContent(content)
		CountStats(&fileJob)
		if fileJob.Code != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Code)
		}
	}
}

// A quote delimiter is written in languages.json as "\"" and JSON already
// escapes it there, so a delimiter carrying a literal backslash in front of
// its quote is one that was escaped twice and matches nothing a real file
// holds. Zig's "\\" is the one true backslash delimiter, being the opener of
// its multiline string literal.
func TestLanguageQuotesAreNotDoubleEscaped(t *testing.T) {
	ProcessConstants()

	for name, language := range languageDatabase {
		for _, quote := range language.Quotes {
			for _, delimiter := range []string{quote.Start, quote.End} {
				if strings.Contains(delimiter, `\"`) {
					t.Errorf("%s: quote delimiter %q holds a literal backslash, so it never matches", name, delimiter)
				}
			}
		}
	}
}

// The double escaped delimiters above left these languages with no working
// double quoted string at all. It only shows on a string running over more
// than one line, because a single line one sits on a line that is code
// whether the string was seen or not. Here the middle line opens with the
// language's own comment token while inside the string, so a language that
// missed the string reads it as a comment.
func TestCountStatsMultiLineStringHoldingACommentToken(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language    string
		lineComment string
	}{
		{"Shell", "#"},
		{"Zsh", "#"},
		{"BASH", "#"},
		{"Korn Shell", "#"},
		{"Fish", "#"},
		{"Ruby", "#"},
		{"Rakefile", "#"},
		{"Gemfile", "#"},
		{"Crystal", "#"},
		{"TOML", "#"},
		{"Julia", "#"},
		{"Nim", "#"},
		{"Dockerfile", "#"},
		{"TCL", "#"},
		{"Puppet", "#"},
		{"CoffeeScript", "#"},
		{"Cython", "#"},
		{"Just", "#"},
		{"Scons", "#"},
		{"Expect", "#"},
		{"Luna", "#"},
		{"Raku", "#"},
		{"Zig", "//"},
		{"Typst", "//"},
		{"Pony", "//"},
		{"Idris", "--"},
		{"Assembly", ";"},
		{"Arturo", ";"},
		{"Basic", "'"},
		{"Visual Basic", "'"},
		{"Visual Basic for Applications", "'"},
		{"Softbridge Basic", "'"},
		{"Fortran Modern", "!"},
	} {
		fileJob := FileJob{Language: test.language}
		fileJob.SetContent("x = \"line one\n" + test.lineComment + " still inside the string\nend\"\ny = 2\n")

		CountStats(&fileJob)

		if fileJob.Lines != 4 {
			t.Errorf("%s: expected 4 lines got %d", test.language, fileJob.Lines)
		}

		if fileJob.Code != 4 {
			t.Errorf("%s: expected 4 code got %d", test.language, fileJob.Code)
		}

		if fileJob.Comment != 0 {
			t.Errorf("%s: expected 0 comment got %d", test.language, fileJob.Comment)
		}

		if fileJob.Blank != 0 {
			t.Errorf("%s: expected 0 blank got %d", test.language, fileJob.Blank)
		}
	}
}

// Vim Script is the one language in the sweep above that must not be given a
// working double quoted string, because the double quote is what opens its
// comments. A string delimiter is matched ahead of a comment one, so handing
// it the quote turns every comment in a vimrc into code.
func TestCountStatsVimScriptDoubleQuoteOpensAComment(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{Language: "Vim Script"}
	fileJob.SetContent("\" a vim comment\nlet x = 1\n\" another comment\nlet y = 2\n")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comment got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank got %d", fileJob.Blank)
	}
}

// Standard SQL escapes a quote by doubling it and the backslash is an ordinary
// character, so a path ending in one closes the string it sits in. Reading the
// backslash as an escape leaves the string open and takes the comment under it
// along with everything below. Oracle's PL/SQL reads the backslash the same
// way. MySQL is the dialect that does escape with it, and it loses here.
func TestCountStatsSQLBackslashDoesNotEscapeTheClosingQuote(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"SQL", "PL/SQL"} {
		fileJob := FileJob{Language: language}
		fileJob.SetContent("UPDATE t SET path = 'C:\\';\n-- a real comment\nSELECT 2;\n")

		CountStats(&fileJob)

		if fileJob.Lines != 3 {
			t.Errorf("%s: expected 3 lines got %d", language, fileJob.Lines)
		}

		if fileJob.Code != 2 {
			t.Errorf("%s: expected 2 code got %d", language, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s: expected 1 comment got %d", language, fileJob.Comment)
		}

		if fileJob.Blank != 0 {
			t.Errorf("%s: expected 0 blank got %d", language, fileJob.Blank)
		}
	}
}

// Groovy writes a string four ways and scc knew one of them, so a comment
// opener inside any of the other three opened a comment. The single quoted
// string is the idiomatic one and the triple quoted ones run over lines, which
// is what lets an unclosed block comment take the rest of the file.
func TestCountStatsGroovyStringForms(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"Groovy", "Gradle"} {
		for _, quote := range []string{"'", `"`} {
			fileJob := FileJob{Language: language}
			fileJob.SetContent("def a = " + quote + "/* not a comment" + quote + "\ndef b = 1\ndef c = 2\n")

			CountStats(&fileJob)

			if fileJob.Lines != 3 {
				t.Errorf("%s %s: expected 3 lines got %d", language, quote, fileJob.Lines)
			}

			if fileJob.Code != 3 {
				t.Errorf("%s %s: expected 3 code got %d", language, quote, fileJob.Code)
			}

			if fileJob.Comment != 0 {
				t.Errorf("%s %s: expected 0 comment got %d", language, quote, fileJob.Comment)
			}
		}

		for _, quote := range []string{"'''", `"""`} {
			fileJob := FileJob{Language: language}
			fileJob.SetContent("def a = " + quote + "\n// this is text inside the string\n/* and so is this\n" + quote + "\ndef b = 1\n")

			CountStats(&fileJob)

			if fileJob.Lines != 5 {
				t.Errorf("%s %s: expected 5 lines got %d", language, quote, fileJob.Lines)
			}

			if fileJob.Code != 5 {
				t.Errorf("%s %s: expected 5 code got %d", language, quote, fileJob.Code)
			}

			if fileJob.Comment != 0 {
				t.Errorf("%s %s: expected 0 comment got %d", language, quote, fileJob.Comment)
			}
		}
	}
}

// CMake opens a bracket comment with any number of equals signs between its
// brackets and only a closer carrying the same number ends it. There is no
// dynamic delimiter here so the levels are enumerated, the way Rust's raw
// strings are, and this walks the ones that are.
func TestCountStatsCMakeBracketComment(t *testing.T) {
	ProcessConstants()

	for level := 0; level <= 8; level++ {
		equals := strings.Repeat("=", level)
		fileJob := FileJob{Language: "CMake"}
		fileJob.SetContent("#[" + equals + "[ a bracket comment\n a second comment line\n]" + equals + "]\nset(X 1)\n")

		CountStats(&fileJob)

		if fileJob.Lines != 4 {
			t.Errorf("level %d: expected 4 lines got %d", level, fileJob.Lines)
		}

		if fileJob.Code != 1 {
			t.Errorf("level %d: expected 1 code got %d", level, fileJob.Code)
		}

		if fileJob.Comment != 3 {
			t.Errorf("level %d: expected 3 comment got %d", level, fileJob.Comment)
		}
	}
}

// A closer carrying fewer equals signs than the opener is text, so the comment
// runs past it to the one that matches.
func TestCountStatsCMakeBracketCommentShorterCloserIsText(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{Language: "CMake"}
	fileJob.SetContent("#[==[ a bracket comment\n]] not the end\n]==]\nset(X 1)\n")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code got %d", fileJob.Code)
	}

	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comment got %d", fileJob.Comment)
	}
}

// The same brackets without the leading # are a bracket argument, which is how
// CMake writes a string running over lines. A # inside one is content.
func TestCountStatsCMakeBracketArgument(t *testing.T) {
	ProcessConstants()

	for level := 0; level <= 8; level++ {
		equals := strings.Repeat("=", level)
		fileJob := FileJob{Language: "CMake"}
		fileJob.SetContent("set(X [" + equals + "[\n# not a comment\n]" + equals + "])\nset(Y 1)\n")

		CountStats(&fileJob)

		if fileJob.Lines != 4 {
			t.Errorf("level %d: expected 4 lines got %d", level, fileJob.Lines)
		}

		if fileJob.Code != 4 {
			t.Errorf("level %d: expected 4 code got %d", level, fileJob.Code)
		}

		if fileJob.Comment != 0 {
			t.Errorf("level %d: expected 0 comment got %d", level, fileJob.Comment)
		}
	}
}

// C removes a backslash and the newline behind it before it looks for comments
// or strings, so a backslash ending a line comment carries the comment onto
// the next line, and a string that does not end in one is over at the newline
// whatever it holds. The cases below are LineJudge 5020 to 5070.
func TestCountStatsLineSplice(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		name    string
		content string
		lines   int64
		code    int64
		comment int64
		blank   int64
	}{
		{
			name:    "a splice inside a line comment carries it on",
			content: "// this comment continues \\\n   onto the next line in C\nint x = 1;\n",
			lines:   3, code: 1, comment: 2, blank: 0,
		},
		{
			name:    "a spliced comment holding no words still carries on",
			content: "int a = 1;\n// ---- \\\nint b = 2;\nint c = 3;\n",
			lines:   4, code: 2, comment: 2, blank: 0,
		},
		{
			name:    "a splice after a closed block comment carries the line comment on",
			content: "int a = 1;\n/* block */ // trailing \\\nint x = 1;\nint y = 2;\n",
			lines:   4, code: 2, comment: 2, blank: 0,
		},
		{
			name:    "code then a spliced line comment carries the comment on",
			content: "int a = 1; // comment \\\n   joined to the comment\nint x = 1;\n",
			lines:   3, code: 2, comment: 1, blank: 0,
		},
		{
			name:    "a blank line inside a spliced comment is comment and not blank",
			content: "// the blank line below is still this comment \\\n\nint x = 1;\n",
			lines:   3, code: 1, comment: 2, blank: 0,
		},
		{
			name:    "a string is carried only by a splice, so a blank line ends it",
			content: "char *s = \"abc \\\n\n// a comment once the string has ended\";\nint x = 1;\n",
			lines:   4, code: 3, comment: 1, blank: 0,
		},
	} {
		fileJob := FileJob{Language: "C"}
		fileJob.SetContent(test.content)

		CountStats(&fileJob)

		if fileJob.Lines != test.lines {
			t.Errorf("%s: expected %d lines got %d", test.name, test.lines, fileJob.Lines)
		}

		if fileJob.Code != test.code {
			t.Errorf("%s: expected %d code got %d", test.name, test.code, fileJob.Code)
		}

		if fileJob.Comment != test.comment {
			t.Errorf("%s: expected %d comment got %d", test.name, test.comment, fileJob.Comment)
		}

		if fileJob.Blank != test.blank {
			t.Errorf("%s: expected %d blank got %d", test.name, test.blank, fileJob.Blank)
		}
	}
}

// A raw string carries over a newline with no splice behind it, so the rule
// that ends an unspliced string has to leave it alone. The ignoreEscape flag
// is what tells the two apart, being set on exactly the delimiters that have
// no escape mechanism to splice with.
func TestCountStatsLineSpliceLeavesRawStringsAlone(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{Language: "C++"}
	fileJob.SetContent("const char *s = R\"(abc\n// not a comment, it is inside the raw string\n)\";\nint x = 1;\n")

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comment got %d", fileJob.Comment)
	}
}

// Only the C family splices. A backslash ending a line comment in a language
// that does not do it must leave the next line alone.
func TestCountStatsLineSpliceIsNotUniversal(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"Java", "C#", "Go", "JavaScript"} {
		fileJob := FileJob{Language: language}
		fileJob.SetContent("// a comment ending in a backslash \\\nint x = 1;\nint y = 2;\n")

		CountStats(&fileJob)

		if fileJob.Code != 2 {
			t.Errorf("%s: expected 2 code got %d", language, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s: expected 1 comment got %d", language, fileJob.Comment)
		}
	}
}

// A markup language holds prose, and prose holds apostrophes. Listing the
// single quote as a string delimiter turns every contraction into the start of
// a string that nothing closes, which then swallows the comments under it and
// runs to the end of the file. HTML, XML, XHTML, Svelte and JSX are all
// written with the double quote alone for that reason, and Svelte is the same
// single file component shape as Vue.
func TestCountStatsMarkupApostropheIsNotAString(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language string
		comment  string
	}{
		{"Vue", "<!-- a comment -->"},
		{"Astro", "<!-- a comment -->"},
		{"Handlebars", "<!-- a comment -->"},
		{"Mustache", "{{! a comment }}"},
		// already written with the double quote alone, here to keep it that way
		{"HTML", "<!-- a comment -->"},
		{"Svelte", "<!-- a comment -->"},
		{"XML", "<!-- a comment -->"},
	} {
		fileJob := FileJob{Language: test.language}
		fileJob.SetContent("<p>it's fine</p>\n" + test.comment + "\n<p>x</p>\n")

		CountStats(&fileJob)

		if fileJob.Lines != 3 {
			t.Errorf("%s: expected 3 lines got %d", test.language, fileJob.Lines)
		}

		if fileJob.Code != 2 {
			t.Errorf("%s: expected 2 code got %d", test.language, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s: expected 1 comment got %d", test.language, fileJob.Comment)
		}
	}
}

// A line comment opened by a word ends where the word does. REM opens a Batch
// comment and REMOVE is a different word, so matching those three letters as a
// prefix reads a command as a comment. The same holds for Autoconf's dnl and
// LOLCODE's BTW.
func TestCountStatsWordCommentNeedsAWordBreak(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language string
		comment  string
		longer   string
	}{
		{"Batch", "REM", "REMOVE /Q file.txt"},
		{"Batch", "rem", "remove /Q file.txt"},
		{"ASP", "REM", "REMOVE x"},
		{"Autoconf", "dnl", "dnlfoo bar"},
		{"LOLCODE", "BTW", "BTWISE x"},
	} {
		fileJob := FileJob{Language: test.language}
		fileJob.SetContent(test.comment + " a comment\n" + test.longer + "\nx\n")

		CountStats(&fileJob)

		if fileJob.Lines != 3 {
			t.Errorf("%s %s: expected 3 lines got %d", test.language, test.comment, fileJob.Lines)
		}

		if fileJob.Code != 2 {
			t.Errorf("%s %s: expected 2 code got %d", test.language, test.comment, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s %s: expected 1 comment got %d", test.language, test.comment, fileJob.Comment)
		}
	}
}

// A word break is wanted after a word, not after a symbol, so :: and // and #
// keep opening a comment with whatever behind them. FORTRAN Legacy is the one
// that would break: a C in the first column opens a comment there whatever
// follows it, which is how CCCCCCCC is written as a rule across the page, so
// the check is only for tokens longer than a single byte.
func TestCountStatsWordBreakLeavesSymbolCommentsAlone(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language string
		content  string
	}{
		{"Batch", "::comment\nx\n"},
		{"C", "//comment\nint x = 1;\n"},
		{"Shell", "#comment\nx=1\n"},
		{"Haskell", "--comment\nx = 1\n"},
		{"FORTRAN Legacy", "CCCCCCCCCCCCCCCCCCCC\n      x = 1\n"},
	} {
		fileJob := FileJob{Language: test.language}
		fileJob.SetContent(test.content)

		CountStats(&fileJob)

		if fileJob.Code != 1 {
			t.Errorf("%s: expected 1 code got %d", test.language, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s: expected 1 comment got %d", test.language, fileJob.Comment)
		}
	}
}

// Forth opens a comment with a backslash standing as its own word, so it wants
// the break after it the way a word does. The delimiter was written \\ before
// this, which matched nothing, so Forth had no line comment at all.
func TestCountStatsForthBackslashComment(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{Language: "Forth"}
	fileJob.SetContent("\\ a forth comment\n: double dup + ;\n")

	CountStats(&fileJob)

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
}

func TestCountStatsWithQuotes(t *testing.T) {
	fileJob := FileJob{}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.SetContent(`var test = "/*";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.SetContent(`t = " if ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}

	fileJob.Code = 0
	fileJob.Comment = 0
	fileJob.Complexity = 0
	fileJob.SetContent(`t = " if switch for while do loop != == && || ";`)
	CountStats(&fileJob)
	if fileJob.Code != 1 {
		t.Errorf("One line expected got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("No line expected got %d", fileJob.Comment)
	}
	if fileJob.Complexity != 0 {
		t.Errorf("No line expected got %d", fileJob.Complexity)
	}
}

func TestCountStatsBlankLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
		Blank:   0,
	}

	CountStats(&fileJob)
	if fileJob.Blank != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent(" ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("\n ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("            ")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("            \n             ")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("\r\n\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Blank)
	}

	fileJob.Blank = 0
	fileJob.SetContent("\r\n")
	CountStats(&fileJob)
	if fileJob.Blank != 1 {
		t.Errorf("One line expected got %d", fileJob.Blank)
	}
}

func TestCountStatsComplexityCount(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if ",
		"	if ",
		"if a.equals(b) {",
		"if(",
		" if(i.equals(0))",
		"    if(",
		"    if( ",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.SetContent(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 1 {
			t.Errorf("Expected complexity of 1 got %d for %s", fileJob.Complexity, check)
		}
	}
}

func TestCountStatsComplexityCountFalse(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	checks := []string{
		"if",
		"aif ",
		"aif(",
	}

	for _, check := range checks {
		fileJob.Complexity = 0
		fileJob.SetContent(check)
		fileJob.Language = "Java"
		CountStats(&fileJob)
		if fileJob.Complexity != 0 {
			t.Errorf("Expected complexity of 0 got %d for %s", fileJob.Complexity, check)
		}
	}

}

func TestCountStatsComplexityRustQuestionOperator(t *testing.T) {
	ProcessConstants()

	checks := []struct {
		content string
		want    int64
	}{
		{"foo()?;", 1},
		{"foo() ?;", 1},
		{"foo()?.bar();", 1},
		{"foo()? as u16;", 1},
		{"foo()??;", 2},
		{"let y = x?;", 1},
	}

	for _, c := range checks {
		fileJob := FileJob{Language: "Rust"}
		fileJob.SetContent(c.content)
		CountStats(&fileJob)
		if fileJob.Complexity != c.want {
			t.Errorf("Expected complexity of %d got %d for %q", c.want, fileJob.Complexity, c.content)
		}
	}
}

// ?Sized is a trait-bound prefix, not the try operator. The postfix matcher
// must skip it whether or not whitespace separates it from the preceding
// token (`T: ?Sized`, `T:?Sized`, `Debug+?Sized` are all valid Rust).
func TestCountStatsComplexityRustQuestionSizedNotCounted(t *testing.T) {
	ProcessConstants()

	checks := []string{
		"fn foo<T: ?Sized>() {}",
		"fn foo<T:?Sized>() {}",
		"fn foo<T:? Sized>() {}",
		"struct Bar<T: ?Sized>(T);",
		"where T: Debug + ?Sized,",
		"where T: Debug+?Sized,",
	}

	for _, content := range checks {
		fileJob := FileJob{Language: "Rust"}
		fileJob.SetContent(content)
		CountStats(&fileJob)
		if fileJob.Complexity != 0 {
			t.Errorf("Expected complexity of 0 got %d for %q (?Sized must not count)", fileJob.Complexity, content)
		}
	}
}

func TestCountStatsComplexityTypeScriptPostfixOperators(t *testing.T) {
	ProcessConstants()

	checks := []struct {
		content string
		want    int64
	}{
		{"obj.attr?.method();", 1},
		{"path ?? url;", 1},
		{"path??url;", 1},
		{"value ??= fallback;", 1},
		{"value??=fallback;", 1},
		{"function get(path?: string) { return path ?? url; }", 1},
		{"interface User { age?: number; method?(): void; }", 0},
	}

	for _, c := range checks {
		fileJob := FileJob{Language: "TypeScript"}
		fileJob.SetContent(c.content)
		CountStats(&fileJob)
		if fileJob.Complexity != c.want {
			t.Errorf("Expected complexity of %d got %d for %q", c.want, fileJob.Complexity, c.content)
		}
	}
}

type linecounter struct {
	blanks   int
	comments int
	code     int
	loc      int
	stop     bool
}

func (l *linecounter) ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool {
	l.loc++
	switch lineType {
	case LINE_BLANK:
		l.blanks++
	case LINE_COMMENT:
		l.comments++
	case LINE_CODE:
		l.code++
	}
	return !l.stop
}

func TestCountStatsCallback(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.SetContent(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 7 {
		t.Errorf("Expected loc of 7 got %d", lc.loc)
	}
	if lc.blanks != 2 {
		t.Errorf("Expected loc of 2 got %d", lc.blanks)
	}
	if lc.comments != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.comments)
	}
	if lc.code != 4 {
		t.Errorf("Expected loc of 4 got %d", lc.code)
	}
}

func TestCountStatsCallbackInterrupt(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{}

	fileJob.SetContent(`package foo

import com.foo.bar;

// this is a comment
class A {
}`)
	var lc linecounter
	lc.stop = true
	fileJob.Language = "Java"
	fileJob.Callback = &lc
	CountStats(&fileJob)
	if lc.loc != 1 {
		t.Errorf("Expected loc of 1 got %d", lc.loc)
	}
}

// Edge case condition where if ending with comment it would be counted
// as code due to how internal state work.
func TestCountStatsEdgeCase1(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.SetContent(`/**/
`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}
}

// Turns out that some languages such as Rust support
// nested comments. Check that it works here
func TestCountStatsNestedComments(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

// Java does not support nested multiline comments
func TestCountStatsNestedCommentsJava(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.SetContent(`/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsNestedCommentsRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`t/*/**/*/`)

	CountStats(&fileJob)

	if fileJob.Lines != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

// Block comments nest in a good many more languages than the ones that had the
// flag set. Each of these opens an inner comment inside an outer one, and the
// text between the inner closer and the outer one is still comment.
func TestCountStatsNestedCommentsByLanguage(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language string
		open     string
		close    string
	}{
		{"Haskell", "{-", "-}"},
		{"Odin", "/*", "*/"},
		{"Scala", "/*", "*/"},
		{"F#", "(*", "*)"},
		{"OCaml", "(*", "*)"},
		{"Coq", "(*", "*)"},
		{"Dart", "/*", "*/"},
		{"Elm", "{-", "-}"},
		{"PureScript", "{-", "-}"},
		{"Agda", "{-", "-}"},
	} {
		fileJob := FileJob{Language: test.language}
		fileJob.SetContent(test.open + " outer " + test.open + " inner " + test.close + " still the outer comment " + test.close + "\nx = 1\n")

		CountStats(&fileJob)

		if fileJob.Lines != 2 {
			t.Errorf("%s: expected 2 lines got %d", test.language, fileJob.Lines)
		}

		if fileJob.Code != 1 {
			t.Errorf("%s: expected 1 code got %d", test.language, fileJob.Code)
		}

		if fileJob.Comment != 1 {
			t.Errorf("%s: expected 1 comment got %d", test.language, fileJob.Comment)
		}

		if fileJob.Blank != 0 {
			t.Errorf("%s: expected 0 blank got %d", test.language, fileJob.Blank)
		}
	}
}

func TestCountStatsSingleCommentRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`t = "
/*
";`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsStringCheck(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`let does_not_start = // "
"until here,
test/*
test"; // a quote: "`)

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comment lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank lines got %d", fileJob.Blank)
	}
}

func TestCountStatsBosque(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Bosque",
	}

	fileJob.SetContent(`//This is a bosque test
method offsetMomentum(px: Float, py: Float, pz: Float): Body {
      return this<~(vx=Float::div(px->negate(), Body::solar_mass), vy=Float::div(py->negate(), Body::solar_mass), vz=Float::div(pz->negate(), Body::solar_mass));
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 3 {
		t.Errorf("Expected 4 code lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 0 comment lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank lines got %d", fileJob.Blank)
	}
}

func TestCheckForMatchNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("one does not simply walk into mordor"),
	}

	matches := &Trie{}
	matches.Insert(TSlcomment, []byte("//"))
	matches.Insert(TSlcomment, []byte("--"))

	match, _, _ := matches.Match(fileJob.Content)

	if match != 0 {
		t.Errorf("Expected no match")
	}
}

func TestCheckForMatchHasMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("// one does not simply walk into mordor"),
	}

	matches := &Trie{}
	matches.Insert(TSlcomment, []byte("//"))
	matches.Insert(TSlcomment, []byte("--"))

	match, _, _ := matches.Match(fileJob.Content)

	if match != TSlcomment {
		t.Errorf("Expected match")
	}
}

func TestCheckForMatchSingleNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("// one does not simply walk into mordor"),
	}

	matches := []byte("*/")

	match := checkForMatchSingle('/', 0, 100, matches, &fileJob)

	if match != false {
		t.Errorf("Expected no match")
	}
}

func TestCheckForMatchSingleMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Rust",
		Content:  []byte("*/ one does not simply walk into mordor"),
	}

	matches := []byte("*/")

	match := checkForMatchSingle('*', 0, 100, matches, &fileJob)

	if match != true {
		t.Errorf("Expected match")
	}
}

func TestCheckComplexityMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("for (int i=0; i<100; i++) {"),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))

	match, n, _ := matches.Match(fileJob.Content)

	if match != TComplexity || n != 4 {
		t.Errorf("Expected match")
	}
}

func TestCheckComplexityNoMatch(t *testing.T) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("far (int i=0; i<100; i++) {"),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))

	match, _, _ := matches.Match(fileJob.Content)

	if match != 0 {
		t.Errorf("Expected no match")
	}
}

func TestCountStatsRubyRegression(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Ruby",
	}

	fileJob.SetContent(`=begin
=end
t`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comment lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blank lines got %d", fileJob.Blank)
	}
}

func TestFileProcessorWorker(t *testing.T) {
	inputChan := make(chan *FileJob, 10000)

	inputChan <- &FileJob{
		Filename:  "testing.go",
		Location:  "./",
		Extension: "go",
		Content:   []byte("this is some content"),
	}

	close(inputChan)
	outputChan := make(chan *FileJob, 10000)

	Duplicates = true

	ctx := processorContext{remap: newRemapConfig("", "")}
	ctx.fileProcessorWorker(inputChan, outputChan)

	for res := range outputChan {
		if res.Bytes == 0 {
			t.Error("Expect bytes to have something")
		}
	}
}

func TestParseRemapRulesIgnoresInvalidEntries(t *testing.T) {
	rules := parseRemapRules("match:Go,invalid,too:many:parts,:Rust")

	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}

	if string(rules[0].pattern) != "match" || rules[0].language != "Go" {
		t.Fatalf("unexpected first rule: %#v", rules[0])
	}

	if string(rules[1].pattern) != "" || rules[1].language != "Rust" {
		t.Fatalf("unexpected second rule: %#v", rules[1])
	}
}

func TestHardRemapLanguageUsesParsedRules(t *testing.T) {
	job := &FileJob{
		Language: "Plain Text",
		Location: "./test.txt",
		Content:  []byte("prefix -*- C++ -*- suffix"),
	}

	ctx := processorContext{remap: newRemapConfig("-*- C++ -*-:C Header", "")}
	remapped := ctx.hardRemapLanguage(job)

	if !remapped {
		t.Fatal("expected file to be remapped")
	}

	if job.Language != "C Header" {
		t.Fatalf("expected remapped language to be C Header, got %s", job.Language)
	}
}

func TestEdgeCase(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C#",
	}

	// For C# we can enter a string using @" or " but if we do the former,
	// and we don't skip over the full length we exit the string in this case
	// which means we pick up the /* and the count is incorrect
	fileJob.SetContent(`@"\ /*"
a`)

	CountStats(&fileJob)

	if fileJob.Lines != 2 {
		t.Errorf("Expected 2 lines")
	}

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}
}

func TestEdgeCaseOther(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C#",
	}

	// For C# we can enter a string using @" or " but if we do the former,
	// and we don't skip over the full length we exit the string in this case
	// which means we pick up the /* and the count is incorrect
	fileJob.SetContent(`@"C:\" /*
a */`)

	CountStats(&fileJob)

	if fileJob.Lines != 2 {
		t.Errorf("Expected 2 lines")
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}
}

func TestCountStatsCSharpIgnoreEscape(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C#",
	}

	fileJob.SetContent(`namespace Ns
{
   public class Cls
   {
       private const string BasePath = @"a:\";

       [Fact]
       public void MyTest()
       {
           // Arrange.
           Foo();

           // Act.
           Bar();

           // Assert.
           Baz();
       }
   }
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 20 {
		t.Errorf("Expected 20 lines")
	}

	if fileJob.Code != 14 {
		t.Errorf("Expected 14 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Blank)
	}
}

func TestCheckBomSkipUTF8(t *testing.T) {
	fileJob := &FileJob{
		Content: []byte{239, 187, 191}, // UTF-8 BOM
	}

	skip := checkBomSkip(fileJob)
	if skip != 3 {
		t.Errorf("Expected skip length to match 3 got %d", skip)
	}
}

func TestCheckBomSkip(t *testing.T) {
	Verbose = true
	for _, v := range ByteOrderMarks {
		fileJob := &FileJob{
			Content: v,
		}

		skip := checkBomSkip(fileJob)
		if skip != 0 {
			t.Errorf("Expected skip length to match %d got %d", len(v), skip)
		}
	}
}

// Captures checking if a quote is prefixed by \ such as in
// a char which should otherwise trigger the string state which is incorrect
func TestCountStatsIssue73(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Java",
	}

	fileJob.SetContent(`'\"'{
code

`)
	fileJob.Bytes = int64(len(fileJob.Content))

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines")
	}

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsIssue106(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent("foo = `\nabc\"\ndef\n`")

	CountStats(&fileJob)
}

func TestMinifiedGeneratedCheck(t *testing.T) {
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890ABCDEF")
	Minified = true
	CountStats(&fileJob)
	Minified = false

	if fileJob.Minified != true {
		t.Error("Expected minified to come back true")
	}
}

func TestMinifiedGeneratedCheckTwoLines(t *testing.T) {
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890ABCDEF\n1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890ABCDEF")
	Minified = true
	CountStats(&fileJob)
	Minified = false

	if fileJob.Minified != true {
		t.Error("Expected minified to come back true")
	}
}

func TestMinifiedGeneratedCheckEdge(t *testing.T) {
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890ABCD")
	Minified = true
	CountStats(&fileJob)
	Minified = false

	if fileJob.Minified != false {
		t.Error("Expected minified to come back false")
	}
}

func TestGenerated(t *testing.T) {
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent(`
// Code generated by some tool, DO NOT EDIT.

// Package some contains something.
package some
`)
	Generated = true
	GeneratedMarkers = []string{"do not edit", "generated"}
	CountStats(&fileJob)
	Generated = false

	if fileJob.Generated != true {
		t.Error("Expected generated to come back true")
	}

	if fileJob.Language != "Go (gen)" {
		t.Errorf("Expected Language \"Go (gen)\", received %q", fileJob.Language)
	}
}

func TestCountStatsIssue182(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Pascal",
	}

	fileJob.SetContent(`uses
    someunit;

{This is a comment}
procedure Something
var
    avar: String;
begin
    Println('Oho');
end;
{This is a comment}
procedure Nothing
begin
end.
`)

	CountStats(&fileJob)

	if fileJob.Code != 11 {
		t.Errorf("Expected 11 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsIssue182Delphi(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Pascal",
	}

	fileJob.SetContent(`// this isnt a comment in pascal but is in delphi
`)

	CountStats(&fileJob)

	if fileJob.Code != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

//////////////////////////////////////////////////
// Content Classification Tests
//////////////////////////////////////////////////

// Classification should not change any line counts
func TestClassifyContentCountInvariance(t *testing.T) {
	ProcessConstants()

	content := `package main

import "fmt"

// main function
func main() {
	fmt.Println("hello") // inline comment
	/* block
	   comment */
}
`
	// Run without classification
	fj1 := FileJob{Language: "Go"}
	fj1.SetContent(content)
	CountStats(&fj1)

	// Run with classification
	fj2 := FileJob{Language: "Go", ClassifyContent: true}
	fj2.SetContent(content)
	CountStats(&fj2)

	if fj1.Lines != fj2.Lines {
		t.Errorf("Lines mismatch: %d vs %d", fj1.Lines, fj2.Lines)
	}
	if fj1.Code != fj2.Code {
		t.Errorf("Code mismatch: %d vs %d", fj1.Code, fj2.Code)
	}
	if fj1.Comment != fj2.Comment {
		t.Errorf("Comment mismatch: %d vs %d", fj1.Comment, fj2.Comment)
	}
	if fj1.Blank != fj2.Blank {
		t.Errorf("Blank mismatch: %d vs %d", fj1.Blank, fj2.Blank)
	}
	if fj1.Complexity != fj2.Complexity {
		t.Errorf("Complexity mismatch: %d vs %d", fj1.Complexity, fj2.Complexity)
	}
}

// ContentByteType should stay nil when ClassifyContent is false (default)
func TestClassifyContentNilGuard(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go"}
	fileJob.SetContent("x := 1\n")
	CountStats(&fileJob)

	if fileJob.ContentByteType != nil {
		t.Error("Expected ContentByteType to be nil when ClassifyContent is false")
	}
}

// Code-only file: all non-whitespace bytes should be ByteTypeCode
func TestClassifyContentCodeOnly(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent("x := 1")
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	for i, b := range fileJob.Content {
		bt := fileJob.ContentByteType[i]
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue // whitespace can be blank or code depending on state
		}
		if bt != ByteTypeCode {
			t.Errorf("byte %d (%q): expected ByteTypeCode(%d), got %d", i, string(b), ByteTypeCode, bt)
		}
	}
}

// Comment-only file: "// comment" bytes after // should be ByteTypeComment
func TestClassifyContentCommentOnly(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent("// this is a comment")
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	// After the first byte that triggers comment state, subsequent bytes should be comment
	hasComment := false
	for i := range fileJob.Content {
		if fileJob.ContentByteType[i] == ByteTypeComment {
			hasComment = true
		}
	}
	if !hasComment {
		t.Error("Expected at least some ByteTypeComment bytes for a comment-only line")
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment line, got %d", fileJob.Comment)
	}
}

// Multi-line comment: inner bytes should be ByteTypeComment
func TestClassifyContentMultiLineComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent("/* hello\nworld */")
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	// Check that "hello" inside the comment is classified as comment
	helloStart := bytes.Index(fileJob.Content, []byte("hello"))
	for i := helloStart; i < helloStart+5; i++ {
		if fileJob.ContentByteType[i] != ByteTypeComment {
			t.Errorf("byte %d (%q): expected ByteTypeComment(%d), got %d", i, string(fileJob.Content[i]), ByteTypeComment, fileJob.ContentByteType[i])
		}
	}
}

// String literal: bytes inside "..." should be ByteTypeString
func TestClassifyContentString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent(`x := "hello"`)
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	// Find "hello" inside the string
	helloStart := bytes.Index(fileJob.Content, []byte("hello"))
	for i := helloStart; i < helloStart+5; i++ {
		if fileJob.ContentByteType[i] != ByteTypeString {
			t.Errorf("byte %d (%q): expected ByteTypeString(%d), got %d", i, string(fileJob.Content[i]), ByteTypeString, fileJob.ContentByteType[i])
		}
	}

	// "x" at the start should be code
	if fileJob.ContentByteType[0] != ByteTypeCode {
		t.Errorf("byte 0 (%q): expected ByteTypeCode(%d), got %d", string(fileJob.Content[0]), ByteTypeCode, fileJob.ContentByteType[0])
	}
}

func TestMojoRawTStringsDoNotCountComplexity(t *testing.T) {
	ProcessConstants()

	for _, prefix := range []string{"rt\"", "tr\"", "rT\"", "tR\""} {
		content := "value = " + prefix + "for item in items\""
		fileJob := FileJob{Language: "Mojo"}
		fileJob.SetContent(content)
		CountStats(&fileJob)

		if fileJob.Complexity != 0 {
			t.Errorf("prefix %q counted string contents as complexity: %d", prefix, fileJob.Complexity)
		}
	}
}

// Mixed line: "x := 1 // comment" has code then comment
func TestClassifyContentMixedLine(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent("x := 1 // comment")
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	// 'x' should be code
	if fileJob.ContentByteType[0] != ByteTypeCode {
		t.Errorf("byte 0 (%q): expected ByteTypeCode, got %d", string(fileJob.Content[0]), fileJob.ContentByteType[0])
	}

	// "comment" text after // should be ByteTypeComment
	commentStart := bytes.Index(fileJob.Content, []byte("comment"))
	for i := commentStart; i < commentStart+7; i++ {
		if fileJob.ContentByteType[i] != ByteTypeComment {
			t.Errorf("byte %d (%q): expected ByteTypeComment, got %d", i, string(fileJob.Content[i]), fileJob.ContentByteType[i])
		}
	}

	// Line should be counted as code (code + comment = code line)
	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code line, got %d", fileJob.Code)
	}
}

// Python docstring: content classified as ByteTypeComment
func TestClassifyContentPythonDocstring(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Python", ClassifyContent: true}
	fileJob.SetContent(`"""
docstring content
"""`)
	CountStats(&fileJob)

	if fileJob.ContentByteType == nil {
		t.Fatal("Expected ContentByteType to be non-nil")
	}

	// "docstring content" should be classified as comment
	docStart := bytes.Index(fileJob.Content, []byte("docstring"))
	if docStart >= 0 {
		for i := docStart; i < docStart+9; i++ {
			if fileJob.ContentByteType[i] != ByteTypeComment {
				t.Errorf("byte %d (%q): expected ByteTypeComment(%d), got %d",
					i, string(fileJob.Content[i]), ByteTypeComment, fileJob.ContentByteType[i])
			}
		}
	}
}

// Python raw-string docstrings (r"""/r”') should be counted as comments just
// like plain """/”' docstrings. https://github.com/boyter/scc raw docstrings
func TestCountStatsPythonRawDocstring(t *testing.T) {
	ProcessConstants()

	content := `r'''This is a module docstring'''

class C:
  r'''
  This is a class docstring
  '''

  def f():
    r"""This is a function docstring.
    simple quotes and double quotes are equivalent
    """
    pass
`

	fileJob := FileJob{Language: "Python"}
	fileJob.SetContent(content)
	CountStats(&fileJob)

	if fileJob.Comment != 7 {
		t.Errorf("Expected 7 comment lines (docstrings) got %d", fileJob.Comment)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code lines got %d", fileJob.Code)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blank lines got %d", fileJob.Blank)
	}

	plain := strings.ReplaceAll(content, "r'''", "'''")
	plain = strings.ReplaceAll(plain, "r\"\"\"", "\"\"\"")

	plainJob := FileJob{Language: "Python"}
	plainJob.SetContent(plain)
	CountStats(&plainJob)

	if plainJob.Comment != fileJob.Comment || plainJob.Code != fileJob.Code || plainJob.Blank != fileJob.Blank {
		t.Errorf("Raw docstring counts (c=%d code=%d b=%d) differ from plain (c=%d code=%d b=%d)",
			fileJob.Comment, fileJob.Code, fileJob.Blank,
			plainJob.Comment, plainJob.Code, plainJob.Blank)
	}
}

// FilterContentByType: returns filtered content correctly
func TestFilterContentByType(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent("x := 1 // comment\n")
	CountStats(&fileJob)

	// Filter to only code
	codeOnly := fileJob.FilterContentByType(ByteTypeCode)
	if codeOnly == nil {
		t.Fatal("Expected non-nil result from FilterContentByType")
	}

	// Should preserve newlines
	if codeOnly[len(codeOnly)-1] != '\n' {
		t.Error("Expected newline to be preserved")
	}

	// 'x' should be preserved
	if codeOnly[0] != 'x' {
		t.Errorf("Expected 'x' to be preserved, got %q", string(codeOnly[0]))
	}

	// "comment" should be replaced with spaces
	commentStart := bytes.Index(fileJob.Content, []byte("comment"))
	for i := commentStart; i < commentStart+7; i++ {
		if codeOnly[i] != ' ' {
			t.Errorf("byte %d: expected space, got %q", i, string(codeOnly[i]))
		}
	}
}

// FilterContentByType returns nil when ContentByteType is nil
func TestFilterContentByTypeNil(t *testing.T) {
	fileJob := FileJob{}
	fileJob.SetContent("hello")

	result := fileJob.FilterContentByType(ByteTypeCode)
	if result != nil {
		t.Error("Expected nil result when ContentByteType is nil")
	}
}

// FilterContentByType with multiple keep types
func TestFilterContentByTypeMultiple(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Go", ClassifyContent: true}
	fileJob.SetContent(`x := "hello" // comment`)
	CountStats(&fileJob)

	// Keep both code and string, filter out comments
	result := fileJob.FilterContentByType(ByteTypeCode, ByteTypeString)
	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// 'x' (code) should be preserved
	if result[0] != 'x' {
		t.Errorf("Expected 'x' preserved, got %q", string(result[0]))
	}

	// "hello" (string) content should be preserved
	helloStart := bytes.Index(fileJob.Content, []byte("hello"))
	for i := helloStart; i < helloStart+5; i++ {
		if result[i] != fileJob.Content[i] {
			t.Errorf("byte %d: expected %q preserved, got %q", i, string(fileJob.Content[i]), string(result[i]))
		}
	}
}

//////////////////////////////////////////////////
// Benchmarks Below
//////////////////////////////////////////////////

func BenchmarkCountStatsLinesEmpty(b *testing.B) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesSingleChar(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesTwoLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a\na\na"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesShortEmptyLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890\n1234567890\n1234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesThreeShortEmptyLines(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("          \n          \n          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890"),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongMixedLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("1234567890          1234567890          1234567890          1234567890          1234567890          "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesLongAlternateLine(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a "),
	}

	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLines(b *testing.B) {
	b.StopTimer()
	content := ""
	for range 500 {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityIf(b *testing.B) {
	b.StopTimer()
	content := ""
	for range 500 {
		content += "iiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiiii\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFiveHundredLongLinesTriggerComplexityFor(b *testing.B) {
	b.StopTimer()
	content := ""
	for range 500 {
		content += "fofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofofo\n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesFourHundredLongLinesMixed(b *testing.B) {
	b.StopTimer()
	content := ""
	for range 100 {
		content += "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890\n"
		content += "1234567890          1234567890          1234567890          1234567890          1234567890          \n"
		content += "                                                                                                    \n"
		content += "a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a a \n"
	}

	fileJob := FileJob{
		Content: []byte(content),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		CountStats(&fileJob)
	}
}

func BenchmarkCheckByteEqualityReflect(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := reflect.DeepEqual(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckByteEqualityBytes(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := bytes.Equal(one[1:], two[1:])

		if equal {
			count++
		}
	}

	b.Log(count)
}

// This appears to be faster than bytes.Equal because it does not need
// to do length comparison checks at the start
func BenchmarkCheckByteEqualityLoop(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		for j := 1; j < len(one); j++ {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

// Check if the 1 offset makes a difference, which it does by ~1 ns
func BenchmarkCheckByteEqualityLoopWithAdditional(b *testing.B) {
	b.StopTimer()
	one := []byte("for")
	two := []byte("for")

	count := 0

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		equal := true

		// Don't start at 1 like the above but 0 to do a full scan
		for j := range one {
			if one[j] != two[j] {
				equal = false
				break
			}
		}

		if equal {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckArrayCheck(b *testing.B) {
	array := []byte{
		'a',
		'b',
		'c',
		'd',
		'e',
		'f',
		'g',
		'h',
		'i',
		'j',
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {
		for index := 0; index < len(array); index++ {
			if array[index] == searchFor {
				found++
				break
			}
		}
	}

	b.Log(found)
}

func BenchmarkCheckMapCheck(b *testing.B) {
	array := map[byte]bool{
		'a': true,
		'b': true,
		'c': true,
		'd': true,
		'e': true,
		'f': true,
		'g': true,
		'h': true,
		'i': true,
		'j': true,
	}

	var searchFor byte = 'j'
	found := 0

	for i := 0; i < b.N; i++ {

		_, ok := array[searchFor]

		if ok {
			found++
		}
	}

	b.Log(found)
}

func BenchmarkStringLoop(b *testing.B) {
	b.StopTimer()

	var str strings.Builder
	for range 10000 {
		str.WriteString("1")
	}
	search := str.String()
	count := 0
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		for j := 0; j < len(search); j++ {
			if search[j] != '\n' {
				count++
			}

		}
	}
	b.Log(count)
}

func BenchmarkByteLoop(b *testing.B) {
	b.StopTimer()

	var str strings.Builder
	for range 10000 {
		str.WriteString("1")
	}
	search := []byte(str.String())
	count := 0
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		for j := range search {
			if search[j] != '\n' {
				count++
			}

		}
	}
	b.Log(count)
}

func BenchmarkLoopInLoop(b *testing.B) {
	search := []byte("this is a long from for string which we will search")
	matches := [][]byte{
		[]byte("if"),
		[]byte("if("),
		[]byte("else"),
		[]byte("while"),
		[]byte("while("),
		[]byte("for"),
		[]byte("foreach"),
	}
	endPoint := len(search)
	b.ResetTimer()

	potentialMatch := true
	for i := 0; i < b.N; i++ {

		potentialMatch = true
		for index := range search {

			for k := range matches {

				for j := 0; j < len(matches[k]); j++ {
					if index+j >= endPoint || matches[k][j] != search[index+j] {
						potentialMatch = false
					}
				}
			}

		}

	}
	b.Log(potentialMatch)
}

func BenchmarkFlattenedLoop(b *testing.B) {
	index := 0
	search := []byte("this is a long from for string which we will search")
	matches := []byte("if if( else while while( for foreach")

	b.ResetTimer()

	potentialMatch := true
	count := 0
	for i := 0; i < b.N; i++ {

		potentialMatch = true
		for j := range matches {
			if matches[j] == ' ' {
				count = 0
			} else {
				if matches[j] != search[index+count] {
					potentialMatch = false
				}

			}
		}

	}

	b.Log(potentialMatch)
}

func BenchmarkCheckComplexity(b *testing.B) {
	ProcessConstants()

	fileJob := FileJob{
		Language: "Java",
		Content:  []byte("A little while ago, I passed my first year mark of working for Google. This also marked the "),
	}

	matches := &Trie{}
	matches.Insert(TComplexity, []byte("for "))
	matches.Insert(TComplexity, []byte("for("))
	matches.Insert(TComplexity, []byte("if "))
	matches.Insert(TComplexity, []byte("if("))
	matches.Insert(TComplexity, []byte("switch "))
	matches.Insert(TComplexity, []byte("while "))
	matches.Insert(TComplexity, []byte("else "))
	matches.Insert(TComplexity, []byte("|| "))
	matches.Insert(TComplexity, []byte("&& "))
	matches.Insert(TComplexity, []byte("!= "))
	matches.Insert(TComplexity, []byte("== "))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := 0; j < len(fileJob.Content); j++ {
			matches.Match(fileJob.Content)
		}
	}
}

func BenchmarkCheckLen(b *testing.B) {
	matches := [][]byte{
		[]byte("for "),
		[]byte("for("),
		[]byte("if "),
		[]byte("if("),
		[]byte("switch "),
		[]byte("while "),
		[]byte("else "),
		[]byte("|| "),
		[]byte("&& "),
		[]byte("!= "),
		[]byte("== "),
	}

	count := 0
	for i := 0; i < b.N; i++ {
		for range matches {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCheckLenPrecalc(b *testing.B) {
	matches := [][]byte{
		[]byte("for "),
		[]byte("for("),
		[]byte("if "),
		[]byte("if("),
		[]byte("switch "),
		[]byte("while "),
		[]byte("else "),
		[]byte("|| "),
		[]byte("&& "),
		[]byte("!= "),
		[]byte("== "),
	}

	count := 0
	for i := 0; i < b.N; i++ {
		l := len(matches)
		for range l {
			count++
		}
	}

	b.Log(count)
}

func BenchmarkCountStatsNoClassify(b *testing.B) {
	ProcessConstants()
	b.StopTimer()
	content := ""
	for range 500 {
		content += "x := 1 // comment\n"
	}
	fileJob := FileJob{
		Language: "Go",
		Content:  []byte(content),
		Bytes:    int64(len(content)),
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		fileJob.Lines = 0
		fileJob.Code = 0
		fileJob.Comment = 0
		fileJob.Blank = 0
		fileJob.Complexity = 0
		fileJob.ComplexityLine = nil
		fileJob.ContentByteType = nil
		CountStats(&fileJob)
	}
}

func BenchmarkCountStatsWithClassify(b *testing.B) {
	ProcessConstants()
	b.StopTimer()
	content := ""
	for range 500 {
		content += "x := 1 // comment\n"
	}
	fileJob := FileJob{
		Language:        "Go",
		Content:         []byte(content),
		Bytes:           int64(len(content)),
		ClassifyContent: true,
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		fileJob.Lines = 0
		fileJob.Code = 0
		fileJob.Comment = 0
		fileJob.Blank = 0
		fileJob.Complexity = 0
		fileJob.ComplexityLine = nil
		fileJob.ContentByteType = nil
		CountStats(&fileJob)
	}
}

func BenchmarkHardRemapLanguage(b *testing.B) {
	job := &FileJob{
		Language: "Plain Text",
		Location: "./test.txt",
		Content:  []byte("prefix -*- C++ -*- suffix"),
	}

	b.Run("match_single_rule", func(b *testing.B) {
		ctx := processorContext{remap: newRemapConfig("-*- C++ -*-:C Header", "")}
		b.ReportAllocs()
		b.SetBytes(int64(len(job.Content)))

		for i := 0; i < b.N; i++ {
			job.Language = "Plain Text"
			_ = ctx.hardRemapLanguage(job)
		}
	})

	b.Run("no_match_single_rule", func(b *testing.B) {
		ctx := processorContext{remap: newRemapConfig("-*- Rust -*-:Rust", "")}
		b.ReportAllocs()
		b.SetBytes(int64(len(job.Content)))

		for i := 0; i < b.N; i++ {
			job.Language = "Plain Text"
			_ = ctx.hardRemapLanguage(job)
		}
	})

	b.Run("match_many_rules_late", func(b *testing.B) {
		rules := []string{
			"-*- Rust -*-:Rust",
			"-*- Python -*-:Python",
			"-*- Ruby -*-:Ruby",
			"-*- Java -*-:Java",
			"-*- Kotlin -*-:Kotlin",
			"-*- Swift -*-:Swift",
			"-*- Scala -*-:Scala",
			"-*- Perl -*-:Perl",
			"-*- Lua -*-:Lua",
			"-*- C++ -*-:C Header",
		}
		ctx := processorContext{remap: newRemapConfig(strings.Join(rules, ","), "")}
		b.ReportAllocs()
		b.SetBytes(int64(len(job.Content)))

		for i := 0; i < b.N; i++ {
			job.Language = "Plain Text"
			_ = ctx.hardRemapLanguage(job)
		}
	})
}

func BenchmarkUnknownRemapLanguage(b *testing.B) {
	job := &FileJob{
		Language: "#!",
		Location: "./test.txt",
		Content:  []byte("#!/bin/sh\nprefix -*- C++ -*- suffix"),
	}

	b.Run("match_single_rule", func(b *testing.B) {
		ctx := processorContext{remap: newRemapConfig("", "-*- C++ -*-:C Header")}
		b.ReportAllocs()
		b.SetBytes(int64(len(job.Content)))

		for i := 0; i < b.N; i++ {
			job.Language = "#!"
			_ = ctx.unknownRemapLanguage(job)
		}
	})

	b.Run("no_match_single_rule", func(b *testing.B) {
		ctx := processorContext{remap: newRemapConfig("", "-*- Rust -*-:Rust")}
		b.ReportAllocs()
		b.SetBytes(int64(len(job.Content)))

		for i := 0; i < b.N; i++ {
			job.Language = "#!"
			_ = ctx.unknownRemapLanguage(job)
		}
	})
}

// F# defines (* *) as multi-line comments but, before this fix, had no string
// literals in languages.json. A (* inside a regular string therefore opened a
// phantom block comment that ate the following code lines. With the string
// literals now declared, the (* inside the string is ignored by the comment
// scanner and each line is classified correctly.
func TestCountStatsFSharpStringHidesBlockComment(t *testing.T) {
	ProcessConstants()

	checks := []struct {
		name    string
		content string
		code    int64
		comment int64
	}{
		{
			name: "regular string hides block-comment opener",
			content: "let s = \"(* not a comment\"\n" +
				"let x = 1\n",
			code:    2,
			comment: 0,
		},
		{
			name: "verbatim string hides block-comment opener",
			content: "let s = @\"(* not a comment\"\n" +
				"let x = 1\n",
			code:    2,
			comment: 0,
		},
		{
			name: "real comment still counted",
			content: "// real comment\n" +
				"let x = 1\n",
			code:    1,
			comment: 1,
		},
	}

	for _, c := range checks {
		fileJob := FileJob{Language: "F#"}
		fileJob.SetContent(c.content)
		CountStats(&fileJob)
		if fileJob.Code != c.code || fileJob.Comment != c.comment {
			t.Errorf("%s: Code=%d Comment=%d (want Code=%d Comment=%d)",
				c.name, fileJob.Code, fileJob.Comment, c.code, c.comment)
		}
	}
}
