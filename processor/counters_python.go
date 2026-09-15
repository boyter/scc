// SPDX-License-Identifier: MIT

package processor

import "bytes"

// A counter for Python, and the last of the sixteen.
//
// Python is the only language in this work with no block comment at all. Its
// comments beyond the hash are docstrings, which are triple quoted strings that
// happen to open a line, and that one distinction is the whole of M7 and M10
// together: a triple quote reached with nothing but whitespace in front of it on
// the line opens SDocString and every line of it counts as comment, while the
// same three bytes reached after code on the line open an ordinary string and
// every line of it counts as code.
//
// SDocString is the one state of the generic loop no other counter models.
//
// Python is also the third of the three languages the architecture doc names as
// having an anchoring collision, on f and r. Like Rust's r, it dissolves: see
// buildPythonStop.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// pythonStop marks every byte the scan has to stop on: the hash of the line
// comment, the two quote bytes that end the opening token of every one of the
// ten string forms, one anchor byte out of every complexity check, and the
// newline and null that end a line and a file of bytes rather than text.
var pythonStop = buildPythonStop()

// pythonStopNoComplexity holds only what changes the state, which is what a file
// counted with --no-complexity is scanned with.
var pythonStopNoComplexity = buildPythonStopNoComplexity()

// The closers of the four shapes a Python string takes, held here so the shared
// string state is handed a slice rather than building one per string.
var (
	pythonDoubleQuote  = []byte{'"'}
	pythonSingleQuote  = []byte{'\''}
	pythonTripleDouble = []byte(`"""`)
	pythonTripleSingle = []byte("'''")
)

// pythonQuoteAnchors is the byte each quote form is stopped on where that is not
// its first byte. Every prefixed form is caught on the quote that follows its r
// or its f, so neither letter is ever in the stop table.
var pythonQuoteAnchors = map[string]byte{
	`r'`:   '\'',
	`r"`:   '"',
	`r"""`: '"',
	`r'''`: '\'',
	`f"""`: '"',
	`f'''`: '\'',
}

// pythonComplexityAnchors is the byte each complexity check of Python is
// stopped on. It is what the structural conformance test holds against
// pythonStop, and it is the written form of the argument in buildPythonStop.
var pythonComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"elif ": 'f', "elif(": 'f',
	"while ": 'w', "while(": 'w',
	"with ": 'w', "with (": 'w',
	"else ": 'l', "else:": 'l',
	"match ": 'h', "match(": 'h',
	"try ": 'y', "try:": 'y',
	"finally ": 'y', "finally:": 'y',
	"except ": 'x', "except:": 'x',
	"and ": 'd', "and(": 'd',
	"or ": 'o', "or(": 'o',
}

