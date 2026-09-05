// SPDX-License-Identifier: MIT

package processor

// A counter written for C and C Header, on the same terms as the Java one: it
// must agree with the generic loop to the line, and where the two differ the
// generic one is right by definition.
//
// C asks for two things Java does not. Lines joined by a backslash carry a line
// comment on into the next line and end a string that is not carried, which is
// the linesplice flag in languages.json. And C has no character literal in its
// table, so a quote inside one opens a string, which is wrong but is what the
// generic loop does and so is what this has to do too.

// cStop marks every byte the scan stops on for C: the slash of both comment
// forms, the one quote, one anchor byte out of every complexity check, and the
// newline and null that end a line and a file of bytes rather than text.
//
// The anchor is not the first byte of the check but the rarest byte in it,
// which is what keeps the scan from stopping on a fifth of the file. Counted
// over 2.5MB of the kernel, stopping on the first byte of every check costs
// 21.5% of all bytes; stopping on the anchors costs 10.5%. The saving is nearly
// all of it e, i and s, which open if, switch and else and are three of the
// five commonest bytes in C.
var cStop = buildCStop(false)

// cHeaderStop is cStop with the c of case, which C Header counts and C does
// not.
var cHeaderStop = buildCStop(true)

// cStopNoComplexity holds only what changes the state, which is what a file
// counted with --no-complexity is scanned with.
var cStopNoComplexity = buildCStopNoComplexity()

// buildCStop marks the anchor of every complexity check of C.
//
//	f   if, for      the f of if is read backwards, the f of for forwards
//	w   while, switch
//	l   else
//	c   case, which C Header counts and C does not
//	|   ||
//	&   &&
//	=   ==, and the = of != read backwards, so ! is not needed at all
func buildCStop(withCase bool) [256]bool {
	table := buildCStopNoComplexity()
	for _, b := range []byte{'f', 'w', 'l', '|', '&', '='} {
		table[b] = true
	}
	if withCase {
		table['c'] = true
	}

	return table
}

func buildCStopNoComplexity() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\n', 0} {
		table[b] = true
	}

	return table
}

// cStopTable picks the table the scan runs with. The global reads as complexity
// having been turned off.
func cStopTable(withCase bool) *[256]bool {
	if Complexity {
		return &cStopNoComplexity
	}
	if withCase {
		return &cHeaderStop
	}

	return &cStop
}

// hasPrefixAt reports whether prefix sits at index, which is the form every
// anchored match is written with. One test covers the bounds of the whole
// comparison, and the compiler then does the comparison a word at a time rather
// than a byte. index is allowed to be negative, since a match read back from
// its anchor can ask about bytes in front of the file.
func hasPrefixAt(content []byte, index int, prefix string) bool {
	if index < 0 || index+len(prefix) > len(content) {
		return false
	}

	return string(content[index:index+len(prefix)]) == prefix
}

// cOpens reports whether a byte closes the keyword of a complexity check. Every
// keyword of C is written twice in the table, once with a space behind it and
// once with the bracket or brace that C is normally written with instead.
func cOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == '('
}

// cComplexityAnchored reports whether a complexity check of C sits on the
// anchor byte at index, which is what cStop stopped the scan on.
//
// Where the anchor is not the first byte of the check the bytes in front of it
// are read back, and the word boundary is tested at the front of the check
// rather than at the anchor. A check can never begin before a quote, a slash or
// a newline, since none of those is a byte any check is spelled with, so
// reading back never crosses out of the code the scan is in.
//
// The pairs are told apart on the byte behind the anchor and cannot both match:
// the f of if has an i behind it and the f of for cannot, i being a byte that
// carries a word on, and the same holds of the w of switch against while and
// the = of != against ==.
func cComplexityAnchored(content []byte, index int, withCase bool) bool {
	switch content[index] {
	case 'f':
		if content[index-1] == 'i' {
			return javaWordStarts(content, index-1) && cOpens(content, index+1)
		}

		return javaWordStarts(content, index) &&
			hasPrefixAt(content, index+1, "or") && cOpens(content, index+3)
	case 'l':
		if content[index-1] != 'e' {
			return false
		}

		return javaWordStarts(content, index-1) &&
			hasPrefixAt(content, index+1, "se") && elseOpens(content, index+3)
	case 'w':
		if content[index-1] == 's' {
			return javaWordStarts(content, index-1) &&
				hasPrefixAt(content, index+1, "itch") && cOpens(content, index+5)
		}

		return javaWordStarts(content, index) &&
			hasPrefixAt(content, index+1, "hile") && cOpens(content, index+5)
	case '=':
		if b := content[index-1]; b != '=' && b != '!' {
			return false
		}

		return javaWordStarts(content, index-1) &&
			index+1 < len(content) && content[index+1] == ' '
	case '|':
		return javaWordStarts(content, index) && hasPrefixAt(content, index+1, "| ")
	case '&':
		return javaWordStarts(content, index) && hasPrefixAt(content, index+1, "& ")
	case 'c':
		return withCase && javaWordStarts(content, index) &&
			hasPrefixAt(content, index+1, "ase ")
	}

	return false
}

