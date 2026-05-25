// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

func TestParseIgnoreFileSkipsCommentsAndBlanks(t *testing.T) {
	body := strings.NewReader("# comment\n\nfoo\n!bar\n   \n")
	pats := parseIgnoreFile(body, nil)
	if len(pats) != 2 {
		t.Fatalf("got %d patterns, want 2", len(pats))
	}
}

func TestHistoryIgnoreMatchesPattern(t *testing.T) {
	pat := gitignore.ParsePattern("vendor/", nil)
	h := &historyIgnore{matcher: gitignore.NewMatcher([]gitignore.Pattern{pat})}

	if !h.Match("vendor/foo.go", false) {
		t.Errorf("vendor/foo.go should match vendor/ pattern")
	}
	if h.Match("foo.go", false) {
		t.Errorf("foo.go should not match vendor/ pattern")
	}
}

func TestHistoryIgnoreNilSafe(t *testing.T) {
	var h *historyIgnore
	if h.Match("anything", false) {
		t.Errorf("nil matcher should match nothing")
	}
}

func TestSplitDomain(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{".", 0},
		{"a", 1},
		{"a/b/c", 3},
	}
	for _, c := range cases {
		got := splitDomain(c.in)
		if len(got) != c.want {
			t.Errorf("splitDomain(%q) = %v, want length %d", c.in, got, c.want)
		}
	}
}
