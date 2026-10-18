// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// makeMergeFixtureRepo builds the four-commit shape from issue #783: a root
// commit on the default branch, a side branch that adds a.go and b.go, a
// further commit on the default branch, and a --no-ff merge of the side
// branch. The first-parent chain is three commits (root, main work, merge);
// the full reachable set is four because the side-branch commit is only
// reachable through the merge's second parent.
func makeMergeFixtureRepo(t *testing.T) string {
	t.Helper()
	ProcessConstants()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	when := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	sig := func(name string, hour int) *object.Signature {
		return &object.Signature{
			Name:  name,
			Email: name + "@example.com",
			When:  when.Add(time.Duration(hour) * time.Hour),
		}
	}
	write := func(path, content string) {
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}

	write("main.go", "package main\n\nfunc main() {\n}\n")
	if _, err := wt.Commit("init", &git.CommitOptions{Author: sig("Bob", 0)}); err != nil {
		t.Fatalf("commit init: %v", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("feature"),
		Create: true,
	}); err != nil {
		t.Fatalf("checkout feature: %v", err)
	}
	aContent := "package main\n\nfunc a() int {\n\tif true {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"
	bContent := "package main\n\nfunc b() int {\n\treturn 2\n}\n"
	write("a.go", aContent)
	write("b.go", bContent)
	featureHash, err := wt.Commit("feature", &git.CommitOptions{Author: sig("Alice", 1)})
	if err != nil {
		t.Fatalf("commit feature: %v", err)
	}

	if err := wt.Checkout(&git.CheckoutOptions{Branch: plumbing.NewBranchReferenceName("master")}); err != nil {
		t.Fatalf("checkout master: %v", err)
	}
	write("main.go", "package main\n\nfunc main() {\n\tprintln(1)\n}\n")
	mainHash, err := wt.Commit("main work", &git.CommitOptions{Author: sig("Bob", 2)})
	if err != nil {
		t.Fatalf("commit main work: %v", err)
	}

	// Re-stage the branch files so the merge commit's tree is the union of
	// both sides, then record both parents explicitly: this is --no-ff.
	write("a.go", aContent)
	write("b.go", bContent)
	if _, err := wt.Commit("Merge feature", &git.CommitOptions{
		Author:  sig("Bob", 3),
		Parents: []plumbing.Hash{mainHash, featureHash},
	}); err != nil {
		t.Fatalf("commit merge: %v", err)
	}

	return dir
}

// TestRunHistoryWalksFirstParentChainOnly is the regression test for issue
// #783: runHistory walked every reachable commit, so a merged branch's work
// was observed once on the branch and again through the merge's first-parent
// diff. The walk must follow first parents only, matching the README's
// `git log --first-parent` semantics.
func TestRunHistoryWalksFirstParentChainOnly(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeMergeFixtureRepo(t)

	cap := &captureObserver{}
	window, err := runHistory(dir, cap)
	if err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	// init, main work, merge. The feature commit is off the first-parent chain.
	if want := 3; window.Commits != want {
		t.Fatalf("window.Commits = %d, want %d (first-parent commits only)", window.Commits, want)
	}
	if len(cap.commits) != window.Commits {
		t.Fatalf("observed %d commits, want %d", len(cap.commits), window.Commits)
	}
}

// TestHotspotsCountsMergedFileOnce pins the user-visible symptom: a file
// introduced on a merged branch was reported with two commits instead of one.
func TestHotspotsCountsMergedFileOnce(t *testing.T) {
	saveDepth, saveFormat := HistoryDepth, Format
	HistoryDepth, Format = 100, "tabular"
	t.Cleanup(func() { HistoryDepth, Format = saveDepth, saveFormat })

	dir := makeMergeFixtureRepo(t)

	obs := newHotspotsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	for _, r := range obs.records {
		if r.File == "a.go" {
			if r.Commits != 1 {
				t.Fatalf("a.go Commits = %d, want 1 (merged branch work counted once)", r.Commits)
			}
			return
		}
	}
	t.Fatalf("a.go missing from hotspots records: %v", obs.records)
}
