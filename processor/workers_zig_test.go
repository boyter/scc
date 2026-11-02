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

// Zig only uses ' for character literals, so a '"' opened a string at the
// double quote that swallowed every line up to the next " in the file.
func TestCountStatsZigCharLiteralHoldingQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Zig",
	}

	fileJob.SetContent(`const std = @import("std");
const quote = '"';
// Prints a greeting.
// Nothing else.
pub fn main() void {
    std.debug.print("hi\n", .{});
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}

func TestCountStatsZigCharLiteralEscapes(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Zig",
	}

	fileJob.SetContent(`const apostrophe = '\'';
const backslash = '\\';
// Both literals above are closed.
const text =
    \\it's a multiline string
;
// Still a comment.`)

	CountStats(&fileJob)

	if fileJob.Lines != 7 {
		t.Errorf("Expected 7 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 5 {
		t.Errorf("Expected 5 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}
