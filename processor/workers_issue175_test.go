// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// https://github.com/boyter/scc/issues/175
// Raw/verbatim strings contain characters that look like string terminators,
// comment starts or escapes. Without dedicated quote rules the embedded "
// prematurely closes the string, the leftover quote opens a phantom string and
// the real comment that follows is swallowed and counted as code.

func TestCountStatsIssue175CPlusPlusRawString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C++",
	}

	fileJob.SetContent(`/* 6 lines 3 code 2 comments 1 blanks */
int main() {
  const char* s = R"(has a " and \n inside)";

  // a real comment
}`)

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

func TestCountStatsIssue175RustRawString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Rust",
	}

	fileJob.SetContent(`// 6 lines 3 code 2 comments 1 blanks
fn main() {
    let s = r#"has a " and /* inside"#;

    // a real comment
}`)

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
