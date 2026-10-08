// SPDX-License-Identifier: MIT
package processor

import (
	"strings"
	"testing"
)

func TestCOBOLFixedComments(t *testing.T) {
	content := "000100* Comment\n000200/ Page comment\n000300 IDENTIFICATION DIVISION.\n000400     COMPUTE X = A * B / C.\n000500\n"
	lines, code, comment, blank := countOne(t, "COBOL", content)
	if lines != 5 || code != 2 || comment != 2 || blank != 1 {
		t.Fatalf("got %d/%d/%d/%d, want 5/2/2/1", lines, code, comment, blank)
	}
}

func TestCOBOLSourceFormats(t *testing.T) {
	for _, tc := range []struct {
		name, content        string
		code, comment, blank int64
	}{
		{"literals and inline comments", "       DISPLAY 'not *> a comment'.\n       DISPLAY \"It\"\"s text\". *> trailing\n       *> comment\n", 2, 1, 0},
		{"debug and continuation indicators", "000100D    DISPLAY 'DEBUG'.\n000200-    MOVE 1 TO X.\n", 2, 0, 0},
		{"free and fixed switches", "       >>SOURCE FORMAT IS FREE\n*> free comment\nCOMPUTE X = A * B / C.\n>>SOURCE FORMAT FIXED\n000100/ fixed comment\n", 3, 2, 0},
		{"commented directive", "000100* >>SOURCE FORMAT FREE\n000200/ still fixed\n", 0, 2, 0},
		{"identification area only", strings.Repeat(" ", 72) + "TAG\n000100" + strings.Repeat(" ", 66) + "TAG\n", 0, 0, 2},
		{"continued literal then comment", "000100     DISPLAY 'continued\n000200* comment between records\n000300-    'text'.\n", 2, 1, 0},
		{"BOM and CRLF", "\xef\xbb\xbf000100* comment\r\n000200/ comment\r\n000300", 0, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, code, comment, blank := countOne(t, "COBOL", tc.content)
			if code != tc.code || comment != tc.comment || blank != tc.blank {
				t.Fatalf("got %d/%d/%d, want %d/%d/%d", code, comment, blank, tc.code, tc.comment, tc.blank)
			}
		})
	}
}
