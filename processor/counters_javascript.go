// SPDX-License-Identifier: MIT

package processor

// A counter written for JavaScript, on the same terms as the C and Java ones:
// it must agree with the generic loop to the line, and where the two differ the
// generic one is right by definition.
//
// JavaScript asks for three things Java does not.
//
// A third quote, the backtick of a template literal. It runs over newlines and
// holds ${} interpolation, but neither matters to a line count: the generic
// loop has it as an ordinary quote with no ignoreEscape, resetState sends
// SString back to itself, and JavaScript does not splice, so every line of one
// counts as code until the closing backtick. A third entry in the same switch.
//
// Postfix complexity checks, which no counter had before. ?. ?? and ??= are
// counted wherever they appear rather than only at the start of a word, which
// is M13 and is modelled on countComplexityPostfix.
//
// And a slash that is a division, a comment, or a regular expression literal
// depending on what came before it. That is M16, it is the one place this
// counter deliberately disagrees with the generic loop, and it is why the
// counter is worth writing even where the speed case is weak. See
// jsRegexAllowed.
//
// Everything that is not the stop table, the complexity matcher and the regex
// question lives in counters_shared.go.

// jsStop marks every byte the scan has to stop on: the slash of both comment
// forms and of a regex literal, the three quotes, one anchor byte out of every
// complexity check, and the newline and the null that end a line and a file of
// bytes rather than text.
var jsStop = buildJsStop()

// jsStopNoComplexity is jsStop without the bytes that only a complexity check
// is spelled with. It is what a file counted with --no-complexity is scanned
// with, the generic loop leaving the checks out of its trie under the same
// flag.
var jsStopNoComplexity = buildJsStopNoComplexity()

// The three quotes of JavaScript, held here so the shared string state is
// handed a slice rather than building one per string.
var (
	jsDoubleQuote = []byte{'"'}
	jsSingleQuote = []byte{'\''}
	jsBacktick    = []byte{'`'}
)

// jsComplexityAnchors is the byte each complexity check of JavaScript is
// stopped on, the postfix three included. It is what the structural conformance
// test holds against jsStop, and it is the written form of the argument in
// buildJsStop.
var jsComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"case ":   'c', "case(": 'c',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
	"?.":  '?',
	"??":  '?',
	"??=": '?',
}

// buildJsStop marks the anchor of every complexity check of JavaScript.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	c   case
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//	?   ?. ?? ??=, the three postfix checks, all spelled with ? first
//
// The anchor is the rarest byte of the check rather than its first. Measured
// over 19.8MB of React, stopping on the first byte of every check costs 19.4%
// of all bytes; the anchors above cost 7.6%. The saving is nearly all of it e,
// i, s and t, which open else, if, switch and the rest and are four of the
// commonest bytes in JavaScript.
//
// Each anchor is the rarest byte of its own check: f 1.07% against i 3.22% for
// if, w 0.52% against s 3.31% for switch, l 2.45% against e 7.68% for else,
// c 2.44% against a 3.61% and s 3.31% for case.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match, which is what lets the scan
// carry on through a matched token rather than stepping over it the way the
// generic loop does:
//
//   - while carries the l of else, but else wants an e in front of its l and
//     finds an i.
//   - switch carries the c of case, but case wants a word boundary in front of
//     its c and finds the t of switch.
//   - the second | of || and the second & of && are read as the start of
//     another check and fail, there being no third byte to match.
//
// So the scan never counts a check twice and never counts one that is not
// there.
func buildJsStop() [256]bool {
	table := buildJsStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'c', '|', '&', '=', '?'} {
		table[b] = true
	}

	return table
}

func buildJsStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '`', '\n', 0} {
		table[b] = true
	}

	return table
}

// jsStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func jsStopTable() *[256]bool {
	if Complexity {
		return &jsStopNoComplexity
	}

	return &jsStop
}

// spaceOpens reports whether a byte closes a keyword that languages.json spells
// with a space behind it and nothing else. JavaScript writes switch, while and
// else that way, where it writes for, if and case twice, once with a space and
// once with the bracket.
func spaceOpens(content []byte, index int) bool {
	return index < len(content) && content[index] == ' '
}

// jsComplexityAnchored reports whether a complexity check of JavaScript sits on
// the anchor byte at index, which is what jsStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of JavaScript is spelled
// with, so reading back never crosses out of the code the scan is in. Nor can
// it read in front of the region the counter owns, every backwards read being
// clamped to floor.
//
// The checks that share an anchor are told apart on the byte behind it and
// cannot both match: the f of if has an i behind it and the f of for cannot,
// i being a byte that carries a word on, and the same holds of the w of switch
// against while and the = of != against ==.
func jsComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		if byteBefore(content, index, floor) == 's' {
			return wordStartsAt(content, index-1, floor) &&
				hasPrefixAt(content, index+1, floor, "itch") && spaceOpens(content, index+5)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case 'l':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "se") && spaceOpens(content, index+3)
	case 'c':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "ase") && cOpens(content, index+4)
	case '=':
		if b := byteBefore(content, index, floor); b != '=' && b != '!' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			index+1 < len(content) && content[index+1] == ' '
	case '|':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// jsComplexityAtLineStart is jsComplexityAnchored for the first byte of code on
