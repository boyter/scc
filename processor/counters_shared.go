// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"fmt"
	"io"
	"sort"
)

// The half of a specialised counter that does not change from one language to
// the next.
//
// A counter is two pieces. The hot one is the switch over a [256]bool stop
// table and the complexity matcher behind it, which is written by hand for each
// language because that is where the per-language reasoning lives. Everything
// around it — the runs of whitespace, the end-of-line accounting, the splice,
// the binary bail, the string and block comment bodies — is the same shape for
// every language and lives here. The states below are entered per token rather
// than per byte, so an argument more or less costs nothing measurable.
//
// Everything here must agree with countLoopGeneric to the line. Where the two
// differ the generic one is right by definition, since it is what the rest of
// the tests are written against and a counter is only ever an accelerator.

// counterState is the state the scan is in. The constants are the generic
// loop's, under a name that says a counter carries the state across a range
// rather than over a whole file.
type counterState = int64

// counterTally is what a counter produces. Counting into one of these rather
// than into the FileJob is what lets a range of one language sitting inside
// another be attributed to the language that owns it, and it keeps the four
// counters the loop increments in registers rather than behind a pointer.
//
// It is the whole of what a counter is allowed to produce. FileJob.ComplexityLine
// is deliberately not here and no counter bumps it: specialisedCounterEligible
// declines any file that asked for per line complexity, so the slice is always
// empty on this path and there is nothing to bump. A counter that wanted to
// produce more would have to be let past the guard first, and the guard has a
// test of its own now.
type counterTally struct {
	Lines      int64
	Code       int64
	Comment    int64
	Blank      int64
	Complexity int64
	Binary     bool
}

// addTo folds a tally into the file it was counted from. Every path out of a
// counter goes through it, including the ones that end the count early, so a
// file cut short reports what was counted before it was.
func (t *counterTally) addTo(fileJob *FileJob) {
	fileJob.Lines += t.Lines
	fileJob.Code += t.Code
	fileJob.Comment += t.Comment
	fileJob.Blank += t.Blank
	fileJob.Complexity += t.Complexity
	if t.Binary {
		fileJob.Binary = true
	}
}

// newlineByte is the separator bytes.Count is handed. Held here rather than
// written at the call site so nothing has to reason about whether the literal
// escapes.
var newlineByte = []byte{'\n'}

// The block comment delimiters of the C family, which is every language counted
// today and most of the sixteen. Held as slices so no call site converts one.
var (
	slashStarOpen  = []byte("/*")
	slashStarClose = []byte("*/")
)

// specialisedCounterEligible reports whether a counter can answer for this file
// at all. A counter produces the four line counts, the complexity count and the
// binary marker and nothing else, so anything that asks for more is left to the
// generic loop. Each of these is a thing the generic loop does inside its own
// state functions that no counter replicates:
//
//	Duplicates            folds every byte it walks past into the file hash
//	Cognitive             carries an indent stack and weights each check by it
//	Trace                 logs what every line was counted as
//	NoLarge               truncates the file and frees its content part way
//	ClassifyContent       records a type for every byte
//	TrackComplexityLines  appends a per-line complexity count
//	Callback              is called for every line, which --history uses
//
// The language is not tested here. That is the dispatch's job.
func specialisedCounterEligible(fileJob *FileJob) bool {
	return SpecialisedCounters &&
		!Duplicates &&
		!Cognitive &&
		!Trace &&
		!NoLarge &&
		!fileJob.ClassifyContent &&
		!fileJob.TrackComplexityLines &&
		fileJob.Callback == nil
}

// hasPrefixAt reports whether prefix sits at index, which is the form every
// anchored match is written with. One test covers the bounds of the whole
// comparison, and the compiler then does the comparison a word at a time rather
// than a byte.
//
// index is allowed to be below floor, since a match read back from its anchor
// can ask about bytes in front of the region the counter owns, and the answer
// for those is no. floor subsumes the test against zero, being never negative.
func hasPrefixAt(content []byte, index, floor int, prefix string) bool {
	if index < floor || index+len(prefix) > len(content) {
		return false
	}

	return string(content[index:index+len(prefix)]) == prefix
}

