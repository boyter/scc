// SPDX-License-Identifier: MIT

package processor

// A counter written for PHP, on the same terms as the rest: it must agree with
// the generic loop to the line, and where the two differ the generic one is
// right by definition.
//
// PHP asks for one thing no counter had before. It spells a line comment two
// ways, # and //, which is M12. That costs a second byte in the stop table and
// a second case in the code and blank states, and nothing else: the two open
// the same state and neither has anything to say about the other.
//
// What PHP does NOT do here is understand <?php. scc counts a .php file as PHP
// from the first byte to the last, and the HTML around the tags is counted as
// PHP code, because that is what the generic loop does and the generic loop is
// the oracle. Telling one language from another inside a file is the embedded
// language work, which is its own phase.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// phpStop marks every byte the scan has to stop on: the slash of both comment
// forms, the hash of the other line comment, the two quotes, one anchor byte
// out of every complexity check, and the newline and the null that end a line
// and a file of bytes rather than text.
var phpStop = buildPhpStop()

// phpStopNoComplexity is phpStop without the bytes that only a complexity check
// is spelled with. It is what a file counted with --no-complexity is scanned
// with, the generic loop leaving the checks out of its trie under the same
// flag.
var phpStopNoComplexity = buildPhpStopNoComplexity()

// The two quotes of PHP, held here so the shared string state is handed a slice
// rather than building one per string.
var (
	phpDoubleQuote = []byte{'"'}
	phpSingleQuote = []byte{'\''}
)

// phpComplexityAnchors is the byte each complexity check of PHP is stopped on.
// It is what the structural conformance test holds against phpStop, and it is
// the written form of the argument in buildPhpStop.
var phpComplexityAnchors = map[string]byte{
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

// buildPhpStop marks the anchor of every complexity check of PHP.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// The anchor is the rarest byte of the check rather than its first. Measured
// over 17.7MB of laravel: f 0.92% against i 3.61%, o 3.27% and r 3.79% for if
// and for; w 0.40% against s 4.20% and h 1.32% for switch and while; l 2.76%
// against e at nearly 9% for else. The operators are already their own rarest
// byte, | at 0.08% and & at 0.03%.
//
// The bytes are chosen so that no check holds the anchor of another check in a
// position where reading back from it can match, which is what lets the scan
// carry on through a matched token rather than stepping over it:
//
//   - while carries the l of else, but else wants an e in front of its l and
//     finds an i.
//   - the second | of || and the second & of && are read as the start of
//     another check and fail, there being no third byte to match.
//
// So the scan never counts a check twice and never counts one that is not
// there.
func buildPhpStop() [256]bool {
	table := buildPhpStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildPhpStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '#', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// phpStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func phpStopTable() *[256]bool {
	if Complexity {
		return &phpStopNoComplexity
	}

	return &phpStop
}

// phpComplexityAnchored reports whether a complexity check of PHP sits on the
// anchor byte at index, which is what phpStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash, a
// hash or a newline, since none of those is a byte any check of PHP is spelled
// with, so reading back never crosses out of the code the scan is in. Nor can
// it read in front of the region the counter owns, every backwards read being
// clamped to floor.
//
// switch, while and else are spelled with a space behind them and nothing else,
// so they take spaceOpens; using cOpens would count a switch( the language
// database does not carry. if and for have both forms.
func phpComplexityAnchored(content []byte, index, floor int) bool {
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

// phpComplexityAtLineStart is phpComplexityAnchored for the first byte of code
// on a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches, since that begins on the byte after this one
// and no check that is anchored on its first byte holds an anchor of its own
// anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, while, ||
// and && are never counted when they open a line, and it is the reason the
// backwards reads above can be written as reads rather than as searches. The
// two are one thing.
func phpComplexityAtLineStart(content []byte, index, floor int) bool {
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

// phpBlankState looks at the first byte of content on a line.
func phpBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte) {
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
	case '#':
		// The other line comment. PHP 8 spells an attribute #[Name], which the
		// generic loop reads as a comment like any other hash, so this does too.
		return index, SComment, nil
	case '"':
		return index, SString, phpDoubleQuote
	case '\'':
		return index, SString, phpSingleQuote
	}

	if !Complexity && phpComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// phpCodeState runs to the end of the line or to whatever token takes it out of
// code.
func phpCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
		case '#':
			return i, SCommentCode, nil
		case '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here
			// on it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				if curByte == '"' {
					return i, SString, phpDoubleQuote
				}

				return i, SString, phpSingleQuote
			}

			return i, SCode, nil
		default:
			if phpComplexityAnchored(content, i, floor) {
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

// countLoopPHP stands in for countLoopGeneric where the language is PHP and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopPHP(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := phpStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for. PHP has two and the code and
	// blank states say which one opened, so it is carried across calls. It is
	// never nil: the states below hand back the quote only when they opened a
	// string, and anything else leaves the last one in place.
	endQuote := phpDoubleQuote
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := phpCodeState(content, &tally, index, endPoint, floor, stop)
			openQuote(quote)

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, false)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, whichever of
			// the two tokens opened it, so the rest of the line is of no
			// interest and IndexByte finds where it ends a vector at a time
			// rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			return counterCommentState(content, index, endPoint, state, slashStarClose, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote := phpBlankState(content, &tally, index, floor)
			openQuote(quote)

			return index, state
		}
	}

	// PHP does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
