// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// MATLAB declared its block comment as %{ ... }% and Racket declared its own
// reversed as |# ... #|. In both cases the real opener never closed, so the
// scanner ran the comment to EOF and swallowed the rest of the file.

func TestCountStatsMatlabBlockCommentCloses(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "MATLAB",
	}

	fileJob.SetContent(`function c = add(a, b)

%{
 A block comment
 over three lines
%}

c = a + b;
end`)

	CountStats(&fileJob)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blanks got %d", fileJob.Blank)
	}
}

func TestCountStatsRacketBlockCommentCloses(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Racket",
	}

	fileJob.SetContent(`#lang racket

#|
 A block comment
 over three lines
|#

(define (add a b)
  (+ a b))`)

	CountStats(&fileJob)

	if fileJob.Lines != 9 {
		t.Errorf("Expected 9 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 4 {
		t.Errorf("Expected 4 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blanks got %d", fileJob.Blank)
	}
}

func TestCountStatsRacketNestedBlockComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Racket",
	}

	fileJob.SetContent(`#lang racket

#|
 outer #| inner |# still outer
|#

(displayln "hi")`)

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blanks got %d", fileJob.Blank)
	}
}
