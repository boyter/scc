// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

// The shared spine, tested directly.
//
// Most of it is covered by the differential test through C and Java, but two
// pieces are not: the nested block comment, which no language counted today
// has, and the bulk skip's line accounting, which the differential test can
// only confirm the sum of. A bug that moved a line from comment to code and
// another back would pass the differential and fail here.

func TestSkipToTerminator(t *testing.T) {
	for _, test := range []struct {
		name     string
		content  string
		from     int
		closer   string
		end      int
		newlines int
	}{
		{"closer on the first byte", "*/rest", 0, "*/", 0, 0},
		{"closer after text", "a b */", 0, "*/", 4, 0},
		{"newlines in front of the closer", "a\nb\nc*/", 0, "*/", 5, 2},
		{"no closer at all", "a\nb\nc", 0, "*/", -1, 2},
		{"starts part way in", "*/a\n*/", 2, "*/", 4, 1},
		{"empty tail", "abc", 3, "*/", -1, 0},
		{"closer split by the end", "a*", 0, "*/", -1, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			end, newlines := skipToTerminator([]byte(test.content), test.from, []byte(test.closer))
			if end != test.end || newlines != test.newlines {
				t.Errorf("got end=%d newlines=%d, want end=%d newlines=%d", end, newlines, test.end, test.newlines)
			}
		})
	}
}

// The line accounting of the bulk skip. A region jumped over in one go has to
// leave the tally exactly as a loop stopping on each of its newlines would.
func TestBulkLines(t *testing.T) {
	for _, test := range []struct {
		name  string
		state counterState
		n     int64
		want  counterTally
		after counterState
	}{
		{"nothing to account for", SMulticomment, 0, counterTally{}, SMulticomment},
		{"one comment line", SMulticomment, 1, counterTally{Lines: 1, Comment: 1}, SMulticomment},
		{"four comment lines", SMulticomment, 4, counterTally{Lines: 4, Comment: 4}, SMulticomment},
		// A block comment opened after code on the line ends that one line as
		// code and every line under it as comment.
		{"code then comment", SMulticommentCode, 1, counterTally{Lines: 1, Code: 1}, SMulticomment},
		{"code then three comment lines", SMulticommentCode, 4, counterTally{Lines: 4, Code: 1, Comment: 3}, SMulticomment},
		// A string is the state resetState leaves alone, so every line of one
		// counts as code.
		{"a string over lines", SString, 3, counterTally{Lines: 3, Code: 3}, SString},
	} {
		t.Run(test.name, func(t *testing.T) {
			var tally counterTally
			after := bulkLines(&tally, test.state, test.n)
			if tally != test.want {
				t.Errorf("tally is %+v, want %+v", tally, test.want)
			}
			if after != test.after {
				t.Errorf("state is %d, want %d", after, test.after)
			}
		})
	}
}

// A blank line inside a block comment is never blank. The state there is
// SMulticomment, which resets to itself, so the line counts as comment.
// SMulticommentBlank is the closing line of a comment and not an empty line
// inside one, which is the distinction that makes the bulk add legal at all.
func TestBulkLinesNeverCountsBlank(t *testing.T) {
	for _, state := range []counterState{SMulticomment, SMulticommentCode, SMulticommentBlank, SString, SCode} {
		var tally counterTally
		bulkLines(&tally, state, 5)
		if tally.Blank != 0 {
			t.Errorf("state %d counted %d blank lines in a jumped over region", state, tally.Blank)
		}
		if tally.Lines != 5 || tally.Code+tally.Comment != 5 {
			t.Errorf("state %d accounted %+v for five lines", state, tally)
		}
	}
}

// The nested block comment, which Rust, Swift, Kotlin and Scala have and none
// of the three languages counted today does. It cannot reach the differential
// test yet, so it is pinned here against the counter that is not nested.
func TestCounterCommentStateNesting(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		// Where the comment body starts, which is the byte after the opener.
		from int
		// nestedEnd and flatEnd are the last byte consumed with the nested
		// behaviour and without it.
		nestedEnd int
		flatEnd   int
	}{
		{
			name:      "an inner comment closes the outer one when nesting is off",
			content:   "/* a /* b */ c */ x",
			from:      2,
			nestedEnd: 16,
			flatEnd:   11,
		},
		{
			name:      "nothing nested behaves the same either way",
			content:   "/* a */ x",
			from:      2,
			nestedEnd: 6,
			flatEnd:   6,
		},
		{
			name:      "two levels deep",
			content:   "/* a /* b /* c */ d */ e */ x",
			from:      2,
			nestedEnd: 26,
			flatEnd:   16,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			content := []byte(test.content)
			endPoint := len(content) - 1

			var tally counterTally
			got, _ := counterCommentState(content, test.from, endPoint, SMulticomment, slashStarOpen, slashStarClose, true, &tally)
			if got != test.nestedEnd {
				t.Errorf("nested ended on %d (%q), want %d", got, content[got], test.nestedEnd)
			}

			tally = counterTally{}
			got, _ = counterCommentState(content, test.from, endPoint, SMulticomment, slashStarOpen, slashStarClose, false, &tally)
			if got != test.flatEnd {
				t.Errorf("flat ended on %d (%q), want %d", got, content[got], test.flatEnd)
			}
		})
	}
}