// wordStartsAt reports whether a complexity check could begin at index, which is
// that the byte in front of it does not carry a word on.
//
// The generic loop matches the check first and applies this after, then steps
// over the token either way. Testing it first is the same thing done in the
// cheaper order: no check holds the opening of another, or a quote, or a slash,
// or a newline, so the bytes the generic loop steps over hold nothing that would
// have been read had it not. It is worth the argument because the letters the
// checks are spelled with are nearly always in the middle of an identifier, and
// this is what keeps the match from being run on every one of them.
//
// floor is the first byte of the region the counter owns. Nothing carries a word
// into it, whether it is the start of the file or the start of a script tag, so
// the floor is a word boundary.
func wordStartsAt(content []byte, index, floor int) bool {
	return index <= floor || !isIdentifierContinue(content[index-1])
}

// byteBefore returns the byte in front of index, or zero where index sits on the
// floor and there is nothing in front of it to read. Zero is not a byte any
// check is spelled with, so a caller comparing it against a letter reads the
// absence of a byte as the mismatch it is.
//
// Every backwards read of every counter goes through this or through
// wordStartsAt. Reading content[index-1] raw is safe only while the reduced
// check set of the line start covers the checks anchored on their own first
// byte, which is an invariant of one language's counter and not of the machinery
// around it, and scc must not crash on a file whatever shape it is.
func byteBefore(content []byte, index, floor int) byte {
	if index <= floor {
		return 0
	}

	return content[index-1]
}

// cOpens reports whether a byte closes the keyword of a complexity check that is
// normally written with a bracket behind it. Every such keyword is written twice
// in languages.json, once with a space and once with the bracket.
func cOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == '('
}

// braceOpens is cOpens for the keywords written with a brace behind them rather
// than a bracket, which is else across the C family and try and finally in Java.
func braceOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == '{'
}

// spaceOpens reports whether a byte closes a keyword that languages.json spells
// with a space behind it and nothing else. JavaScript and C# write switch and
// while that way, where the rest of the family writes them twice, once with a
// space and once with the bracket. Using cOpens for those would count a
// switch( the language database does not carry.
func spaceOpens(content []byte, index int) bool {
	return index < len(content) && content[index] == ' '
}

// bytesIndexNewline is bytes.IndexByte under a name that says what it is for.
// It is written in assembly for every architecture scc is built for, comparing a
// vector of bytes at a time, which is what makes skipping a run worth doing
// rather than walking it.
func bytesIndexNewline(content []byte) int {
	return bytes.IndexByte(content, '\n')
}

// skipBlankRun returns the first byte that is not a space, a tab or a carriage
// return, given an index that is one of those, or endPoint where the run reaches
// it. That is the byte the loop has something to say about, and it is what the
// generic loop moves to.
//
// This is tuning 7, and the generic loop has had it for some time while
// neither counter did: leading indentation is a fifth of the bytes of a Java
// file and a sixth of a Python one, and walking it a byte at a time costs the
// whole of the loop body for each of them.
//
// The newline is deliberately not in isBlankRun, because a line ends on it and
// that is the one piece of whitespace the loop has to stop for.
func skipBlankRun(content []byte, index, endPoint int) int {
	next := index + 1
	for next < endPoint && isBlankRun[content[next]] {
		next++
	}

	return next
}

// skipToTerminator finds closer in content[i:] and counts the newlines in front
// of it, both a vector at a time rather than a byte at a time. end is the index
// of the first byte of closer, or -1 where it is not there at all, in which case
// newlines is counted over the whole of content[i:].
//
// This is tuning 8, and it is what lets any region whose only exits are a fixed
// terminator and the end of the file be jumped over rather than walked. A fifth
// of a C file sits inside a block comment and a fifth of a Python one inside a
// triple quote.
//
// It deliberately does not look for a nul. isBinary is called only from the code
// state, in both loops, so neither of them marks a file binary on a nul inside a
// comment or a string. Looking for one here reads like an improvement and would
// break conformance with the generic loop.
func skipToTerminator(content []byte, i int, closer []byte) (end, newlines int) {
	rel := bytes.Index(content[i:], closer)
	if rel < 0 {
		return -1, bytes.Count(content[i:], newlineByte)
	}

	return i + rel, bytes.Count(content[i:i+rel], newlineByte)
}

