// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// makeShallowFixtureRepo builds a normal multi-commit repo and then mutates it
// on disk to look like a shallow clone: the oldest commit's object is removed
// and the next-oldest commit is recorded in .git/shallow as the new boundary.
// The boundary commit still carries its (now dangling) parent hash, so go-git's
// commit walker hits plumbing.ErrObjectNotFound when it tries to advance past
// the boundary — exactly what `git clone --depth N` produces in CI.
//
// Returns the repo path and the number of commits the walk should observe.
// That is two fewer than the original count: the removed root is gone, and the
// boundary commit is only diffable as the parent baseline of the commit above
// it — it is never observed in its own right because diffing it against its
// missing parent is the very thing we must not attempt.
func makeShallowFixtureRepo(t *testing.T) (dir string, observed int) {
	t.Helper()

	dir = makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\nfunc A() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\nfunc D() {}\n"},
	})

	repo, err := git.PlainOpen(dir)
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("head: %v", err)
	}
	iter, err := repo.Log(&git.LogOptions{From: head.Hash()})
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	var hashes []plumbing.Hash
	if err := iter.ForEach(func(c *object.Commit) error {
		hashes = append(hashes, c.Hash)
		return nil
	}); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if len(hashes) < 3 {
		t.Fatalf("need at least 3 commits to simulate a shallow boundary, got %d", len(hashes))
	}

	root := hashes[len(hashes)-1]     // oldest commit; its object we remove
	boundary := hashes[len(hashes)-2] // becomes the shallow boundary

	// Remove the root commit's loose object so resolving the boundary's parent
	// fails with ErrObjectNotFound.
	objPath := filepath.Join(dir, ".git", "objects", root.String()[:2], root.String()[2:])
	if err := os.Remove(objPath); err != nil {
		t.Fatalf("remove root object %s: %v", objPath, err)
	}

	// Record the boundary in .git/shallow, as a real shallow clone would.
	shallowPath := filepath.Join(dir, ".git", "shallow")
	if err := os.WriteFile(shallowPath, []byte(boundary.String()+"\n"), 0o644); err != nil {
		t.Fatalf("write .git/shallow: %v", err)
	}

	return dir, len(hashes) - 2 // minus the removed root, minus the unobserved boundary
}

// TestRunHistoryOnShallowClone is the regression test for the shallow-clone
// crash: before the fix runHistory returned "collect commits: object not
// found" whenever the walk depth reached the shallow boundary. It must now
// stop cleanly at the boundary and return the commits it could walk.
func TestRunHistoryOnShallowClone(t *testing.T) {
	saveDepth := HistoryDepth
	// Depth well past the number of commits on disk forces the walk to march
	// straight into the shallow boundary — the case that used to error.
	HistoryDepth = 1000
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir, observed := makeShallowFixtureRepo(t)

	cap := &captureObserver{}
	window, err := runHistory(dir, cap)
	if err != nil {
		t.Fatalf("runHistory on shallow clone errored: %v", err)
	}

	if window.Commits != observed {
		t.Fatalf("window.Commits = %d, want %d (commits walked up to the shallow boundary)", window.Commits, observed)
	}
	if len(cap.commits) != observed {
		t.Fatalf("observed %d commits, want %d", len(cap.commits), observed)
	}

	// Author data must still be populated for the walked commits.
	for i, c := range cap.commits {
		if c.Author == "" {
			t.Fatalf("commit %d has empty author", i)
		}
	}

	// And the diff/churn pipeline must have produced changes for at least one
	// commit (i.e. we didn't silently degrade to an empty window).
	sawChange := false
	for _, changes := range cap.changes {
		if len(changes) > 0 {
			sawChange = true
			break
		}
	}
	if !sawChange {
		t.Fatalf("expected populated churn data, got no FileChanges across %d commits", len(cap.changes))
	}
}
