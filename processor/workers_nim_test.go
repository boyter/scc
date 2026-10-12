// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Nim has #[ ]# block comments and ##[ ]## documentation block comments, and
// both nest. Without them every line inside a block comment counted as code.
func TestCountStatsNimBlockComments(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Nim",
	}

	fileJob.SetContent(`## Module documentation
#[ outer
  #[ inner ]#
  proc old() = discard
]#
let a = 1 #[ inline ]# + 2
let b = "#[ not a comment"
##[ documentation
  #[ nested ]#
]##
let c = 3`)

	CountStats(&fileJob)

	if fileJob.Lines != 11 {
		t.Errorf("Expected 11 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 3 {
		t.Errorf("Expected 3 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 8 {
		t.Errorf("Expected 8 comments got %d", fileJob.Comment)
	}
}
