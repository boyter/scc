// SPDX-License-Identifier: MIT

package processor

// A counter written for Kotlin, on the same terms as the others: it must agree
// with the generic loop to the line, and where the two differ the generic one
// is right by definition.
//
// Kotlin is Java with two differences that matter here. It spells its multi-way
// branch when rather than switch, and its block comments nest, so /* a /* b */
// still runs until a second closer. The nesting is the shared comment state's
// job; this file is the stop table and the matcher.
//
// One quote, the plain ", with the ordinary backslash escape. The raw string of
// three quotes is not in languages.json and so is not modelled here either: a
// counter that handled it would disagree with the generic loop, which is the
// one thing a counter may not do.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// kotlinStop marks every byte the scan has to stop on: the slash of both
// comment forms, the one quote, one anchor byte out of every complexity check,
// and the newline and the null that end a line and a file of bytes rather than
// text.
var kotlinStop = buildKotlinStop()

// kotlinStopNoComplexity is kotlinStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
var kotlinStopNoComplexity = buildKotlinStopNoComplexity()

// kotlinQuote is the one quote of Kotlin.
var kotlinQuote = []byte{'"'}

// kotlinComplexityAnchors is the byte each complexity check of Kotlin is
// stopped on. It is what the structural conformance test holds against
// kotlinStop, and it is the written form of the argument in buildKotlinStop.
var kotlinComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"finally ": 'f', "finally{": 'f',
	"when ": 'w', "when(": 'w',
	"while ": 'w', "while(": 'w',
	"else ": 'l', "else{": 'l',
	"try ": 'y', "try{": 'y',
	"catch ": 'h', "catch(": 'h',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
}

// buildKotlinStop marks the anchor of every complexity check of Kotlin.
//
//	f   if, for, finally   the f of if is read backwards, the other two forwards
//	w   when, while        told apart forwards, on the byte after their shared wh
//	l   else
//	y   try
//	h   catch
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each anchor is the rarest byte of its own check. Measured over 10.0MB of
// Kotlin, w is 0.42% of all bytes against i at 3.68% and e at 7.64%; y is 0.71%
// against t at 5.72%; f is 0.88% against i at 3.68% and o at 3.46%; h is 1.16%
// against c at 2.11% and a at 4.37%; l is 2.88% against e at 7.64%.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match, which is what lets the scan
// carry on through a matched token rather than stepping over it the way the
// generic loop does:
//
//   - finally carries two l and a y, but else wants an e in front of its l and
//     finds an a and then an l, and try wants an r in front of its y and finds
//     an l.
//   - when and while both carry an h, but catch wants the four bytes catc in
//     front of its h and finds neither.
//   - while carries an l, but else wants an e in front of it and finds an i.
//
// So the scan never counts a check twice and never counts one that is not
// there.
func buildKotlinStop() [256]bool {
	table := buildKotlinStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'y', 'h', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildKotlinStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// kotlinStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func kotlinStopTable() *[256]bool {
	if Complexity {
		return &kotlinStopNoComplexity
	}

	return &kotlinStop
}

// kotlinComplexityAnchored reports whether a complexity check of Kotlin sits on
// the anchor byte at index, which is what kotlinStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of Kotlin is spelled with,
// so reading back never crosses out of the code the scan is in. Nor can it read
// in front of the region the counter owns, every backwards read being clamped
// to floor.
func kotlinComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}
		if !wordStartsAt(content, index, floor) {
			return false
		}

		return (hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)) ||
			(hasPrefixAt(content, index+1, floor, "inally") && braceOpens(content, index+7))
	case 'w':
		if !wordStartsAt(content, index, floor) {
			return false
		}

		// when and while share their first two bytes and part on the third.
		return (hasPrefixAt(content, index+1, floor, "hen") && cOpens(content, index+4)) ||
			(hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5))
	case 'l':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "se") && braceOpens(content, index+3)
	case 'y':
		if byteBefore(content, index, floor) != 'r' {
			return false
		}

		return wordStartsAt(content, index-2, floor) &&
			hasPrefixAt(content, index-2, floor, "try") && braceOpens(content, index+1)
	case 'h':
		return wordStartsAt(content, index-4, floor) &&
			hasPrefixAt(content, index-4, floor, "catch") && cOpens(content, index+1)
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

// kotlinComplexityAtLineStart is kotlinComplexityAnchored for the first byte of
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
// This is not an optimisation that can be left out. Without it for, finally,
// when, while, || and && are never counted when they open a line, and it is the
// reason the backwards reads above can be written as reads rather than as
// searches. The two are one thing.
func kotlinComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return (hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)) ||
			(hasPrefixAt(content, index+1, floor, "inally") && braceOpens(content, index+7))
	case 'w':
		return (hasPrefixAt(content, index+1, floor, "hen") && cOpens(content, index+4)) ||
			(hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5))
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// kotlinBlankState looks at the first byte of content on a line.
func kotlinBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState) {
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

	if !Complexity && kotlinComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode
}

// kotlinCodeState runs to the end of the line or to whatever token takes it out
// of code.
func kotlinCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState) {
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
			if kotlinComplexityAnchored(content, i, floor) {
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

// countLoopKotlin stands in for countLoopGeneric where the language is Kotlin
// and none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopKotlin(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := kotlinStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// How deep the block comment the scan is inside runs. Kotlin nests them, so a
	// comment left open at the end of a line is open to a depth the next line
	// has to know; the generic loop keeps the same count in endComments.
	commentDepth := 1

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state := kotlinCodeState(content, &tally, index, endPoint, floor, stop)
			if state == SMulticommentCode {
				commentDepth = 1
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, kotlinQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			// Kotlin nests its block comments, so the closer that ends this one is
			// the one that brings the depth back to zero.
			index, state, commentDepth = counterNestedCommentState(content, index, endPoint, state, slashStarOpen, slashStarClose, commentDepth)

			return index, state
		default: // SBlank and SMulticommentBlank
			index, state := kotlinBlankState(content, &tally, index, floor)
			if state == SMulticomment {
				commentDepth = 1
			}

			return index, state
		}
	}

	// Kotlin does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
