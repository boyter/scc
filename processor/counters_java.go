// SPDX-License-Identifier: MIT

package processor

// A counter written for one language rather than driven by the tries built from
// languages.json. The generic loop asks a trie what token sits at a byte, which
// the profile puts at a quarter of the whole program; Java needs to know about
// four tokens and can answer that with a comparison.
//
// Everything here must agree with the generic loop to the line. Where the two
// differ the generic one is right by definition, since it is what the rest of
// the tests are written against, and countLoopJava is only ever an accelerator.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// javaStop marks every byte the scan has to stop on: the slash of both comment
// forms, the two quotes, one anchor byte out of every complexity check, and the
// newline and the null that end a line and a file of bytes rather than text.
// One load and one branch answer for a byte, where asking each question in turn
// costs several.
var javaStop = buildJavaStop()

// javaStopNoComplexity is javaStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
var javaStopNoComplexity = buildJavaStopNoComplexity()

// The two quotes of Java, held here so the shared string state is handed a
// slice rather than building one per string.
var (
	javaDoubleQuote = []byte{'"'}
	javaSingleQuote = []byte{'\''}
)

// javaComplexityAnchors is the byte each complexity check of Java is stopped
// on. It is what the structural conformance test holds against javaStop, and it
// is the written form of the argument in buildJavaStop.
var javaComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"finally ": 'f', "finally{": 'f',
	"switch ": 'w', "switch(": 'w',
	"while ": 'w', "while(": 'w',
	"else ": 'l', "else{": 'l',
	"try ": 'y', "try{": 'y',
	"catch ": 'h', "catch(": 'h',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
}

// buildJavaStop marks the anchor of every complexity check of Java.
//
//	f   if, for, finally   the f of if is read backwards, the other two forwards
//	w   while, switch
//	l   else
//	y   try
//	h   catch
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// The anchor is the rarest byte of the check rather than its first, which is
// what the C counter does and what Java did not. Stopping on the first byte of
// every check means stopping on f, i, s, w, e, t and c, better than a third of
// the bytes of a Java file; the anchors above are f, w, l, y and h, which are
// nearer a tenth. Counted over 1.8MB of real Java the two rates are 27.1% and
// 11.1%.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match. finally carries an l and a y,
// but the l of else wants an e in front of it and finds an a or an l, and the y
// of try wants an r and finds an l. switch carries the h of catch, but reading
// four bytes back from it gives witc and not catc. So the scan never counts a
// check twice and never counts one that is not there.
func buildJavaStop() [256]bool {
	table := buildJavaStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'y', 'h', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildJavaStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// javaStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func javaStopTable() *[256]bool {
	if Complexity {
		return &javaStopNoComplexity
	}

	return &javaStop
}

// javaComplexityAnchored reports whether a complexity check of Java sits on the
// anchor byte at index, which is what javaStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of Java is spelled with,
// so reading back never crosses out of the code the scan is in. Nor can it read
// in front of the region the counter owns, every backwards read being clamped
// to floor.
//
// The checks that share an anchor are told apart on the byte behind it and
// cannot both match: the f of if has an i behind it and the f of for or finally
// cannot, i being a byte that carries a word on, and the same holds of the w of
// switch against while and the = of != against ==.
func javaComplexityAnchored(content []byte, index, floor int) bool {
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
		if byteBefore(content, index, floor) == 's' {
			return wordStartsAt(content, index-1, floor) &&
				hasPrefixAt(content, index+1, floor, "itch") && cOpens(content, index+5)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
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

// javaComplexityAtLineStart is javaComplexityAnchored for the first byte of
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
// while, || and && are never counted when they open a line, and it is the
// reason the backwards reads above can be written as reads rather than as
// searches. The two are one thing.
func javaComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return (hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)) ||
			(hasPrefixAt(content, index+1, floor, "inally") && braceOpens(content, index+7))
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// javaBlankState looks at the first byte of content on a line.
func javaBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte) {
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
		return index, SString, javaDoubleQuote
	case '\'':
		return index, SString, javaSingleQuote
	}

	if !Complexity && javaComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// javaAnchorBit gives each anchor byte of Java a bit; javaAnchorPrev[prev]
// holds the bits of every anchor whose first test could still pass with prev in
// front of it. The AND of the two is zero exactly where javaComplexityAnchored
// would have returned false on its own first test, so a zero skips a call that
// could only have said no.
//
// The table is a conservative superset: a bit that need not be set costs a call
// that would have been paid anyway, while a bit wrongly cleared loses a count.
// TestAnchorPrefilterIsConservative holds every cleared pair to being a pair the
// function really does reject, over all 256 bytes and every anchor.
const (
	javaAnchorBitF uint8 = 1 << iota
	javaAnchorBitW
	javaAnchorBitL
	javaAnchorBitY
	javaAnchorBitH
	javaAnchorBitEq
	javaAnchorBitPipe
	javaAnchorBitAmp
)

