// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Terraform (an HCL dialect) used to ship with an empty quotes list, unlike the
// HCL language entry which declares the "..." string literal. As a result a
// string such as "logs/*" was parsed as code: the /* inside opened a block
// comment that ran to the next */ (or to EOF), swallowing every subsequent
// line as a comment. Adding the "..." quote rule lets the scanner treat the
// glob as a string so the embedded /* is ignored.

func TestCountStatsTerraformGlobStringNotComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Terraform",
	}

	fileJob.SetContent(`variable "glob" {
  default = "logs/*"
}
output "result" {
  value = "still real code"
}`)

	CountStats(&fileJob)

	if fileJob.Lines != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 6 {
		t.Errorf("Expected 6 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}