// elseOpens is cOpens for else, which is written with a brace rather than a
// bracket behind it.
func elseOpens(content []byte, index int) bool {
	if index >= len(content) {
		return false
	}
	b := content[index]

	return b == ' ' || b == '{'
}

// cBlankState looks at the first byte of content on a line.
func cBlankState(fileJob *FileJob, index int, withCase bool) (int, int64) {
	content := fileJob.Content

	switch content[index] {
	case '/':
		if index+1 < len(content) {
			switch content[index+1] {
			case '/':
				return index, SComment
			case '*':
				return index + 1, SMulticomment
			}
		}
	case '"':
		return index, SString
	}

	if !Complexity && cComplexityAtLineStart(content, index, withCase) {
		fileJob.Complexity++
		fileJob.bumpComplexityLine()
	}

	return index, SCode
}

// cComplexityAtLineStart is cComplexityAnchored for the first byte of code on a
// line, which has nothing in front of it to read back to. Only the checks
// anchored on their own first byte are looked for here; the rest are anchored
// on a byte the code scan reaches, since that begins on the byte after this one
// and no check that is anchored on its first byte holds an anchor of its own
// anywhere else.
//
// Nothing carries a word into the first byte of code on a line — whitespace or
// the slash of a closed block comment is all that can sit in front of it — so
// the word boundary needs no test.
func cComplexityAtLineStart(content []byte, index int, withCase bool) bool {
	switch content[index] {
	case 'f':
		return hasPrefixAt(content, index+1, "or") && cOpens(content, index+3)
	case 'w':
		return hasPrefixAt(content, index+1, "hile") && cOpens(content, index+5)
	case '|':
		return hasPrefixAt(content, index+1, "| ")
	case '&':
		return hasPrefixAt(content, index+1, "& ")
	case 'c':
		return withCase && hasPrefixAt(content, index+1, "ase ")
	}

	return false
}

// cCodeState runs to the end of the line or to whatever token takes it out of
// code.
func cCodeState(fileJob *FileJob, index, endPoint int, stop *[256]bool, withCase bool) (int, int64) {
	content := fileJob.Content
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
			return i, SCode
		case 0:
			if isBinary(i, curByte) {
				fileJob.Binary = true
				return i, SCode
			}
		case '/':
			if i+1 < len(content) {
				switch content[i+1] {
				case '/':
					return i, SCommentCode
				case '*':
					return i + 1, SMulticommentCode
				}
			}
		case '"':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing. A quote
			// at the first byte has nothing in front of it and so is not
			// escaped; the state machine cannot reach here at i of zero, having
			// started blank, but the check does not depend on that holding.
			if i == 0 || content[i-1] != '\\' {
				return i, SString
			}

			return i, SCode
		default:
			if cComplexityAnchored(content, i, withCase) {
				fileJob.Complexity++
				fileJob.bumpComplexityLine()
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which is
	// the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode
	}

	return index, SCode
}

