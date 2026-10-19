// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Kotlin had no raw string, so """ was read as an empty string and a plain
// quote. Whatever the raw string held decided what came after it: here the
// apostrophe in it's opens a character literal that turns the comments and
// the blank line under the string into code.
func TestCountStatsKotlinRawString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Kotlin",
	}

	fileJob.SetContent(`fun main() {
    val json = """
        {"key": "it's"}
    """
    // a comment

    /* another */
    println(json)
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 6 {
		t.Errorf("Expected 6 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// A raw string has no escape, so a backslash at the end of one is an ordinary
// character and does not carry the string on past its closer.
func TestCountStatsKotlinRawStringEndingInBackslash(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Kotlin",
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

// A raw string that ends in a quote is closed by the last three quotes of the
// run, so """say "hi"""" holds say "hi". Closing on the first three would leave
// the last quote to open a string that swallows the comment.
func TestCountStatsKotlinRawStringEndingInQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Kotlin",
	}

	fileJob.SetContent(`val s = """say "hi""""
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
