// SPDX-License-Identifier: MIT

package processor

import "strings"

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
// The plain " and the """ of a multi-line string both have the ordinary
// backslash escape. A raw string opens with up to three hashes in front of
// either and closes with the same number behind it, #"…"# or #"""…"""#, and a
// backslash in front of its closer does not carry it on. And #"""# is not the
// opener of a multi-line raw string but a one-line one holding a quote, which
// Swift tells apart by its closer being on the same line, and languages.json
// spells as a quote of its own.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// swiftStop marks every byte the scan has to stop on: the slash of both comment
// forms, the quote and the hash a raw string opens with, one anchor byte out of
// every complexity check, and the newline and the null that end a line and a
// file of bytes rather than text.
var swiftStop = buildSwiftStop()

// swiftStopNoComplexity is swiftStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
var swiftStopNoComplexity = buildSwiftStopNoComplexity()

// The closers of the plain and the multi-line string, held here so the shared
// string state is handed a slice rather than building one per string.
var (
	swiftQuote      = []byte{'"'}
	swiftMultiQuote = []byte(`"""`)
)

// maxSwiftRawHashes is the longest run of hashes languages.json gives a raw
// string. A run longer than this opens no raw string at its first hash in
// either loop, the generic loop's trie having no node for it, and both go on to
// open one on its last three.
const maxSwiftRawHashes = 3

// swiftRawEnds and swiftRawMultiEnds are what a raw string closes with, indexed
// by how many hashes its opening token carried. Built once rather than assembled
// per string.
var swiftRawEnds, swiftRawMultiEnds = buildSwiftRawEnds()

func buildSwiftRawEnds() ([maxSwiftRawHashes + 1][]byte, [maxSwiftRawHashes + 1][]byte) {
	var ends, multiEnds [maxSwiftRawHashes + 1][]byte
	for hashes := 1; hashes <= maxSwiftRawHashes; hashes++ {
		pound := strings.Repeat("#", hashes)
		ends[hashes] = []byte(`"` + pound)
		multiEnds[hashes] = []byte(`"""` + pound)
	}

	return ends, multiEnds
}