// a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches, since that begins on the byte after this one
// and no check that is anchored on its first byte holds an anchor of its own
// anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, while,
// case, || and && are never counted when they open a line, and it is the reason
// the backwards reads above can be written as reads rather than as searches.
// The two are one thing. The postfix three are handled separately, since they
// ask a different question of what sits in front of them.
func jsComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case 'c':
		return hasPrefixAt(content, index+1, floor, "ase") && cOpens(content, index+4)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// jsPostfixToken reports the length of the postfix complexity check that begins
// at index, or zero where none does. The trie the generic loop asks takes the
// longest match, so ??= is answered before ??, and the caller does not step
// over what it matched: the generic loop does not either, which is why a??.b
// counts twice, once for the ?? and once for the ?. that overlaps it.
func jsPostfixToken(content []byte, index int) int {
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

// jsPostfixCounts is countComplexityPostfix for JavaScript, which declares no
// postfix excludes. A postfix check counts wherever it sits except at the very
// first byte of the file, or where nothing but whitespace comes before it.
//
// The test is against absolute zero rather than against floor because that is
// what the generic loop tests, and agreeing with the generic loop is the whole
// contract. A counter handed a range rather than a file will want floor here.
func jsPostfixCounts(content []byte, index int) bool {
	if index == 0 {
		return false
	}

	if isWhitespace(byteBefore(content, index, 0)) && !hasNonWhitespaceBefore(content, index-1) {
		return false
	}

	return true
}

// jsRegexOperators marks the bytes after which a slash opens a regular
// expression literal rather than dividing. See jsRegexAllowed.
//
// < and > are in here because they are comparison operators, and a slash after
// one divides nothing. The side effect is that the closing tag of a JSX element
// reads as a pattern opening: in <b>x</b> the slash sits behind a <, so the
// scan runs on to the next slash on the line and treats what lies between as
// pattern body. Real JSX in a .js file therefore has runs of element text
// skipped rather than scanned.
//
// It is left that way deliberately, on two grounds. Measured over 5610 files of
// React and three.js it never once made a count worse than the generic loop,
// and where it changes one it changes it for the better: the generic loop reads
// the apostrophe of a word like doesn't in element text as opening a string
// that never closes, and swallows the rest of the file into it, where skipping
// the run avoids that. And the alternative, telling JSX from comparison, needs
// a parse of the element rather than a byte of look behind.
//
// It is still reasoning the counter arrives at by accident, so it is written
// down here rather than left to be rediscovered. A JSX aware counter would drop
// < and > from this table and handle elements as their own state.
var jsRegexOperators = buildJsRegexOperators()

func buildJsRegexOperators() [256]bool {
	var table [256]bool
	for _, b := range []byte("(,=:[!&|?{};+-*%<>~^") {
		table[b] = true
	}

	return table
}

// jsRegexKeywords are the words a regular expression literal may follow. Every
// other word ends an expression, and a slash behind one of those is a division.
var jsRegexKeywords = []string{
	"return", "typeof", "case", "in", "of", "new", "delete",
	"void", "do", "yield", "await", "throw", "else", "instanceof",
}

// jsRegexLookBehind bounds how far back jsRegexAllowed will walk over
// whitespace looking for the token in front of a slash. The token is within a
// byte or two of it in anything anyone writes, and a bound is what keeps a file
// of nothing but whitespace and slashes from costing time in the square of its
// length.
const jsRegexLookBehind = 64

// jsRegexLongestKeyword is the length of the longest word in jsRegexKeywords,
// which bounds how far back the word in front of a slash is read.
const jsRegexLongestKeyword = len("instanceof")

// jsRegexAllowed reports whether the slash at index can open a regular
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
// comment, or behind more than jsRegexLookBehind bytes of whitespace, reads as
// a division.
func jsRegexAllowed(content []byte, index, floor int) bool {
	limit := index - jsRegexLookBehind
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
		return jsRegexOperators[content[i]]
	}

	// An identifier or a number ends an expression and the slash divides it,
	// unless the word is one of the keywords that cannot end one. Only as many
	// bytes as the longest of those are read back; a longer word is an
	// identifier whatever it spells.
	start := i
	bound := i - jsRegexLongestKeyword + 1
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
	for _, keyword := range jsRegexKeywords {
		if word == keyword {
			return true
		}
	}

	return false
}

