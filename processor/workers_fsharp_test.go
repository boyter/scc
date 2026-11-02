// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// F# reads (*) as the multiplication operator, not as the start of a comment,
// so that List.fold (*) 1 xs is a line of code. Read as a comment it ran on to
// the next *) in the file, here the end of the real comment below it, and
// counted the code and blank lines in between as comments.
func TestCountStatsFSharpMultiplicationOperator(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`module Numbers

let product xs = List.fold (*) 1 xs

// sums every item
let sum xs = List.fold (+) 0 xs

(* the average,
   or zero for an empty list *)
let average xs = if List.isEmpty xs then 0 else sum xs / List.length xs`)

	CountStats(&fileJob)

	if fileJob.Lines != 10 {
		t.Errorf("Expected 10 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 3 {
		t.Errorf("Expected 3 blank got %d", fileJob.Blank)
	}
}

// A line holding nothing but (*) is code as well.
func TestCountStatsFSharpMultiplicationOperatorOnItsOwnLine(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`let product =
    List.fold
        (*)
        1`)

	CountStats(&fileJob)

	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comments got %d", fileJob.Comment)
	}
}

// Only (*) itself is the operator. (* followed by anything else still opens a
// comment, including (**), which is an empty one.
func TestCountStatsFSharpBlockCommentStillOpens(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`(**)
let x = 1
(*x*)
(* a
   b *)
let y = (* inline *) 2`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 comments got %d", fileJob.Comment)
	}
}

// The quote in the character literal '"' opened a string that ran on to the
// next quote in the file, which turned the comments in between into code. A
// plain ' can't be a quote in F#, since it also starts a type parameter such as
// 'a, so '" is a quote closed by an apostrophe instead, the same as in Scala.
func TestCountStatsFSharpQuoteCharLiteral(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`module Quote

let isQuote c = c = '"' || c = '\"'

// the type parameter is not a quote
let id (x: 'a) = x

(* a block comment *)
let quote = "\""`)

	CountStats(&fileJob)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 3 {
		t.Errorf("Expected 3 blank got %d", fileJob.Blank)
	}
}

// A verbatim string holds a quote by writing it twice, so @"""" is one quote
// and @"""""" is two. Closing on the first quote of a pair left the rest of
// the run to open a triple-quoted string that ran on to the next """ in the
// file, which turned the comment in between into code.
func TestCountStatsFSharpVerbatimDoubledQuotes(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`let csv (s: string) = s.Replace(@"""", @"""""")
// a comment

let attr = @"<book author=""Milton, John"">"
// another comment
let x = """a "quoted" word"""
`)

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// A doubled quote inside a verbatim string does not end it, so the backslash
// in front of its real closer is not read as escaping that.
func TestCountStatsFSharpVerbatimDoubledQuoteThenBackslash(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`let s = @"say ""hi"" from C:\"
// a comment
let t = 1
`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comments got %d", fileJob.Comment)
	}
}

// @$" opens an interpolated verbatim string the same as $@", so a backslash in
// front of its closing quote does not escape it.
func TestCountStatsFSharpInterpolatedVerbatimString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "F#",
	}

	fileJob.SetContent(`let a = @$"{root}\"
// a comment
let b = $@"{root}\"

(* a comment *)
`)

	CountStats(&fileJob)

	if fileJob.Lines != 5 {
		t.Errorf("Expected 5 lines got %d", fileJob.Lines)
	}
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
