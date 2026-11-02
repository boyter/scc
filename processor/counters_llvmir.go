// SPDX-License-Identifier: MIT

package processor

// A counter written for LLVM IR, on the same terms as the others: it must agree
// with the generic loop to the line, and where the two differ the generic one is
// right by definition.
//
// LLVM IR is the simplest language of the set by token count and the most
// expensive by check count. It has one line comment, a semicolon; one quote, a
// plain double quote that escapes with a backslash; and no block comment at all,
// so three of the eight states the shared loop knows about are unreachable here.
// Against that it carries sixteen complexity checks whose first bytes are l, b,
// s, i, c, r, a, o and x, which is most of the alphabet a machine written
// language is spelled with: the generic loop stops on 23.8% of an LLVM IR file
// before it has looked at a single one of them.
//
// That is the whole reason this counter is worth writing. Anchoring takes the
// same sixteen checks down to seven bytes and 6.1% of the file, and nothing else
// about the language is hard.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// llvmQuote is the one quote of LLVM IR. Held as a slice so the string state is
// handed the same shape every other counter hands it.
var llvmQuote = []byte{'"'}

// llvmStop marks every byte the scan stops on: the semicolon that opens a line
// comment, the one quote, one anchor byte out of every complexity check, and the
// newline and the null that end a line and a file of bytes rather than text.
var llvmStop = buildLLVMStop()

// llvmStopNoComplexity is llvmStop without the bytes only a complexity check is
// spelled with, which is what a file counted with --no-complexity is scanned
// with. For this language that is four bytes against eleven, and it is the
// cheapest table of any counter here.
var llvmStopNoComplexity = buildLLVMStopNoComplexity()

// llvmComplexityAnchors is the byte each complexity check of LLVM IR is stopped
// on. It is what the structural conformance test holds against llvmStop, and it
// is the written form of the argument in buildLLVMStop.
var llvmComplexityAnchors = map[string]byte{
	"llvm.loop": 'm',
	"resume ":   'm',

	"br ":         'b',
	"callbr ":     'b',
	"indirectbr ": 'b',

	"switch ":      'w',
	"catchswitch ": 'w',

	"invoke ": 'k',

	"shl ":  'h',
	"lshr ": 'h',
	"ashr ": 'h',

	"or ":         'r',
	"xor ":        'r',
	"catchret ":   'r',
	"cleanupret ": 'r',

	"and ": 'a',
}

// buildLLVMStop marks the anchor of every complexity check of LLVM IR.
//
//	w   switch, catchswitch
//	k   invoke
//	b   br, callbr, indirectbr
//	h   shl, lshr, ashr
//	m   llvm.loop, resume
//	r   or, xor, catchret, cleanupret
//	a   and
//
// Each anchor is the rarest byte of its check, measured over 27MB of the .ll
// files of llvm-project: w is 0.08% of all bytes, k 0.10%, b 0.36%, h 0.48%,
// m 0.82%, r 2.00% and a 2.24%, and the seven together are 6.08%. The naive
// table, stopping on the first byte of each check the way the generic loop's
// TokenFirst does, is l, b, s, i, c, r, a, o and x and costs 23.75%.
//
// Two checks are not anchored on their own rarest byte, and both for the reason
// the Go counter gives for select: the table is what the scan pays for, not the
// individual check. "xor " is rarest at x, 1.47%, and takes r instead, which
// "or " has already bought. "catchret " is rarest at h, 0.48%, and "cleanupret "
// at p, 0.85%, and both take r for the same reason; h is in the table anyway for
// shl but r costs nothing more either way and keeps every ret check in one arm
// of the matcher.
//
// "and " and "or " are the two checks with no rare byte to be had. "and " is
// spelled only with a at 2.24%, n at 2.55% and d at 2.51%, and "or " only with
// o at 4.65% and r at 2.00%, so the cheapest anchor for each is still a common
// byte and those two alone are 4.24 of the 6.08 points. They are the floor on
// what this table can cost.
//
// Where two checks share an anchor the byte behind it tells them apart, and no
// check holds the anchor of another check in a position a backwards read could
// match from. Taken one anchor at a time:
//
//	m   llv-m and res-u-m are told apart on the byte behind, v against u.
//	b   br is bare, callbr has l behind its b and indirectbr has t. Both of
//	    those carry a word on, so a bare br can never match inside either.
//	h   all three have s behind the h; shl is told from the other two on the
//	    byte in front, l against r, and lshr from ashr on the byte two back.
//	w   both have s behind the w, and switch inside catchswitch has h two bytes
//	    back, which carries a word on and so cannot start a check.
//	r   or and xor both have o behind the r, and the x of xor carries a word on
//	    so a bare or cannot match inside it. catchret and cleanupret are told
//	    apart on the byte behind the r, h against p, and both are told from
//	    or and xor on the byte in front, e against a space.
//
// The backwards reads are sound because nothing a check of LLVM IR is spelled
// with also opens or closes a quote or a comment: the delimiters are the
// semicolon and the double quote and neither is a letter.
// TestCounterAnchoringCollisions records that as an empty collision set and
// fails if a check added to languages.json ever breaks it.
func buildLLVMStop() [256]bool {
	table := buildLLVMStopNoComplexity()
	for _, b := range []byte{'w', 'k', 'b', 'h', 'm', 'r', 'a'} {
		table[b] = true
	}

	return table
}

func buildLLVMStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{';', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// llvmStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func llvmStopTable() *[256]bool {
	if Complexity {
		return &llvmStopNoComplexity
	}

	return &llvmStop
}

// llvmComplexityAnchored reports whether a complexity check of LLVM IR sits on
// the anchor byte at index, which is what llvmStop stopped the scan on.
//
// Every check but "and " is anchored inside itself, so the bytes in front of the
// anchor are read back and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a semicolon, a quote
// or a newline, none of those being a byte any check is spelled with, so reading
// back never crosses out of the code the scan is in. Nor can it read in front of
// the region the counter owns, every backwards read being clamped to floor.
//
// The byte behind the anchor is tested before anything else in each arm. It is
// one load and one compare, it rejects nearly every a and r in the file, and
// those two are two thirds of what this table stops on.
func llvmComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'a':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "nd ")
	case 'r':
		switch byteBefore(content, index, floor) {
		case 'o':
			// "or " and "xor " both sit on an o. The x in front of the o of xor
			// carries a word on, so the two matches are mutually exclusive and
			// the bare or is tried first because it is by far the commoner.
			if index+1 >= len(content) || content[index+1] != ' ' {
				return false
			}
			if wordStartsAt(content, index-1, floor) {
				return true
			}

			return hasPrefixAt(content, index-2, floor, "xor") &&
				wordStartsAt(content, index-2, floor)
		case 'h':
			return hasPrefixAt(content, index-5, floor, "catchret ") &&
				wordStartsAt(content, index-5, floor)
		case 'p':
			return hasPrefixAt(content, index-7, floor, "cleanupret ") &&
				wordStartsAt(content, index-7, floor)
		}

		return false
	case 'b':
		if !hasPrefixAt(content, index+1, floor, "r ") {
			return false
		}

		switch byteBefore(content, index, floor) {
		case 'l':
			return hasPrefixAt(content, index-4, floor, "callbr") &&
				wordStartsAt(content, index-4, floor)
		case 't':
			return hasPrefixAt(content, index-8, floor, "indirectbr") &&
				wordStartsAt(content, index-8, floor)
		}

		return wordStartsAt(content, index, floor)
	case 'h':
		if byteBefore(content, index, floor) != 's' {
			return false
		}
		if hasPrefixAt(content, index+1, floor, "l ") {
			return wordStartsAt(content, index-1, floor)
		}
		if !hasPrefixAt(content, index+1, floor, "r ") {
			return false
		}

		return (hasPrefixAt(content, index-2, floor, "lsh") ||
			hasPrefixAt(content, index-2, floor, "ash")) &&
			wordStartsAt(content, index-2, floor)
	case 'm':
		switch byteBefore(content, index, floor) {
		case 'v':
			// The one check of the language with no space behind it. It is the
			// metadata attachment !llvm.loop, and the bang in front of it is not
			// a byte that carries a word on, so the boundary test still passes
			// where it should.
			return hasPrefixAt(content, index-3, floor, "llvm.loop") &&
				wordStartsAt(content, index-3, floor)
		case 'u':
			return hasPrefixAt(content, index-4, floor, "resume ") &&
				wordStartsAt(content, index-4, floor)
		}

		return false
	case 'w':
		if byteBefore(content, index, floor) != 's' ||
			!hasPrefixAt(content, index+1, floor, "itch ") {
			return false
		}
		if hasPrefixAt(content, index-6, floor, "catchswitch") {
			return wordStartsAt(content, index-6, floor)
		}

		return wordStartsAt(content, index-1, floor)
	case 'k':
		if byteBefore(content, index, floor) != 'o' {
			return false
		}

		return hasPrefixAt(content, index-4, floor, "invoke ") &&
			wordStartsAt(content, index-4, floor)
	}

	return false
}