// buildPythonStop marks the anchor of every complexity check of Python.
//
//	f   for, if, elif   for forwards, if and elif read backwards
//	w   while, with
//	l   else            read backwards, e sitting in front of it
//	h   match           read backwards, matc sitting in front of it
//	y   try, finally    both read backwards
//	x   except          read backwards, ex sitting in front of it
//	d   and             read backwards, an sitting in front of it
//	o   or
//
// Each is the rarest byte of its check, measured over 42MB of cpython: w 0.45%
// for while and with against h at 0.96% and i, t, l and e all above 2.8%; x
// 0.58% for except against c at 1.89% and e at 6.78%; y 0.59% for try and
// finally against t at 4.58% and r at 3.56%; h 0.96% for match against m at
// 1.22% and c at 1.89%; d 1.78% for and against n at 2.89% and a at 3.59%; f
// 1.96% for for, if and elif against i at 2.90%, o at 2.94% and r at 3.56%; l
// 2.82% for else against s at 4.61% and e at 6.78%. or is the one check whose
// rarest byte is expensive: o at 2.94% against r at 3.56%, and there is nothing
// else in a two letter word to choose.
//
// Three checks share f and two share y, and the byte behind the anchor tells
// them apart. if and elif both carry an i behind their f, so elif is tested
// first and the two cannot both match: elif wants elif spelled out with a word
// boundary in front of the e, and if wants a word boundary in front of its own
// i, which the l of elif is not.
//
// No check holds the anchor of another check where reading back from it can
// match, so the scan carries on through a matched token rather than stepping
// over it. while and with both carry an h, but match wants matc in front of its
// h and finds neither. while carries an l, but else wants an e in front of its l
// and finds an i. for carries an o, but or wants a word boundary and finds the f.
// finally carries an l and an f: the l has an a in front of it rather than an e,
// and the f is the first byte of the word so elif and if both fail their
// backwards read on it.
//
// # ANCHORING SOUNDNESS, which Python does not get for free
//
// Spec 07 03-architecture §2.1 names Python one of three languages whose
// complexity checks share bytes with the tokens that open a string. Python's
// colliding bytes are f and r: r opens r', r", r""" and r”', f opens f""" and
// f”', and the checks carry an f in for, if, elif and finally and an r in for,
// try and or.
//
// The collision dissolves, and for the same reason it dissolved for Rust: the
// fast choice and the safe choice are the same choice.
//
//  1. r is never an anchor, and neither is f as a quote byte. Every prefixed
//     string form is caught on the quote that follows its prefix, never on the
//     prefix itself, exactly as C++ catches R"( and Rust catches br#". So r is
//     in neither stop table at all, and f is in the table only as the anchor of
//     for, if and elif, never as the opening of a string.
//
//  2. The one read that touches a prefix cannot reach past it. pythonQuotePrefix
//     reads exactly one byte behind the quote and asks whether it is an r or an
//     f. It reads no further and is clamped to floor.
//
//  3. Finding that r or f inside a word is not a mistake, because the generic
//     loop does the same. Neither loop asks for a word boundary in front of a
//     quote, so forr"x" opens a raw string in both and deff"""x""" opens a
//     string in both. They agree by construction rather than by luck.
//
//  4. No complexity read can reach into a string opening token. Every backwards
//     read here is at most six bytes and wants letters: elif wants elif, finally
//     wants finall, match wants matc, except wants ex, and wants an, else wants
//     e. None of ', " or # is a byte any check carries behind its anchor, so a
//     read that began on an anchor inside code can never land inside a quote's
//     opening token and come out matching.
//
// So the f and r the collision table flags cost nothing, and the table is doing
// what it is for: it made the question be asked.
func buildPythonStop() [256]bool {
	table := buildPythonStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'h', 'y', 'x', 'd', 'o'} {
		table[b] = true
	}

	return table
}

func buildPythonStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'#', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// pythonStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func pythonStopTable() *[256]bool {
	if Complexity {
		return &pythonStopNoComplexity
	}

	return &pythonStop
}

// pythonComplexityAnchored reports whether a complexity check of Python sits on
// the anchor byte at index, which is what pythonStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. Every read is clamped to floor, and see
// buildPythonStop for why none of them can cross out of the code the scan is in.
func pythonComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			// elif is tested before if. Both carry an i behind the f, and they
			// cannot both match: if wants a word boundary in front of its i,
			// which the l of elif is not.
			if wordStartsAt(content, index-3, floor) &&
				hasPrefixAt(content, index-3, floor, "elif") && cOpens(content, index+1) {
				return true
			}

			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		if !wordStartsAt(content, index, floor) {
			return false
		}

		return (hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)) ||
			(hasPrefixAt(content, index+1, floor, "ith") && pythonWithOpens(content, index+4))
	case 'l':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index-1, floor, "else") && colonOpens(content, index+3)
	case 'h':
		if byteBefore(content, index, floor) != 'c' {
			return false
		}

		return wordStartsAt(content, index-4, floor) &&
			hasPrefixAt(content, index-4, floor, "match") && cOpens(content, index+1)
	case 'y':
		switch byteBefore(content, index, floor) {
		case 'r':
			return wordStartsAt(content, index-2, floor) &&
				hasPrefixAt(content, index-2, floor, "try") && colonOpens(content, index+1)
		case 'l':
			return wordStartsAt(content, index-6, floor) &&
				hasPrefixAt(content, index-6, floor, "finally") && colonOpens(content, index+1)
		}

		return false
	case 'x':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index-1, floor, "except") && colonOpens(content, index+5)
	case 'd':
		if byteBefore(content, index, floor) != 'n' {
			return false
		}

		return wordStartsAt(content, index-2, floor) &&
			hasPrefixAt(content, index-2, floor, "and") && cOpens(content, index+1)
	case 'o':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "r") && cOpens(content, index+2)
	}

	return false
}

