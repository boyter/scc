// SPDX-License-Identifier: MIT

package processor

// A counter written for Ruby, on the same terms as the rest: it must agree with
// the generic loop to the line, and where the two differ the generic one is
// right by definition.
//
// Ruby asks for one thing no counter had before. Its block comment is spelled
// =begin and =end rather than a pair of punctuation marks, which is M11. That
// is the first multi-byte block comment delimiter of the sixteen, and it is why
// Ruby is one of the three languages where anchoring is not sound for free. The
// argument is at buildRubyStop and it is the most important thing in this file.
//
// What Ruby does NOT do here is understand a heredoc. The generic loop has no
// heredoc token, so <<~SQL and the text under it are counted as ordinary code
// and a quote inside one opens a string like any other. That is wrong about
// Ruby and right about the oracle, so it is what this does too.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// rubyBlockOpen and rubyBlockClose are the block comment delimiters. Ruby wants
// them in the first column; the generic loop does not ask, and matches one
// indented or sitting after code on a line, so neither does this. See
// rubyBlankState.
var (
	rubyBlockOpen  = []byte("=begin")
	rubyBlockClose = []byte("=end")
)

// rubyStop marks every byte the scan has to stop on: the hash of the line
// comment, the two quotes, the equals that opens a block comment, one anchor
// byte out of every complexity check, and the newline and the null that end a
// line and a file of bytes rather than text.
var rubyStop = buildRubyStop()

// rubyStopNoComplexity is rubyStop without the bytes that only a complexity
// check is spelled with. It is what a file counted with --no-complexity is
// scanned with, the generic loop leaving the checks out of its trie under the
// same flag.
//
// The equals stays in it. Every other counter drops its complexity anchors
// wholesale under that flag, but Ruby's = is not only an anchor: it is the first
// byte of =begin, and a scan that stopped stopping on it would read a whole
// block comment as code.
var rubyStopNoComplexity = buildRubyStopNoComplexity()

// The two quotes of Ruby, held here so the shared string state is handed a slice
// rather than building one per string.
var (
	rubyDoubleQuote = []byte{'"'}
	rubySingleQuote = []byte{'\''}
)

// rubyComplexityAnchors is the byte each complexity check of Ruby is stopped on.
// It is what the structural conformance test holds against rubyStop, and it is
// the written form of the argument in buildRubyStop.
var rubyComplexityAnchors = map[string]byte{
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

// buildRubyStop marks the anchor of every complexity check of Ruby.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//
// The anchor is the rarest byte of the check rather than its first. Measured
// over 19.3MB of rails: f 1.18% against i 3.82%, o 4.40% and r 4.21% for if and
// for; w 0.56% against s 4.50%, h 1.20% and c 2.53% for switch and while; l
// 2.83% against e at nearly 9% for else. | at 0.15% and & at 0.06% are already
// their own rarest byte.
//
// ! at 0.06% is rarer than = at 0.58%, so != would be cheaper anchored on its
// own first byte if it were the only check that wanted one. It is not: == has
// to be anchored on an =, and = has to be in the table regardless because it
// opens =begin, so reading != backwards from its = costs nothing at all and
// keeps ! out of the table.
//
// # ANCHORING SOUNDNESS, which Ruby does not get for free
//
// Spec 07 03-architecture §2.1 names Ruby one of three languages whose
// complexity checks share bytes with the tokens that open or close a quote, a
// line comment or a block comment. Ruby's colliding bytes are =, e and i: =begin
// and =end are spelled with all three, and so are the checks, == and != carrying
// the =, else the e, and if, while and switch the i.
//
// The choice §2.1 offers is to anchor the colliding checks on their first byte
// and pay the extra stops, or to prove the backwards read still cannot cross a
// token boundary. This counter proves it. The argument is in four parts.
//
// 1. e and i are never anchors. The anchor set is f, w, l, |, & and =. The scan
// therefore never stops on an e or an i and never begins a backwards read at
// one. They are only ever the target of a read that began somewhere else, which
// is a much weaker thing than the collision table's blanket intersection
// suggests: that table is computed over every byte a check is spelled with, and
// most of those bytes the scan never looks at.
//
// 2. A read can never begin inside =begin or =end. Every byte of those two
// tokens other than the leading = is one of b, e, g, i, n, d, and none of them
// is an anchor, so the scan does not stop there. The leading = is an anchor, but
// rubyCodeState tests =begin before it tests any complexity check, and the two
// are mutually exclusive on the byte after the =, b against = or !. So the scan
// leaves the code state at the = and the interior bytes of the delimiter are
// never scanned as code at all.
//
// 3. A read can never reach into a delimiter from outside it. Every backwards
// read here is one byte deep, plus the word boundary that reads one more: f
// wants an i behind it, w an s, l an e, and = an = or a !. For any of those to
// land inside a delimiter the anchor would have to sit immediately behind one,
// and =begin ends in n while =end ends in d, neither of which is a byte any
// check carries behind its anchor. Every such read fails on its first test.
//
// The case that gets furthest is the l of else, which wants an e behind it, and
// both delimiters carry an e. Neither can supply it: the e of =begin is followed
// by g and the e of =end by n, never by an l. Where the file spells something
// like =belse the l does have an e behind it, but that e is part of no delimiter,
// =begin having failed to match at the =, and the word boundary then fails on
// the b in front of it.
//
// 4. Nothing inside a comment is read at all. Once =begin has opened, the scan
// is in the comment state, which finds =end with bytes.Index and runs no
// complexity matcher and no backwards read. So the body of a block comment
// cannot produce a false count however it is spelled.
//
// The ordinary argument then still holds for the rest: no check holds the anchor
// of another check where reading back from it can match, which is what lets the
// scan carry on through a matched token rather than stepping over it. while
// carries the l of else, but else wants an e in front of its l and finds an i.
// The second | of || and the second & of && are read as the start of another
// check and fail for want of a third byte.
func buildRubyStop() [256]bool {
	table := buildRubyStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&'} {
		table[b] = true
	}

	return table
}

func buildRubyStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'#', '"', '\'', '=', '\n', 0} {
		table[b] = true
	}

	return table
}

// rubyStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func rubyStopTable() *[256]bool {
	if Complexity {
		return &rubyStopNoComplexity
	}

	return &rubyStop
}

// rubyBlockCommentOpens reports whether a block comment opens at index, and
// where the cursor is left when it does.
//
// The second return is not simply the end of the token. The generic loop steps
// on by what Trie.Match reported, and Match counts the byte it stopped on rather
// than the bytes it matched, so a token running to the very last byte of the
// file comes back one short. Ruby's opener is six bytes where every other
// counter's is two, so the gap is wide enough to be worth reproducing exactly
// rather than reasoning about each time.
func rubyBlockCommentOpens(content []byte, index, floor int) (int, bool) {
	if !hasPrefixAt(content, index, floor, "=begin") {
		return index, false
	}

	jump := len(rubyBlockOpen)
	if len(content)-index <= len(rubyBlockOpen) {
		jump--
	}

	return index + max(jump, 1) - 1, true
}

// rubyComplexityAnchored reports whether a complexity check of Ruby sits on the
// anchor byte at index, which is what rubyStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. See buildRubyStop for why that read cannot cross
// out of the code the scan is in, which for Ruby takes an argument rather than
// the one-liner the C family gets. Every backwards read is clamped to floor, so
// it cannot read in front of the region the counter owns either.
//
// switch, while and else are spelled with a space behind them and nothing else,
// so they take spaceOpens; using cOpens would count a switch( the language
// database does not carry. if and for have both forms.
func rubyComplexityAnchored(content []byte, index, floor int) bool {
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

// rubyComplexityAtLineStart is rubyComplexityAnchored for the first byte of code
// on a line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored on
// a byte the code scan reaches, since that begins on the byte after this one and
// no check that is anchored on its first byte holds an anchor of its own
// anywhere else that can match.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test.
//
// This is not an optimisation that can be left out. Without it for, while, || and
// && are never counted when they open a line, and it is the reason the backwards
// reads above can be written as reads rather than as searches. The two are one
// thing.
func rubyComplexityAtLineStart(content []byte, index, floor int) bool {
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

// rubyBlankState looks at the first byte of content on a line.
//
// =begin is matched here with no test that it sits in the first column, because
// the generic loop has no such test: it reaches this state at the first byte of
// content on the line whatever the indentation, and opens a comment on what it
// finds. Real Ruby wants the delimiter in column one. The oracle does not ask,
// so neither does this.
func rubyBlankState(content []byte, tally *counterTally, index, floor int) (int, counterState, []byte) {
	switch content[index] {
	case '#':
		return index, SComment, nil
	case '=':
		if end, ok := rubyBlockCommentOpens(content, index, floor); ok {
			return end, SMulticomment, nil
		}
	case '"':
		return index, SString, rubyDoubleQuote
	case '\'':
		return index, SString, rubySingleQuote
	}

	if !Complexity && rubyComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil
}

// rubyCodeState runs to the end of the line or to whatever token takes it out of
// code.
func rubyCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte) {
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
		case '#':
			return i, SCommentCode, nil
		case '=':
			// The block comment is looked for before the complexity check, and
			// the two cannot both match: =begin wants a b behind its = and the
			// check wants another = or a !. See buildRubyStop.
			if end, ok := rubyBlockCommentOpens(content, i, floor); ok {
				return end, SMulticommentCode, nil
			}

			if !Complexity && rubyComplexityAnchored(content, i, floor) {
				tally.Complexity++
			}
		case '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code. A quote on the floor has nothing in front
			// of it and so is not escaped; the state machine cannot reach here on
			// it, having started blank, but the check does not depend on that
			// holding.
			if byteBefore(content, i, floor) != '\\' {
				if curByte == '"' {
					return i, SString, rubyDoubleQuote
				}

				return i, SString, rubySingleQuote
			}

			return i, SCode, nil
		default:
			if rubyComplexityAnchored(content, i, floor) {
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

// countLoopRuby stands in for countLoopGeneric where the language is Ruby and
// none of the extra outputs are wanted. It returns false when it ended the count
// early, the same way the generic loop does.
func countLoopRuby(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := rubyStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// The quote the string state is looking for. Ruby has two and the code and
	// blank states say which one opened, so it is carried across calls. It is
	// never nil: the states below hand back the quote only when they opened a
	// string, and anything else leaves the last one in place.
	endQuote := rubyDoubleQuote
	openQuote := func(quote []byte) {
		if quote != nil {
			endQuote = quote
		}
	}

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote := rubyCodeState(content, &tally, index, endPoint, floor, stop)
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
			return counterCommentState(content, index, endPoint, state, rubyBlockClose, &tally)
		default: // SBlank and SMulticommentBlank
			index, state, quote := rubyBlankState(content, &tally, index, floor)
			openQuote(quote)

			return index, state
		}
	}

	// Ruby does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}
