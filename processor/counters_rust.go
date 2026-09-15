// SPDX-License-Identifier: MIT

package processor

// A counter for Rust, which carries more machinery than any other language in
// this work: twenty quote forms, eighteen of them raw with an opening token of
// up to eleven bytes and a closer of up to nine, block comments that nest, a
// postfix complexity check with an exclusion, and a character literal the
// language database cannot describe.
//
// Two things here are not in any other counter.
//
// The raw strings are anchored on the quote that ends their opening token and
// the r and the hashes are read backwards, which is the trick the C++ counter
// found. It is worth more here than anywhere. Measured over 145MB of
// rust-lang/rust, r is 3.16% of a Rust file and b another 0.82%, so a counter
// that stopped on the first byte of a raw string would pay 3.98% for the
// privilege, against 7.37% for all nine complexity anchors put together.
// Anchoring on the quote costs 0.51% instead, and r and b stay out of the stop
// table entirely.
//
// And the character literal is the one place this counter is deliberately
// better than the generic loop. See rustCharLiterals.
//
// Everything that is not the stop table and the complexity matcher lives in
// counters_shared.go.

// rustStop marks every byte the scan has to stop on: the slash of both comment
// forms, the quote that ends the opening token of every string form, the single
// quote of a character literal, one anchor byte out of every complexity check,
// and the newline and null that end a line and a file of bytes rather than text.
var rustStop = buildRustStop()

// rustStopNoComplexity holds only what changes the state, which is what a file
// counted with --no-complexity is scanned with.
var rustStopNoComplexity = buildRustStopNoComplexity()

// rustCharQuote closes the byte string that b' opens, which is the one quote of
// Rust the language database spells with a single quote.
var rustCharQuote = []byte{'\''}

// rustPlainQuote is the ordinary string.
var rustPlainQuote = []byte{'"'}

// maxRustRawHashes is the longest run of hashes languages.json gives a raw
// string, which is what bounds the backwards read at a quote. A file spelling
// more than this opens no raw string in either loop: the generic loop's trie
// has no node for it and falls back to the plain quote, and the read below
// stops looking and does the same.
const maxRustRawHashes = 8

// rustRawEnds is what a raw string closes with, indexed by how many hashes its
// opening token carried. Built once rather than assembled per string.
var rustRawEnds = buildRustRawEnds()

func buildRustRawEnds() [maxRustRawHashes + 1][]byte {
	var ends [maxRustRawHashes + 1][]byte
	for hashes := range ends {
		end := make([]byte, 0, hashes+1)
		end = append(end, '"')
		for range hashes {
			end = append(end, '#')
		}
		ends[hashes] = end
	}

	return ends
}

// rustQuoteAnchors is the byte each quote form is stopped on where that is not
// its first byte. Every raw form is caught on the quote that closes its opening
// token, and the byte string on the quote that follows its b.
var rustQuoteAnchors = buildRustQuoteAnchors()

func buildRustQuoteAnchors() map[string]byte {
	anchors := map[string]byte{`b'`: '\''}
	for hashes := range maxRustRawHashes + 1 {
		start := "r"
		for range hashes {
			start += "#"
		}
		start += `"`
		anchors[start] = '"'
		anchors["b"+start] = '"'
	}

	return anchors
}

// rustComplexityAnchors is the byte each complexity check of Rust is stopped
// on. It is what the structural conformance test holds against rustStop, and it
// is the written form of the argument in buildRustStop.
var rustComplexityAnchors = map[string]byte{
	"for ": 'f', "for(": 'f',
	"if ": 'f', "if(": 'f',
	"while ": 'w', "while(": 'w',
	"loop ": 'p', "loop{": 'p',
	"else ": 'l', "else{": 'l',
	"match ": 'h', "match(": 'h',
	"|| ": '|',
	"&& ": '&',
	"!= ": '=',
	"== ": '=',
	"?":   '?',
}