// bulkLines accounts for n whole lines that all sit inside a region the state
// does not change across, and returns the state the last of them left behind.
//
// The line accounting is exactly what the outer loop would have done had it
// stopped on each of those newlines, minus the splice, which cannot matter here:
// resetLineState differs from resetState only for a line comment and a string,
// and a region that is jumped over in bulk is neither. Only the first line is a
// special case, since resetState is idempotent from the second one on: a block
// comment opened after code on the line ends that one line as code and every
// line under it as comment.
//
// A blank line inside a block comment is never counted blank. The state there is
// SMulticomment, which resets to itself, so the line counts as comment; the
// blank-looking SMulticommentBlank is the closing line of a comment rather than
// an empty line inside one.
func bulkLines(tally *counterTally, state counterState, n int64) counterState {
	if n <= 0 {
		return state
	}

	tally.Lines += n
	if isCodeLineState(state) {
		tally.Code++
	} else {
		tally.Comment++
	}

	state = resetState(state)
	if n--; n > 0 {
		if isCodeLineState(state) {
			tally.Code += n
		} else {
			tally.Comment += n
		}
	}

	return state
}

// isCodeLineState reports whether a line ending in this state counts as code,
// which is the same split the end-of-line switch of every loop is written with.
// SBlank is not among them and cannot reach here, a blank line being the one
// thing no region is jumped over in.
func isCodeLineState(state counterState) bool {
	switch state {
	case SCode, SString, SCommentCode, SMulticommentCode:
		return true
	}

	return false
}

// counterStringState runs to the closing quote, which a run of escapes of odd
// length in front of it does not count as. A raw quote has no escape mechanism
// at all and passes ignoreEscape, which is what stops a backslash at the end of
// one ending it.
//
// endQuote is a slice so a language whose quote closes with more than one byte
// is covered, but the single byte case is every quote of C and Java and is the
// one worth keeping cheap, so the first byte is compared before anything else.
func counterStringState(content []byte, index, endPoint, floor int, endQuote []byte, ignoreEscape bool) (int, counterState) {
	first := endQuote[0]
	single := len(endQuote) == 1

	for i := index; i < endPoint; i++ {
		index = i

		if content[i] == '\n' {
			return i, SString
		}

		if content[i] != first {
			continue
		}

		if !ignoreEscape && escapedAt(content, i, floor) {
			continue
		}

		if single {
			return i, SCode
		}

		if i+len(endQuote) <= endPoint && string(content[i:i+len(endQuote)]) == string(endQuote) {
			// Step past the whole terminator. For a multi byte one such as the
			// C++ raw string )" the trailing byte is itself a quote start, so
			// leaving the cursor on it would open a new string. See #175.
			return i + len(endQuote) - 1, SCode
		}
	}

	return index, SString
}

// escapedAt reports whether the byte at index is escaped, which is that the run
// of backslashes in front of it is of odd length. An even run is a run of
// backslashes that escape each other and leave the byte alone.
//
// The escape is the backslash and is not read from the language, because none
// of the sixteen escapes with anything else. PowerShell escapes with a backtick
// and has no counter; a language like it would have to thread langFeatures.Escape
// through to here.
//
// The generic loop counts the run the same way and stops at index 1 rather than
// at 0, which makes a file opening with a backslash come out one escape short.
// It is reproduced rather than fixed, since a counter that is right where the
// generic loop is wrong is a counter that disagrees with it.
func escapedAt(content []byte, index, floor int) bool {
	if index <= floor || content[index-1] != '\\' {
		return false
	}

	escapes := 0
	for j := index - 1; j > floor; j-- {
		if content[j] != '\\' {
			break
		}
		escapes++
	}

	return escapes%2 != 0
}