// swiftQuotes is every quote of Swift as languages.json spells it, start and end
// in turn, which the structural conformance test holds the database to. Built
// rather than written out, since nine of the eleven differ only in how many
// hashes they carry.
func swiftQuotes() []string {
	quotes := []string{`"`, `"`, `"""`, `"""`}
	for hashes := 1; hashes <= maxSwiftRawHashes; hashes++ {
		pound := strings.Repeat("#", hashes)
		quotes = append(quotes,
			pound+`"`, `"`+pound,
			pound+`"""`, `"""`+pound,
			pound+`"""`+pound, `"`+pound,
		)
	}

	return quotes
}

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
	for _, b := range []byte{'/', '"', '#', '\n', 0} {
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

// swiftAnchorBit gives each anchor byte of Swift a bit; swiftAnchorPrev[prev]
// holds the bits of every anchor whose first test could still pass with prev in
// front of it. The AND of the two is zero exactly where swiftComplexityAnchored
// would have returned false on its first test.
//
// Nine anchors, so these are uint16 where the other counters hold theirs in a
// byte. Two tables of 512 bytes against two of 256 is not a difference the scan
// can feel, and giving the ? a bit of its own is worth more than the byte: the
// ? of an optional is the one anchor of Swift that is nearly always preceded by
// a letter, which is exactly the case this table settles.
const (
	swiftAnchorBitF uint16 = 1 << iota
	swiftAnchorBitW
	swiftAnchorBitL
	swiftAnchorBitH
	swiftAnchorBitG
	swiftAnchorBitQuestion
	swiftAnchorBitEq
	swiftAnchorBitAmp
	swiftAnchorBitPipe
)

var swiftAnchorBit = buildSwiftAnchorBit()

var swiftAnchorPrev = buildSwiftAnchorPrev()

func buildSwiftAnchorBit() [256]uint16 {
	var table [256]uint16
	table['f'] = swiftAnchorBitF
	table['w'] = swiftAnchorBitW
	table['l'] = swiftAnchorBitL
	table['h'] = swiftAnchorBitH
	table['g'] = swiftAnchorBitG
	table['?'] = swiftAnchorBitQuestion
	table['='] = swiftAnchorBitEq
	table['&'] = swiftAnchorBitAmp
	table['|'] = swiftAnchorBitPipe

	return table
}

// buildSwiftAnchorPrev writes down, for each byte that can sit in front of an
// anchor, which anchors a match could still begin behind. It is the first test
// of each arm of swiftComplexityAnchored and nothing more:
//
//	f   if, for         an i in front for if, or a word boundary for for
//	w   switch, while   an s in front for switch, or a word boundary for while
//	l   else            an e in front
//	h   catch           a c in front
//	g   guard           anchored on its own first byte, so a word boundary
//	?   the optional    one byte, so the boundary is tested where it stands
//	=   ==, !=          an = or a ! in front
//	&, |                anchored on their own first byte, so a word boundary
//
// catch is the one arm whose first test is not itself a read of the byte in
// front: it asks for a word boundary four bytes back and then for catc there.
// That prefix cannot hold unless the byte in front of the h is the c of catc,
// so a c in front is a condition of the arm returning true even though it is
// not the order the arm tests it in.
//
// A word boundary is the byte in front not carrying a word on, which is what
// wordStartsAt asks. byteBefore reads a zero where the anchor sits on the floor
// and there is nothing in front of it, and zero carries no word on, so the
// floor comes out of the table as the boundary wordStartsAt already calls it.
func buildSwiftAnchorPrev() [256]uint16 {
	var table [256]uint16
	for value := range 256 {
		previous := byte(value)
		boundary := !isIdentifierContinue(previous)

		var mask uint16
		if boundary {
			mask |= swiftAnchorBitG | swiftAnchorBitQuestion | swiftAnchorBitAmp | swiftAnchorBitPipe
		}
		if previous == 'i' || boundary {
			mask |= swiftAnchorBitF
		}
		if previous == 's' || boundary {
			mask |= swiftAnchorBitW
		}
		if previous == 'e' {
			mask |= swiftAnchorBitL
		}
		if previous == 'c' {
			mask |= swiftAnchorBitH
		}
		if previous == '=' || previous == '!' {
			mask |= swiftAnchorBitEq
		}
		table[value] = mask
	}

	return table
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

// swiftMultiLineAt reports whether the quote at index opens a multi-line
// string, which is three of them in a row. The generic loop takes the longest
// quote it can match, and does not test what sits in front of one of three bytes
// for an escape, since the byte it tests is the second quote of the three.
func swiftMultiLineAt(content []byte, index int) bool {
	return index+2 < len(content) && content[index+1] == '"' && content[index+2] == '"'
}

// swiftRawStringAt reports whether a raw string opens on the hash at index, and
// if so the quote its opening token is read up to and what closes it.
//
// The generic loop finds these forwards from its trie, which takes the longest
// token at the earliest byte, so a run of more hashes than any token carries
// opens nothing on its first hash and is tried again on the next. Whichever of
// the three tokens it matched, the cursor is left on the quote after the
// hashes: prepareString steps to the end of the first quote in languages.json
// whose start is there, and #" is listed ahead of #""" and #"""#. That is what
// lets #"""# close on its own last two bytes. The token decides only the
// closer.
func swiftRawStringAt(content []byte, index, floor int) (int, []byte, bool) {
	hashes := 0
	for index+hashes < len(content) && content[index+hashes] == '#' {
		hashes++
		if hashes > maxSwiftRawHashes {
			return index, nil, false
		}
	}

	quote := index + hashes
	if quote >= len(content) || content[quote] != '"' {
		return index, nil, false
	}

	if !swiftMultiLineAt(content, quote) {
		return quote, swiftRawEnds[hashes], true
	}

	// The hashes straight after the three quotes make it #"""#, a raw string of
	// one line holding a quote, rather than the opener of a multi-line one.
	if hasPrefixAt(content, quote+3, floor, string(swiftRawEnds[hashes][1:])) {
		return quote, swiftRawEnds[hashes], true
	}

	return quote, swiftRawMultiEnds[hashes], true
}

// swiftBlankState looks at the first byte of content on a line. When it opens a
// string it hands back the closer and whether the string is a raw one, which
// has no escape.
func swiftBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte, bool) {
	switch content[index] {
	case '/':
		if index+1 < len(content) {
			switch content[index+1] {
			case '/':
				return index, SComment, nil, false
			case '*':
				return index + 1, SMulticomment, nil, false
			}
		}
	case '"':
		// The cursor is left on the last quote of the three, so the scan begins
		// after them rather than reading the second as closing an empty string.
		if swiftMultiLineAt(content, index) {
			return index + 2, SString, swiftMultiQuote, false
		}

		return index, SString, swiftQuote, false
	case '#':
		if quote, endQuote, ok := swiftRawStringAt(content, index, floor); ok {
			return quote, SString, endQuote, true
		}
	}

	if !Complexity && swiftComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// swiftCodeState runs to the end of the line or to whatever token takes it out
// of code.
func swiftCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
			return i, SCode, nil, false
		case 0:
			if isBinary(i, curByte) {
				tally.Binary = true
				return i, SCode, nil, false
			}
		case '/':
			if i+1 < len(content) {
				switch content[i+1] {
				case '/':
					return i, SCommentCode, nil, false
				case '*':
					return i + 1, SMulticommentCode, nil, false
				}
			}
		case '"':
			if swiftMultiLineAt(content, i) {
				return i + 2, SString, swiftMultiQuote, false
			}

			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString, swiftQuote, false
			}

			return i, SCode, nil, false
		case '#':
			// A raw string opens whatever is in front of it, the byte the generic
			// loop tests for an escape being the hash in front of the quote.
			if quote, endQuote, ok := swiftRawStringAt(content, i, floor); ok {
				return quote, SString, endQuote, true
			}
		default:
			// Instrumented over what Swift there is to hand, 46 files: 73.0% of
			// the stops this loop makes are anchor stops, 95.3% of those count
			// nothing, and 77.9% of the failures are settled by the single byte
			// in front of the anchor. That byte is the first thing
			// swiftComplexityAnchored tests and the last thing the caller knows
			// before paying for the call, so it is tested here instead: a zero
			// AND is exactly the case the function rejects on its own first
			// test, so skipping it cannot change a count. It removes 74.3% of
			// the calls.
			if swiftAnchorPrev[byteBefore(content, i, floor)]&swiftAnchorBit[curByte] == 0 {
				continue
			}

			if swiftComplexityAnchored(content, i, floor) {
				tally.Complexity++
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which is
	// the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode, nil, false
	}

	return index, SCode, nil, false
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

	// The quote the string state is looking for, and whether it is a raw one,
	// which has no escape. The code and blank states say which one opened, so
	// both are carried across calls. The states hand back a quote only when they
	// opened a string, and anything else leaves the last one in place.
	endQuote := swiftQuote
	rawString := false
	openQuote := func(quote []byte, raw bool) {
		if quote != nil {
			endQuote = quote
			rawString = raw
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, raw := swiftCodeState(content, &tally, index, endPoint, floor, stop)
			openQuote(quote, raw)
			if state == SMulticommentCode {
				commentDepth = 1
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, rawString)
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
			index, state, quote, raw := swiftBlankState(content, &tally, index, floor)
			openQuote(quote, raw)
			if state == SMulticomment {
				commentDepth = 1
			}

			return index, state
		}
	}

	// Swift does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
