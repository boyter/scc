// Package fnmatch provides string matching based on fnmatch.3.
//
// This is a fork of github.com/danwakefield/fnmatch, which is a clone of
// kballard's golang fnmatch gist (https://gist.github.com/kballard/272720).
// See LICENSE for the original copyright, which is retained. Upstream was last
// changed in 2016 and does not take fixes, and the two defects below are both
// reachable from an ordinary .gitignore, so it is carried here instead.
//
// Changes from upstream:
//
//   - rangematch read the byte after a character class member before checking
//     that there was one, so a pattern ending in "[" plus a single multi byte
//     rune -- "[中", "[é", "[😀", or "[" followed by any invalid UTF-8, which
//     the .gitignore lexer turns into U+FFFD -- panicked with an index out of
//     range. Such a pattern now matches nothing, which is both what this
//     implementation already returned for the ASCII cases that did not panic
//     and what git does with an unterminated class.
//
//   - Match read the first byte of the subject to apply FNM_PERIOD after a
//     "*" without checking that there was one, so Match("*", "", FNM_PERIOD)
//     panicked. An empty subject has no leading period to object to. This one
//     is not reachable from this package, which never passes FNM_PERIOD, but
//     it is a trap for anyone who starts.
//
//   - Match recursed over every remaining suffix of the subject at each "*",
//     so a pattern with several of them cost O(len(s)^stars): "*0*0*!*01"
//     against two thousand characters took nine seconds. Subproblems that have
//     already failed are now remembered, which bounds the work without
//     changing any answer, since it only declines to recompute a result it
//     already has.
//
// There are a few issues that I believe to be bugs, but this implementation is
// based as closely as possible on BSD fnmatch. These bugs are present in the
// source of BSD fnmatch, and so are replicated here. The issues are as follows:
//
//   - FNM_PERIOD is no longer observed after the first * in a pattern
//     This only applies to matches done with FNM_PATHNAME as well
//   - FNM_PERIOD doesn't apply to ranges. According to the documentation,
//     a period must be matched explicitly, but a range will match it too
package fnmatch

import (
	"unicode"
	"unicode/utf8"
)

const (
	FNM_NOESCAPE = (1 << iota)
	FNM_PATHNAME
	FNM_PERIOD

	FNM_LEADING_DIR
	FNM_CASEFOLD

	FNM_IGNORECASE = FNM_CASEFOLD
	FNM_FILE_NAME  = FNM_PATHNAME
)

func unpackRune(str *string) rune {
	rune, size := utf8.DecodeRuneInString(*str)
	*str = (*str)[size:]
	return rune
}

// Matches the pattern against the string, with the given flags,
// and returns true if the match is successful.
// This function should match fnmatch.3 as closely as possible.
func Match(pattern, s string, flags int) bool {
	return match(pattern, s, flags, &matchState{})
}

// matchKey identifies a subproblem of one top level Match. Every pattern the
// recursion below is given is a suffix of the pattern Match was called with,
// and every subject a suffix of its subject, so the two lengths name the pair
// exactly. flags is part of the key because the recursion clears FNM_PERIOD.
type matchKey struct {
	pattern int32
	subject int32
	flags   int32
} // matchKey{}

// matchState is carried through one top level Match so that the "*" recursion
// does not solve the same subproblem twice.
//
// Left to itself that recursion tries every remaining suffix of the subject at
// every "*", so a pattern with several of them costs O(len(s)^stars), and
// "*0*0*!*01" against two thousand characters took nine seconds. Remembering
// the subproblems that failed bounds it. Only failures are worth recording, as
// a success returns straight out of the whole recursion.
//
// The map is not created until the recursion has done enough work to be worth
// the allocation, so the ordinary patterns that reach here -- one "*" against a
// file name, a few hundred steps at most -- still match without allocating.
type matchState struct {
	steps  int
	failed map[matchKey]struct{}
} // matchState{}

// memoThreshold is how many "*" expansions one Match may do before it starts
// remembering them. A single "*" against the longest name most filesystems
// allow is 255 expansions, and nothing that stays under this can blow up, so
// the common patterns never pay for the map.
//
// A variable rather than a constant only so that the tests can run the same
// inputs either side of it and check the answers do not move.
var memoThreshold = 1024

// expand runs one branch of the "*" recursion, consulting and recording the
// memo once there has been enough work to justify it.
func (m *matchState) expand(pattern, s string, flags int) bool {
	m.steps++

	if m.failed == nil {
		if m.steps < memoThreshold {
			return match(pattern, s, flags, m)
		}

		m.failed = make(map[matchKey]struct{})
	}

	_key := matchKey{pattern: int32(len(pattern)), subject: int32(len(s)), flags: int32(flags)}
	if _, _seen := m.failed[_key]; _seen {
		return false
	}

	if match(pattern, s, flags, m) {
		return true
	}

	m.failed[_key] = struct{}{}

	return false
} // expand()

