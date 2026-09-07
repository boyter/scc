// SPDX-License-Identifier: MIT

package processor

import "testing"

// countOne counts one file's worth of content and returns the four line counts.
func countOne(t *testing.T, language, content string) (lines, code, comment, blank int64) {
	t.Helper()
	ProcessConstants()

	fileJob := &FileJob{Language: language, Content: []byte(content), Bytes: int64(len(content))}
	CountStats(fileJob)

	return fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank
}

// TestEmptySpecialStringClosesItself covers the strings whose opener is also
// most of their closer. The cursor used to be left past the opening token, so
// the byte the caller looked at next was the one after the closer, and an empty
// string never closed: everything under it counted as string until something
// else happened to close it. A comment on the following line is what shows it.
func TestEmptySpecialStringClosesItself(t *testing.T) {
	tests := []struct {
		name     string
		language string
		content  string
	}{
		{"sql empty string", "SQL", "x = ''\n-- comment line\nSELECT 1;\n"},
		{"plsql empty string", "PL/SQL", "x := '';\n-- comment line\nnull;\n"},
		{"lua empty long string", "Lua", "s = [[]]\n-- comment\nprint(1)\n"},
		{"go empty raw string", "Go", "s := ``\n// comment\nvar x int\n"},
		{"python empty docstring", "Python", "s = ''''''\n# comment\nx = 1\n"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lines, code, comment, blank := countOne(t, tc.language, tc.content)
			if lines != 3 || code != 2 || comment != 1 || blank != 0 {
				t.Errorf("got lines=%d code=%d comment=%d blank=%d, want 3/2/1/0", lines, code, comment, blank)
			}
		})
	}
}

// TestRawStringLongestDelimiter pins the bound on the delimiter of a C++ raw
// string. Sixteen bytes is the longest the standard allows, so the bracket that
// ends it sits sixteen bytes in and the scan has to reach it; stopping a byte
// short left the string unrecognised and the quote it fell back to ran to the
// end of the file. Seventeen is not a raw string and must still be rejected.
func TestRawStringLongestDelimiter(t *testing.T) {
	sixteen := "aaaaaaaaaaaaaaaa"
	if len(sixteen) != 16 {
		t.Fatalf("delimiter is %d bytes", len(sixteen))
	}

	content := "const char *s = R\"" + sixteen + "(x)" + sixteen + "\";\n// comment\nint b;\n"
	lines, code, comment, _ := countOne(t, "C++", content)
	if lines != 3 || code != 2 || comment != 1 {
		t.Errorf("16 byte delimiter: got lines=%d code=%d comment=%d, want 3/2/1", lines, code, comment)
	}

	// A delimiter of fifteen bytes, comfortably inside the bound.
	fifteen := "aaaaaaaaaaaaaaa"
	content = "const char *s = R\"" + fifteen + "(x)" + fifteen + "\";\n// comment\nint b;\n"
	lines, code, comment, _ = countOne(t, "C++", content)
	if lines != 3 || code != 2 || comment != 1 {
		t.Errorf("15 byte delimiter: got lines=%d code=%d comment=%d, want 3/2/1", lines, code, comment)
	}
}

// TestUnterminatedBlockCommentIsLinear pins the position the comment states
// report when nothing closes the comment and no line ends. Returning the index
// they started from had the caller step on one byte and hand the whole tail
// back, so a block comment left open on a single line cost time in the square
// of its length: eleven seconds for 250KB, three minutes for a megabyte.
func TestUnterminatedBlockCommentIsLinear(t *testing.T) {
	ProcessConstants()

	content := append([]byte("/*"), make([]byte, 4096)...)
	for i := 2; i < len(content); i++ {
		content[i] = 'a'
	}

	fileJob := &FileJob{Content: content}
	endPoint := len(content)

	if got, _ := cCommentState(fileJob, 2, endPoint, SMulticomment); got != endPoint-1 {
		t.Errorf("cCommentState reported %d on exhaustion, want %d", got, endPoint-1)
	}
	if got, _ := javaCommentState(fileJob, 2, endPoint, SMulticomment); got != endPoint-1 {
		t.Errorf("javaCommentState reported %d on exhaustion, want %d", got, endPoint-1)
	}

	// An index already at or past the end must not be moved backwards, which
	// would walk the outer loop over the same bytes forever.
	if got, _ := cCommentState(fileJob, endPoint, endPoint, SMulticomment); got != endPoint {
		t.Errorf("cCommentState moved an exhausted index to %d, want %d", got, endPoint)
	}
}

// TestWorkerPoolFloor covers the flags that size the pools. A count of zero
// started no goroutines, so nothing drained the queue, the stage below saw it
// closed, and the run reported a tree of no files and exited successfully.
func TestWorkerPoolFloor(t *testing.T) {
	for _, workers := range []int{-1, 0, 1, 8} {
		want := workers
		if want < 1 {
			want = 1
		}
		if got := atLeastOneWorker(workers); got != want {
			t.Errorf("atLeastOneWorker(%d) = %d, want %d", workers, got, want)
		}
	}
}
