// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Scala had no multi-line string, so """ was read as an empty string and a
// plain quote. Whatever the string held then decided what came after it: here
// the lone quote in it opens a string that turns the comments and the blank
// line under it into code, while the line that is part of the string is
// counted as a comment.
func TestCountStatsScalaMultiLineString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Scala",
	}

	fileJob.SetContent(`object Main {
  val text = """
    a lone " quote
    // not a comment
  """
  // a comment

  /* another */
  println(text)
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 10 {
		t.Errorf("Expected 10 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 7 {
		t.Errorf("Expected 7 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// A multi-line string has no escape, so a backslash at the end of one is an
// ordinary character and does not carry the string on past its closer.
func TestCountStatsScalaMultiLineStringEndingInBackslash(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Scala",
	}

	fileJob.SetContent(`val dir = """C:\"""
// a comment
val x = 1`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
}

// A multi-line string that ends in a quote is closed by the last three quotes
// of the run, so """"([^"]*)"""" holds "([^"]*)". Closing on the first three
// would leave the last quote to open a string that swallows the comment.
func TestCountStatsScalaMultiLineStringEndingInQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Scala",
	}

	fileJob.SetContent(`val Quoted = """"([^"]*)"""".r
// a comment
val x = 1`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
}

// The quote in the character literal '"' opened a string that ran on to the
// next quote in the file and turned the lines in between into code.
func TestCountStatsScalaCharLiteralQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Scala",
	}

	fileJob.SetContent(`def isQuote(c: Char): Boolean = c == '"'
// a comment

/* another */
def split(line: String): Array[String] = line.split(",")`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// Only '" opens a character literal. An apostrophe on its own also opens a
// symbol literal or a Scala 3 quote, neither of which is closed by another, so
// it is left as code rather than opening a string that runs to the next one.
func TestCountStatsScalaApostropheIsNotAQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Scala",
	}

	fileJob.SetContent(`val s = 'name
// a comment
val e = '{ x + 1 }
/* another */
val c = '"'
// a third`)

	CountStats(&fileJob)

	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
}
