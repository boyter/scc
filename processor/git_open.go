// SPDX-License-Identifier: MIT

package processor

import "github.com/go-git/go-git/v5"

// openRepository opens the git repository at, or above, path. It is the only
// place scc opens a repository, because both options below have to be set
// together and getting either wrong fails quietly rather than loudly.
//
// DetectDotGit walks up to find the repository, and follows the `gitdir:`
// pointer inside the .git *file* that a linked worktree has in place of a
// .git directory. That lands go-git in .git/worktrees/<name>, which holds the
// worktree's own HEAD but an empty refs/ — the real refs and objects live in
// the main repository, reachable only through the `commondir` file sitting
// next to that HEAD. EnableDotGitCommonDir is what makes go-git read it.
//
// With DetectDotGit alone a worktree therefore opens successfully and then
// resolves nothing: HEAD comes back as plumbing.ErrReferenceNotFound, which
// is indistinguishable from a freshly initialised repository, so every
// history report renders "no commits" against a repo with full history.
//
// Both options are inert outside a worktree. An ordinary checkout has no
// commondir file, so go-git skips the lookup entirely.
func openRepository(path string) (*git.Repository, error) {
	return git.PlainOpenWithOptions(path, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
}