// counterCommentState runs a block comment to its closer, jumping over the body
// rather than walking it, and accounts for every whole line it jumped over.
//
// It returns the last byte it consumed and the state that byte left behind, the
// same contract every state of every loop is written to. Where nothing closes
// the comment it hands the last newline of the file back to the outer loop
// rather than swallowing it, so the file's trailing line is worked out in the
// one place that knows how.
//
// A language whose comments nest wants counterNestedCommentState instead. This
// one jumps straight to the first closer, which is only the right closer when
// nothing between here and there can open another comment.
func counterCommentState(content []byte, index, endPoint int, state counterState, closer []byte, tally *counterTally) (int, counterState) {
	// Nothing is left to scan, which is the shape the outer loop hands back on
	// the last byte of a file.
	if index >= endPoint {
		return index, state
	}

	// Everything the counters do is bounded by endPoint, which is the byte
	// before the last one of the file. A closer whose last byte sits on that
	// final byte is not seen by the generic loop either.
	region := content[:endPoint]

	i := index

	closeAt, newlines := skipToTerminator(region, i, closer)
	if closeAt < 0 {
		return commentRunsOut(content, index, endPoint, state, newlines, tally)
	}

	state = bulkLines(tally, state, int64(newlines))

	// Only a comment that opened and closed on the one line can still be holding
	// the state that says there was code in front of it.
	if state == SMulticommentCode {
		return closeAt + len(closer) - 1, SCode
	}

	return closeAt + len(closer) - 1, SMulticommentBlank
}

// counterNestedCommentState runs a block comment for a language whose comments
// count their own openers, which is Rust, Swift, Kotlin and Scala among the
// sixteen. It takes the nearer of the next opener and the next closer, so
// /* a /* b */ is still open and needs a second closer.
//
// It carries depth in and out, because a nested comment left open at the end of
// a line is open to a depth the next line has to know. The generic loop keeps
// the same count in endComments and hands it round the outer loop the same way.
//
// It stops at the newline rather than jumping the whole comment the way
// counterCommentState does. A depth is only meaningful at a position, so a scan
// that hopped several openers and then ran out of file cannot hand back both
// the newline it should stop on and the depth it had reached there: it would
// have to report a depth from further ahead, and the tokens it had already
// passed would be read a second time. Stopping on the line keeps the two in
// step. The search within the line is still two vector scans rather than a byte
// loop, which is what spec 07 03-architecture §4.2.2 asks for.
func counterNestedCommentState(content []byte, index, endPoint int, state counterState, opener, closer []byte, depth int) (int, counterState, int) {
	// Nothing is left to scan, which is the shape the outer loop hands back on
	// the last byte of a file.
	if index >= endPoint {
		return index, state, depth
	}

	// Bound the scan to this line. The outer loop counts the line that ends on
	// the newline, so the state must hand it back rather than swallow it.
	limit := endPoint
	if next := bytesIndexNewline(content[index:endPoint]); next >= 0 {
		limit = index + next
	}

	region := content[:limit]
	i := index

	for {
		closeAt, _ := skipToTerminator(region, i, closer)
		openAt, _ := skipToTerminator(region, i, opener)

		// The nearer of the two wins, and on a tie the closer does, which is the
		// order the generic loop tests them in.
		if openAt >= 0 && (closeAt < 0 || openAt < closeAt) {
			depth++
			i = openAt + len(opener)

			continue
		}

		if closeAt < 0 {
			break
		}

		depth--
		if depth == 0 {
			// Only a comment that opened and closed on the one line can still be
			// holding the state that says there was code in front of it.
			if state == SMulticommentCode {
				return closeAt + len(closer) - 1, SCode, 0
			}

			return closeAt + len(closer) - 1, SMulticommentBlank, 0
		}

		i = closeAt + len(closer)
	}

	// Nothing closed it on this line, so hand back the newline and the depth it
	// is still open to.
	if limit < endPoint {
		return limit, state, depth
	}

	// No newline either, so the scan reached the end of the file. Saying so, the
	// way the other states do, is what stops the outer loop stepping on one byte
	// and handing the whole remaining tail back to be scanned again.
	if index < endPoint {
		return endPoint - 1, state, depth
	}

	return index, state, depth
}