// pythonAnchorBit gives each anchor byte of Python a bit; pythonAnchorPrev[prev]
// holds the bits of every anchor whose first test could still pass with prev in
// front of it. The AND of the two is zero exactly where pythonComplexityAnchored
// would have returned false on its first test.
const (
	pythonAnchorBitF uint8 = 1 << iota
	pythonAnchorBitW
	pythonAnchorBitL
	pythonAnchorBitH
	pythonAnchorBitY
	pythonAnchorBitX
	pythonAnchorBitD
	pythonAnchorBitO
)

var pythonAnchorBit = buildPythonAnchorBit()

var pythonAnchorPrev = buildPythonAnchorPrev()

func buildPythonAnchorBit() [256]uint8 {
	var table [256]uint8
	table['f'] = pythonAnchorBitF
	table['w'] = pythonAnchorBitW
	table['l'] = pythonAnchorBitL
	table['h'] = pythonAnchorBitH
	table['y'] = pythonAnchorBitY
	table['x'] = pythonAnchorBitX
	table['d'] = pythonAnchorBitD
	table['o'] = pythonAnchorBitO

	return table
}

func buildPythonAnchorPrev() [256]uint8 {
	var table [256]uint8
	for value := range 256 {
		previous := byte(value)
		boundary := !isIdentifierContinue(previous)

		var mask uint8
		// elif and if both take the i path; for wants the boundary. Both arms
		// of the f are live, so both are in the mask.
		if previous == 'i' || boundary {
			mask |= pythonAnchorBitF
		}
		// while and with are anchored on their own first byte.
		if boundary {
			mask |= pythonAnchorBitW | pythonAnchorBitO
		}
		if previous == 'e' {
			mask |= pythonAnchorBitL | pythonAnchorBitX
		}
		if previous == 'c' {
			mask |= pythonAnchorBitH
		}
		// try reads back over an r, finally over an l.
		if previous == 'r' || previous == 'l' {
			mask |= pythonAnchorBitY
		}
		if previous == 'n' {
			mask |= pythonAnchorBitD
		}
		table[value] = mask
	}

	return table
}

// pythonComplexityAtLineStart is pythonComplexityAnchored for the first byte of
// code on a line, which has nothing in front of it to read back to. Only the
// checks anchored on their own first byte are looked for here; the rest are
// anchored on a byte the code scan reaches, since that begins on the byte after
// this one.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test.
//
// for, while, with and or are the four anchored on their first byte. if, elif,
// else, match, try, except, finally and and are all anchored later and are
// deliberately absent.
func pythonComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return (hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)) ||
			(hasPrefixAt(content, index+1, floor, "ith") && pythonWithOpens(content, index+4))
	case 'o':
		return hasPrefixAt(content, index+1, floor, "r") && cOpens(content, index+2)
	}

	return false
}

// colonOpens reports whether a byte closes a keyword Python spells with a colon
// behind it as well as a space, which is every clause introducer: else:, try:,
// except:, finally:.
func colonOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == ':'
}

// pythonWithOpens reports whether a byte closes the with of a with statement.
// languages.json spells the bracket form "with (" with the space still in it
// rather than "with(", so both forms end on a space and the bracket is not
// looked at.
func pythonWithOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}

	return content[index] == ' '
}

// pythonQuotePrefix reports the r or f that sits in front of the quote at index,
// or zero where nothing does.
//
// No word boundary is asked for, because the generic loop asks for none either:
// its trie matches r" wherever the two bytes fall, so forr"x" opens a raw string
// in both loops. See buildPythonStop.
func pythonQuotePrefix(content []byte, index, floor int) byte {
	switch byteBefore(content, index, floor) {
	case 'r':
		return 'r'
	case 'f':
		return 'f'
	}

	return 0
}

// pythonTripleAt reports whether three of the same quote byte sit at index.
func pythonTripleAt(content []byte, index int) bool {
	return index+2 < len(content) &&
		content[index+1] == content[index] && content[index+2] == content[index]
}

