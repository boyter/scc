// SPDX-License-Identifier: MIT

package processor

// A counter written for Swift, on the same terms as the others: it must agree
// with the generic loop to the line, and where the two differ the generic one
// is right by definition.
//
// Swift asks for one thing no counter had before: a complexity check of a
// single byte that is not a word. The ? of an optional is in complexitychecks,
// as a prefix check, so it carries the ordinary word boundary and counts only
// where the byte in front of it does not carry a word on. That makes the common
// spellings score nothing — foo?, Int?, try?, as? all have a letter in front of
// the ? — while a ?? b scores twice, the first ? having a space in front of it
// and the second a ?, neither of which carries a word on.
//
// That is the generic loop's rule rather than a reading of Swift, and this
// reproduces it. A counter that scored optionals the way a Swift programmer
// would is a counter that disagrees with the generic loop.
//
// Swift also writes most of its keywords with a space and no bracket form:
// switch, while, else, catch and guard are all spelled once in languages.json,
// with a trailing space, so switch(x) scores nothing where switch (x) scores
// one. Only for and if carry the bracket form.
//
// Block comments nest, which is the shared comment state's job.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// swiftStop marks every byte the scan has to stop on: the slash of both comment
// forms, the one quote, one anchor byte out of every complexity check, and the
// newline and the null that end a line and a file of bytes rather than text.
var swiftStop = buildSwiftStop()

// swiftStopNoComplexity is swiftStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
var swiftStopNoComplexity = buildSwiftStopNoComplexity()

// swiftQuote is the one quote of Swift.
var swiftQuote = []byte{'"'}

// swiftComplexityAnchors is the byte each complexity check of Swift is stopped
// on. It is what the structural conformance test holds against swiftStop, and
// it is the written form of the argument in buildSwiftStop.
var swiftComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"catch ":  'h',
	"guard ":  'g',
	"?":       '?',
	"|| ":     '|',
	"&& ":     '&',
	"!= ":     '=',
	"== ":     '=',
}

// buildSwiftStop marks the anchor of every complexity check of Swift.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	h   catch
//	g   guard
//	?   the optional, which is one byte and is its own anchor
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each anchor is the rarest byte of its own check. Measured over 9.9MB of
// Swift, w is 0.65% of all bytes against s at 3.66% and i at 3.45%; g is 0.72%
// against u and d at 2.03% and r at 4.24%; h is 1.41% against c at 2.02% and a
// at 3.78%; f is 1.46% against i at 3.45% and o at 3.52%; l is 2.91% against e
// at 7.90%. The ? is 0.08% and is the whole of its own check.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match, which is what lets the scan
// carry on through a matched token rather than stepping over it the way the
// generic loop does:
//
//   - switch carries an h, but catch wants the four bytes catc in front of its
//     h and finds witc.
//   - while carries an h with nothing but its own w in front of it, and an l,
//     but else wants an e in front of its l and finds an i.
//
// So the scan never counts a check twice and never counts one that is not
// there.
func buildSwiftStop() [256]bool {
	table := buildSwiftStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'h', 'g', '?', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildSwiftStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// swiftStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func swiftStopTable() *[256]bool {
	if Complexity {
		return &swiftStopNoComplexity
	}

	return &swiftStop
}

// swiftComplexityAnchored reports whether a complexity check of Swift sits on
// the anchor byte at index, which is what swiftStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of Swift is spelled with,
// so reading back never crosses out of the code the scan is in. Nor can it read
// in front of the region the counter owns, every backwards read being clamped
// to floor.
func swiftComplexityAnchored(content []byte, index, floor int) bool {
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
	case 'h':
		return wordStartsAt(content, index-4, floor) &&
			hasPrefixAt(content, index-4, floor, "catch") && spaceOpens(content, index+1)
	case 'g':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "uard") && spaceOpens(content, index+5)
	case '?':
		// One byte, so the anchor is the whole check and the word boundary is
		// tested where the scan already stands.
		return wordStartsAt(content, index, floor)
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

// swiftComplexityAtLineStart is swiftComplexityAnchored for the first byte of
// code on a line, which has nothing in front of it to read back to. Only the
// checks anchored on their own first byte are looked for here; the rest are
// anchored on a byte the code scan reaches, since that begins on the byte after
// this one and no check that is anchored on its first byte holds an anchor of
// its own anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test, and the ? that opens a line is counted for
// exactly that reason.
//
// This is not an optimisation that can be left out. Without it for, while,
// guard, ?, || and && are never counted when they open a line, and it is the
// reason the backwards reads above can be written as reads rather than as
// searches. The two are one thing.
func swiftComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case 'g':
		return hasPrefixAt(content, index+1, floor, "uard") && spaceOpens(content, index+5)
	case '?':
		return true
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// swiftBlankState looks at the first byte of content on a line.
func swiftBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState) {
	switch content[index] {
	case '/':
		if index+1 < len(content) {
			switch content[index+1] {
			case '/':
				return index, SComment
			case '*':
				return index + 1, SMulticomment
			}
		}
	case '"':
		return index, SString
	}

	if !Complexity && swiftComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode
}

// swiftCodeState runs to the end of the line or to whatever token takes it out
// of code.
func swiftCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState) {
	if endPoint > len(content) {
		endPoint--
	}

	mask := stopMask(stop)

	for i := index; i < endPoint; i++ {
		// Eight bytes answered at a time, without a branch between them. See
		// scanToStop.
		at := scanToStop(content, i, endPoint, mask)
		if at < 0 {
			break
		}
		i = at
		curByte := content[i]

		switch curByte {
		case '\n':
			return i, SCode
		case 0:
			if isBinary(i, curByte) {
				tally.Binary = true
				return i, SCode
			}
		case '/':
			if i+1 < len(content) {
				switch content[i+1] {
				case '/':
					return i, SCommentCode
				case '*':
					return i + 1, SMulticommentCode
				}
			}
		case '"':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString
			}

			return i, SCode
		default:
			if swiftComplexityAnchored(content, i, floor) {
				tally.Complexity++
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which is
	// the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode
	}

	return index, SCode
}

// countLoopSwift stands in for countLoopGeneric where the language is Swift and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopSwift(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := swiftStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// How deep the block comment the scan is inside runs. Swift nests them, so a
	// comment left open at the end of a line is open to a depth the next line
	// has to know; the generic loop keeps the same count in endComments.
	commentDepth := 1

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state := swiftCodeState(content, &tally, index, endPoint, floor, stop)
			if state == SMulticommentCode {
				commentDepth = 1
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, swiftQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			// Swift nests its block comments, so the closer that ends this one is
			// the one that brings the depth back to zero.
			index, state, commentDepth = counterNestedCommentState(content, index, endPoint, state, slashStarOpen, slashStarClose, commentDepth)

			return index, state
		default: // SBlank and SMulticommentBlank
			index, state := swiftBlankState(content, &tally, index, floor)
			if state == SMulticomment {
				commentDepth = 1
			}

			return index, state
		}
	}

	// Swift does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