// buildRustStop marks the anchor of every complexity check of Rust.
//
//	f   if, for       the f of if is read backwards, the f of for forwards
//	w   while
//	p   loop          read backwards, loo sitting in front of it
//	l   else
//	h   match         read backwards, matc sitting in front of it
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
//	?   the postfix check, which is its own first and only byte
//
// Each is the rarest byte of its check, measured over 145MB of rust-lang/rust:
// w 0.42% for while against h at 0.98% and i, l and e all above 2%; h 0.98% for
// match against m at 1.46%, c at 1.85% and t above 5%; p 1.43% for loop against
// l at 2.25% and o at 3%; f 1.46% for if and for against i above 3% and r at
// 3.16%; l 2.25% for else against e near 7% and s above 3%. The operators are
// already their own rarest byte, ? at 0.04%, | at 0.09%, & at 0.27% and = at
// 0.43%.
//
// loop and match are the two anchored on their last byte rather than an
// interior one, which costs a four byte and a three byte backwards read and
// buys the difference between p at 1.43% and l at 2.25%, and between h at 0.98%
// and m at 1.46%. There is a second reason besides the frequency: a check
// anchored on its own first byte has to be looked for again at the first byte
// of a line, where there is nothing behind it to read, and every check that
// does not need that is one less thing for rustComplexityAtLineStart to carry.
//
// No check holds another's anchor where reading back from it would match, so
// the scan carries on through a matched token rather than stepping over it.
// while carries the h of match, but reading four bytes back from an h inside
// while gives nothing that spells match. while and loop and else all carry an
// l, but else wants an e in front of its l and finds an i inside while and an
// o inside loop. loop carries a p only at its end.
//
// # ANCHORING SOUNDNESS, which Rust does not get for free
//
// Spec 07 03-architecture §2.1 names Rust one of three languages whose
// complexity checks share a byte with the tokens that open a quote, a line
// comment or a block comment. Rust's colliding byte is r: every raw string
// begins r" or br" or r#" and so on, and for is spelled with an r too.
//
// The choice §2.1 offers is to anchor the colliding checks on their first byte
// and pay the extra stops, or to prove the backwards read still cannot cross a
// token boundary. This counter proves it, and the proof is the same fact that
// makes the counter fast. The argument is in four parts.
//
// 1. r is never an anchor. The complexity anchors are f, w, p, l, h, |, &, =
// and ?, and the quote anchors are " and '. A raw string is caught on the quote
// that ends its opening token, never on the r that begins it, so r is in
// neither table. The scan does not stop on an r, does not begin a backwards
// read at one, and never asks what an r means.
//
// 2. The one read that touches an r cannot reach past it. rustRawStringEnd
// begins at a quote the scan stopped on and walks back over at most eight
// hashes to exactly one r. It reads no byte beyond that r, so it cannot reach
// into whatever is in front, and it is clamped to floor besides. The r it finds
// is the last byte of the opening token it is reading, not a byte belonging to
// some other token.
//
// 3. Finding that r inside a word is not a mistake, because the generic loop
// does the same. Neither loop asks for a word boundary in front of a quote, so
// for" opens a raw string in both: the trie reaches the r of for going
// forwards and matches r", and the read here reaches the same r going
// backwards. The two agree by construction rather than by luck, which is the
// property that matters, and it is why a collision on r is harmless where a
// collision that changed the answer would not be.
//
// 4. No complexity read can reach into a raw string's opening token. Every
// complexity read here is at most four bytes deep and asks for letters: f wants
// an i, p wants loo, h wants matc, l wants an e, = wants an = or a !. A raw
// string's opening token is r, hashes and a quote. For a complexity read to
// land inside one, the anchor would have to sit within four bytes of it with
// only those bytes between, and none of r, # or " is a byte any check carries
// behind its anchor. Every such read fails on its first test.
//
// The collision table of §2.1 is computed over every byte a check is spelled
// with, which for Rust flags the r of for. Only the bytes the scan actually
// stops on and the one or two a read touches can matter, and by that measure
// Rust collides on nothing at all.
func buildRustStop() [256]bool {
	table := buildRustStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'p', 'l', 'h', '|', '&', '=', '?'} {
		table[b] = true
	}

	return table
}

func buildRustStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', '\n', 0} {
		table[b] = true
	}

	return table
}

// rustStopTable picks the table the scan runs with. The global reads as
// complexity having been turned off.
func rustStopTable() *[256]bool {
	if Complexity {
		return &rustStopNoComplexity
	}

	return &rustStop
}