// commentRunsOut is counterCommentState where nothing closes the comment before
// the end of the file. Every line but the last is accounted for here and the
// last newline is handed back, which is where the byte at a time version
// returned on each of them.
func commentRunsOut(content []byte, index, endPoint int, state counterState, newlines int, tally *counterTally) (int, counterState) {
	if newlines == 0 {
		// Saying the scan reached the end, the way the other states do, is what
		// stops the outer loop stepping on one byte and handing the whole
		// remaining tail back to be scanned again: an unterminated block comment
		// with no newline in it took time in the square of its length, eleven
		// seconds for 250KB and three minutes for a megabyte.
		if index < endPoint {
			return endPoint - 1, state
		}

		return index, state
	}

	state = bulkLines(tally, state, int64(newlines-1))

	return index + bytes.LastIndexByte(content[index:endPoint], '\n'), state
}

// counterStep is the per-language half of a counter. It is handed the byte the
// scan stopped on and the state it is in, and runs until the state changes or
// the line ends, returning the last byte it consumed. Everything around it is
// countLoopShared.
type counterStep func(index int, state counterState) (int, counterState)

// countLoopShared is the outer loop every counter runs under. It is the generic
// loop with the parts no counter supports taken out: there is no callback, no
// trace, no cognitive indent stack, no byte classification and no large file
// truncation, because specialisedCounterEligible declined the file if any of
// them was asked for.
//
// It reports whether it ran to the end of the file, the same way the generic
// loop does. A binary marker or a state leaving the index past the end both end
// the count there.
func countLoopShared(fileJob *FileJob, tally *counterTally, bomSkip, endPoint int, splice spliceRule, step counterStep) bool {
	content := fileJob.Content
	total := int(fileJob.Bytes)
	state := counterState(SBlank)

	for index := bomSkip; index < total; index++ {
		curByte := content[index]

		if index < endPoint && isBlankRun[curByte] {
			index = skipBlankRun(content, index, endPoint)
			curByte = content[index]
		}

		if !isWhitespace(curByte) {
			index, state = step(index, state)

			// Only a state above moves the index or marks the file binary, so
			// both of the checks that follow belong here rather than on the
			// whitespace the loop walked over to get to one.
			if index >= len(content) {
				tally.addTo(fileJob)

				return false
			}

			if index < 10000 && tally.Binary {
				tally.addTo(fileJob)

				return false
			}

			curByte = content[index]
		}

		if curByte == '\n' || index >= endPoint {
			tally.Lines++

			switch state {
			case SCode, SString, SCommentCode, SMulticommentCode:
				tally.Code++
				state = resetCounterLineState(content, index, state, splice)
			case SComment, SMulticomment, SMulticommentBlank:
				tally.Comment++
				state = resetCounterLineState(content, index, state, splice)
			case SBlank:
				tally.Blank++
			}
		}
	}

	tally.addTo(fileJob)

	return true
}

// spliceRule says how a line hands its state to the line under it.
//
// A language that splices joins a line ending in a backslash to the next one
// before it looks for a comment or a string, which carries a line comment on and
// ends a string that is not carried. That second half is the trap: a raw string
// has no escape mechanism at all, so a backslash at the end of one is an
// ordinary byte and the string runs on whether or not it looks spliced. Ending
// it at the newline would break every C++ raw string that spans lines.
type spliceRule struct {
	// Splices marks a language whose backslash at the end of a line joins it to
	// the next, which among the counted languages is only the C family.
	Splices bool
	// InRawString reports whether the string the scan is currently inside was
	// opened by a quote with no escape mechanism. It is read at the end of every
	// line, so the counter that owns it writes it when it opens a string and
	// leaves it alone otherwise. nil for a language with no raw quote, which is
	// every splicing language except C++ and C++ Header.
	InRawString *bool
}

