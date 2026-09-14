// SPDX-License-Identifier: MIT

package processor

// The half of a counter that JavaScript and TypeScript share.
//
// TypeScript is JavaScript with two more equality checks and a type system the
// counting loop cannot see, so everything below reads the same for both: the
// three quotes, the postfix checks of M13, and the regular expression
// disambiguation of M16. Only the stop table and the complexity matcher differ,
// and those stay in the two counters.
//
// Written as one file rather than duplicated because M16 is the one piece of
// per-language reasoning in this work that is genuinely subtle, and two copies
// of it would drift.

// The three quotes of the ECMAScript family, held here so the shared string state is
// handed a slice rather than building one per string.
var (
	ecmaDoubleQuote = []byte{'"'}
	ecmaSingleQuote = []byte{'\''}
	ecmaBacktick    = []byte{'`'}
)

// ecmaPostfixToken reports the length of the postfix complexity check that begins
// at index, or zero where none does. The trie the generic loop asks takes the
// longest match, so ??= is answered before ??, and the caller does not step
// over what it matched: the generic loop does not either, which is why a??.b
// counts twice, once for the ?? and once for the ?. that overlaps it.
func ecmaPostfixToken(content []byte, index int) int {
	if index+1 >= len(content) {
		return 0
	}

	switch content[index+1] {
	case '?':
		if index+2 < len(content) && content[index+2] == '=' {
			return 3
		}

		return 2
	case '.':
		return 2
	}

	return 0
}

// ecmaPostfixCounts is countComplexityPostfix for JavaScript, which declares no
// postfix excludes. A postfix check counts wherever it sits except at the very
// first byte of the file, or where nothing but whitespace comes before it.
//
// The test is against absolute zero rather than against floor because that is
// what the generic loop tests, and agreeing with the generic loop is the whole
// contract. A counter handed a range rather than a file will want floor here.
func ecmaPostfixCounts(content []byte, index int) bool {
	if index == 0 {
		return false
	}

	if isWhitespace(byteBefore(content, index, 0)) && !hasNonWhitespaceBefore(content, index-1) {
		return false
	}

	return true
}

// ecmaRegexOperators marks the bytes after which a slash opens a regular
// expression literal rather than dividing. See ecmaRegexAllowed.
//
// < and > are deliberately NOT in here, though they are comparison operators
// and a slash after one divides nothing. Phase 1 had them in, on the grounds
// that the closing tag of a JSX element then read as a pattern and the element
// text between two tags was skipped rather than scanned, which over React and
// three.js never made a count worse and sometimes made one better.
//
// TypeScript showed that reasoning to be wrong. A nested template literal
// holding markup, which is ordinary in a compiler test,
//
//	const a = `<div>${repeat(`<span></span>`, 3)}</div>`
//
// has the slash of </span> sitting behind a <, so the scan ran on to the slash
// of </div> and swallowed the backtick between them. The line ends inside a
// string that is not open, and every line under it is counted as code. The
// generic loop gets that line right and the counter got it wrong, which no
// argument about JSX covers: a counter is allowed to disagree with the generic
// loop only where the disagreement is enumerated and the counter is right.
//
// Taking the two bytes out fixes it and costs nothing measurable. A slash
// immediately behind a < or a > is a regular expression only in code like
// a < /re/.test(b), which nobody writes; everywhere else the byte in front of
// the slash is an identifier or a bracket and the answer is unchanged. Measured
// over 19,295 TypeScript and 5,610 JavaScript files, taking them out removed
// every divergence where the counter was wrong and kept every one where it was
// right.
//
// The apostrophe in JSX element text, as in doesn't, still opens a string in
// both loops that never closes. That is the generic loop's bug and belongs to
// the embedded language work, not to a side effect of the regex question.
var ecmaRegexOperators = buildEcmaRegexOperators()

func buildEcmaRegexOperators() [256]bool {
	var table [256]bool
	for _, b := range []byte("(,=:[!&|?{};+-*%~^") {
		table[b] = true
	}

	return table
}

// ecmaRegexKeywords are the words a regular expression literal may follow. Every
// other word ends an expression, and a slash behind one of those is a division.
var ecmaRegexKeywords = []string{
	"return", "typeof", "case", "in", "of", "new", "delete",
	"void", "do", "yield", "await", "throw", "else", "instanceof",
}

// ecmaRegexLookBehind bounds how far back ecmaRegexAllowed will walk over
// whitespace looking for the token in front of a slash. The token is within a
// byte or two of it in anything anyone writes, and a bound is what keeps a file
// of nothing but whitespace and slashes from costing time in the square of its
// length.
const ecmaRegexLookBehind = 64