// pythonStringOpen works out what the quote at index opens, which is the whole
// of M4 and M5 for Python and is fiddlier than it looks.
//
// It returns the index the cursor lands on, the closer to look for, whether the
// string ignores escapes, and whether the form is a docstring.
//
// The cursor is where the generic loop's prepareString leaves it, and that is
// not always the last byte of the opening token. prepareString walks the quote
// list in the order languages.json spells it and takes the first entry that
// matches, which for the tripled r forms is the two byte r quote rather than the
// four byte one, so the cursor lands on the first of the three quotes. The f
// forms have no shorter entry to match first, so theirs lands on the last. The
// difference is visible on a file spelling a bare r followed by six quotes and
// has to be reproduced rather than tidied.
//
// A prefix that the database does not carry opens nothing, and the quote behind
// it is an ordinary one. That is every f in front of a single quote: there is an
// f""" and an f”' but no f" and no f'.
func pythonStringOpen(content []byte, index, floor int) (int, []byte, bool, bool) {
	quote := content[index]
	triple := pythonTripleAt(content, index)

	closer, tripleCloser := pythonDoubleQuote, pythonTripleDouble
	if quote == '\'' {
		closer, tripleCloser = pythonSingleQuote, pythonTripleSingle
	}

	switch pythonQuotePrefix(content, index, floor) {
	case 'r':
		if triple {
			// The trie matches the four byte form and hands back the triple
			// closer, while prepareString matches the shorter two byte one and
			// leaves the cursor on the first quote. Both halves belong to the
			// generic loop and both are kept.
			return index, tripleCloser, true, true
		}

		return index, closer, true, false
	case 'f':
		if triple {
			return index + 2, tripleCloser, true, true
		}
		// Falls through to the plain quote: the f is ordinary code.
	}

	if triple {
		return index + 2, tripleCloser, true, true
	}

	return index, closer, false, false
}

// pythonDocStringState runs a docstring to its closer, counting the lines it
// jumps over, and works out whether the line the closer sits on counts as
// comment or as code.
//
// It is docStringState of the generic loop with the byte loop taken out. The
// closer is found with bytes.Index rather than walked to, which is tuning 8 and
// matters more here than anywhere: 19.2% of the bytes of a Python file sit
// inside a triple quoted string. The jump loops over candidate closers rather
// than taking the first, because a closer with a backslash in front of it does
// not close, but each hop is still a vector scan.
//
// The backslash is tested whatever the quote form says about escapes. The
// generic loop's docstring state does not consult ignoreEscape at all, so a
// closer behind a backslash fails to close even a raw docstring, which is wrong
// about Python and right about the oracle.
func pythonDocStringState(content []byte, index, endPoint, floor int, closer []byte, tally *counterTally) (int, counterState) {
	if index >= endPoint {
		return index, SDocString
	}

	region := content[:endPoint]
	newlines := 0
	i := index

	for i < endPoint {
		rel := bytes.Index(region[i:], closer)
		if rel < 0 {
			break
		}

		at := i + rel

		// checkForMatchSingle fails the match where the closer would reach
		// endPoint, so a closer whose last byte sits on the final byte of the
		// file does not close.
		if at+len(closer) > endPoint {
			break
		}

		newlines += bytes.Count(region[i:at], newlineByte)

		if byteBefore(content, at, floor) != '\\' {
			bulkLines(tally, SDocString, int64(newlines))

			return at, pythonDocStringCloses(content, at+len(closer), endPoint)
		}

		i = at + 1
	}

	// Nothing closes the docstring before the end of the file. Every line but
	// the last is accounted for here and the last newline is handed back, which
	// is where the byte at a time version returned on each of them.
	newlines = bytes.Count(region[index:], newlineByte)
	if newlines == 0 {
		if index < endPoint {
			return endPoint - 1, SDocString
		}

		return index, SDocString
	}

	bulkLines(tally, SDocString, int64(newlines-1))

	return index + bytes.LastIndexByte(region[index:], '\n'), SDocString
}

// pythonDocStringCloses decides what the line holding the closer counts as.
//
// Only whitespace may follow the closer for the line to count as comment.
// Anything else, a hash included, makes it code. That is the generic loop's rule
// and it is looser than Python's own: a docstring followed by a comment counts
// as a line of code.
//
// The two debug lines are the generic loop's, emitted here so --debug prints the
// same thing whichever loop ran. The architecture doc §6 records this as the one
// hole in the eligibility guard, and the choice taken is the one it recommends:
// the counter says what the generic loop says rather than the guard quietly
// sending --debug down a different loop, which would be a bad property for the
// flag you reach for when a count looks wrong.
func pythonDocStringCloses(content []byte, from, endPoint int) counterState {
	for j := from; j <= endPoint && j < len(content); j++ {
		if content[j] == '\n' {
			printDebug("Found newline so docstring is comment")

			return SComment
		}

		if !isWhitespace(content[j]) {
			printDebugF("Found something not whitespace so is code: %s", string(content[j]))

			return SCode
		}
	}

	return SCode
}

