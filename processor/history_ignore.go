// SPDX-License-Identifier: MIT

package processor

import (
	"bufio"
	"io"
	"path"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// historyIgnore wraps a gitignore.Matcher built from .ignore / .sccignore
// files found in the HEAD tree. .gitignore is already applied by git itself,
// so files it excludes never appear in the tree — only the scc/ripgrep
// conventions need extra handling.
type historyIgnore struct {
	matcher gitignore.Matcher
}

func (h *historyIgnore) Match(p string, isDir bool) bool {
	if h == nil || h.matcher == nil {
		return false
	}
	parts := strings.Split(p, "/")
	return h.matcher.Match(parts, isDir)
}

// buildHistoryIgnore scans the HEAD tree for .ignore and .sccignore blobs,
// parses them, and produces a matcher. Respects the existing --no-ignore
// (Ignore) and --no-scc-ignore (SccIgnore) flag globals.
func buildHistoryIgnore(repo *git.Repository, head plumbing.Hash) (*historyIgnore, error) {
	commit, err := repo.CommitObject(head)
	if err != nil {
		return nil, err
	}
	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var patterns []gitignore.Pattern
	err = tree.Files().ForEach(func(f *object.File) error {
		if f.Mode == filemode.Dir || f.Mode == filemode.Submodule || f.Mode == filemode.Symlink {
			return nil
		}

		base := path.Base(f.Name)
		switch {
		case base == ".ignore" && !Ignore:
		case base == ".sccignore" && !SccIgnore:
		default:
			return nil
		}

		reader, err := f.Reader()
		if err != nil {
			return nil
		}
		defer reader.Close()

		domain := splitDomain(path.Dir(f.Name))
		patterns = append(patterns, parseIgnoreFile(reader, domain)...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(patterns) == 0 {
		return &historyIgnore{}, nil
	}
	return &historyIgnore{matcher: gitignore.NewMatcher(patterns)}, nil
}

func splitDomain(dir string) []string {
	if dir == "" || dir == "." {
		return nil
	}
	return strings.Split(dir, "/")
}

func parseIgnoreFile(r io.Reader, domain []string) []gitignore.Pattern {
	var out []gitignore.Pattern
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := strings.TrimRight(scan.Text(), "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		out = append(out, gitignore.ParsePattern(line, domain))
	}
	return out
}
