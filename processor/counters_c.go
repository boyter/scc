// SPDX-License-Identifier: MIT

package processor

// A counter written for C and C Header, on the same terms as the Java one: it
// must agree with the generic loop to the line, and where the two differ the
// generic one is right by definition.
//
// C asks for two things Java does not. Lines joined by a backslash carry a line
// comment on into the next line and end a string that is not carried, which is
// the linesplice flag in languages.json. And C has no character literal in its
// table, so a quote inside one opens a string, which is wrong but is what the
// generic loop does and so is what this has to do too.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// cStop marks every byte the scan stops on for C: the slash of both comment
// forms, the one quote, one anchor byte out of every complexity check, and the
// newline and null that end a line and a file of bytes rather than text.
//
// The anchor is not the first byte of the check but the rarest byte in it,
// which is what keeps the scan from stopping on a fifth of the file. Counted
// over 2.5MB of the kernel, stopping on the first byte of every check costs
// 21.5% of all bytes; stopping on the anchors costs 10.5%. The saving is nearly
// all of it e, i and s, which open if, switch and else and are three of the
// five commonest bytes in C.
var cStop = buildCStop()

// cStopNoComplexity holds only what changes the state, which is what a file
// counted with --no-complexity is scanned with.
var cStopNoComplexity = buildCStopNoComplexity()

// cQuote is the one quote of C. Held here so the shared string state is handed
// a slice it can compare without building one per call.
var cQuote = []byte{'"'}

// cComplexityAnchors is the byte each complexity check of C is stopped on. It
// is what the structural conformance test holds against cStop, and it is the
// written form of the argument in buildCStop.
var cComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w', "switch(": 'w',
	"case ":  'c',
	"while ": 'w', "while(": 'w',
	"else ": 'l', "else{": 'l',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
}

// buildCStop marks the anchor of every complexity check of C.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	c   case, which C Header counts and C does not
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
func buildCStop() [256]bool {
	table := buildCStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'c', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildCStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// cStopTable picks the table the scan runs with. The global reads as complexity
// having been turned off.
func cStopTable() *[256]bool {
	if Complexity {
		return &cStopNoComplexity
	}

	return &cStop
}

// cComplexityAnchored reports whether a complexity check of C sits on the
// anchor byte at index, which is what cStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check is spelled with, so
// reading back never crosses out of the code the scan is in. Nor can it read in
// front of the region the counter owns, every backwards read being clamped to
// floor.
//
// The pairs are told apart on the byte behind the anchor and cannot both match:
// the f of if has an i behind it and the f of for cannot, i being a byte that
// carries a word on, and the same holds of the w of switch against while and
// the = of != against ==.
func cComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'l':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "se") && braceOpens(content, index+3)
	case 'w':
		if byteBefore(content, index, floor) == 's' {
			return wordStartsAt(content, index-1, floor) &&
				hasPrefixAt(content, index+1, floor, "itch") && cOpens(content, index+5)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
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
	case 'c':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "ase ")
	}

	return false
}

// cBlankState looks at the first byte of content on a line.
func cBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState) {
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

	if !Complexity && cComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode
}

// cComplexityAtLineStart is cComplexityAnchored for the first byte of code on a
// line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches, since that begins on the byte after this one
// and no check that is anchored on its first byte holds an anchor of its own
// anywhere else.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
//
// This is not an optimisation that can be left out. Without it the checks
// anchored on their first byte are never found at all, and it is the reason the
// backwards reads of cComplexityAnchored can be written as reads rather than as
// searches. The two are one thing.
func cComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	case 'c':
		return hasPrefixAt(content, index+1, floor, "ase ")
	}

	return false
}

// cAnchorBit gives each anchor byte of C a bit; cAnchorPrev[prev] holds the bits
// of every anchor whose first test could still pass with prev in front of it.
// The AND of the two is zero exactly where cComplexityAnchored would have
// returned false on its first test.
const (
	cAnchorBitF uint8 = 1 << iota
	cAnchorBitL
	cAnchorBitW
	cAnchorBitEq
	cAnchorBitC
	cAnchorBitAmp
	cAnchorBitPipe
)

var cAnchorBit = buildCAnchorBit()

var cAnchorPrev = buildCAnchorPrev()

func buildCAnchorBit() [256]uint8 {
	var table [256]uint8
	table['f'] = cAnchorBitF
	table['l'] = cAnchorBitL
	table['w'] = cAnchorBitW
	table['='] = cAnchorBitEq
	table['c'] = cAnchorBitC
	table['&'] = cAnchorBitAmp
	table['|'] = cAnchorBitPipe

	return table
}

func buildCAnchorPrev() [256]uint8 {
	var table [256]uint8
	for value := range 256 {
		previous := byte(value)
		boundary := !isIdentifierContinue(previous)

		var mask uint8
		if previous == 'i' || boundary {
			mask |= cAnchorBitF
		}
		if previous == 'e' {
			mask |= cAnchorBitL
		}
		if previous == 's' || boundary {
			mask |= cAnchorBitW
		}
		if previous == '=' || previous == '!' {
			mask |= cAnchorBitEq
		}
		if boundary {
			mask |= cAnchorBitC | cAnchorBitAmp | cAnchorBitPipe
		}
		table[value] = mask
	}

	return table
}

// cCodeState runs to the end of the line or to whatever token takes it out of
// code.
func cCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState) {
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
			// run of them, so a quote behind a backslash opens nothing. A quote
			// on the floor has nothing in front of it and so is not escaped; the
			// state machine cannot reach here on it, having started blank, but
			// the check does not depend on that holding.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString
			}

			return i, SCode
		default:
			// 94% of anchor stops fail, and 88% of those are settled by the
			// byte in front of the anchor alone. This is that test, hoisted in
			// front of the call that would otherwise be paid to reach it: a
			// zero AND is exactly the case cComplexityAnchored rejects on its
			// own first test, so it cannot change a count. It skips four calls
			// in five over the kernel.
			if cAnchorPrev[byteBefore(content, i, floor)]&cAnchorBit[curByte] == 0 {
				continue
			}

			if cComplexityAnchored(content, i, floor) {
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

// countLoopC stands in for countLoopGeneric where the language is C or C
// Header. It returns false when it ended the count early, the same way the
// generic loop does.
func countLoopC(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := cStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			return cCodeState(content, &tally, index, endPoint, floor, stop)
		case SString:
			return counterStringState(content, index, endPoint, floor, cQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state before the
			// newline, so the rest of the line is skipped a vector at a time.
			// Whether it carries on past the newline is the splice, worked out
			// in the shared loop where the line is counted.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			return counterCommentState(content, index, endPoint, state, slashStarClose, &tally)
		default: // SBlank and SMulticommentBlank
			return cBlankState(content, &tally, index, floor)
		}
	}

	// C joins a line ending in a backslash to the one under it before it looks
	// for a comment or a string, which carries a line comment on and ends a
	// string that is not carried.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{Splices: true}, step)
}