// pythonBlankState looks at the first byte of content on a line, which for
// Python is the one place a docstring can open.
//
// The blank state is the one place the quote anchoring of buildPythonStop does
// not reach. Everywhere else the scan stops on the quote that ends a string's
// opening token and reads the prefix backwards, but here it is handed the first
// byte of content on the line whatever that byte is, so a line opening with a
// prefixed docstring arrives on the prefix. So this looks one byte forwards
// where the code scan looks one byte backwards, and the two meet at the same
// quote.
func pythonBlankState(content []byte, tally *counterTally, index, endPoint, floor int) (int, counterState, []byte, bool) {
	switch content[index] {
	case '#':
		return index, SComment, nil, false
	case '"', '\'':
		at, closer, ignoreEscape, docString := pythonStringOpen(content, index, floor)
		if docString {
			return at, SDocString, closer, ignoreEscape
		}

		return at, SString, closer, ignoreEscape
	case 'r', 'f':
		// A prefix opens a string only where the quote it wants sits behind it.
		// The r forms take either quote, singly or tripled; the f forms are
		// tripled only, the language database carrying no two byte f quote at
		// all. Where the quote is there but the form is not, the prefix is
		// ordinary code and the quote behind it is left to the code scan, which
		// is what the generic loop does with it: its trie walks one byte past
		// the f, finds no token, and reports nothing rather than falling back.
		if index+1 < len(content) && (content[index+1] == '"' || content[index+1] == '\'') {
			if content[index] == 'r' || pythonTripleAt(content, index+1) {
				at, closer, ignoreEscape, docString := pythonStringOpen(content, index+1, floor)
				if docString {
					return at, SDocString, closer, ignoreEscape
				}

				return at, SString, closer, ignoreEscape
			}
		}
	}

	if !Complexity && pythonComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// which is the whole of the distinction Python's comment counting rests on.
func pythonCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
		case '#':
			return i, SCommentCode, nil, false
		case '"', '\'':
			at, closer, ignoreEscape, _ := pythonStringOpen(content, i, floor)

			// The generic loop tests the byte in front of where the opening
			// token left the cursor and declines the string where it is a
			// backslash, counting the byte rather than the run. For a prefixed
			// or tripled form that byte is a quote or the prefix itself and is
			// never a backslash, so only the plain one byte quote can be
			// declined.
			if byteBefore(content, at, floor) == '\\' {
				return at, SCode, nil, false
			}

			return at, SString, closer, ignoreEscape
		default:
			// The byte in front of the anchor is the first thing
			// pythonComplexityAnchored tests and the last thing the caller knows
			// before paying for the call. A zero AND is exactly the case that
			// function rejects on its own first test, so skipping it cannot
			// change a count.
			if pythonAnchorPrev[byteBefore(content, i, floor)]&pythonAnchorBit[curByte] == 0 {
				continue
			}

			if pythonComplexityAnchored(content, i, floor) {
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

// countLoopPython stands in for countLoopGeneric where the language is Python.
// It returns false when it ended the count early, the same way the generic loop
// does.
func countLoopPython(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := pythonStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// What closes the string or docstring the scan is in, and whether that form
	// ignores escapes. Python's triple quoted forms close with three bytes, so
	// this is not always one.
	endQuote := pythonDoubleQuote
	ignoreEscape := false

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, raw := pythonCodeState(content, &tally, index, endPoint, floor, stop)
			if quote != nil {
				endQuote, ignoreEscape = quote, raw
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, ignoreEscape)
		case SDocString:
			return pythonDocStringState(content, index, endPoint, floor, endQuote, &tally)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		default: // SBlank and SMulticommentBlank
			index, state, quote, raw := pythonBlankState(content, &tally, index, endPoint, floor)
			if quote != nil {
				endQuote, ignoreEscape = quote, raw
			}

			return index, state
		}
	}

	// Python does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}

// pythonQuotes is every quote of Python as languages.json spells it, start and
// end in turn, which the structural conformance test holds the database to.
func pythonQuotes() []string {
	return []string{
		`"`, `"`,
		`'`, `'`,
		`r'`, `'`,
		`r"`, `"`,
		`"""`, `"""`,
		`'''`, `'''`,
		`r"""`, `"""`,
		`r'''`, `'''`,
		`f"""`, `"""`,
		`f'''`, `'''`,
	}
}
