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
// ecmaRegexAllowed.
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
		if end := ecmaSkipRegex(content, index, endPoint, floor); end >= 0 {
			return end, SCode, nil
		}

		return index, SCode, nil
	case '"':
		return index, SString, ecmaDoubleQuote
	case '\'':
		return index, SString, ecmaSingleQuote
	case '`':
		return index, SString, ecmaBacktick
	case '?':
		if !Complexity && ecmaPostfixToken(content, index) != 0 && ecmaPostfixCounts(content, index) {
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
			if end := ecmaSkipRegex(content, i, endPoint, floor); end >= 0 {
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
					return i, SString, ecmaDoubleQuote
				case '\'':
					return i, SString, ecmaSingleQuote
				}

				return i, SString, ecmaBacktick
			}

			return i, SCode, nil
		case '?':
			if ecmaPostfixToken(content, i) != 0 && ecmaPostfixCounts(content, i) {
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
	endQuote := ecmaDoubleQuote
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
			return counterCommentState(content, index, endPoint, state, slashStarClose, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote := jsBlankState(content, &tally, index, endPoint, floor)
			openQuote(quote)

			return index, state
		}
	}

	// JavaScript does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, false, step)
}