// llvmAnchorBit gives each anchor byte of LLVM IR a bit; llvmAnchorPrev[prev]
// holds the bits of every anchor whose first test could still pass with prev in
// front of it. The AND of the two is zero exactly where llvmComplexityAnchored
// would have returned false on its first test.
//
// It is worth more here than anywhere else in the set. Sixteen checks share
// seven anchors, a and r alone are two thirds of what the table stops on, and
// both of those arms reject on the byte behind before they read anything else.
const (
	llvmAnchorBitA uint8 = 1 << iota
	llvmAnchorBitR
	llvmAnchorBitB
	llvmAnchorBitH
	llvmAnchorBitM
	llvmAnchorBitW
	llvmAnchorBitK
)

var llvmAnchorBit = buildLLVMAnchorBit()

var llvmAnchorPrev = buildLLVMAnchorPrev()

func buildLLVMAnchorBit() [256]uint8 {
	var table [256]uint8
	table['a'] = llvmAnchorBitA
	table['r'] = llvmAnchorBitR
	table['b'] = llvmAnchorBitB
	table['h'] = llvmAnchorBitH
	table['m'] = llvmAnchorBitM
	table['w'] = llvmAnchorBitW
	table['k'] = llvmAnchorBitK

	return table
}

func buildLLVMAnchorPrev() [256]uint8 {
	var table [256]uint8
	for value := range 256 {
		previous := byte(value)
		boundary := !isIdentifierContinue(previous)

		var mask uint8
		// and is anchored on its own first byte and wants a boundary there.
		if boundary {
			mask |= llvmAnchorBitA
		}
		// or and xor read back over an o, catchret over an h, cleanupret over
		// a p. Nothing else reaches past the switch on the byte behind.
		if previous == 'o' || previous == 'h' || previous == 'p' {
			mask |= llvmAnchorBitR
		}
		// callbr has an l behind its b and indirectbr a t; a bare br wants a
		// boundary. All three arms are live, so all three are in the mask.
		if previous == 'l' || previous == 't' || boundary {
			mask |= llvmAnchorBitB
		}
		// shl, lshr and ashr all carry an s behind the h, and switch and
		// catchswitch both carry an s behind the w.
		if previous == 's' {
			mask |= llvmAnchorBitH | llvmAnchorBitW
		}
		// llvm.loop reads back over a v, resume over a u.
		if previous == 'v' || previous == 'u' {
			mask |= llvmAnchorBitM
		}
		if previous == 'o' {
			mask |= llvmAnchorBitK
		}
		table[value] = mask
	}

	return table
}

// llvmComplexityAtLineStart is llvmComplexityAnchored for the first byte of code
// on a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored on
// a byte the code scan reaches, since that begins on the byte after this one.
//
// For LLVM IR that is "and " and "br " and nothing else. The other fourteen
// checks are anchored two to seven bytes in, and a check that begins at the
// first byte of code on a line puts its anchor on a byte the code scan is going
// to stop on anyway.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the start of the file is all that can sit in front of it — so the word
// boundary needs no test.
//
// This is not an optimisation that can be left out. Without it an and or a br
// opening a line is never counted, and it is the reason the backwards reads
// above can be written as reads rather than as searches. The two are one thing.
func llvmComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'a':
		return hasPrefixAt(content, index+1, floor, "nd ")
	case 'b':
		return hasPrefixAt(content, index+1, floor, "r ")
	}

	return false
}

// llvmBlankState looks at the first byte of content on a line. There is no block
// comment in LLVM IR, so a line either opens a comment, opens a string, or is
// code.
func llvmBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState) {
	switch content[index] {
	case ';':
		return index, SComment
	case '"':
		return index, SString
	}

	if !Complexity && llvmComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode
}

// llvmCodeState runs to the end of the line or to whatever token takes it out of
// code.
func llvmCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState) {
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
		case ';':
			return i, SCommentCode
		case '"':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here on
			// it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) == '\\' {
				return i, SCode
			}

			return i, SString
		default:
			// The byte in front of the anchor is the first thing
			// llvmComplexityAnchored tests and the last thing the caller knows
			// before paying for the call. A zero AND is exactly the case that
			// function rejects on its own first test, so skipping it cannot
			// change a count.
			if llvmAnchorPrev[byteBefore(content, i, floor)]&llvmAnchorBit[curByte] == 0 {
				continue
			}

			if llvmComplexityAnchored(content, i, floor) {
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

// countLoopLLVMIR stands in for countLoopGeneric where the language is LLVM IR
// and none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopLLVMIR(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := llvmStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			return llvmCodeState(content, &tally, index, endPoint, floor, stop)
		case SString:
			return counterStringState(content, index, endPoint, floor, llvmQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		default:
			// SBlank, and SMulticommentBlank which LLVM IR can never be in: the
			// language has no block comment, so nothing ever opens one and the
			// three multiline states are unreachable here.
			return llvmBlankState(content, &tally, index, floor)
		}
	}

	// LLVM IR does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
