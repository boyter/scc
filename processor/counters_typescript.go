// SPDX-License-Identifier: MIT

package processor

// A counter written for TypeScript. It shares the quotes, the postfix checks
// and the regular expression disambiguation of counters_ecmascript.go with
// JavaScript, and differs from it in exactly one place: TypeScript carries
// === and !== as well as == and !=, and those four overlap.
//
// That overlap is the whole of what is interesting here. Everywhere else a
// counter anchors on the rarest byte of a check and reads backwards, because no
// two checks can match at the same place. The equality family can: === holds
// == inside it, and the generic loop resolves that by taking the longest match
// at the first byte and stepping the cursor past what it matched. An anchored
// read cannot see that, so for these four the counter does what the generic
// loop does, at the same position, with the same skip. See tsEqualityToken.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go and counters_ecmascript.go.

// tsStop marks every byte the scan has to stop on: the slash of both comment
// forms and of a regex literal, the three quotes, one anchor byte out of every
// complexity check, and the newline and the null that end a line and a file of
// bytes rather than text.
var tsStop = buildTsStop()

// tsStopNoComplexity is tsStop without the bytes that only a complexity check
// is spelled with. It is what a file counted with --no-complexity is scanned
// with, the generic loop leaving the checks out of its trie under the same
// flag.
var tsStopNoComplexity = buildTsStopNoComplexity()

// tsComplexityAnchors is the byte each complexity check of TypeScript is
// stopped on, the postfix three included. It is what the structural conformance
// test holds against tsStop, and it is the written form of the argument in
// buildTsStop.
var tsComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"case ":   'c', "case(": 'c',
	"|| ":  '|',
	"&& ":  '&',
	"!= ":  '!',
	"!== ": '!',
	"== ":  '=',
	"=== ": '=',
	"?.":   '?',
	"??":   '?',
	"??=":  '?',
}

// buildTsStop marks the anchor of every complexity check of TypeScript.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	c   case
//	|   ||
//	&   &&
//	!   != and !==, matched forwards at the first byte
//	=   == and ===, matched forwards at the first byte
//	?   ?. ?? ??=, the three postfix checks, all spelled with ? first
//
// Each anchor is the rarest byte of its own check, measured over 28MB of the
// TypeScript compiler: f 1.24% against i 3.69% and o 4.02% for if and for,
// w 0.31% against s 3.51% and h 0.98% for switch and while, l 2.28% against e
// at nearly 8% for else, c 2.16% against a 4.01% and s 3.51% for case. The
// operators are already their own rarest byte: & 0.09%, ? 0.12%, | 0.20%,
// = 0.65%, and ! at 0.06% is the rarest byte in the language bar q.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match:
//
//   - while carries the l of else, but else wants an e in front of its l and
//     finds an i.
//   - switch carries the c of case, but case wants a word boundary in front of
//     its c and finds the t of switch.
//   - the second | of || and the second & of && are read as the start of
//     another check and fail, there being no third byte to match.
//
// The equality family is the exception and does not read backwards at all.
func buildTsStop() [256]bool {
	table := buildTsStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'c', '|', '&', '=', '!', '?'} {
		table[b] = true
	}

	return table
}

func buildTsStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '`', '\n', 0} {
		table[b] = true
	}

	return table
}

// tsStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func tsStopTable() *[256]bool {
	if Complexity {
		return &tsStopNoComplexity
	}

	return &tsStop
}

// tsEqualityToken reports the length of the equality check that begins at
// index, or zero where none does, taking the longest of the four the way the
// trie the generic loop asks does.
//
// The four are == , === , != and !== , each ending in a space. === holds == and
// !== holds != , so the length matters: matching the shorter one and carrying
// on would count the same operator twice.
//
// The bound is the whole content and not endPoint, because the trie the generic
// loop asks is handed content[i:] and reads to the end of the file. A check
// whose trailing space is the very last byte therefore matches for the generic
// loop, and has to match here too. What endPoint governs is only whether the
// caller may step the cursor past it, which is the caller's business.
func tsEqualityToken(content []byte, index int) int {
	if index+2 >= len(content) {
		return 0
	}

	switch content[index] {
	case '!':
		if content[index+1] != '=' {
			return 0
		}
	case '=':
		if content[index+1] != '=' {
			return 0
		}
	default:
		return 0
	}

	if content[index+2] == '=' {
		if index+3 < len(content) && content[index+3] == ' ' {
			return 4
		}

		return 0
	}

	if content[index+2] == ' ' {
		return 3
	}

	return 0
}

// tsComplexityAnchored reports whether a complexity check of TypeScript sits on
// the anchor byte at index, for the checks that are read backwards. The
// equality family is not among them; it is handled where the cursor can be
// moved, since matching it needs the same forward step the generic loop takes.
func tsComplexityAnchored(content []byte, index, floor int) bool {
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
	case '|':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// tsComplexityAtLineStart is tsComplexityAnchored for the first byte of code on
// a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches.
//
// The equality family is not here either. A line can open with one, and the
// caller looks for it separately, because a match has to step the cursor past
// what it matched and this reports only whether there was one.
func tsComplexityAtLineStart(content []byte, index, floor int) bool {
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

// tsBlankState looks at the first byte of content on a line.
func tsBlankState(content []byte, tally *counterTally, index, endPoint, floor int) (int, counterState, []byte) {
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
	case '!', '=':
		// The generic loop counts the check here and steps the cursor past it,
		// so the line-start case has to move the index too. Without the step the
		// code scan would start inside === and read the == that overlaps it as a
		// second check.
		if length := tsEqualityToken(content, index); length != 0 {
			if !Complexity && wordStartsAt(content, index, floor) {
				tally.Complexity++
			}

			return index + length - 1, SCode, nil
		}

		return index, SCode, nil
	}

	if !Complexity && tsComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// tsCodeState runs to the end of the line or to whatever token takes it out of
// code.
func tsCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
			// line carries on as code.
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
		case '!', '=':
			// The equality family, matched forwards at its first byte and
			// stepped over, which is what the generic loop does and the only way
			// to keep === from being counted twice once as itself and once as
			// the == inside it.
			if length := tsEqualityToken(content, i); length != 0 {
				if wordStartsAt(content, i, floor) {
					tally.Complexity++
				}

				// The generic loop gives up the rest of the line when the step
				// would land on or past the end, reporting the byte it matched
				// at rather than the last byte of the file.
				if i+length >= endPoint {
					return i, SCode, nil
				}

				i += length - 1
			}
		default:
			if tsComplexityAnchored(content, i, floor) {
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

// countLoopTypeScript stands in for countLoopGeneric where the language is
// TypeScript and none of the extra outputs are wanted. It returns false when it
// ended the count early, the same way the generic loop does.
func countLoopTypeScript(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := tsStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	endQuote := ecmaDoubleQuote
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := tsCodeState(content, &tally, index, endPoint, floor, stop)
			openQuote(quote)

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, false)
		case SComment, SCommentCode:
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			return counterCommentState(content, index, endPoint, state, slashStarClose, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote := tsBlankState(content, &tally, index, endPoint, floor)
			openQuote(quote)

			return index, state
		}
	}

	// TypeScript does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
