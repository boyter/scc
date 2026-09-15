// SPDX-License-Identifier: MIT

package processor

// A counter written for C#, on the same terms as the others: it must agree with
// the generic loop to the line, and where the two differ the generic one is
// right by definition.
//
// C# asks for one thing none of the counters before it did. Its verbatim string
// opens with two bytes rather than one, @" , and has no escape mechanism at all,
// so a backslash inside it is an ordinary byte of a Windows path. That is the
// multi-byte quote start and the ignoreEscape flag together, which is most of
// what Go, Python, Rust and C++ will want later.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// csharpStop marks every byte the scan stops on: the slash of both comment
// forms, the three quote openings, one anchor byte out of every complexity
// check, and the newline and the null that end a line and a file of bytes
// rather than text.
var csharpStop = buildCsharpStop()

// csharpStopNoComplexity is csharpStop without the bytes only a complexity
// check is spelled with, which is what a file counted with --no-complexity is
// scanned with.
var csharpStopNoComplexity = buildCsharpStopNoComplexity()

// The quotes of C#. The verbatim one opens with @" and closes with a single
// quote, so its opener and its closer are not the same token, which is the
// first time that has been true of any counter here.
var (
	csharpQuote     = []byte{'"'}
	csharpCharQuote = []byte{'\''}
)

// csharpComplexityAnchors is the byte each complexity check of C# is stopped
// on. It is what the structural conformance test holds against csharpStop, and
// it is the written form of the argument in buildCsharpStop.
var csharpComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"foreach ": 'f', "foreach(": 'f',
	"switch ": 'w',
	"while ":  'w',
	"else ":   'l',
	"|| ":     '|',
	"&& ":     '&',
	"!= ":     '=',
	"== ":     '=',
}

// buildCsharpStop marks the anchor of every complexity check of C#.
//
//	f   if, for, foreach   the f of if is read backwards, the other two forwards
//	w   while, switch
//	l   else
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each anchor is the rarest byte of its check, measured over 256MB of roslyn:
// f is 0.73% of all bytes against i at 4.99% and o at 3.50%, w is 0.32% against
// s at 3.25%, l is 2.42% against e at 9.91%. The operators are already their own
// rarest byte, & and | costing 0.03% and 0.04%.
//
// No check holds the anchor of another check where reading back from it would
// match, so the scan carries on through a matched token rather than stepping
// over it. while carries the l of else, but else wants an e in front of its l
// and while has an i there. Nothing else collides at all: foreach holds no w,
// no l and no operator, and switch holds no l.
//
// The @ is not a complexity anchor. It is here because it opens the verbatim
// string, and it is the rarest byte in the whole language at 0.04%.
func buildCsharpStop() [256]bool {
	table := buildCsharpStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildCsharpStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '@', '\n', 0} {
		table[b] = true
	}

	return table
}

// csharpStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func csharpStopTable() *[256]bool {
	if Complexity {
		return &csharpStopNoComplexity
	}

	return &csharpStop
}

// csharpComplexityAnchored reports whether a complexity check of C# sits on the
// anchor byte at index, which is what csharpStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of C# is spelled with, so
// reading back never crosses out of the code the scan is in. Nor can it read in
// front of the region the counter owns, every backwards read being clamped to
// floor.
//
// The checks that share an anchor are told apart on the byte behind it: the f
// of if has an i behind it and the f of for or foreach cannot, i being a byte
// that carries a word on, and the same holds of the w of switch against while
// and the = of != against ==.
func csharpComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}
		if !wordStartsAt(content, index, floor) {
			return false
		}

		return (hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)) ||
			(hasPrefixAt(content, index+1, floor, "oreach") && cOpens(content, index+7))
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
	case '=':
		if b := byteBefore(content, index, floor); b != '=' && b != '!' {
			return false
		}

		return wordStartsAt(content, index-1, floor) && spaceOpens(content, index+1)
	case '|':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return wordStartsAt(content, index, floor) && hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// csharpComplexityAtLineStart is csharpComplexityAnchored for the first byte of
// code on a line, which has nothing in front of it to read back to. Only the
// checks anchored on their own first byte are looked for here; the rest are
// anchored on a byte the code scan reaches, since that begins on the byte after
// this one and no check that is anchored on its first byte holds an anchor of
// its own anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, foreach,
// while, || and && are never counted when they open a line, and it is the
// reason the backwards reads above can be written as reads rather than as
// searches. The two are one thing.
func csharpComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return (hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)) ||
			(hasPrefixAt(content, index+1, floor, "oreach") && cOpens(content, index+7))
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && spaceOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// csharpString reports the string a quote at index opens: where the scan carries
// on from, the quote that closes it, and whether that quote can be escaped.
//
// The verbatim form is the whole of the difference from every counter before
// this one. @" is two bytes, so the cursor is left on the second of them and
// the scan begins after both, which is what stops the quote of the opener being
// read as the quote that closes an empty string. And a verbatim string has no
// escape at all, so a backslash in front of its closer does not carry it on:
// @"C:\" is a complete string ending in a backslash.
//
// The generic loop opens a verbatim string without testing what sits in front
// of the @ , since the byte it tests is the @ itself, so a backslash in front of
// one opens a string there too. Reproduced rather than argued with.
func csharpString(content []byte, index, floor int) (int, []byte, bool, bool) {
	switch content[index] {
	case '@':
		if index+1 < len(content) && content[index+1] == '"' {
			return index + 1, csharpQuote, true, true
		}

		return index, nil, false, false
	case '"':
		if byteBefore(content, index, floor) != '\\' {
			return index, csharpQuote, false, true
		}
	case '\'':
		if byteBefore(content, index, floor) != '\\' {
			return index, csharpCharQuote, false, true
		}
	}

	return index, nil, false, false
}

// csharpBlankState looks at the first byte of content on a line.
func csharpBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte, bool) {
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
	case '@', '"', '\'':
		if at, quote, ignoreEscape, opened := csharpString(content, index, floor); opened {
			return at, SString, quote, ignoreEscape
		}

		return index, SCode, nil, false
	}

	if !Complexity && csharpComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// csharpCodeState runs to the end of the line or to whatever token takes it out
// of code.
func csharpCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
		case '@', '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if at, quote, ignoreEscape, opened := csharpString(content, i, floor); opened {
				return at, SString, quote, ignoreEscape
			}

			return i, SCode, nil, false
		default:
			if csharpComplexityAnchored(content, i, floor) {
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

// countLoopCsharp stands in for countLoopGeneric where the language is C# and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopCsharp(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := csharpStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for and whether it can be escaped.
	// C# has three and the code and blank states say which one opened, so both
	// are carried across calls. Neither is ever read before a state has set it:
	// the states below hand a quote back only when they opened a string, and
	// anything else leaves the last one in place rather than clearing it.
	endQuote := csharpQuote
	ignoreEscape := false
	openString := func(quote []byte, raw bool) {
		if quote != nil {
			endQuote, ignoreEscape = quote, raw
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, raw := csharpCodeState(content, &tally, index, endPoint, floor, stop)
			openString(quote, raw)

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, ignoreEscape)
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
			index, state, quote, raw := csharpBlankState(content, &tally, index, floor)
			openString(quote, raw)

			return index, state
		}
	}

	// C# does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
