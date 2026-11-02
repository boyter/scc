// SPDX-License-Identifier: MIT

package gitignore

import (
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/boyter/gocodewalker/go-gitignore/internal/fnmatch"
)

// Pattern represents per-line patterns within a .gitignore file
type Pattern interface {
	Match

	// Match returns true if the given path matches the name pattern. If the
	// pattern is meant for directories only, and the path is not a directory,
	// Match will return false. The matching is performed by fnmatch(). It
	// is assumed path is relative to the base path of the owning GitIgnore.
	Match(string, bool) bool
}

// pathInfo carries the values derived from a path that more than one pattern
// in an ignore file would otherwise each recompute for itself. A .gitignore
// holds one pattern list but is asked about one path at a time, so anything
// that depends only on the path can be worked out once for the whole list
// rather than once per pattern. The Linux kernel's ignore rules put several
// hundred name patterns in scope at the root of the tree, and every one of
// them wants the same base name.
type pathInfo struct {
	// path is the path as handed to Relative, relative to the ignore file's
	// base directory and in slash form.
	path string

	// base is the last component of path, the only part a non-anchored name
	// pattern is ever matched against.
	base string

	// hasSep records whether path contains a '/', which is what tells an
	// anchored name pattern that it cannot match.
	hasSep bool
}

// newPathInfo derives the per-path values from path. It reproduces exactly what
// name.Match works out for itself, so the two remain interchangeable.
func newPathInfo(path string) pathInfo {
	// the base name of the path. filepath.Split was previously used here, but
	// it scans backwards a byte at a time through os.IsPathSeparator, so for a
	// tree the size of the Linux kernel it was several percent of the whole
	// walk. The GOOS test is a compile time constant, so on Unix this is a
	// single vectorised LastIndexByte.
	_i := strings.LastIndexByte(path, '/')
	_hasSep := _i >= 0
	if runtime.GOOS == "windows" {
		if _j := strings.LastIndexByte(path, '\\'); _j > _i {
			_i = _j
		}
	}

	return pathInfo{path: path, base: path[_i+1:], hasSep: _hasSep}
} // newPathInfo()

// fastPattern is the internal matching interface. It is Pattern with the
// addition of matchInfo, which takes the already derived pathInfo instead of
// re-deriving it. Pattern is exported, so its method set is left alone and this
// sits alongside it; ignore stores its patterns as fastPattern so the loop in
// Relative does not pay for a type assertion per pattern.
type fastPattern interface {
	Pattern

	// matchInfo is Match, given the pathInfo for the path being tested. It
	// must agree with Match(info.path, isdir) for every path and pattern.
	matchInfo(info pathInfo, isdir bool) bool
}

// pattern is the base implementation of a .gitignore pattern
type pattern struct {
	_negated   bool
	_anchored  bool
	_directory bool
	_string    string
	_fnmatch   string
	_position  Position
} // pattern()

// matchType classifies name patterns for fast-path matching.
type matchType int

const (
	matchComplex  matchType = iota // requires fnmatch
	matchExact                     // no glob chars → string ==
	matchSuffix                    // leading * + literal tail → HasSuffix
	matchPrefix                    // literal head + trailing * → HasPrefix
	matchContains                  // leading and trailing * + literal middle → Contains
	matchAny                       // bare "*" → matches every name
)

// name represents patterns matching a file or path name (i.e. the last
// component of a path)
type name struct {
	pattern
	_matchType matchType
	_literal   string
} // name{}

// path represents a pattern that contains at least one path separator within
// the pattern (i.e. not at the start or end of the pattern)
type path struct {
	pattern
	_depth int
} // path{}

// anyToken is a single non-separator token of an "any" pattern, pre-resolved
// into the form the matcher needs. Token.Token() builds a string from a []rune
// on every call, and the matcher calls it once per path component per pattern,
// so the string is built once here instead.
type anyToken struct {
	_any       bool // true for the "**" token
	_word      string
	_matchType matchType
	_literal   string
} // anyToken{}

// any represents a pattern that contains at least one "any" token "**"
// allowing for recursive matching.
type any struct {
	pattern
	_tokens []*Token
	_words  []anyToken
} // any{}

