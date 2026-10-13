// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// A Zig multiline string line runs from \\ to the end of its line. The newline
// that closes it was never matched, so the string carried on to the end of the
// file and every comment after it counted as code.
func TestCountStatsZigMultilineString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Zig",
	}

	fileJob.SetContent(`const text =
    \\first line with a " in it
    \\
;
// Still a comment.
// So is this.`)

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}
