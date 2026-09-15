// SPDX-License-Identifier: MIT

package processor

// One counter for C++ and C++ Header. The two are the same language to
// languages.json: identical complexitychecks, quotes, line_comment, multi_line
// and linesplice, differing only in extensions and in C++ Header's eight
// heuristics, which are detection's business and not counting's. So this is one
// counter and two entries in the registry.
//
// C++ is C with two things added. Its check list carries try and catch, which C
// does not, and it has five raw string forms whose closer is written into the
// file rather than fixed by the language. Everything else is the shared spine.

// cppStop marks every byte the scan stops on: the slash of both comment forms,
// the one quote every string form ends with, one anchor byte out of every
// complexity check, and the newline and null that end a line and a file of bytes
// rather than text.
var cppStop = buildCppStop()

// cppStopNoComplexity holds only what changes the state, which is what a file
// counted with --no-complexity is scanned with.
var cppStopNoComplexity = buildCppStopNoComplexity()

// cppPlainQuote is the ordinary string of C++. The raw forms name their own
// closer and build it as they go.
var cppPlainQuote = []byte{'"'}

// cppRawFallbackEnd is what a raw string closes with when the delimiter was
// never readable: the End the language database gives the raw quotes, which is
// what an empty delimiter closes with. See cppOpensRawString.
var cppRawFallbackEnd = []byte(`)"`)

// cppRawPrefixes are the bytes that sit in front of the quote of a raw string,
// longest first so the longest match wins the way the generic loop's trie does.
// u8R has to be tested before uR, since both end in R.
var cppRawPrefixes = []string{"u8R", "uR", "UR", "LR", "R"}

// cppQuoteAnchors is the byte each raw string form is stopped on, which is the
// quote that ends its opening token rather than the letter that begins it. See
// buildCppStop for what that is worth, and cppOpensRawString for the read.
var cppQuoteAnchors = map[string]byte{
	`R"`:   '"',
	`u8R"`: '"',
	`uR"`:  '"',
	`UR"`:  '"',
	`LR"`:  '"',
}

// cppComplexityAnchors is the byte each complexity check of C++ is stopped on.
var cppComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
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

// buildCppStop marks the anchor of every complexity check of C++.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	y   try
//	h   catch
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// Each is the rarest byte of its check, measured over 20MB of protobuf and
// googletest: w 0.25% for while and switch against s at 3.78%, y 0.62% for try
// against t at 5.47%, h 0.73% for catch against c at 1.76% and a at 3.70%, f
// 1.36% for if and for against i at 4.10%, l 2.78% for else against e at 7.55%.
//
// No check holds another's anchor where reading back from it would match, so the
// scan carries on through a matched token rather than stepping over it. while
// carries the h of catch, but reading four bytes back from it gives nothing that
// spells catch; while carries the l of else, but else wants an e in front of its
// l and finds an i; try and catch share no anchor with anything.
//
// R, u, U and L are deliberately NOT in this table, although every raw string
// starts with one of them. They are 3.32% of a C++ file between them, which is
// half as much again as every complexity anchor put together. A raw string is
// recognised at its quote instead, reading the prefix backwards, and the quote
// is a byte the scan already stops on at 0.63%. Anchoring is usually described
// as a trick for keywords; it works just as well on the opening of a string.
func buildCppStop() [256]bool {
	table := buildCppStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', 'y', 'h', '|', '&', '='} {
		table[b] = true
	}

	return table
}

func buildCppStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// cppStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func cppStopTable() *[256]bool {
	if Complexity {
		return &cppStopNoComplexity
	}

	return &cppStop
}

// cppComplexityAnchored reports whether a complexity check of C++ sits on the
// anchor byte at index, which is what cppStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. No check of C++ is spelled with a quote, a slash or
// a newline, so reading back never crosses out of the code the scan is in, and
// every read is clamped to floor besides.
func cppComplexityAnchored(content []byte, index, floor int) bool {
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

// cppComplexityAtLineStart is cppComplexityAnchored for the first byte of code
// on a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored on
// a byte the code scan reaches, since that begins on the byte after this one.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test. A raw string opening a line needs nothing here either:
// its prefix is ordinary code to this state and the quote that follows is read
// by the code scan.
func cppComplexityAtLineStart(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, floor, "| ")
	case '&':
		return hasPrefixAt(content, index+1, floor, "& ")
	}

	return false
}

