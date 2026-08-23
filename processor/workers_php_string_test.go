// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// The PHP language entry declared its double quote as the two byte sequence
// \" so the scanner never entered the string state on a plain double quote.
// A glob such as "modules/*/config.php" was then parsed as code, the /*
// inside it opened a block comment, and every line to the next */ (or to
// EOF) was counted as a comment.

func TestCountStatsPhpDoubleQuotedGlobNotComment(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "PHP",
	}

	fileJob.SetContent(`<?php
$files = glob("$root/modules/*/config.php");
$out = [];
foreach ($files as $file) {
    $out[] = require $file;
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

func TestCountStatsPhpEscapedQuoteInString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "PHP",
	}

	fileJob.SetContent(`<?php
$a = "say \"hi\" /*";
$b = 1;
$c = 2;`)

	CountStats(&fileJob)

	if fileJob.Lines != 4 {
		t.Errorf("Expected 4 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 4 {
		t.Errorf("Expected 4 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 0 {
		t.Errorf("Expected 0 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 blanks got %d", fileJob.Blank)
	}
}