func match(pattern, s string, flags int, state *matchState) bool {
	// The implementation for this function was patterned after the BSD fnmatch.c
	// source found at http://src.gnu-darwin.org/src/contrib/csup/fnmatch.c.html
	noescape := (flags&FNM_NOESCAPE != 0)
	pathname := (flags&FNM_PATHNAME != 0)
	period := (flags&FNM_PERIOD != 0)
	leadingdir := (flags&FNM_LEADING_DIR != 0)
	casefold := (flags&FNM_CASEFOLD != 0)
	// the following is some bookkeeping that the original fnmatch.c implementation did not do
	// We are forced to do this because we're not keeping indexes into C strings but rather
	// processing utf8-encoded strings. Use a custom unpacker to maintain our state for us
	sAtStart := true
	sLastAtStart := true
	sLastSlash := false
	sLastUnpacked := rune(0)
	unpackS := func() rune {
		sLastSlash = (sLastUnpacked == '/')
		sLastUnpacked = unpackRune(&s)
		sLastAtStart = sAtStart
		sAtStart = false
		return sLastUnpacked
	}
	for len(pattern) > 0 {
		c := unpackRune(&pattern)
		switch c {
		case '?':
			if len(s) == 0 {
				return false
			}
			sc := unpackS()
			if pathname && sc == '/' {
				return false
			}
			if period && sc == '.' && (sLastAtStart || (pathname && sLastSlash)) {
				return false
			}
		case '*':
			// collapse multiple *'s
			// don't use unpackRune here, the only char we care to detect is ASCII
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			// an empty subject has no leading period to object to, and reading
			// s[0] to find that out panicked
			if period && len(s) > 0 && s[0] == '.' && (sAtStart || (pathname && sLastUnpacked == '/')) {
				return false
			}
			// optimize for patterns with * at end or before /
			if len(pattern) == 0 {
				if pathname {
					return leadingdir || (strchr(s, '/') == -1)
				} else {
					return true
				}
			} else if pathname && pattern[0] == '/' {
				offset := strchr(s, '/')
				if offset == -1 {
					return false
				} else {
					// we already know our pattern and string have a /, skip past it
					s = s[offset:] // use unpackS here to maintain our bookkeeping state
					unpackS()
					pattern = pattern[1:] // we know / is one byte long
					break
				}
			}
			// general case, recurse
			for test := s; len(test) > 0; unpackRune(&test) {
				// I believe the (flags &^ FNM_PERIOD) is a bug when FNM_PATHNAME is specified
				// but this follows exactly from how fnmatch.c implements it
				if state.expand(pattern, test, (flags &^ FNM_PERIOD)) {
					return true
				} else if pathname && test[0] == '/' {
					break
				}
			}
			return false
		case '[':
			if len(s) == 0 {
				return false
			}
			if pathname && s[0] == '/' {
				return false
			}
			sc := unpackS()
			if !rangematch(&pattern, sc, flags) {
				return false
			}
		case '\\':
			if !noescape {
				if len(pattern) > 0 {
					c = unpackRune(&pattern)
				}
			}
			fallthrough
		default:
			if len(s) == 0 {
				return false
			}
			sc := unpackS()
			switch {
			case sc == c:
			case casefold && unicode.ToLower(sc) == unicode.ToLower(c):
			default:
				return false
			}
		}
	}
	return len(s) == 0 || (leadingdir && s[0] == '/')
}

func rangematch(pattern *string, test rune, flags int) bool {
	if len(*pattern) == 0 {
		return false
	}
	casefold := (flags&FNM_CASEFOLD != 0)
	noescape := (flags&FNM_NOESCAPE != 0)
	if casefold {
		test = unicode.ToLower(test)
	}
	var negate, matched bool
	if (*pattern)[0] == '^' || (*pattern)[0] == '!' {
		negate = true
		(*pattern) = (*pattern)[1:]
	}
	for !matched && len(*pattern) > 1 && (*pattern)[0] != ']' {
		c := unpackRune(pattern)
		if !noescape && c == '\\' {
			if len(*pattern) > 1 {
				c = unpackRune(pattern)
			} else {
				return false
			}
		}
		if casefold {
			c = unicode.ToLower(c)
		}
		// the length test has to come first: unpackRune above may have consumed
		// the whole of the remaining pattern, which happens when a class member
		// is the last thing in it and is more than one byte long, as in "[中"
		if len(*pattern) > 1 && (*pattern)[0] == '-' && (*pattern)[1] != ']' {
			unpackRune(pattern) // skip the -
			c2 := unpackRune(pattern)
			if !noescape && c2 == '\\' {
				if len(*pattern) > 0 {
					c2 = unpackRune(pattern)
				} else {
					return false
				}
			}
			if casefold {
				c2 = unicode.ToLower(c2)
			}
			// this really should be more intelligent, but it looks like
			// fnmatch.c does simple int comparisons, therefore we will as well
			if c <= test && test <= c2 {
				matched = true
			}
		} else if c == test {
			matched = true
		}
	}
	// skip past the rest of the pattern
	ok := false
	for !ok && len(*pattern) > 0 {
		c := unpackRune(pattern)
		if c == '\\' && len(*pattern) > 0 {
			unpackRune(pattern)
		} else if c == ']' {
			ok = true
		}
	}
	return ok && matched != negate
}

// define strchr because strings.Index() seems a bit overkill
// returns the index of c in s, or -1 if there is no match
func strchr(s string, c rune) int {
	for i, sc := range s {
		if sc == c {
			return i
		}
	}
	return -1
}
