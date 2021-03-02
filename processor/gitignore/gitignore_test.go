package gitignore

import (
	"testing"
)

// Note that this works on the following hence the bug
// https://golang.org/pkg/path/filepath/#Match
// it needs to include glob patterns and https://golang.org/pkg/path/filepath/#Glob might help

func TestNewGitIgnore(t *testing.T) {
	ignore, _ := NewGitIgnore(".ignoretest")

	if ignore.Match("", false) != false {
		t.Error("empty should never match")
	}
}
