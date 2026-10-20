// SPDX-License-Identifier: MIT

package processor

// A counter written for Scala, on the same terms as the others: it must agree
// with the generic loop to the line, and where the two differ the generic one
// is right by definition.
//
// Scala asks for one thing no counter had before. Four of its complexity checks
// are not words at all: >= , >	, <=  and < , spelled with a trailing space and
// carrying no letters. They are ordinary prefix checks and so still want the
// word boundary the generic loop tests, which has a consequence worth knowing:
// a > written tight against an identifier, as in a> b, is not counted, while
// a > b is. That is the generic loop's rule and this reproduces it.
//
// It also means the arrow of a lambda is counted. In x => y the > has an = in
// front of it, which is not a byte that carries a word on, and a space behind
// it, so it matches >  and scores. Very likely not what anyone intended when
// the check list was written, but it is what scc has always done and a counter
// that fixed it would be a counter that disagrees with the generic loop.
//
// Block comments nest, which is the shared comment state's job.
//
// Three quotes. The plain " has the ordinary backslash escape. Three of them
// open a multi-line string, which has no escape at all, so a backslash in front
// of its closer does not carry it on and a run of more than three quotes closes
// on the last three. And '" opens the character literal '"', which the
// apostrophe after it closes. An apostrophe on its own is not a quote: it also
// opens a symbol literal such as 'name and a Scala 3 quote such as '{ x }, and
// neither of those has a closer, so as a quote it would run on to the next
// apostrophe in the file.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// scalaStop marks every byte the scan has to stop on: the slash of both comment
// forms, the two bytes a quote opens with, one anchor byte out of every
// complexity check, and the newline and the null that end a line and a file of
// bytes rather than text.
var scalaStop = buildScalaStop()

// scalaStopNoComplexity is scalaStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
var scalaStopNoComplexity = buildScalaStopNoComplexity()

// The closers of Scala's quotes, held here so the shared string state is handed
// a slice rather than building one per string.
var (
	scalaDoubleQuote = []byte{'"'}
	scalaTripleQuote = []byte(`"""`)
	scalaApostrophe  = []byte{'\''}
)

// scalaComplexityAnchors is the byte each complexity check of Scala is stopped
// on. It is what the structural conformance test holds against scalaStop, and
// it is the written form of the argument in buildScalaStop.
var scalaComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"|| ":     '|',
	"&& ":     '&',
	"!= ":     '=',
	"== ":     '=',
	">= ":     '>',
	"> ":      '>',
	"<= ":     '<',
	"< ":      '<',
}

// buildScalaStop marks the anchor of every complexity check of Scala.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	>   >= and >, told apart forwards on the byte after the bracket
//	<   <= and <
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each anchor is the rarest byte of its own check. Measured over 99.3MB of
// Scala, w is 0.62% of all bytes against s at 3.85% and i at 3.77%; f is 1.23%
// against i at 3.77% and o at 3.61%; l is 2.89% against e at 7.49%. The four
// bracket checks are anchored on their own first byte, < at 0.04% and > at
// 0.20%, both already the rarest byte they are spelled with.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match, which is what lets the scan
// carry on through a matched token rather than stepping over it the way the
// generic loop does:
//
//   - while carries an l, but else wants an e in front of it and finds an i.
//   - >=  and <=  carry an =, but == wants an = or a ! in front of its = and
//     finds a bracket, so the pair is counted once rather than twice.
//
// So the scan never counts a check twice and never counts one that is not
// there.
func buildScalaStop() [256]bool {
	table := buildScalaStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&', '=', '>', '<'} {
		table[b] = true
	}

	return table
}

func buildScalaStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// scalaStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func scalaStopTable() *[256]bool {
	if Complexity {
		return &scalaStopNoComplexity
	}

	return &scalaStop
}

// scalaBracketCheck reports whether one of the four bracket checks sits at
// index, which is a >  or a <  with an optional = between the bracket and the
// space. The caller has already established the word boundary.
func scalaBracketCheck(content []byte, index, floor int) bool {
	if hasPrefixAt(content, index+1, floor, "= ") {
		return true
	}

	return spaceOpens(content, index+1)
}

