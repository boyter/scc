// SPDX-License-Identifier: MIT

package processor

import "testing"

// Coq uses doubled quotes inside strings; backslashes are ordinary bytes.
func TestCountStatsCoqStrings(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Coq"}
	fileJob.SetContent(`(* 16 lines 8 code 5 comments 3 blanks *)
Definition backslash := "\".
(* A backslash does not escape the closing quote. *)

Definition backslashes := "\\\".
(* Neither does an odd run of backslashes. *)

Definition quote := """".
Definition quoted := "a ""(* still a string *)"" \".
(* Doubled quotes and backslashes leave this a comment. *)

Definition multiline := "first ""
(* still a string *)
"" last\".
(* The multiline string has ended. *)
Definition empty := "".
`)
	CountStats(&fileJob)

	if fileJob.Lines != 16 || fileJob.Code != 8 || fileJob.Comment != 5 || fileJob.Blank != 3 {
		t.Errorf("got lines/code/comments/blanks = %d/%d/%d/%d, want 16/8/5/3",
			fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank)
	}
}