// NewPattern returns a Pattern from the ordered slice of Tokens. The tokens are
// assumed to represent a well-formed .gitignore pattern. A Pattern may be
// negated, anchored to the start of the path (relative to the base directory
// of tie containing .gitignore), or match directories only.
func NewPattern(tokens []*Token) Pattern {
	// if we have no tokens there is no pattern
	if len(tokens) == 0 {
		return nil
	}

	// extract the pattern position from first token
	_position := tokens[0].Position
	_string := tokenset(tokens).String()

	// is this a negated pattern?
	_negated := false
	if tokens[0].Type == NEGATION {
		_negated = true
		tokens = tokens[1:]
	}

	// is this pattern anchored to the start of the path?
	_anchored := false
	if tokens[0].Type == SEPARATOR {
		_anchored = true
		tokens = tokens[1:]
	}

	// is this pattern for directories only?
	_directory := false
	_last := len(tokens) - 1
	if len(tokens) != 0 {
		if tokens[_last].Type == SEPARATOR {
			_directory = true
			tokens = tokens[:_last]
		}
	}

	// build the pattern expression
	_fnmatch := tokenset(tokens).String()
	_pattern := &pattern{
		_negated:   _negated,
		_anchored:  _anchored,
		_position:  _position,
		_directory: _directory,
		_string:    _string,
		_fnmatch:   _fnmatch,
	}
	return _pattern.compile(tokens)
} // NewPattern()

// compile generates a specific Pattern (i.e. name, path or any)
// represented by the list of tokens.
func (p *pattern) compile(tokens []*Token) Pattern {
	// what tokens do we have in this pattern?
	//      - ANY token means we can match to any depth
	//      - SEPARATOR means we have path rather than file matching
	_separator := false
	for _, _token := range tokens {
		switch _token.Type {
		case ANY:
			return p.any(tokens)
		case SEPARATOR:
			_separator = true
		}
	}

	// should we perform path or name/file matching?
	if _separator {
		return p.path(tokens)
	} else {
		return p.name(tokens)
	}
} // compile()

// Ignore returns true if the pattern describes files or paths that should be
// ignored.
func (p *pattern) Ignore() bool { return !p._negated }

// Include returns true if the pattern describes files or paths that should be
// included (i.e. not ignored)
func (p *pattern) Include() bool { return p._negated }

// Position returns the position of the first token of this pattern.
func (p *pattern) Position() Position { return p._position }

// String returns the string representation of the pattern.
func (p *pattern) String() string { return p._string }

//
// name patterns
//      - designed to match trailing file/directory names only
//

// containsGlob reports whether s contains any fnmatch special characters.
func containsGlob(s string) bool {
	return strings.ContainsAny(s, "*?[\\")
}

// classifyGlob buckets an fnmatch expression that will be matched against a
// single path component, returning the match type and the literal it should be
// compared against.
func classifyGlob(fn string) (matchType, string) {
	switch {
	case strings.ContainsRune(fn, utf8.RuneError):
		// fnmatch compares decoded runes, and every byte of invalid UTF-8 in
		// the target decodes to U+FFFD, so a U+FFFD in the pattern -- which is
		// what the lexer produces for invalid UTF-8 in the .gitignore file --
		// matches any invalid byte. Byte comparison cannot reproduce that, so
		// leave these to fnmatch.
		return matchComplex, ""
	case !containsGlob(fn):
		// exact literal (e.g. "node_modules", ".DS_Store")
		return matchExact, fn
	case fn == "*":
		// bare "*" matches any name, so there is nothing to compare
		return matchAny, ""
	case fn[0] == '*' && !containsGlob(fn[1:]):
		// suffix match (e.g. "*.o", "*.pyc")
		return matchSuffix, fn[1:]
	case fn[len(fn)-1] == '*' && !containsGlob(fn[:len(fn)-1]):
		// prefix match (e.g. ".*", "vmlinuz*") — very common, and the Linux
		// kernel's root .gitignore leads with ".*", which was previously
		// evaluated by fnmatch for every file in the tree
		return matchPrefix, fn[:len(fn)-1]
	case len(fn) > 2 && fn[0] == '*' && fn[len(fn)-1] == '*' && !containsGlob(fn[1:len(fn)-1]):
		// contains match (e.g. "*.o.*")
		return matchContains, fn[1 : len(fn)-1]
	default:
		return matchComplex, ""
	}
} // classifyGlob()