// resetCounterLineState hands the state at the end of a line to the line under
// it. See spliceRule, and spec 07 03-architecture §7.1.
func resetCounterLineState(content []byte, index int, state counterState, splice spliceRule) counterState {
	if !splice.Splices {
		return resetState(state)
	}

	ignoreEscape := false
	if splice.InRawString != nil {
		ignoreEscape = *splice.InRawString
	}

	return resetLineState(state, endsWithLineSplice(content, index), ignoreEscape)
}

// counterFn is a counter's entry point, the shape countLoopGeneric has minus
// the language features it no longer needs to be handed. It reports whether it
// ran to the end of the file.
type counterFn func(fileJob *FileJob, bomSkip, endPoint int) bool

// counterSpec is what a counter declares about itself so it can be held against
// languages.json with no corpus, no env var and no file counted. It is the
// mechanism that makes hand-writing a counter safe: edit languages.json without
// touching the counter and go test fails, which is the failure a differential
// test behind an env var will not catch.
//
// See spec 07 04-testing §2 for the four things the test asserts with it.
type counterSpec struct {
	// Language is the languages.json name the counter answers for.
	Language string
	// Count is the counter itself, which the dispatch resolves to.
	Count counterFn
	// Extension is what a corpus of this language is sampled by, which the
	// anchor measurement walks a tree for. It is not how a file's language is
	// decided; that is detection's job and it is far cleverer than this.
	Extension string
	// Anchors maps every complexity check the counter handles to the byte the
	// scan stops on for it, which for an anchored check is the rarest byte of
	// the check rather than its first.
	Anchors map[string]byte
	// QuoteAnchors maps a quote's start token to the byte the counter stops on
	// for it, where that is not its first byte. Anchoring is usually described
	// as a trick for keywords, but a quote whose opening token is several bytes
	// can be caught on a later one just as well: C++ stops on the " that ends
	// R" and u8R" and reads the prefix backwards, which keeps R, u, U and L out
	// of a table they would cost 3.32% of a C++ file to sit in. Empty for a
	// language whose quotes are all found at their first byte.
	QuoteAnchors map[string]byte
	// LineComments, BlockComments and Quotes are the rest of what the counter
	// handles, spelled exactly as languages.json spells them.
	LineComments  []string
	BlockComments [][]string
	Quotes        []string
	// Stop is the table the scan runs with, and StopNoComplexity the smaller
	// one it runs with under --no-complexity.
	Stop             *[256]bool
	StopNoComplexity *[256]bool
	// Collisions is the bytes this language's complexity checks are spelled
	// with that also open or close a quote, a line comment or a block comment,
	// sorted and with no repeats. Those are the bytes a backwards read from an
	// anchor could cross, so where this is not empty the counter has to carry
	// the argument for why the read is still sound. Thirteen of the sixteen
	// languages are empty; Python is "fr", Ruby is "=ei" and Rust is "r".
	Collisions string
}

