// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
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

			got, _, _ := counterNestedCommentState(content, test.from, endPoint, SMulticomment, slashStarOpen, slashStarClose, 1)
			if got != test.nestedEnd {
				t.Errorf("nested ended on %d (%q), want %d", got, content[got], test.nestedEnd)
			}

			var tally counterTally
			got, _ = counterCommentState(content, test.from, endPoint, SMulticomment, slashStarClose, &tally)
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

	// The nested state stops on the first newline, not the last: the depth it
	// hands back has to belong to the position it hands back, or the caller
	// re-reads the tokens it already passed.
	got, state, depth := counterNestedCommentState(content, 2, endPoint, SMulticomment, slashStarOpen, slashStarClose, 1)

	first := strings.IndexByte(string(content[:endPoint]), '\n')
	if got != first {
		t.Errorf("ended on %d, want the first newline at %d", got, first)
	}
	if state != SMulticomment {
		t.Errorf("left state %d, want SMulticomment", state)
	}
	if depth != 2 {
		t.Errorf("left depth %d, want 2: the line opened a second comment and closed neither", depth)
	}
}

// The depth has to survive being handed back and in again, which is the whole
// reason the nested state carries it. This walks the shape the fuzzer found:
// a nested opener, then a newline, then a single closer that must NOT end the
// outer comment.
func TestCounterNestedCommentStateCarriesDepth(t *testing.T) {
	// /*00000/*0\n*/0 — the closer on the second line brings the depth from two
	// to one, so the comment is still open and the last line is still comment.
	content := []byte("/*00000/*0\n*/0")
	endPoint := len(content) - 1

	index, state, depth := counterNestedCommentState(content, 2, endPoint, SMulticomment, slashStarOpen, slashStarClose, 1)
	if depth != 2 {
		t.Fatalf("first line left depth %d, want 2", depth)
	}
	if content[index] != '\n' {
		t.Fatalf("first line ended on %q, want the newline", content[index])
	}

	_, state, depth = counterNestedCommentState(content, index+1, endPoint, state, slashStarOpen, slashStarClose, depth)
	if depth != 1 {
		t.Errorf("second line left depth %d, want 1: one closer cannot end two comments", depth)
	}
	if state != SMulticomment {
		t.Errorf("second line left state %d, want SMulticomment", state)
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

// diffCorpus reads every file of one extension under the tree named by an env
// var both ways and requires the counts to be identical. The C, Java and
// JavaScript tests each grew their own copy of this walk; the counters added
// after them share one.
//
// It is skipped rather than failed when the tree is not there, since a corpus is
// a checkout and not something the repository carries.
func diffCorpus(t *testing.T, language, envVar, extension string) {
	t.Helper()
	ProcessConstants()

	if testing.Short() {
		t.Skip("walks a whole source tree")
	}

	corpus := os.Getenv(envVar)
	if corpus == "" {
		t.Skipf("set %s to a tree of real %s", envVar, language)
	}

	// The live heap here is the trie built for every one of the languages, which
	// is large and all pointers, so the default collector rescans it on every
	// cycle and the walk below spends its time in the garbage collector rather
	// than in either counter.
	defer debug.SetGCPercent(debug.SetGCPercent(1600))

	limit := 0
	if v := os.Getenv("SCC_DIFF_LIMIT"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	checked := 0
	disagreed := 0
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, extension) {
			return nil
		}

		if limit != 0 && checked >= limit {
			return filepath.SkipAll
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		fast, generic := countBothWays(t, language, content)
		checked++
		if countsDiffer(fast, generic) {
			disagreed++
			if disagreed <= 5 {
				compareCounts(t, language, path, fast, generic)
			}
		}

		return nil
	})

	if checked == 0 {
		t.Skipf("no %s found in the corpus", language)
	}

	t.Logf("checked %d files, %d disagreed", checked, disagreed)
}

// benchmarkCorpus is the shared shape of every counter's corpus benchmark: read
// a bounded number of real files once, then measure only the counting.
func benchmarkCorpus(b *testing.B, language, envVar, extension string, specialised bool) {
	b.Helper()
	ProcessConstants()

	corpus := os.Getenv(envVar)
	if corpus == "" {
		b.Skipf("set %s to a tree of real %s", envVar, language)
	}

	var files [][]byte
	var total int64
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, extension) || len(files) >= 400 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, content)
		total += int64(len(content))

		return nil
	})

	if len(files) == 0 {
		b.Skipf("no %s found in the corpus", language)
	}

	previous := SpecialisedCounters
	SpecialisedCounters = specialised
	defer func() { SpecialisedCounters = previous }()

	b.SetBytes(total)
	b.ResetTimer()

	for b.Loop() {
		for _, content := range files {
			fileJob := FileJob{Language: language, Content: content, Bytes: int64(len(content))}
			CountStats(&fileJob)
		}
	}
}

// diffCorpusRegex is diffCorpus for the two languages that carry M16, the
// regular expression disambiguation, which is the one place a counter is meant
// to disagree with the generic loop.
//
// M16 fires wherever a pattern holds a quote, a comment opener or a complexity
// token, which is far more than the two LineJudge inputs it is named by, so
// "the counter agrees with the generic loop on every file" is only testable
// with the difference taken out. It runs twice: once with the fix off, where
// exact agreement is still required and any disagreement is a bug, and once
// with it on, where the divergences are counted and reported.
func diffCorpusRegex(t *testing.T, language, envVar, extension string) {
	t.Helper()
	diffCorpusToggle(t, language, envVar, extension, &ecmaRegexLiterals, "regex literals")
}

// diffCorpusToggle is diffCorpus for a counter that carries a deliberate
// divergence, which is a switch it can be turned off at.
//
// The corpus is walked twice. With the divergence off the counter has nothing
// left to disagree about and exact agreement is required, which is what keeps
// the differential worth anything. With it on the disagreements are counted and
// reported rather than failed, every one of them having been read by hand and
// found to be the generic loop being wrong.
func diffCorpusToggle(t *testing.T, language, envVar, extension string, toggle *bool, name string) {
	t.Helper()
	ProcessConstants()

	if testing.Short() {
		t.Skip("walks a whole source tree")
	}

	corpus := os.Getenv(envVar)
	if corpus == "" {
		t.Skipf("set %s to a tree of real %s", envVar, language)
	}

	defer debug.SetGCPercent(debug.SetGCPercent(1600))

	limit := 0
	if v := os.Getenv("SCC_DIFF_LIMIT"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	for _, on := range []bool{false, true} {
		previous := *toggle
		*toggle = on

		checked := 0
		disagreed := 0
		_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, extension) {
				return nil
			}

			if limit != 0 && checked >= limit {
				return filepath.SkipAll
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			fast, generic := countBothWays(t, language, content)
			checked++
			if countsDiffer(fast, generic) {
				disagreed++
				if !on && disagreed <= 5 {
					compareCounts(t, language, path, fast, generic)
				}
			}

			return nil
		})

		*toggle = previous

		if checked == 0 {
			t.Skipf("no %s found in the corpus", language)
		}

		if on {
			t.Logf("with %s on: checked %d files, %d diverged", name, checked, disagreed)
		} else {
			t.Logf("with %s off: checked %d files, %d disagreed", name, checked, disagreed)
			if disagreed != 0 {
				t.Errorf("%d files disagree with the generic loop for a reason that is not the %s fix", disagreed, name)
			}
		}
	}
}