// An unterminated nested comment must still be linear, and must still hand the
// last newline back rather than swallowing it.
func TestCounterCommentStateNestedUnterminated(t *testing.T) {
	content := []byte("/* a /* b\n c\n d")
	endPoint := len(content) - 1

	var tally counterTally
	got, state := counterCommentState(content, 2, endPoint, SMulticomment, slashStarOpen, slashStarClose, true, &tally)

	last := strings.LastIndexByte(string(content[:endPoint]), '\n')
	if got != last {
		t.Errorf("ended on %d, want the last newline at %d", got, last)
	}
	if state != SMulticomment {
		t.Errorf("left state %d, want SMulticomment", state)
	}
	if tally.Lines != 1 || tally.Comment != 1 {
		t.Errorf("accounted %+v, want one comment line held back for the outer loop", tally)
	}
}

// floor is what stops a backwards read walking out of the region a counter
// owns. Every helper that reads backwards is held to it here, because the
// embedded language work that needs it has no test of its own yet.
func TestBackwardsReadsRespectTheFloor(t *testing.T) {
	content := []byte("abc\\\"def")

	if !wordStartsAt(content, 3, 3) {
		t.Error("the floor is a word boundary and wordStartsAt said otherwise")
	}
	if wordStartsAt(content, 3, 0) {
		t.Error("a byte after a letter is not a word start")
	}
	if got := byteBefore(content, 0, 0); got != 0 {
		t.Errorf("byteBefore read %q in front of the floor, want nothing", got)
	}
	if got := byteBefore(content, 4, 0); got != '\\' {
		t.Errorf("byteBefore read %q, want a backslash", got)
	}
	if got := byteBefore(content, 4, 4); got != 0 {
		t.Errorf("byteBefore read %q across the floor, want nothing", got)
	}
	if hasPrefixAt(content, 0, 3, "abc") {
		t.Error("hasPrefixAt matched in front of the floor")
	}
	if !hasPrefixAt(content, 5, 3, "def") {
		t.Error("hasPrefixAt failed to match above the floor")
	}
	if hasPrefixAt(content, 6, 0, "defg") {
		t.Error("hasPrefixAt matched past the end of the content")
	}

	// The quote at index 4 is escaped by the backslash at 3, but only while the
	// floor is below it. A region starting at the quote owns no backslash.
	if !escapedAt(content, 4, 0) {
		t.Error("a quote behind a backslash is escaped")
	}
	if escapedAt(content, 4, 4) {
		t.Error("a quote on the floor has nothing in front of it to escape it")
	}
}

// The blank run skip lands on the first byte that is not blank, which is the
// byte the loop has something to say about, and never past endPoint.
func TestSkipBlankRun(t *testing.T) {
	// index: 0 and 1 spaces, 2 a tab, 3 a carriage return, 4 a space, 5 an x,
	// 6 the newline that ends the line.
	content := []byte("  \t\r x\n")
	endPoint := len(content) - 1

	if got := skipBlankRun(content, 0, endPoint); got != 5 {
		t.Errorf("skipped to %d, want the x at 5", got)
	}
	if got := skipBlankRun(content, 4, endPoint); got != 5 {
		t.Errorf("a run of one moved to %d, want 5", got)
	}

	// A newline is deliberately not blank run whitespace, because a line ends
	// on it and the loop has to stop there.
	if got := skipBlankRun([]byte(" \n "), 0, 2); got != 1 {
		t.Errorf("skipped over a newline to %d, want to stop on it at 1", got)
	}

	// A run that reaches the end stops on endPoint rather than walking past it,
	// which is the byte the outer loop reads next.
	allBlank := []byte("      ")
	if got := skipBlankRun(allBlank, 0, len(allBlank)-1); got != len(allBlank)-1 {
		t.Errorf("skipped to %d, want to stop on endPoint at %d", got, len(allBlank)-1)
	}
}