// scalaComplexityAnchored reports whether a complexity check of Scala sits on
// the anchor byte at index, which is what scalaStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of Scala is spelled with,
// so reading back never crosses out of the code the scan is in. Nor can it read
// in front of the region the counter owns, every backwards read being clamped
// to floor.
func scalaComplexityAnchored(content []byte, index, floor int) bool {
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
	case '>', '<':
		return wordStartsAt(content, index, floor) && scalaBracketCheck(content, index, floor)
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

// scalaComplexityAtLineStart is scalaComplexityAnchored for the first byte of
// code on a line, which has nothing in front of it to read back to. Only the
// checks anchored on their own first byte are looked for here; the rest are
// anchored on a byte the code scan reaches, since that begins on the byte after
// this one and no check that is anchored on its first byte holds an anchor of
// its own anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, while, the
// four bracket checks, || and && are never counted when they open a line, and
// it is the reason the backwards reads above can be written as reads rather
// than as searches. The two are one thing.
func scalaComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case '>', '<':
		return scalaBracketCheck(content, index, floor)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// scalaTripleQuoteAt reports whether the quote at index opens a multi-line
// string, which is three of them in a row. The generic loop takes the longest
// quote it can match, and does not test what sits in front of one of three bytes
// for an escape, since the byte it tests is the second quote of the three.
func scalaTripleQuoteAt(content []byte, index int) bool {
	return index+2 < len(content) && content[index+1] == '"' && content[index+2] == '"'
}

// scalaCharQuoteAt reports whether the apostrophe at index opens the character
// literal '"'. The generic loop tests the byte in front of the quote that ends
// the opener for an escape, which here is the apostrophe, so nothing in front
// of the apostrophe keeps it from opening.
func scalaCharQuoteAt(content []byte, index int) bool {
	return index+1 < len(content) && content[index+1] == '"'
}

// scalaBlankState looks at the first byte of content on a line.
func scalaBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte) {
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
	case '"':
		// The cursor is left on the last byte of the opener, so the scan begins
		// after it rather than reading the second quote as closing an empty
		// string.
		if scalaTripleQuoteAt(content, index) {
			return index + 2, SString, scalaTripleQuote
		}

		return index, SString, scalaDoubleQuote
	case '\'':
		if scalaCharQuoteAt(content, index) {
			return index + 1, SString, scalaApostrophe
		}
	}

	if !Complexity && scalaComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// scalaCodeState runs to the end of the line or to whatever token takes it out
// of code.
func scalaCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
		case '"':
			if scalaTripleQuoteAt(content, i) {
				return i + 2, SString, scalaTripleQuote
			}

			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString, scalaDoubleQuote
			}

			return i, SCode, nil
		case '\'':
			if scalaCharQuoteAt(content, i) {
				return i + 1, SString, scalaApostrophe
			}
		default:
			if scalaComplexityAnchored(content, i, floor) {
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

// countLoopScala stands in for countLoopGeneric where the language is Scala and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopScala(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := scalaStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// How deep the block comment the scan is inside runs. Scala nests them, so a
	// comment left open at the end of a line is open to a depth the next line
	// has to know; the generic loop keeps the same count in endComments.
	commentDepth := 1

	// The quote the string state is looking for, and whether it is the one of
	// three, which has no escape. The code and blank states say which one
	// opened, so both are carried across calls. The states hand back a quote
	// only when they opened a string, and anything else leaves the last one in
	// place.
	endQuote := scalaDoubleQuote
	multiLine := false
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
			multiLine = len(quote) == len(scalaTripleQuote)
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := scalaCodeState(content, &tally, index, endPoint, floor, stop)
			openQuote(quote)
			if state == SMulticommentCode {
				commentDepth = 1
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, multiLine)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			// Scala nests its block comments, so the closer that ends this one is
			// the one that brings the depth back to zero.
			index, state, commentDepth = counterNestedCommentState(content, index, endPoint, state, slashStarOpen, slashStarClose, commentDepth)

			return index, state
		default: // SBlank and SMulticommentBlank
			index, state, quote := scalaBlankState(content, &tally, index, floor)
			openQuote(quote)
			if state == SMulticomment {
				commentDepth = 1
			}

			return index, state
		}
	}

	// Scala does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
