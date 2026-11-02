// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"testing"
)

// makeWorktreeFixtureRepo builds a normal multi-commit repo and then fabricates
// the on-disk layout that `git worktree add` produces for a linked worktree:
//
//	<wt>/.git                         file, "gitdir: <repo>/.git/worktrees/wt"
//	<repo>/.git/worktrees/wt/HEAD     the worktree's own HEAD
//	<repo>/.git/worktrees/wt/commondir  "../.." — points back at the real store
//	<repo>/.git/worktrees/wt/gitdir   absolute path of the <wt>/.git file
//
// The refs and objects stay in the main repository; the worktree directory has
// no refs/ of its own, which is the whole point — resolving HEAD from inside it
// is only possible by following commondir.
//
// Built by hand rather than by shelling out to `git worktree add` so the test
// needs no git binary, matching how makeShallowFixtureRepo fabricates a shallow
// clone. Only the git metadata is created: runHistory reads commits out of the
// object store and never looks at the checked-out files, so there is nothing to
// gain from populating the working tree.
//
// Returns the worktree path and the number of commits its history holds.
func makeWorktreeFixtureRepo(t *testing.T, commits []map[string]string) (wtDir string, count int) {
	t.Helper()

	repo := makeFixtureRepo(t, commits)
	wtDir = filepath.Join(t.TempDir(), "wt")
	if err := os.MkdirAll(wtDir, 0o755); err != nil {
		t.Fatalf("mkdir worktree: %v", err)
	}

	adminDir := filepath.Join(repo, ".git", "worktrees", "wt")
	if err := os.MkdirAll(adminDir, 0o755); err != nil {
		t.Fatalf("mkdir worktree admin dir: %v", err)
	}

	dotGitFile := filepath.Join(wtDir, ".git")
	if err := os.WriteFile(dotGitFile, []byte("gitdir: "+adminDir+"\n"), 0o644); err != nil {
		t.Fatalf("write worktree .git file: %v", err)
	}

	// The worktree's HEAD is a copy of the main repo's: a symbolic ref naming a
	// branch whose ref file lives in the common dir, never here.
	head, err := os.ReadFile(filepath.Join(repo, ".git", "HEAD"))
	if err != nil {
		t.Fatalf("read repo HEAD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "HEAD"), head, 0o644); err != nil {
		t.Fatalf("write worktree HEAD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatalf("write commondir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(adminDir, "gitdir"), []byte(dotGitFile+"\n"), 0o644); err != nil {
		t.Fatalf("write gitdir: %v", err)
	}

	return wtDir, len(commits)
}

var worktreeFixtureCommits = []map[string]string{
	{"a.go": "package a\nfunc A() {}\n"},
	{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
	{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"},
}

// TestRunHistoryInLinkedWorktree is the regression test for issue #765: every
// history report rendered "no commits" when run inside a git worktree. Opening
// with DetectDotGit alone lands go-git in .git/worktrees/<name>, where refs/ is
// empty, so HEAD failed with ErrReferenceNotFound and was mistaken for a
// freshly initialised repo. openRepository now sets EnableDotGitCommonDir so
// the shared refs are reachable.
func TestRunHistoryInLinkedWorktree(t *testing.T) {
	dir, want := makeWorktreeFixtureRepo(t, worktreeFixtureCommits)

	cap := &captureObserver{}
	window, err := runHistory(dir, cap)
	if err != nil {
		t.Fatalf("runHistory in worktree errored: %v", err)
	}

	if window.Commits != want {
		t.Fatalf("window.Commits = %d, want %d (a worktree sees the full history of its repo)", window.Commits, want)
	}
	if len(cap.commits) != want {
		t.Fatalf("observed %d commits, want %d", len(cap.commits), want)
	}
	if window.Head.IsZero() {
		t.Fatal("window.Head is zero, HEAD did not resolve through commondir")
	}
	if len(cap.snapshot.Files) == 0 {
		t.Fatal("HEAD snapshot is empty, the tree did not resolve through commondir")
	}
}

// TestResolveCouplingTargetInLinkedWorktree covers the second opener: --coupling
// verifies its target against HEAD before walking, so it opened the repo
// independently of runHistory.
//
// The bogus-target half is what actually discriminates. On an unresolvable HEAD
// the function takes its empty-repo path and hands back the cleaned input
// unchanged, so a valid target still "resolves" by accident — but a typo that
// should be rejected sails through too. In a worktree that meant --coupling-for
// silently accepted any path and then reported nothing for it.
func TestResolveCouplingTargetInLinkedWorktree(t *testing.T) {
	dir, _ := makeWorktreeFixtureRepo(t, worktreeFixtureCommits)

	got, err := resolveCouplingTarget(dir, "a.go")
	if err != nil {
		t.Fatalf("resolveCouplingTarget in worktree errored: %v", err)
	}
	if got != "a.go" {
		t.Fatalf("resolveCouplingTarget = %q, want %q", got, "a.go")
	}

	if _, err := resolveCouplingTarget(dir, "nope.go"); err == nil {
		t.Fatal("resolveCouplingTarget accepted a path absent from HEAD, want an error")
	}
}

// TestDetectGitInLinkedWorktree pins the third opener. This one already
// returned true before the fix — a worktree opens fine, it just resolves
// nothing — which is exactly why the failure surfaced as a plausible empty
// table instead of "not a git repository".
func TestDetectGitInLinkedWorktree(t *testing.T) {
	dir, _ := makeWorktreeFixtureRepo(t, worktreeFixtureCommits)

	if !detectGit(dir) {
		t.Fatal("detectGit = false in a linked worktree, want true")
	}
}