// jsRegexEnd returns the index of the slash that closes the regular expression
// literal opened at index, or -1 where nothing does before the line or the
// region ends.
//
// A slash inside a character class does not close the pattern, which is the
// whole of LineJudge 7020, and a slash behind a backslash does not either. A
// pattern cannot run over a newline, so a line that ends without closing one
// was never a pattern and the caller falls back to reading the slash the way
// the generic loop does.
func jsRegexEnd(content []byte, index, endPoint int) int {
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

// jsRegexLiterals turns the one deliberate disagreement with the generic loop on
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
var jsRegexLiterals = true

// jsSkipRegex reports the last byte of the regular expression literal opened by
// the slash at index, or -1 where the slash opens no pattern. The caller carries
// on from the byte after it, so nothing inside the pattern is read as a quote or
// a comment.
func jsSkipRegex(content []byte, index, endPoint, floor int) int {
	if !jsRegexLiterals || !jsRegexAllowed(content, index, floor) {
		return -1
	}

	return jsRegexEnd(content, index, endPoint)
}

// jsBlankState looks at the first byte of content on a line.
func jsBlankState(content []byte, tally *counterTally, index, endPoint, floor int) (int, counterState, []byte) {
	switch content[index] {
	case '/':
		if index+1 < len(content) {
			switch content[index+1] {
			case '/':
				return index, SComment, nil
			case '*':
				return index + 1, SMulticomment, nil
			}
		}

		// A line can open with a pattern, as in /^a/.test(s), and the token that
		// says so is on a line above. Reading it here rather than leaving it to
		// the code scan is what stops a quote inside that pattern opening a
		// string.
		if end := jsSkipRegex(content, index, endPoint, floor); end >= 0 {
			return end, SCode, nil
		}

		return index, SCode, nil
	case '"':
		return index, SString, jsDoubleQuote
	case '\'':
		return index, SString, jsSingleQuote
	case '`':
		return index, SString, jsBacktick
	case '?':
		if !Complexity && jsPostfixToken(content, index) != 0 && jsPostfixCounts(content, index) {
			tally.Complexity++
		}

		return index, SCode, nil
	}

	if !Complexity && jsComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// jsCodeState runs to the end of the line or to whatever token takes it out of
// code.
func jsCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
	if endPoint > len(content) {
		endPoint--
	}

	for i := index; i < endPoint; i++ {
		curByte := content[i]

		if !stop[curByte] {
			continue
		}

		switch curByte {
		case '\n':
			return i, SCode, nil
		case 0:
			if isBinary(i, curByte) {
				tally.Binary = true
				return i, SCode, nil
			}
		case '/':
			if i+1 < len(content) {
				switch content[i+1] {
				case '/':
					return i, SCommentCode, nil
				case '*':
					return i + 1, SMulticommentCode, nil
				}
			}

			// Neither comment opens here, so the slash divides or it opens a
			// pattern. Where it opens one the scan carries on from the byte
			// after the pattern closes, and the quotes and slashes inside it are
			// never read as anything.
			if end := jsSkipRegex(content, i, endPoint, floor); end >= 0 {
				i = end
			}
		case '"', '\'', '`':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				switch curByte {
				case '"':
					return i, SString, jsDoubleQuote
				case '\'':
					return i, SString, jsSingleQuote
				}

				return i, SString, jsBacktick
			}

			return i, SCode, nil
		case '?':
			if jsPostfixToken(content, i) != 0 && jsPostfixCounts(content, i) {
				tally.Complexity++
			}
		default:
			if jsComplexityAnchored(content, i, floor) {
				tally.Complexity++
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which is
	// the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode, nil
	}

	return index, SCode, nil
}

// countLoopJavaScript stands in for countLoopGeneric where the language is
// JavaScript and none of the extra outputs are wanted. It returns false when it
// ended the count early, the same way the generic loop does.
func countLoopJavaScript(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := jsStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for. JavaScript has three and the
	// code and blank states say which one opened, so it is carried across calls.
	// It is never nil: the states below hand back the quote only when they
	// opened a string, and anything else leaves the last one in place rather
	// than clearing it.
	endQuote := jsDoubleQuote
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := jsCodeState(content, &tally, index, endPoint, floor, stop)
			openQuote(quote)

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			return counterCommentState(content, index, endPoint, state, slashStarOpen, slashStarClose, false, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote := jsBlankState(content, &tally, index, endPoint, floor)
			openQuote(quote)

			return index, state
		}
	}

	// JavaScript does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, false, step)
}

// useJavaScriptCounter reports whether the JavaScript counter can answer for
// this file.
func useJavaScriptCounter(fileJob *FileJob) bool {
	if fileJob.Language != "JavaScript" {
		return false
	}

	return specialisedCounterEligible(fileJob)
}