// name returns a Pattern designed to match file or directory names, with no
// path elements.
func (p *pattern) name(tokens []*Token) Pattern {
	n := &name{pattern: *p}

	// Classify the fnmatch expression for fast-path dispatch. A name pattern is
	// only ever matched against a single path component, which never contains a
	// '/', so an fnmatch '*' here is equivalent to "any run of characters" and
	// the prefix/suffix/contains forms classifyGlob picks are exact
	// substitutions for it.
	n._matchType, n._literal = classifyGlob(p._fnmatch)

	return n
} // name()

// Match returns true if the given path matches the name pattern. If the
// pattern is meant for directories only, and the path is not a directory,
// Match will return false. The matching is performed by fnmatch(). It
// is assumed path is relative to the base path of the owning GitIgnore.
func (n *name) Match(path string, isdir bool) bool {
	return n.matchInfo(newPathInfo(path), isdir)
} // Match()

// matchInfo is Match with the base name of the path already derived. It is the
// form used by the pattern loop in Relative, where the whole list of patterns
// shares one pathInfo.
func (n *name) matchInfo(info pathInfo, isdir bool) bool {
	// are we expecting a directory?
	if n._directory && !isdir {
		return false
	}

	// determine the string to match against
	_target := info.base
	if n._anchored {
		// an anchored name pattern is a single path component anchored to the
		// base directory, so it can only ever match a single-segment path. A
		// multi-segment target (e.g. "keep/sub" against "/*/") must not match,
		// otherwise glob patterns such as "*" would incorrectly span the '/'
		// separator and ignore nested directories that git keeps.
		if info.hasSep {
			return false
		}
		_target = info.path
	}

	// fast-path dispatch avoids expensive fnmatch for simple patterns
	switch n._matchType {
	case matchExact:
		return _target == n._literal
	case matchSuffix:
		return strings.HasSuffix(_target, n._literal)
	case matchPrefix:
		return strings.HasPrefix(_target, n._literal)
	case matchContains:
		return strings.Contains(_target, n._literal)
	case matchAny:
		return true
	default:
		return fnmatch.Match(n._fnmatch, _target, 0)
	}
} // matchInfo()

//
// path patterns
//      - designed to match complete or partial paths (not just filenames)
//

// path returns a Pattern designed to match paths that include at least one
// path separator '/' neither at the end nor the start of the pattern.
func (p *pattern) path(tokens []*Token) Pattern {
	// how many directory components are we expecting?
	_depth := 0
	for _, _token := range tokens {
		if _token.Type == SEPARATOR {
			_depth++
		}
	}

	// return the pattern instance
	return &path{pattern: *p, _depth: _depth}
} // path()

// Match returns true if the given path matches the path pattern. If the
// pattern is meant for directories only, and the path is not a directory,
// Match will return false. The matching is performed by fnmatch()
// with flags set to FNM_PATHNAME. It is assumed path is relative to the
// base path of the owning GitIgnore.
func (p *path) Match(path string, isdir bool) bool {
	// are we expecting a directory
	if p._directory && !isdir {
		return false
	}

	return fnmatch.Match(p._fnmatch, path, fnmatch.FNM_PATHNAME)
} // Match()

// matchInfo is Match for path patterns. A path pattern is matched against the
// whole path, so there is nothing precomputed for it to use.
func (p *path) matchInfo(info pathInfo, isdir bool) bool {
	return p.Match(info.path, isdir)
} // matchInfo()

//
// "any" patterns
//

// any returns a Pattern designed to match paths that include at least one
// any pattern '**', specifying recursive matching.
func (p *pattern) any(tokens []*Token) Pattern {
	// consider only the non-SEPARATOR tokens, as these will be matched
	// against the path components
	_tokens := make([]*Token, 0)
	for _, _token := range tokens {
		if _token.Type != SEPARATOR {
			_tokens = append(_tokens, _token)
		}
	}

	// pre-resolve each token into the form the matcher wants: the token word as
	// a string (Token() otherwise rebuilds it from []rune on every call), and
	// the same glob classification the name patterns use. A token is only ever
	// matched against a single path component, which contains no '/', so an
	// fnmatch '*' is just "any run of characters" and the literal forms below
	// are exact substitutions for it -- the same argument that makes the name
	// fast paths correct, and FNM_PATHNAME makes no difference for the same
	// reason.
	_words := make([]anyToken, len(_tokens))
	for i, _token := range _tokens {
		_word := _token.Token()
		_words[i] = anyToken{
			_any:  _token.Type == ANY,
			_word: _word,
		}
		_words[i]._matchType, _words[i]._literal = classifyGlob(_word)
	}

	return &any{*p, _tokens, _words}
} // any()

