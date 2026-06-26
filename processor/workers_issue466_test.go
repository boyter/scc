// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"testing"
)

// https://github.com/boyter/scc/issues/466
// Complexity was never counted for languages whose tokens are not separated by
// whitespace (e.g. wenyan, where keywords are delimited by CJK punctuation).
// The left word-boundary guard required the preceding byte to be whitespace,
// which never holds for these languages. The thread blamed UTF-8/UTF-16
// encoding, but the byte sequences match fine; the boundary guard was the bug.
//
// A second bug surfaced by the same fixture: wenyan string literals are
// 「「 ... 」」 but the quote rule had start/end reversed, which corrupted the
// string regions (and the blank count) and would have let 遍 inside a string be
// miscounted as complexity.

func TestCountStatsIssue466Wenyan(t *testing.T) {
	ProcessConstants()

	content, err := os.ReadFile("../examples/language/wenyan.wy")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	fileJob := FileJob{Language: "wenyan"}
	fileJob.SetContent(string(content))
	CountStats(&fileJob)

	if fileJob.Lines != 12 {
		t.Errorf("Expected 12 lines got %d", fileJob.Lines)
	}
	if fileJob.Blank != 4 {
		t.Errorf("Expected 4 blank got %d", fileJob.Blank)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
	if fileJob.Code != 7 {
		t.Errorf("Expected 7 code got %d", fileJob.Code)
	}
	// 恆為是, 若, 等於 on line 5. 遍 on line 8 is inside a 「「...」」 string and
	// must NOT be counted, and 恆為是 must not double count as 恆為是 + 為是.
	if fileJob.Complexity != 3 {
		t.Errorf("Expected 3 complexity got %d", fileJob.Complexity)
	}
}
