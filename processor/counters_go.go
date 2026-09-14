// SPDX-License-Identifier: MIT

package processor

// A counter written for Go, on the same terms as the others: it must agree with
// the generic loop to the line, and where the two differ the generic one is
// right by definition.
//
// Go asks for the raw string, a quote with no escape mechanism at all, which is
// the ignoreEscape flag. Unlike the verbatim string of C# it opens and closes
// with the same single byte, so it costs the stop table one byte and the string
// state a flag.
//
// Go also carries two checks nothing else does, go and select, which is the
// whole of what makes its check list seventeen long against Java's twenty and
// C's fourteen.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// goStop marks every byte the scan stops on: the slash of both comment forms,
// the three quote characters, one anchor byte out of every complexity check,
// and the newline and the null that end a line and a file of bytes rather than
// text.
var goStop = buildGoStop()

// goStopNoComplexity is goStop without the bytes only a complexity check is
// spelled with, which is what a file counted with --no-complexity is scanned
// with.
var goStopNoComplexity = buildGoStopNoComplexity()

// The three quotes of Go. The backtick has no escape mechanism, so a backslash
// in front of its closer does not carry the string on.
var (
	goQuote     = []byte{'"'}
	goCharQuote = []byte{'\''}
	goRawQuote  = []byte{'`'}
)

// goComplexityAnchors is the byte each complexity check of Go is stopped on. It
// is what the structural conformance test holds against goStop, and it is the
// written form of the argument in buildGoStop.
var goComplexityAnchors = map[string]byte{
	"go ":  'g',
	"for ": 'f', "for(": 'f', "for{": 'f',
	"if ": 'f', "if(": 'f',
	"switch ": 'w', "switch(": 'w', "switch{": 'w',
	"select ": 'l', "select{": 'l',
	"else ": 'l', "else{": 'l',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
}

// buildGoStop marks the anchor of every complexity check of Go.
//
//	g   go
//	f   if, for          the f of if is read backwards, the f of for forwards
//	w   while is not a keyword of Go, so w is switch alone
//	l   else, select     both read backwards, and by a different distance
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each anchor is the rarest byte of its check, measured over 92MB of the Go
// standard library: w is 0.39% of all bytes against s at 2.83%, g is 1.08%
// against o at 2.61%, f is 1.63% against i at 2.94% and r at 3.63%.
//
// select is the one place the rarest byte is not taken. Its own rarest is c at
// 1.78% against l at 1.83%, a difference of five hundredths of a percent, but l
// is already in the table for else and c is not in it at all. The table is what
// the scan pays for, not the individual check, so sharing an anchor that is
// already bought beats adding one that is marginally rarer.
//
// No check holds the anchor of another check anywhere, so the scan can carry on
// through a matched token rather than stepping over it, and nothing can be
// counted twice. else and select are told apart where they share the l: both
// have an e in front of it, so the byte behind cannot separate them and the
// bytes in front and after do. else is e-l-s-e and select is s-e-l-e-c-t, so
// the byte after the l is s for one and e for the other, and the two matches are
// mutually exclusive.
func buildGoStop() [256]bool {
	table := buildGoStopNoComplexity()
	for _, b := range []byte{'g', 'f', 'w', 'l', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildGoStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '`', '\n', 0} {
		table[b] = true
	}

	return table
}

// goStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func goStopTable() *[256]bool {
	if Complexity {
		return &goStopNoComplexity
	}

	return &goStop
}

// goBlockOpens reports whether a byte closes a keyword of Go that is written
// with a space, a bracket or a brace behind it, which for and switch both are.
func goBlockOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == '(' || b == '{'
}

// goComplexityAnchored reports whether a complexity check of Go sits on the
// anchor byte at index, which is what goStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check of Go is spelled with, so
// reading back never crosses out of the code the scan is in. Nor can it read in
// front of the region the counter owns, every backwards read being clamped to
// floor.
func goComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'g':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "o ")
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "or") && goBlockOpens(content, index+3)
	case 'w':
		if byteBefore(content, index, floor) != 's' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "itch") && goBlockOpens(content, index+5)
	case 'l':
		// else and select both carry an e in front of their l, so the two are
		// told apart on what follows instead: else is e-l-s-e and select is
		// s-e-l-e-c-t, so the byte after the l is s for one and e for the other
		// and both matches cannot hold at once.
		if byteBefore(content, index, floor) != 'e' {
			return false
		}
		if wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "se") && braceOpens(content, index+3) {
			return true
		}

		return wordStartsAt(content, index-2, floor) &&
			hasPrefixAt(content, index-2, floor, "select") && braceOpens(content, index+4)
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

// goComplexityAtLineStart is goComplexityAnchored for the first byte of code on
// a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored on
// a byte the code scan reaches, since that begins on the byte after this one and
// no check that is anchored on its first byte holds an anchor of its own
// anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test.
//
// This is not an optimisation that can be left out. Without it go, for, || and
// && are never counted when they open a line, and it is the reason the backwards
// reads above can be written as reads rather than as searches. The two are one
// thing.
func goComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'g':
		return hasPrefixAt(content, index+1, floor, "o ")
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && goBlockOpens(content, index+3)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// goString reports the string a quote at index opens: the quote that closes it
// and whether that quote can be escaped.
//
// All three of Go's quotes open and close with one byte, so unlike C# there is
// no cursor to move. The raw one is told from the other two only by the flag it
// carries, which is what stops a backslash in front of its closer carrying it
// on: `C:\` is a complete raw string ending in a backslash.
//
// The backtick is tested for an escape in front of it the same as the other two,
// because the generic loop tests the byte in front of every quote it opens
// whatever kind it is, so a backtick behind a backslash opens nothing.
func goString(content []byte, index, floor int) ([]byte, bool, bool) {
	if byteBefore(content, index, floor) == '\\' {
		return nil, false, false
	}

	switch content[index] {
	case '"':
		return goQuote, false, true
	case '\'':
		return goCharQuote, false, true
	case '`':
		return goRawQuote, true, true
	}

	return nil, false, false
}

// goBlankState looks at the first byte of content on a line.
func goBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte, bool) {
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
	case '"', '\'', '`':
		if quote, ignoreEscape, opened := goString(content, index, floor); opened {
			return index, SString, quote, ignoreEscape
		}

		return index, SCode, nil, false
	}

	if !Complexity && goComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// goCodeState runs to the end of the line or to whatever token takes it out of
// code.
func goCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
		case '"', '\'', '`':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if quote, ignoreEscape, opened := goString(content, i, floor); opened {
				return i, SString, quote, ignoreEscape
			}

			return i, SCode, nil, false
		default:
			if goComplexityAnchored(content, i, floor) {
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

// countLoopGo stands in for countLoopGeneric where the language is Go and none
// of the extra outputs are wanted. It returns false when it ended the count
// early, the same way the generic loop does.
func countLoopGo(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := goStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for and whether it can be escaped.
	// Go has three and the code and blank states say which one opened, so both
	// are carried across calls. Neither is ever read before a state has set it:
	// the states below hand a quote back only when they opened a string, and
	// anything else leaves the last one in place rather than clearing it.
	endQuote := goQuote
	ignoreEscape := false
	openString := func(quote []byte, raw bool) {
		if quote != nil {
			endQuote, ignoreEscape = quote, raw
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, raw := goCodeState(content, &tally, index, endPoint, floor, stop)
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
			index, state, quote, raw := goBlankState(content, &tally, index, floor)
			openString(quote, raw)

			return index, state
		}
	}

	// Go does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, false, step)
}
