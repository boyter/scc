// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

func countCsharp(t *testing.T, content string) FileJob {
	t.Helper()
	ProcessConstants()
	fileJob := FileJob{
		Language: "C#",
	}

	fileJob.SetContent(content)
	CountStats(&fileJob)

	return fileJob
}

func checkCsharpCounts(t *testing.T, fileJob FileJob, code, comment, blank int64) {
	t.Helper()
	if fileJob.Code != code {
		t.Errorf("Expected %d code got %d", code, fileJob.Code)
	}
	if fileJob.Comment != comment {
		t.Errorf("Expected %d comments got %d", comment, fileJob.Comment)
	}
	if fileJob.Blank != blank {
		t.Errorf("Expected %d blank got %d", blank, fileJob.Blank)
	}
}

// A raw string opens on three quotes and closes on three, so a lone quote
// inside one does not end it. Read as an empty string and a plain one, the lone
// quote opened a string that ran on to the next quote in the file and turned
// the comment and blank line in between into code.
func TestCountStatsCsharpRawStringHoldingQuote(t *testing.T) {
	fileJob := countCsharp(t, `var s = """
Use a " to quote.
""";
// a comment

var t = 1;
`)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	checkCsharpCounts(t, fileJob, 4, 1, 1)
}

// Everything inside a raw string is the string, including the blank lines and
// what looks like a comment.
func TestCountStatsCsharpRawStringOverLines(t *testing.T) {
	fileJob := countCsharp(t, `var source = """
    class C
    {

        // not a comment
        char c = '"';
    }
    """;

/* a comment */
var x = 1;
`)

	if fileJob.Lines != 11 {
		t.Errorf("Expected 11 lines got %d", fileJob.Lines)
	}
	checkCsharpCounts(t, fileJob, 9, 1, 1)
}

// A raw string can open and close on the same line.
func TestCountStatsCsharpSingleLineRawString(t *testing.T) {
	fileJob := countCsharp(t, `var s = """Use a " to quote.""";
// a comment

var t = 1;
`)

	checkCsharpCounts(t, fileJob, 2, 1, 1)
}

// A raw string opened with four quotes holds a run of three, and an
// interpolated one opens on the quotes after the dollars.
func TestCountStatsCsharpRawStringOfFourQuotes(t *testing.T) {
	fileJob := countCsharp(t, `var a = """"
    var s = """
        a " quote
        """;
    """";
// a comment
var b = $$"""
    {"name": "{{name}}"}
    """;

/* a comment */
`)

	if fileJob.Lines != 11 {
		t.Errorf("Expected 11 lines got %d", fileJob.Lines)
	}
	checkCsharpCounts(t, fileJob, 8, 2, 1)
}

// A verbatim string holds a quote by writing it twice, so @"""" is one quote
// and @"""""" is two. Closing on the first quote of a pair left the rest of
// the run to open a raw string.
func TestCountStatsCsharpVerbatimDoubledQuotes(t *testing.T) {
	fileJob := countCsharp(t, `var csv = value.Replace(@"""", @"""""");
// a comment

var attr = @"[LoggerMessage(Message = """")]";
// another comment
`)

	if fileJob.Lines != 5 {
		t.Errorf("Expected 5 lines got %d", fileJob.Lines)
	}
	checkCsharpCounts(t, fileJob, 2, 2, 1)
}

// A doubled quote inside a verbatim string does not end it, so the backslash
// in front of its real closer is not read as escaping that.
func TestCountStatsCsharpVerbatimDoubledQuoteThenBackslash(t *testing.T) {
	fileJob := countCsharp(t, `var s = @"say ""hi"" from C:\";
// a comment
var t = 1;
`)

	checkCsharpCounts(t, fileJob, 2, 1, 0)
}

// @$" opens a verbatim string the same as $@", so a backslash in front of its
// closing quote does not escape it.
func TestCountStatsCsharpInterpolatedVerbatimString(t *testing.T) {
	fileJob := countCsharp(t, `var a = @$"{root}\";
// a comment
var b = $@"{root}\";

/* a comment */
`)

	if fileJob.Lines != 5 {
		t.Errorf("Expected 5 lines got %d", fileJob.Lines)
	}
	checkCsharpCounts(t, fileJob, 2, 2, 1)
}
