// SPDX-License-Identifier: MIT

/*
Package gitignore provides an interface for parsing .gitignore files,
either individually, or within a repository, and
matching paths against the retrieved patterns. Path matching is done using
fnmatch as specified by git (see https://git-scm.com/docs/gitignore), with
support for recursive matching via the "**" pattern.
*/
package gitignore