// rustComplexityAnchored reports whether a complexity check of Rust sits on the
// anchor byte at index, which is what rustStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. Every read is clamped to floor, and see
// buildRustStop for why none of them can cross out of the code the scan is in.
//
// The postfix ? is not here. It is not a word and carries no boundary, and the
// generic loop counts it under its own rules, which rustPostfixCounts keeps.
func rustComplexityAnchored(content []byte, index, floor int) bool {
	switch content[index] {
	case 'f':
		if byteBefore(content, index, floor) == 'i' {
			return wordStartsAt(content, index-1, floor) && cOpens(content, index+1)
		}

		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "or") && cOpens(content, index+3)
	case 'w':
		return wordStartsAt(content, index, floor) &&
			hasPrefixAt(content, index+1, floor, "hile") && cOpens(content, index+5)
	case 'p':
		if byteBefore(content, index, floor) != 'o' {
			return false
		}

		return wordStartsAt(content, index-3, floor) &&
			hasPrefixAt(content, index-3, floor, "loop") && braceOpens(content, index+1)
	case 'l':
		if byteBefore(content, index, floor) != 'e' {
			return false
		}

		return wordStartsAt(content, index-1, floor) &&
			hasPrefixAt(content, index+1, floor, "se") && braceOpens(content, index+3)
	case 'h':
		if byteBefore(content, index, floor) != 'c' {
			return false
		}

		return wordStartsAt(content, index-4, floor) &&
			hasPrefixAt(content, index-4, floor, "match") && cOpens(content, index+1)
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

// rustComplexityAtLineStart is rustComplexityAnchored for the first byte of
// code on a line, which has nothing in front of it to read back to. Only the
// checks anchored on their own first byte are looked for here; the rest are
// anchored on a byte the code scan reaches, since that begins on the byte after
// this one.
//
// Nothing carries a word into the first byte of code on a line, so the word
// boundary needs no test.
//
// loop, match, if, else and the equality pair are all anchored on a later byte
// and are deliberately absent. Neither is the postfix ?, which the caller
// handles before it gets here, since it counts under rules of its own rather
// than as a word.
func rustComplexityAtLineStart(content []byte, index, floor int) bool {
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

// rustPostfixCounts reports whether the postfix ? at index counts, which is
// countComplexityPostfix's rule plus Rust's one exclusion.
//
// The ? of a ?Sized bound is not a try operator and does not count. The generic
// loop reads the exclusion with whitespace allowed in between, so ? Sized is
// excluded too, and requires a word boundary after it so ?Sizedness is not.
//
// The generic loop hands hasPostfixExclude the length its trie walk reported,
// which is one for a ? with anything after it and zero for a ? that is the last
// byte of the file. The zero case cannot be an exclusion, there being no room
// for Sized behind it, so one is the only length that reaches the comparison.
func rustPostfixCounts(content []byte, index int) bool {
	if index == 0 {
		return false
	}

	if isWhitespace(byteBefore(content, index, 0)) && !hasNonWhitespaceBefore(content, index-1) {
		return false
	}

	next := nextNonWhitespaceIndex(content, index+1)
	if next+len("Sized") > len(content) || string(content[next:next+len("Sized")]) != "Sized" {
		return true
	}

	after := next + len("Sized")

	return after != len(content) && isIdentifierContinue(content[after])
}

// rustRawStringEnd reports what closes the raw string whose opening token ends
// on the quote at index, and whether there is one at all.
//
// The opening token is read backwards: at most eight hashes and then exactly
// one r. The b of br" is not looked for, because it changes nothing. br" and r"
// close alike, br#" and r#" close alike, and a raw string opens on either, so
// the r is the whole of what has to be found.
//
// The generic loop finds the same strings forwards from its trie, which takes
// the longest token at the earliest byte. Neither loop asks for a word boundary
// in front, so for" opens a raw string in both. See buildRustStop.
func rustRawStringEnd(content []byte, index, floor int) ([]byte, bool) {
	hashes := 0
	j := index - 1
	for hashes < maxRustRawHashes && j >= floor && content[j] == '#' {
		hashes++
		j--
	}

	if j < floor || content[j] != 'r' {
		return nil, false
	}

	return rustRawEnds[hashes], true
}

// rustCharLiterals turns the one deliberate disagreement with the generic loop
// on and off. It is on, and nothing but a test turns it off.
//
// M15. Rust's quotes carry b' but no plain ', because a plain ' cannot be a
// quote in a table: 'a is a lifetime and would open a string that never closes.
// So the table leaves the character literal out altogether, and '"' lets the
// quote inside it open a string instead, which swallows the rest of the file.
// That is LineJudge 4010 and 4020.
//
// A counter can hold both rules at once, which is the whole argument for
// writing counters by hand:
//
//	'  one character  '   is a literal
//	'  \ escape       '   is a literal
//	'  identifier bytes   with no closing ' is a lifetime
//
// A lifetime is ordinary code to both loops and needs nothing done to it, so
// this changes an answer only where a real character literal holds a byte the
// generic loop would have read as a token. The fixture for
// 4050-lifetime_and_apostrophe_in_string is the regression test that says so.
var rustCharLiterals = true

// rustCharLiteralEnd reports the index of the quote that closes the character
// literal opening at index, or -1 where what opens there is a lifetime, is not
// a literal at all, or runs off the line.
//
// A literal holds exactly one character, which may be several bytes of UTF-8,
// or one escape. Nothing here may cross a newline, and the scan is bounded to
// the longest literal Rust can spell so that a file of nothing but quotes costs
// a fixed amount per quote rather than a search of the rest of the file.
//
// The newline is the case that matters and it is not theoretical. A line
// holding a single ' , which rust-analyzer's own lexer fixtures are full of,
// puts a newline where the character should be and the quote that opens the
// next line where the closer should be. Reading that as a literal swallows the
// line ending and the file comes out short, which is a wrong line count rather
// than a divergence.
func rustCharLiteralEnd(content []byte, index, endPoint int) int {
	if !rustCharLiterals {
		return -1
	}

	// The longest literal Rust can spell is '\u{10FFFF}' at twelve bytes.
	limit := index + maxRustCharLiteral
	if limit > endPoint {
		limit = endPoint
	}

	j := index + 1
	if j >= limit {
		return -1
	}

	if content[j] == '\\' {
		j = rustEscapeEnd(content, j, limit)
		if j < 0 {
			return -1
		}
	} else {
		if content[j] == '\n' {
			return -1
		}
		j += rustRuneLen(content[j])
	}

	if j >= limit || content[j] != '\'' {
		return -1
	}

	return j
}

// maxRustCharLiteral is the length of the longest character literal Rust can
// spell, '\u{10FFFF}' , plus a byte of slack. It bounds every read below.
const maxRustCharLiteral = 13

// rustEscapeEnd reports the index one past the escape sequence beginning at the
// backslash at index, or -1 where it runs off the line or the region.
//
// Rust spells four shapes: a single character such as \n or \' , a byte as
// \xNN, a unicode scalar as \u{NNNNNN}, and nothing else. Every one of them is
// bounded by limit, and a newline anywhere inside one ends the attempt: a
// literal cannot hold a raw line ending, and reading one that appears to would
// lose the line.
func rustEscapeEnd(content []byte, index, limit int) int {
	j := index + 1
	if j >= limit {
		return -1
	}

	switch content[j] {
	case '\n':
		return -1
	case 'x':
		if j+3 > limit || content[j+1] == '\n' || content[j+2] == '\n' {
			return -1
		}

		return j + 3
	case 'u':
		if j+1 >= limit || content[j+1] != '{' {
			return j + 1
		}

		for k := j + 2; k < limit; k++ {
			if content[k] == '}' {
				return k + 1
			}
			if content[k] == '\n' {
				return -1
			}
		}

		return -1
	}

	return j + 1
}

// rustRuneLen reports how many bytes the UTF-8 sequence beginning with b runs
// to. A byte that begins no sequence is read as one byte, which is what a
// character literal holding invalid UTF-8 amounts to.
func rustRuneLen(b byte) int {
	switch {
	case b < 0x80:
		return 1
	case b >= 0xF0:
		return 4
	case b >= 0xE0:
		return 3
	case b >= 0xC0:
		return 2
	}

	return 1
}

// rustBlankState looks at the first byte of content on a line.
func rustBlankState(content []byte, tally *counterTally, index, endPoint, floor int) (int, counterState, []byte, bool) {
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
		// A raw string cannot open a line on its quote, its own prefix having
		// to sit in front of it, but the read is made rather than assumed since
		// it costs one compare on the first byte of a line.
		if endQuote, ok := rustRawStringEnd(content, index, floor); ok {
			return index, SString, endQuote, true
		}

		return index, SString, rustPlainQuote, false
	case '\'':
		// b' cannot open a line on its quote either, the b having to sit in
		// front. What can is a character literal, and skipping it here is what
		// stops the byte inside it being read as a token by the code scan.
		if end := rustCharLiteralEnd(content, index, endPoint); end >= 0 {
			return end, SCode, nil, false
		}

		return index, SCode, nil, false
	case '?':
		if !Complexity && rustPostfixCounts(content, index) {
			tally.Complexity++
		}

		return index, SCode, nil, false
	}

	if !Complexity && rustComplexityAtLineStart(content, index, floor) {
		tally.Complexity++
	}

	return index, SCode, nil, false
}

// rustCodeState runs to the end of the line or to whatever token takes it out
// of code.
func rustCodeState(content []byte, tally *counterTally, index, endPoint, floor int, stop *[256]bool) (int, counterState, []byte, bool) {
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
			// A raw string is recognised first, and its own escape question
			// does not arise: the generic loop tests the byte in front of where
			// the opening token left the cursor, which for a raw string is a
			// hash or the r and never a backslash, so it never declines one.
			if endQuote, ok := rustRawStringEnd(content, i, floor); ok {
				return i, SString, endQuote, true
			}

			// The plain quote, which the generic loop declines when a backslash
			// sits in front of it, counting the byte rather than the run.
			if byteBefore(content, i, floor) != '\\' {
				return i, SString, rustPlainQuote, false
			}

			return i, SCode, nil, false
		case '\'':
			// The byte string of b', which is a quote the language database
			// does carry, so it opens a string in both loops. The cursor lands
			// on the quote, which is where prepareString leaves it.
			if byteBefore(content, i, floor) == 'b' {
				return i, SString, rustCharQuote, false
			}

			// A character literal, which the database cannot carry. Skipping it
			// is the one place this counter is deliberately better than the
			// generic loop. Anything else is a lifetime and is ordinary code.
			if end := rustCharLiteralEnd(content, i, endPoint); end >= 0 {
				i = end
			}
		case '?':
			if rustPostfixCounts(content, i) {
				tally.Complexity++
			}
		default:
			if rustComplexityAnchored(content, i, floor) {
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

// countLoopRust stands in for countLoopGeneric where the language is Rust. It
// returns false when it ended the count early, the same way the generic loop
// does.
func countLoopRust(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	stop := rustStopTable()
	floor := bomSkip
	lastByte := int(fileJob.Bytes) - 1

	var tally counterTally

	// What closes the string the scan is in. Rust's raw forms close with a
	// quote and up to eight hashes, so this is not always one byte.
	endQuote := rustPlainQuote

	// Whether that string ignores escapes, which every raw form does and the
	// plain quote and b' do not.
	raw := false

	// How deep the block comment the scan is inside runs. Rust nests them, so a
	// comment left open at the end of a line is open to a depth the next line
	// has to know; the generic loop keeps the same count in endComments.
	commentDepth := 1

	step := func(index int, state counterState) (int, counterState) {
		switch state {
		case SCode:
			index, state, quote, isRaw := rustCodeState(content, &tally, index, endPoint, floor, stop)
			if quote != nil {
				endQuote, raw = quote, isRaw
			}
			if state == SMulticommentCode {
				commentDepth = 1
			}

			return index, state
		case SString:
			return counterStringState(content, index, endPoint, floor, endQuote, raw)
		case SComment, SCommentCode:
			// Nothing inside a line comment can change the state, so the rest of
			// the line is of no interest and IndexByte finds where it ends a
			// vector at a time rather than a byte.
			if next := bytesIndexNewline(content[index:]); next >= 0 {
				return index + next, state
			}

			return lastByte, state
		case SMulticomment, SMulticommentCode:
			// Rust nests its block comments, so the closer that ends this one is
			// the one that brings the depth back to zero.
			index, state, commentDepth = counterNestedCommentState(content, index, endPoint, state, slashStarOpen, slashStarClose, commentDepth)

			return index, state
		default: // SBlank and SMulticommentBlank
			index, state, quote, isRaw := rustBlankState(content, &tally, index, endPoint, floor)
			if quote != nil {
				endQuote, raw = quote, isRaw
			}
			if state == SMulticomment {
				commentDepth = 1
			}

			return index, state
		}
	}

	// Rust does not splice lines, so a line hands its state on unchanged.
	return countLoopShared(fileJob, &tally, bomSkip, endPoint, spliceRule{}, step)
}

// rustQuotes is every quote of Rust as languages.json spells it, start and end
// in turn, which the structural conformance test holds the database to. Built
// rather than written out, since eighteen of the twenty differ only in how many
// hashes they carry.
func rustQuotes() []string {
	quotes := make([]string, 0, 40)
	for _, prefix := range []string{"r", "br"} {
		for hashes := range maxRustRawHashes + 1 {
			start := prefix
			end := `"`
			for range hashes {
				start += "#"
				end += "#"
			}
			quotes = append(quotes, start+`"`, end)
		}
	}

	return append(quotes, `b'`, `'`, `"`, `"`)
}
