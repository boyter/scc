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