// match reports whether a single path component matches this token.
func (t *anyToken) match(component string) bool {
	switch t._matchType {
	case matchExact:
		return component == t._literal
	case matchSuffix:
		return strings.HasSuffix(component, t._literal)
	case matchPrefix:
		return strings.HasPrefix(component, t._literal)
	case matchContains:
		return strings.Contains(component, t._literal)
	case matchAny:
		return true
	default:
		return fnmatch.Match(t._word, component, fnmatch.FNM_PATHNAME)
	}
} // match()

// Match returns true if the given path matches the any pattern. If the
// pattern is meant for directories only, and the path is not a directory,
// Match will return false. The matching is performed by recursively applying
// fnmatch() with flags set to FNM_PATHNAME. It is assumed path is relative to
// the base path of the owning GitIgnore.
func (a *any) Match(path string, isdir bool) bool {
	// are we expecting a directory?
	if a._directory && !isdir {
		return false
	}

	// walk the path components in place, rather than splitting the path into a
	// slice: an 'any' pattern is evaluated against every entry of the tree, and
	// the split was an allocation per pattern per path.
	return a.matchFrom(path, 0, a._words)
} // Match()

// matchInfo is Match for "any" patterns. An "any" pattern walks the whole path
// component by component, so the derived base name is of no use to it. There is
// nothing else left to hoist here either: walking the components from a byte
// offset is what removed the per-path allocation these patterns used to make.
func (a *any) matchInfo(info pathInfo, isdir bool) bool {
	return a.Match(info.path, isdir)
} // matchInfo()

// matchFrom performs the recursive matching for 'any' patterns over the
// components of path beginning at byte offset off. An 'any' token '**' may
// match any path component, or no path component.
//
// The offset encodes the remaining components the way the []string it replaces
// did: off <= len(path) means at least one component remains (off == len(path)
// is the empty trailing component of a path ending in '/', which is what
// strings.Split yields), and off > len(path) means every component has been
// consumed.
func (a *any) matchFrom(path string, off int, tokens []anyToken) bool {
	_empty := off > len(path)

	// if we have no more tokens, then we have matched this path
	// if there are also no more path elements, otherwise there's no match
	if len(tokens) == 0 {
		return _empty
	}

	// the current path component, and the offset of the one after it
	var _component string
	_next := off
	if !_empty {
		if i := strings.IndexByte(path[off:], byte(_SEPARATOR)); i >= 0 {
			_component = path[off : off+i]
			_next = off + i + 1
		} else {
			_component = path[off:]
			_next = len(path) + 1
		}
	}

	// what token are we trying to match?
	_token := &tokens[0]
	if _token._any {
		// A trailing "**" matches everything *inside* the directory named
		// by the preceding tokens, so it must consume at least one path
		// component: gitignore(5) gives "abc/**" as matching all files
		// inside "abc", and git does not ignore "abc" itself. A leading or
		// embedded "**" is the opposite case and may match no component at
		// all, so "**/foo" matches "foo" and "a/**/b" matches "a/b".
		if len(tokens) == 1 {
			return !_empty
		}
		if _empty {
			return a.matchFrom(path, off, tokens[1:])
		}
		return a.matchFrom(path, off, tokens[1:]) || a.matchFrom(path, _next, tokens)
	}

	// if we have a non-ANY token, then we must have a non-empty path; if the
	// current path element matches this token, we match if the remainder of the
	// path matches the remaining tokens
	if !_empty && _token.match(_component) {
		return a.matchFrom(path, _next, tokens[1:])
	}

	// if we are here, then we have no match
	return false
} // matchFrom()

// ensure the patterns confirm to the Pattern interface
var _ Pattern = &name{}
var _ Pattern = &path{}
var _ Pattern = &any{}

// ensure the patterns also offer the internal precomputed form
var _ fastPattern = &name{}
var _ fastPattern = &path{}
var _ fastPattern = &any{}