var javaAnchorBit = buildJavaAnchorBit()

var javaAnchorPrev = buildJavaAnchorPrev()

func buildJavaAnchorBit() [256]uint8 {
	var table [256]uint8
	table['f'] = javaAnchorBitF
	table['w'] = javaAnchorBitW
	table['l'] = javaAnchorBitL
	table['y'] = javaAnchorBitY
	table['h'] = javaAnchorBitH
	table['='] = javaAnchorBitEq
	table['|'] = javaAnchorBitPipe
	table['&'] = javaAnchorBitAmp

	return table
}

func buildJavaAnchorPrev() [256]uint8 {
	var table [256]uint8
	for value := range 256 {
		previous := byte(value)
		boundary := !isIdentifierContinue(previous)

		var mask uint8
		// if reads its f backwards over the i, so an i in front keeps it; every
		// other check anchored on f is read forwards from a word boundary.
		if previous == 'i' || boundary {
			mask |= javaAnchorBitF
		}
		// switch reads its w backwards over the s; while reads forwards from a
		// word boundary.
		if previous == 's' || boundary {
			mask |= javaAnchorBitW
		}
		// else, whose l is read back over the e.
		if previous == 'e' {
			mask |= javaAnchorBitL
		}
		// try, whose y is read back over the r.
		if previous == 'r' {
			mask |= javaAnchorBitY
		}
		// catch, whose h is read back over catc, so the byte in front of the
		// anchor is the second c and nothing else.
		if previous == 'c' {
			mask |= javaAnchorBitH
		}
		// == and != are both read back over their first byte.
		if previous == '=' || previous == '!' {
			mask |= javaAnchorBitEq
		}
		// || and && begin on the byte they are anchored on, so the word
		// boundary is the whole of their first test.
		if boundary {
			mask |= javaAnchorBitPipe | javaAnchorBitAmp
		}
		table[value] = mask
	}

	return table
}

// javaCodeState runs to the end of the line or to whatever token takes it out
// of code.
func javaCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
		case '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				if curByte == '"' {
					return i, SString, javaDoubleQuote
				}

				return i, SString, javaSingleQuote
			}

			return i, SCode, nil
		default:
			// The byte in front of the anchor is the first thing
			// javaComplexityAnchored looks at, and it settles most of the
			// anchor stops that go on to count nothing. This is that test
			// hoisted in front of the call that would otherwise be paid to
			// reach it: a zero AND is exactly the case the function rejects on
			// its own first test, so skipping it cannot change a count.
			if javaAnchorPrev[byteBefore(content, i, floor)]&javaAnchorBit[curByte] == 0 {
				continue
			}

			if javaComplexityAnchored(content, i, floor) {
				tally.Complexity++
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which
	// is the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode, nil
	}

	return index, SCode, nil
}

// countLoopJava stands in for countLoopGeneric where the language is Java and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopJava(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := javaStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for. Java has two and the code and
	// blank states say which one opened, so it is carried across calls. It is
	// never nil: the states below hand back the quote only when they opened a
	// string, and anything else leaves the last one in place rather than
	// clearing it. Reading it depends on no invariant that way.
	endQuote := javaDoubleQuote
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := javaCodeState(content, &tally, index, endPoint, floor, stop)
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
			index, state, quote := javaBlankState(content, &tally, index, floor)
			openQuote(quote)

			return index, state
		}
	}

	// Java does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}

// SpecialisedCounters turns on the counters written for one language, which
// stand in for the generic loop where there is one for the file's language.
//
// On by default as of 4.2.0. They were introduced off, behind a flag spelled
// exp- to say that they might change or go away, because the generic loop is
// what every count had been compared against for years. What they have behind
// them now is a differential test that reads every file of several real trees
// both ways, a fuzzer that holds each counter against the generic loop, and
// byte identical output over llvm-project, the Linux kernel, kubernetes,
// cpython, ruby and lucene: 296,599 files, per file and summary.
//
// The flag now turns them off rather than on, which is --no-per-language-counters,
// and it is there for the same reason the differential test is: the generic loop
// is the oracle, so being able to ask it directly is worth keeping.
var SpecialisedCounters = true

// GenericCounterOnly is what --no-per-language-counters sets, and it turns
// SpecialisedCounters off. Held apart from it so the flag reads as the thing it
// is named, a negative, while the code that asks the question keeps asking it
// in the positive.
var GenericCounterOnly bool