// cppOpensRawString reports whether the quote at index closes the opening token
// of a raw string, and if so where the scan carries on from and what closes it.
//
// The prefix is read backwards, which is what keeps R, u, U and L out of the
// stop table. The generic loop finds the same strings forwards from its trie:
// it takes the longest token at the earliest byte, so u8R" beats uR" and both
// beat R", which is why cppRawPrefixes is ordered longest first. Neither loop
// asks for a word boundary in front, so fooR"(x)" opens a raw string in both.
//
// A raw string names its own closer in the bytes between the quote and the
// bracket, so the closer is read out of the file. Where there is no readable
// delimiter, because a byte that may not appear in one arrived first or the file
// ended, the language database's End stands and the string closes at )" . That
// is what the generic loop falls back to, and a file holding R" and nothing else
// leans on it.
func cppOpensRawString(content []byte, index, floor int) (next int, endQuote []byte, ok bool) {
	prefix := ""
	for _, candidate := range cppRawPrefixes {
		if hasPrefixAt(content, index-len(candidate), floor, candidate) {
			prefix = candidate
			break
		}
	}

	if prefix == "" {
		return 0, nil, false
	}

	// rawStringEnd reads the delimiter, bounded by maxRawStringDelimiter, and
	// reports the bracket that ends it. It is the generic loop's own reader, so
	// a delimiter neither loop accepts is rejected identically.
	if rawEnd, bracket := rawStringEnd(content, index+1, '"'); bracket != -1 {
		return bracket, rawEnd, true
	}

	return index, cppRawFallbackEnd, true
}

// cppBlankState looks at the first byte of content on a line.
func cppBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte, bool) {
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
		// A line can open on the quote of a raw string only where its prefix
		// closed the line above, which no prefix can do, so this is the plain
		// quote. The read is made anyway rather than assumed, since it costs one
		// compare on the first byte of a line.
		if next, endQuote, ok := cppOpensRawString(content, index, floor); ok {
			return next, SString, endQuote, true
		}

		return index, SString, cppPlainQuote, false
	}

	if !Complexity && cppComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// cppCodeState runs to the end of the line or to whatever token takes it out of
// code.
func cppCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
		case '"':
			// A raw string is recognised first, and its own escape question does
			// not arise: the generic loop tests the byte in front of where the
			// opening token left the cursor, which for a raw string is the
			// bracket or the quote and never a backslash, so it never declines
			// one. A delimiter may not hold a backslash either.
			if next, endQuote, ok := cppOpensRawString(content, i, floor); ok {
				return next, SString, endQuote, true
			}

			// The plain quote, which the generic loop declines when a backslash
			// sits in front of it, counting the byte rather than the run.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString, cppPlainQuote, false
			}

			return i, SCode, nil, false
		default:
			if cppComplexityAnchored(content, i, floor) {
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

// countLoopCpp stands in for countLoopGeneric where the language is C++ or C++
// Header. It returns false when it ended the count early, the same way the
// generic loop does.
func countLoopCpp(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := cppStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// What closes the string the scan is in, and whether that string is a raw
	// one. C++ is the only counted language that both splices lines and has a
	// quote with no escape mechanism, so it is the only one where the second of
	// these matters: an ordinary string ends at a newline it is not spliced
	// across, and a raw string does not. See spliceRule.
	endQuote := cppPlainQuote
	inRawString := false

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, raw := cppCodeState(content, &tally, index, endPoint, floor, stop)
			if quote != nil {
				endQuote, inRawString = quote, raw
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, inRawString)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state before the
			// newline, so the rest of the line is skipped a vector at a time.
			// Whether it carries on past the newline is the splice, worked out
			// in the shared loop where the line is counted.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			return counterCommentState(content, index, endPoint, state, slashStarClose, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote, raw := cppBlankState(content, &tally, index, floor)
			if quote != nil {
				endQuote, inRawString = quote, raw
			}

			return index, state
		}
	}

	return countLoopShared(fileJob, &tally, bomSkip, endPoint,
		spliceRule{Splices: true, InRawString: &inRawString}, step)
}