// ecmaRegexLongestKeyword is the length of the longest word in ecmaRegexKeywords,
// which bounds how far back the word in front of a slash is read.
const ecmaRegexLongestKeyword = len("instanceof")

// ecmaRegexAllowed reports whether the slash at index can open a regular
// expression literal rather than being a division.
//
// This is the one question about JavaScript that a table cannot answer, and it
// is the whole argument for writing counters by hand. `/` is a division, a
// comment or the start of a pattern depending on the token in front of it, and
// the generic loop has no way to keep that one token of context. It reads the
// quote in /["']/ as opening a string and the slashes in /[//]/ as opening a
// comment, which is LineJudge 7010 and 7020, the only group scc scores zero on.
//
// The rule is the one every JavaScript lexer uses, read conservatively: a
// pattern may follow an operator, an opening bracket, a separator, or one of
// the keywords that cannot end an expression. Anything else — an identifier, a
// digit, a closing bracket, a quote — ends an expression, and a slash behind
// one of those divides.
//
// Conservative matters. Reading a division as a pattern would swallow code and
// diverge from the generic loop somewhere it is right; failing to spot a
// pattern only leaves scc counting what it counts today. So every case that is
// not clearly a pattern is left alone: a slash behind the end of a block
// comment, or behind more than ecmaRegexLookBehind bytes of whitespace, reads as
// a division.
func ecmaRegexAllowed(content []byte, index, floor int) bool {
	limit := index - ecmaRegexLookBehind
	if limit < floor {
		limit = floor
	}

	i := index - 1
	for i >= limit && isWhitespace(content[i]) {
		i--
	}

	if i < limit {
		// Either the region starts here, in which case there is no expression
		// for the slash to divide and a pattern may open, or the look behind
		// bound was reached and the answer is unknown, which reads as a
		// division.
		return limit == floor
	}

	if !isIdentifierContinue(content[i]) {
		return ecmaRegexOperators[content[i]]
	}

	// An identifier or a number ends an expression and the slash divides it,
	// unless the word is one of the keywords that cannot end one. Only as many
	// bytes as the longest of those are read back; a longer word is an
	// identifier whatever it spells.
	start := i
	bound := i - ecmaRegexLongestKeyword + 1
	if bound < floor {
		bound = floor
	}
	for start > bound && isIdentifierContinue(content[start-1]) {
		start--
	}
	if start > floor && isIdentifierContinue(content[start-1]) {
		return false
	}

	word := string(content[start : i+1])
	for _, keyword := range ecmaRegexKeywords {
		if word == keyword {
			return true
		}
	}

	return false
}

// ecmaRegexEnd returns the index of the slash that closes the regular expression
// literal opened at index, or -1 where nothing does before the line or the
// region ends.
//
// A slash inside a character class does not close the pattern, which is the
// whole of LineJudge 7020, and a slash behind a backslash does not either. A
// pattern cannot run over a newline, so a line that ends without closing one
// was never a pattern and the caller falls back to reading the slash the way
// the generic loop does.
func ecmaRegexEnd(content []byte, index, endPoint int) int {
	inClass := false

	for i := index + 1; i < endPoint; i++ {
		switch content[i] {
		case '\\':
			i++
		case '\n':
			return -1
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return i
			}
		}
	}

	return -1
}

// ecmaRegexLiterals turns the one deliberate disagreement with the generic loop on
// and off. It is on, and nothing but a test turns it off.
//
// M16 is not confined to the two LineJudge inputs it is named by: a regular
// expression literal holding a quote, a comment opener or a complexity token
// reads differently from the generic loop wherever one appears, which on real
// code is a little under one file in a hundred. That makes "the counter agrees
// with the generic loop on every file of the corpus" untestable for JavaScript
// unless the one difference can be taken out, so this takes it out. The corpus
// differential runs once with it off, where exact agreement is still required
// and still holds, and once with it on, where it reports what the fix moved.
//
// The behaviour itself is pinned by the fixtures of counterDivergences, which
// assert both answers, and by the hand written table in the test file.
var ecmaRegexLiterals = true

// ecmaSkipRegex reports the last byte of the regular expression literal opened by
// the slash at index, or -1 where the slash opens no pattern. The caller carries
// on from the byte after it, so nothing inside the pattern is read as a quote or
// a comment.
func ecmaSkipRegex(content []byte, index, endPoint, floor int) int {
	if !ecmaRegexLiterals || !ecmaRegexAllowed(content, index, floor) {
		return -1
	}

	return ecmaRegexEnd(content, index, endPoint)
}
