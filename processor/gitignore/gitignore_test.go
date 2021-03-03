package gitignore

import (
	"testing"
)

// Note that this works on the following hence the bug
// https://golang.org/pkg/path/filepath/#Match
// it needs to include glob patterns and https://golang.org/pkg/path/filepath/#Glob might help

// https://github.com/zealic/xignore perhaps?
// https://github.com/zabawaba99/go-gitignore

func TestNewGitIgnore(t *testing.T) {
	ignore, _ := NewGitIgnore(".ignoretest")

	if ignore.Match("", false) != false {
		t.Error("empty should never match")
	}
}
