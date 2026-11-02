// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Protocol Buffers declared an empty quotes list despite using "..." string
// literals (syntax = "proto3"; option (x) = "..."; string field defaults).
// With no string rule an option value such as "see /* legacy block" was parsed
// as code: the embedded /* opened a block comment that ran to the next */ or
// to EOF, swallowing every following line as a comment. Adding the "..." quote
// rule lets the scanner treat the value as a string so the embedded /* is
// ignored.

func TestCountStatsProtobufStringNotComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Protocol Buffers",
	}

	fileJob.SetContent(`syntax = "proto3";
package demo;

option (desc) = "see /* legacy block";

message Foo {
  string name = 1;
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 8 {
		t.Errorf("Expected 8 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 6 {
		t.Errorf("Expected 6 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 blanks got %d", fileJob.Blank)
	}
}