// cStringState runs to the closing quote, which a run of backslashes of odd
// length in front of it does not count as.
func cStringState(fileJob *FileJob, index, endPoint int) (int, int64) {
	content := fileJob.Content

	for i := index; i < endPoint; i++ {
		index = i

		if content[i] == '\n' {
			return i, SString
		}

		isEscaped := false
		if i > 0 && content[i-1] == '\\' {
			escapes := 0
			for j := i - 1; j > 0; j-- {
				if content[j] != '\\' {
					break
				}
				escapes++
			}
			if escapes%2 != 0 {
				isEscaped = true
			}
		}

		if !isEscaped && content[i] == '"' {
			return i, SCode
		}
	}

	return index, SString
}

// cCommentState runs to the closer of a block comment, which does not nest in
// C.
func cCommentState(fileJob *FileJob, index, endPoint int, currentState int64) (int, int64) {
	content := fileJob.Content

	for i := index; i < endPoint; i++ {
		if content[i] == '\n' {
			return i, currentState
		}

		if content[i] == '*' && i+1 < endPoint && content[i+1] == '/' {
			if currentState == SMulticommentCode {
				currentState = SCode
			} else {
				currentState = SMulticommentBlank
			}

			return i + 1, currentState
		}
	}

	// Nothing closed the comment and no line ended, so the scan reached the end
	// and the last byte it looked at is the one before endPoint. Saying so, the
	// way the other states do, is what stops the outer loop stepping on one byte
	// and handing the whole remaining tail back to be scanned again: an
	// unterminated block comment with no newline in it took time in the square
	// of its length, eleven seconds for 250KB and three minutes for a megabyte.
	if index < endPoint {
		return endPoint - 1, currentState
	}

	return index, currentState
}

// countLoopC stands in for countLoopGeneric where the language is C or C
// Header. It returns false when it ended the count early, the same way the
// generic loop does.
func countLoopC(fileJob *FileJob, bomSkip, endPoint int, withCase bool) bool {
	content := fileJob.Content
	currentState := SBlank
	stop := cStopTable(withCase)

	for index := bomSkip; index < int(fileJob.Bytes); index++ {
		if !isWhitespace(content[index]) {
			switch currentState {
			case SCode:
				index, currentState = cCodeState(fileJob, index, endPoint, stop, withCase)
			case SString:
				index, currentState = cStringState(fileJob, index, endPoint)
			case SComment, SCommentCode:
				// Nothing inside a line comment can change the state before the
				// newline, so the rest of the line is skipped a vector at a time.
				// Whether it carries on past the newline is the splice, worked out
				// below where the line is counted.
				if next := bytesIndexNewline(content[index:]); next >= 0 {
					index += next
				} else {
					index = int(fileJob.Bytes) - 1
				}
			case SMulticomment, SMulticommentCode:
				index, currentState = cCommentState(fileJob, index, endPoint, currentState)
			case SBlank, SMulticommentBlank:
				index, currentState = cBlankState(fileJob, index, withCase)
			}
		}

		if index >= len(content) {
			return false
		}

		if index < 10000 && fileJob.Binary {
			return false
		}

		if content[index] == '\n' || index >= endPoint {
			fileJob.Lines++

			// C joins a line ending in a backslash to the one under it before it
			// looks for a comment or a string, which carries a line comment on and
			// ends a string that is not carried. No quote of C is a raw one so the
			// escape is never ignored.
			spliced := endsWithLineSplice(content, index)

			switch currentState {
			case SCode, SString, SCommentCode, SMulticommentCode:
				fileJob.Code++
				currentState = resetLineState(currentState, spliced, false)
			case SComment, SMulticomment, SMulticommentBlank:
				fileJob.Comment++
				currentState = resetLineState(currentState, spliced, false)
			case SBlank:
				fileJob.Blank++
			}
		}
	}

	return true
}

// useCCounter reports whether the C counter can answer for this file. It
// produces the four line counts and the complexity count and nothing else, so
// anything that asks for more is left to the generic loop.
func useCCounter(fileJob *FileJob) bool {
	if !SpecialisedCounters {
		return false
	}

	if fileJob.Language != "C" && fileJob.Language != "C Header" {
		return false
	}

	return !Duplicates &&
		!Cognitive &&
		!Trace &&
		!NoLarge &&
		!fileJob.ClassifyContent &&
		!fileJob.TrackComplexityLines &&
		fileJob.Callback == nil
}
