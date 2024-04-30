# go-gitignore

Package `go-gitignore` provides an interface for parsing `.gitignore` files,
either individually, or within a repository, and
matching paths against the retrieved patterns. Path matching is done using
[fnmatch](https://github.com/danwakefield/fnmatch) as specified by
[git](https://git-scm.com/docs/gitignore), with
support for recursive matching via the `**` pattern.

```go
import "github.com/denormal/go-gitignore"

// match a file against a particular .gitignore
ignore, err := gitignore.NewFromFile("/my/.gitignore")
if err != nil {
    panic(err)
}
match := ignore.Match("/my/file/to.check")
if match != nil {
    if match.Ignore() {
        return true
    }
}

// or match against a repository
//  - here we match a directory path relative to the repository
ignore, err := gitignore.NewRepository( "/my/git/repository" )
if err != nil {
    panic(err)
}
match := ignore.Relative("src/examples", true)
if match != nil {
    if match.Include() {
        fmt.Printf(
            "include src/examples/ because of pattern %q at %s",
			match, match.Position(),
		)
    }
}

// if it's not important whether a path matches, but whether it is
// ignored or included...
if ignore.Ignore("src/test") {
    fmt.Println("ignore src/test")
} else if ignore.Include("src/github.com") {
    fmt.Println("include src/github.com")
}
```

For more information see `godoc github.com/denormal/go-gitignore`.

## Patterns

`go-gitignore` supports the same `.gitignore` pattern format and matching rules as defined by [git](https://git-scm.com/docs/gitignore):

* A blank line matches no files, so it can serve as a separator for readability.

* A line starting with `#` serves as a comment. Put a backslash `\` in front of the first hash for patterns that begin with a hash.

* Trailing spaces are ignored unless they are quoted with backslash `\`.

* An optional prefix `!` which negates the pattern; any matching file excluded by a previous pattern will become included again. It is not possible to re-include a file if a parent directory of that file is excluded. Git doesn’t list excluded directories for performance reasons, so any patterns on contained files have no effect, no matter where they are defined. Put a backslash `\` in front of the first `!` for patterns that begin with a literal `!`, for example, `\!important!.txt`.

* If the pattern ends with a slash, it is removed for the purpose of the following description, but it would only find a match with a directory. In other words, `foo/` will match a directory foo and paths underneath it, but will not match a regular file or a symbolic link `foo` (this is consistent with the way how pathspec works in general in Git).

* If the pattern does not contain a slash `/`, Git treats it as a shell glob pattern and checks for a match against the pathname relative to the location of the `.gitignore` file (relative to the toplevel of the work tree if not from a `.gitignore` file).

* Otherwise, Git treats the pattern as a shell glob suitable for consumption by `fnmatch(3)` with the `FNM_PATHNAME` flag: wildcards in the pattern will not match a `/` in the pathname. For example, `Documentation/*.html` matches `Documentation/git.html` but not `Documentation/ppc/ppc.html` or `tools/perf/Documentation/perf.html`.

* A leading slash matches the beginning of the pathname. For example, `/*.c` matches `cat-file.c` but not `mozilla-sha1/sha1.c`.

Two consecutive asterisks `**` in patterns matched against full pathname may have special meaning:

* A leading `**` followed by a slash means match in all directories. For example, `**/foo` matches file or directory `foo` anywhere, the same as pattern `foo`. `**/foo/bar` matches file or directory `bar` anywhere that is directly under directory `foo`.

* A trailing `/**` matches everything inside. For example, `abc/**` matches all files inside directory `abc`, relative to the location of the `.gitignore` file, with infinite depth.

* A slash followed by two consecutive asterisks then a slash matches zero or more directories. For example, `a/**/b` matches `a/b`, `a/x/b`, `a/x/y/b` and so on.

* Other consecutive asterisks are considered invalid.

## Installation

`go-gitignore` can be installed using the standard Go approach:

```go
go get github.com/denormal/go-gitignore
```

## License

Copyright (c) 2016 Denormal Limited

[MIT License](LICENSE)
