// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// makeFixtureRepo initialises a real on-disk repo in a temp dir and lets the
// caller commit a sequence of (path -> content) snapshots. Returns the repo
// path so tests can pass it to runHistory.
//
// We use PlainInit (not an in-memory storer) because scc's classifier needs
// to detect languages from file names — same as a normal scc run — and the
// engine itself only talks to go-git, so no shell-out happens.
func makeFixtureRepo(t *testing.T, commits []map[string]string) string {
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
	for i, snap := range commits {
		for path, content := range snap {
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
		_, err := wt.Commit("commit "+itoa(i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Author " + itoa(i%2),
				Email: "author" + itoa(i%2) + "@example.com",
				When:  when.Add(time.Duration(i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	return dir
}

// captureObserver records observe calls so tests can assert on them.
type captureObserver struct {
	commits  []CommitInfo
	changes  [][]FileChange
	window   HistoryWindow
	snapshot HeadSnapshot
}

func (c *captureObserver) Observe(info CommitInfo, changes []FileChange) {
	c.commits = append(c.commits, info)
	c.changes = append(c.changes, changes)
}

func (c *captureObserver) Finalise(w HistoryWindow, s HeadSnapshot) {
	c.window = w
	c.snapshot = s
}

func TestRunHistoryWalksOldestFirstAndCollectsChanges(t *testing.T) {
	// Set depth/flags to defaults this test cares about.
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\n\nfunc A() {}\n"},
		{"a.go": "package a\n\nfunc A() {}\nfunc B() {}\n"},
		{"a.go": "package a\n\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"},
	})

	cap := &captureObserver{}
	window, err := runHistory(dir, cap)
	if err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if want := 3; window.Commits != want {
		t.Fatalf("window.Commits = %d, want %d", window.Commits, want)
	}
	if len(cap.commits) != 3 {
		t.Fatalf("observed %d commits, want 3", len(cap.commits))
	}
	// Oldest-first ordering: each commit's When should be >= previous.
	for i := 1; i < len(cap.commits); i++ {
		if cap.commits[i].When.Before(cap.commits[i-1].When) {
			t.Fatalf("commits not oldest-first: commit %d %s before %s",
				i, cap.commits[i].When, cap.commits[i-1].When)
		}
	}
	// Snapshot should contain a.go in HEAD.
	if _, ok := cap.snapshot.Files["a.go"]; !ok {
		t.Fatalf("HEAD snapshot missing a.go; got %v", cap.snapshot.Files)
	}
}

func TestRunHistoryDepthCap(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 2
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\nfunc A() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\nfunc D() {}\n"},
	})

	cap := &captureObserver{}
	if _, err := runHistory(dir, cap); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	if len(cap.commits) != 2 {
		t.Fatalf("with --depth=2 observed %d commits, want 2", len(cap.commits))
	}
}

func TestRunHistorySkipsBinaryAndUnknown(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 10
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeFixtureRepo(t, []map[string]string{
		{
			"main.go":            "package main\nfunc main() {}\n",
			"weird-extension.xx": "no language for this\n",
			"data.bin":           "\x00\x01\x02\x03\x00binary",
		},
	})

	cap := &captureObserver{}
	if _, err := runHistory(dir, cap); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	if len(cap.changes) != 1 {
		t.Fatalf("expected one Observe call, got %d", len(cap.changes))
	}
	// Only main.go should survive.
	paths := []string{}
	for _, fc := range cap.changes[0] {
		paths = append(paths, fc.Path)
	}
	if len(paths) != 1 || paths[0] != "main.go" {
		t.Fatalf("expected only main.go, got %v", paths)
	}
}

// TestRunHistoryWithoutGitInPath confirms the engine has no shell-out to the
// git binary by running the walk with PATH stripped to /nonexistent.
func TestRunHistoryWithoutGitInPath(t *testing.T) {
	savePath := os.Getenv("PATH")
	saveDepth := HistoryDepth
	HistoryDepth = 10
	t.Cleanup(func() {
		os.Setenv("PATH", savePath)
		HistoryDepth = saveDepth
	})

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\nfunc A() {}\n"},
		{"a.go": "package a\nfunc A() { if true {} }\n"},
	})

	// Strip PATH so any accidental exec.Command would fail.
	if err := os.Setenv("PATH", "/nonexistent"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	cap := &captureObserver{}
	if _, err := runHistory(dir, cap); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	if len(cap.commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(cap.commits))
	}
}

func TestLineCountHelper(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"a", 1},
		{"a\n", 1},
		{"a\nb", 2},
		{"a\nb\n", 2},
		{"\n", 1},
	}
	for _, c := range cases {
		if got := lineCount(c.in); got != c.want {
			t.Errorf("lineCount(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
