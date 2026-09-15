// SPDX-License-Identifier: MIT

package processor

// A counter written for Assembly, on the same terms as the others: it must
// agree with the generic loop to the line, and where the two differ the generic
// one is right by definition.
//
// Assembly is the C counter with three tokens moved and one check list trimmed.
// The comment is a semicolon rather than a double slash, so a double slash is
// ordinary code here and the only thing a slash can open is the block comment C
// also has. There are two quotes rather than one, the second being the single
// quote, which the language database gives Assembly and does not give C.
//
// The check list is C's own eleven, which is what makes this counter worth so
// little thought: an assembler has no if, no while and no switch, and the
// checks are there because scc counts the C-like macro dialects the same way.
// What differs is how they are spelled. C writes switch, while and else twice,
// once with a space and once with the bracket or brace it is normally written
// with; Assembly writes each of those three once, with a space and nothing
// else, so the opener test for them is spaceOpens rather than cOpens. Reading
// that off languages.json rather than off C is the whole of the per-language
// work here.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// The two quotes of Assembly. Both open and close with one byte and both
// escape with a backslash, so neither needs anything the shared string state
// does not already do.
var (
	asmQuote     = []byte{'"'}
	asmCharQuote = []byte{'\''}
)

// asmStop marks every byte the scan stops on: the semicolon that opens a line
// comment, the slash that opens a block comment, the two quotes, one anchor
// byte out of every complexity check, and the newline and the null that end a
// line and a file of bytes rather than text.
var asmStop = buildAsmStop()

// asmStopNoComplexity is asmStop without the bytes only a complexity check is
// spelled with, which is what a file counted with --no-complexity is scanned
// with.
var asmStopNoComplexity = buildAsmStopNoComplexity()

// asmComplexityAnchors is the byte each complexity check of Assembly is stopped
// on. It is what the structural conformance test holds against asmStop, and it
// is the written form of the argument in buildAsmStop.
var asmComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"|| ":     '|',
	"&& ":     '&',
	"!= ":     '=',
	"== ":     '=',
}

// buildAsmStop marks the anchor of every complexity check of Assembly.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// The anchors are C's, and so is the argument for them, but the frequencies are
// not: measured over 6MB of the .s files of llvm-project, f is 1.16% of all
// bytes, w 0.21%, l 1.87%, = 0.20%, | 0.05% and & under 0.04%, which is 3.5%
// together. Stopping on the first byte of every check the way the generic loop
// does costs 9.5%, nearly two thirds of it the e of else at 3.88% and the i of
// if at 2.18%.
//
// else is the one check with no cheap anchor. It is spelled e, l, s, e, and its
// rarest byte on this corpus is s at 1.81% against l at 1.87%, a difference of
// six hundredths of a percent. l is taken anyway, because it is what C reads
// and the two matchers are then the same argument written twice rather than
// two arguments.
//
// The backwards reads are sound because nothing a check of Assembly is spelled
// with also opens or closes a quote or a comment: the delimiters are the
// semicolon, the slash, the star and the two quotes, and none of those is a
// byte a check is spelled with. TestCounterAnchoringCollisions records that as
// an empty collision set and fails if a check added to languages.json ever
// breaks it.
func buildAsmStop() [256]bool {
	table := buildAsmStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildAsmStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{';', '/', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// asmStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func asmStopTable() *[256]bool {
	if Complexity {
		return &asmStopNoComplexity
	}

	return &asmStop
}

// asmComplexityAnchored reports whether a complexity check of Assembly sits on
// the anchor byte at index, which is what asmStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash, a
// semicolon or a newline, none of those being a byte any check is spelled with,
// so reading back never crosses out of the code the scan is in. Nor can it read
// in front of the region the counter owns, every backwards read being clamped
// to floor.
//
// The pairs are told apart on the byte behind the anchor and cannot both match:
// the f of if has an i behind it and the f of for cannot, i being a byte that
// carries a word on, and the same holds of the w of switch against while and
// the = of != against ==.
//
// The l of while is not a false anchor for else: it has an i behind it where
// else has an e, and the first thing the else arm does is test that byte.
func asmComplexityAnchored(content []byte, index, floor int) bool {
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
			hasPrefixAt(content, index+1, floor, "se") && spaceOpens(content, index+3)
	case 'w':
		if byteBefore(content, index, floor) == 's' {
			return wordStartsAt(content, index-1, floor) &&
				hasPrefixAt(content, index+1, floor, "itch") && spaceOpens(content, index+5)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
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

// asmComplexityAtLineStart is asmComplexityAnchored for the first byte of code
// on a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches, since that begins on the byte after this one
// and no check that is anchored on its first byte holds an anchor of its own
// anywhere else.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, while, ||
// and && are never counted when they open a line, and it is the reason the
// backwards reads above can be written as reads rather than as searches. The
// two are one thing.
func asmComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// asmString reports the string a quote at index opens: the quote that closes
// it, and nil where a backslash in front of it means it opens nothing.
func asmString(content []byte, index, floor int) []byte {
	if byteBefore(content, index, floor) == '\\' {
		return nil
	}

	if content[index] == '\'' {
		return asmCharQuote
	}

	return asmQuote
}

// asmBlankState looks at the first byte of content on a line.
//
// A slash here opens a block comment or nothing at all. The double slash that
// opens a comment in C is code in Assembly, whose only line comment is the
// semicolon, and the generic loop reads it the same way: its trie walk stops at
// the slash node, which carries no token of its own.
func asmBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte) {
	switch content[index] {
	case ';':
		return index, SComment, nil
	case '/':
		if index+1 < len(content) && content[index+1] == '*' {
			return index + 1, SMulticomment, nil
		}
	case '"', '\'':
		// Unconditionally, the way blankState and every sibling counter do it.
		// asmString refuses a quote behind a backslash, which is a thing that
		// can happen in code and cannot happen here: the blank state is only
		// ever entered with a newline, a space, a tab, a carriage return, the
		// slash that closed a block comment or the BOM in front of the byte.
		// Asking anyway would be a difference from the generic loop that the
		// entry conditions happen to hide.
		if content[index] == '\'' {
			return index, SString, asmCharQuote
		}

		return index, SString, asmQuote
	}

	if !Complexity && asmComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// asmCodeState runs to the end of the line or to whatever token takes it out of
// code.
func asmCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
		case ';':
			return i, SCommentCode, nil
		case '/':
			if i+1 < len(content) && content[i+1] == '*' {
				return i + 1, SMulticommentCode, nil
			}
		case '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here on
			// it, having started blank, but the check does not depend on that
			// holding.
			if quote := asmString(content, i, floor); quote != nil {
				return i, SString, quote
			}

			return i, SCode, nil
		default:
			if asmComplexityAnchored(content, i, floor) {
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

// countLoopAssembly stands in for countLoopGeneric where the language is
// Assembly and none of the extra outputs are wanted. It returns false when it
// ended the count early, the same way the generic loop does.
func countLoopAssembly(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := asmStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for. Assembly has two and the code
	// and blank states say which one opened, so it is carried across calls. It
	// is never read before a state has set it: the states above hand a quote
	// back only when they opened a string, and anything else leaves the last one
	// in place rather than clearing it.
	endQuote := asmQuote
	openString := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := asmCodeState(content, &tally, index, endPoint, floor, stop)
			openString(quote)

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
			index, state, quote := asmBlankState(content, &tally, index, floor)
			openString(quote)

			return index, state
		}
	}

	// Assembly does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