// counterSpecs is every counter there is, and the only place a new one is
// registered. The dispatch, the conformance test, the fuzz target, the bounds
// walk, the anchor measurement and the flag help all derive from it, so adding a
// language is one entry here rather than five lists that have to agree.
func counterSpecs() []counterSpec {
	cComments := []string{"//"}
	cBlocks := [][]string{{"/*", "*/"}}

	return []counterSpec{
		{
			Language:         "C",
			Count:            func(f *FileJob, b, e int) bool { return countLoopC(f, b, e, false) },
			Extension:        ".c",
			Anchors:          cComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`},
			Stop:             &cStop,
			StopNoComplexity: &cStopNoComplexity,
		},
		{
			Language:         "C Header",
			Count:            func(f *FileJob, b, e int) bool { return countLoopC(f, b, e, true) },
			Extension:        ".h",
			Anchors:          cHeaderComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`},
			Stop:             &cHeaderStop,
			StopNoComplexity: &cStopNoComplexity,
		},
		{
			Language:         "Java",
			Count:            countLoopJava,
			Extension:        ".java",
			Anchors:          javaComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `'`, `'`},
			Stop:             &javaStop,
			StopNoComplexity: &javaStopNoComplexity,
		},
		{
			Language:         "C#",
			Count:            countLoopCsharp,
			Extension:        ".cs",
			Anchors:          csharpComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`@"`, `"`, `"`, `"`, `'`, `'`},
			Stop:             &csharpStop,
			StopNoComplexity: &csharpStopNoComplexity,
		},
		{
			Language:         "Go",
			Count:            countLoopGo,
			Extension:        ".go",
			Anchors:          goComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, "`", "`", `'`, `'`},
			Stop:             &goStop,
			StopNoComplexity: &goStopNoComplexity,
		},
		{
			Language:         "PHP",
			Count:            countLoopPHP,
			Extension:        ".php",
			Anchors:          phpComplexityAnchors,
			LineComments:     []string{"#", "//"},
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `'`, `'`},
			Stop:             &phpStop,
			StopNoComplexity: &phpStopNoComplexity,
		},
		{
			Language:         "TypeScript",
			Count:            countLoopTypeScript,
			Extension:        ".ts",
			Anchors:          tsComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `'`, `'`, "`", "`"},
			Stop:             &tsStop,
			StopNoComplexity: &tsStopNoComplexity,
		},
		{
			Language:         "C++",
			Count:            countLoopCpp,
			Extension:        ".cpp",
			Anchors:          cppComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `R"`, `)"`, `u8R"`, `)"`, `uR"`, `)"`, `UR"`, `)"`, `LR"`, `)"`},
			QuoteAnchors:     cppQuoteAnchors,
			Stop:             &cppStop,
			StopNoComplexity: &cppStopNoComplexity,
		},
		{
			Language:         "C++ Header",
			Count:            countLoopCpp,
			Extension:        ".hpp",
			Anchors:          cppComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `R"`, `)"`, `u8R"`, `)"`, `uR"`, `)"`, `UR"`, `)"`, `LR"`, `)"`},
			QuoteAnchors:     cppQuoteAnchors,
			Stop:             &cppStop,
			StopNoComplexity: &cppStopNoComplexity,
		},
		{
			Language:         "JavaScript",
			Count:            countLoopJavaScript,
			Extension:        ".js",
			Anchors:          jsComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`, `'`, `'`, "`", "`"},
			Stop:             &jsStop,
			StopNoComplexity: &jsStopNoComplexity,
		},
		{
			Language:         "Kotlin",
			Count:            countLoopKotlin,
			Extension:        ".kt",
			Anchors:          kotlinComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`},
			Stop:             &kotlinStop,
			StopNoComplexity: &kotlinStopNoComplexity,
		},
		{
			Language:         "Scala",
			Count:            countLoopScala,
			Extension:        ".scala",
			Anchors:          scalaComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`},
			Stop:             &scalaStop,
			StopNoComplexity: &scalaStopNoComplexity,
		},
		{
			Language:         "Swift",
			Count:            countLoopSwift,
			Extension:        ".swift",
			Anchors:          swiftComplexityAnchors,
			LineComments:     cComments,
			BlockComments:    cBlocks,
			Quotes:           []string{`"`, `"`},
			Stop:             &swiftStop,
			StopNoComplexity: &swiftStopNoComplexity,
		},
	}
}

// counterDivergence records a case where a specialised counter deliberately
// disagrees with the generic loop, because the generic loop is wrong and the
// fix needs reasoning a trie cannot carry.
//
// Exact agreement with countLoopGeneric is what makes the differential test and
// the fuzzer worth anything, so the exception is a closed set: a divergence
// that is not on this list fails the build. Each entry carries a LineJudge case
// id and a fixture that pins both answers, so the difference is asserted rather
// than merely tolerated, and the list shrinks as fixes turn out to be
// expressible in languages.json after all.
//
// See spec 07 01-conformance.md §4. At run time there is no such concept, only
// a counter that counts correctly; the differential test and the fuzz oracle
// are the only readers.
type counterDivergence struct {
	// Language is the languages.json name of the counter that diverges.
	Language string
	// Case is the LineJudge case id, which is what makes the divergence a named
	// one rather than a counter being "a bit different".
	Case string
	// Fixture is the path under examples/linejudge/ holding the input.
	Fixture string
	// Reason is one line, present tense, saying what the counter does instead.
	Reason string
}

// counterDivergences is every deliberate disagreement there is.
//
// The fixtures under examples/linejudge/ are reconstructions written from the
// case descriptions in spec 07 01-conformance.md, not the suite's own files:
// LineJudge is not checked out here. They reproduce the counts the spec records
// for each case, which is what makes them useful for pinning the behaviour, but
// a claim about the recorded suite score wants the real suite run against it.
func counterDivergences() []counterDivergence {
	return []counterDivergence{
		{
			Language: "JavaScript",
			Case:     "7010-regex_literal_holding_a_quote",
			Fixture:  "examples/linejudge/7010-regex_literal_holding_a_quote.js",
			Reason:   "a quote inside a regular expression literal opens no string, where the generic loop reads it as opening one that never closes",
		},
		{
			Language: "JavaScript",
			Case:     "7020-regex_holding_a_comment_opener",
			Fixture:  "examples/linejudge/7020-regex_holding_a_comment_opener.js",
			Reason:   "a slash pair inside a regular expression literal opens no comment, where the generic loop reads it as opening a line comment",
		},
		{
			Language: "TypeScript",
			Case:     "7010-regex_literal_holding_a_quote",
			Fixture:  "examples/linejudge/7010-regex_literal_holding_a_quote.ts",
			Reason:   "a quote inside a regular expression literal opens no string, where the generic loop reads it as opening one that never closes",
		},
		{
			Language: "TypeScript",
			Case:     "7020-regex_holding_a_comment_opener",
			Fixture:  "examples/linejudge/7020-regex_holding_a_comment_opener.ts",
			Reason:   "a slash pair inside a regular expression literal opens no comment, where the generic loop reads it as opening a line comment",
		},
	}
}

// counterDispatch resolves a language to its counter once, rather than walking a
// chain of predicates per file. Built from counterSpecs at startup, which is
// safe: --count-as remapping and the (gen) and (min) suffixes all leave Language
// an arbitrary string, and the suffixes are applied after counting, so a lookup
// that misses simply falls through to the generic loop.
var counterDispatch = buildCounterDispatch()

func buildCounterDispatch() map[string]counterFn {
	dispatch := make(map[string]counterFn, len(counterSpecs()))
	for _, spec := range counterSpecs() {
		dispatch[spec.Language] = spec.Count
	}

	return dispatch
}

// counterFor returns the counter that may answer for this file, or nil where
// the generic loop has to. One map lookup per file, behind the guard.
func counterFor(fileJob *FileJob) counterFn {
	if !specialisedCounterEligible(fileJob) {
		return nil
	}

	return counterDispatch[fileJob.Language]
}

// counterLanguages is every language with a counter, sorted, which is what the
// tests walk and what --list-counters prints.
func counterLanguages() []string {
	languages := make([]string, 0, len(counterSpecs()))
	for _, spec := range counterSpecs() {
		languages = append(languages, spec.Language)
	}
	sort.Strings(languages)

	return languages
}

// PrintCounters writes the languages that have a counter of their own, which is
// what --list-counters asks for. Derived from the one registry, so it cannot
// fall out of step with what the dispatch actually resolves.
func PrintCounters(w io.Writer) {
	languages := counterLanguages()

	fmt.Fprintf(w, "%d of %d languages have a scanner written for them, used with --exp-per-language-counters:\n\n", len(languages), len(languageDatabase))
	for _, language := range languages {
		fmt.Fprintf(w, "  %s\n", language)
	}
	fmt.Fprintln(w, "\nEvery other language is counted by the generic loop, which is also what\nthese are held to: a counter that disagrees with it is a bug in the counter.")
}
