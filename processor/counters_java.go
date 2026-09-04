// SPDX-License-Identifier: MIT

package processor

// A counter written for one language rather than driven by the tries built from
// languages.json. The generic loop asks a trie what token sits at a byte, which
// the profile puts at a quarter of the whole program; Java needs to know about
// four tokens and can answer that with a comparison.
//
// Everything here must agree with the generic loop to the line. Where the two
// differ the generic one is right by definition, since it is what the rest of
// the tests are written against, and countLoopJava is only ever an accelerator.

// javaStop marks every byte the scan has to stop on: the slash of both comment
// forms, the two quotes, the first byte of every complexity check, and the
// newline and the null that end a line and a file of bytes rather than text.
// One load and one branch answer for a byte, where asking each question in turn
// costs several.
var javaStop = buildJavaStop()

// javaInteresting marks only the bytes that begin a token, which is javaStop
// without the two that end things.
var javaInteresting = buildJavaInteresting()

func buildJavaStop() [256]bool {
	table := buildJavaInteresting()
	table['\n'] = true
	table[0] = true

	return table
}

func buildJavaInteresting() [256]bool {
	var table [256]bool
	for _, b := range []byte{'/', '"', '\'', 'f', 'i', 's', 'w', 'e', 't', 'c', '|', '&', '!', '='} {
		table[b] = true
	}

	return table
}

// javaComplexityToken reports the length of the complexity check that begins at
// the front of content, or zero when none does. None of Java's checks is a
// prefix of another, so the first that matches is the longest.
func javaComplexityToken(content []byte) int {
	switch content[0] {
	case 'f':
		if hasPrefix(content, "for ") || hasPrefix(content, "for(") {
			return 4
		}
		if hasPrefix(content, "finally ") || hasPrefix(content, "finally{") {
			return 8
		}
	case 'i':
		if hasPrefix(content, "if ") || hasPrefix(content, "if(") {
			return 3
		}
	case 's':
		if hasPrefix(content, "switch ") || hasPrefix(content, "switch(") {
			return 7
		}
	case 'w':
		if hasPrefix(content, "while ") || hasPrefix(content, "while(") {
			return 6
		}
	case 'e':
		if hasPrefix(content, "else ") || hasPrefix(content, "else{") {
			return 5
		}
	case 't':
		if hasPrefix(content, "try ") || hasPrefix(content, "try{") {
			return 4
		}
	case 'c':
		if hasPrefix(content, "catch ") || hasPrefix(content, "catch(") {
			return 6
		}
	case '|':
		if hasPrefix(content, "|| ") {
			return 3
		}
	case '&':
		if hasPrefix(content, "&& ") {
			return 3
		}
	case '!':
		if hasPrefix(content, "!= ") {
			return 3
		}
	case '=':
		if hasPrefix(content, "== ") {
			return 3
		}
	}

	return 0
}

func hasPrefix(content []byte, prefix string) bool {
	if len(content) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if content[i] != prefix[i] {
			return false
		}
	}

	return true
}

// javaWordStarts reports whether a complexity check could begin at index, which
// is that the byte in front of it does not carry a word on.
//
// The generic loop matches the check first and applies this after, then steps
// over the token either way. Testing it first is the same thing done in the
// cheaper order: no check of Java holds the opening of another, or a quote, or
// a slash, or a newline, so the bytes the generic loop steps over hold nothing
// that would have been read had it not. It is worth the argument because an f,
// i, s, w, e, t or c is nearly always in the middle of an identifier, and this
// is what keeps the match from being run on every one of them.
func javaWordStarts(content []byte, index int) bool {
	return index == 0 || !isIdentifierContinue(content[index-1])
}

// javaCountComplexity counts a check that has already been found to begin a
// word. The global reads as complexity having been turned off, which is what
// the generic loop tests when it decides whether to put the checks into its
// trie at all, so the two agree on a file counted with --no-complexity.
func javaCountComplexity(fileJob *FileJob) {
	if Complexity {
		return
	}

	fileJob.Complexity++
	fileJob.bumpComplexityLine()
}

// javaBlankState looks at the first byte of content on a line.
func javaBlankState(fileJob *FileJob, index int) (int, int64, byte) {
	content := fileJob.Content

	switch content[index] {
	case '/':
		if index+1 < len(content) {
			switch content[index+1] {
			case '/':
				return index, SComment, 0
			case '*':
				return index + 1, SMulticomment, 0
			}
		}
	case '"', '\'':
		return index, SString, content[index]
	}

	if javaInteresting[content[index]] && javaWordStarts(content, index) {
		if length := javaComplexityToken(content[index:]); length != 0 {
			javaCountComplexity(fileJob)
			return index + length - 1, SCode, 0
		}
	}

	return index, SCode, 0
}

// javaCodeState runs to the end of the line or to whatever token takes it out
// of code.
func javaCodeState(fileJob *FileJob, index, endPoint int) (int, int64, byte) {
	content := fileJob.Content
	if endPoint > len(content) {
		endPoint--
	}

	for i := index; i < endPoint; i++ {
		curByte := content[i]

		if !javaStop[curByte] {
			continue
		}

		switch curByte {
		case '\n':
			return i, SCode, 0
		case 0:
			if isBinary(i, curByte) {
				fileJob.Binary = true
				return i, SCode, 0
			}
		case '/':
			if i+1 < len(content) {
				switch content[i+1] {
				case '/':
					return i, SCommentCode, 0
				case '*':
					return i + 1, SMulticommentCode, 0
				}
			}
		case '"', '\'':
			// The generic loop tests the byte in front rather than counting the
			// run of them, so a quote behind a backslash opens nothing and the
			// line carries on as code.
			if content[i-1] != '\\' {
				return i, SString, curByte
			}

			return i, SCode, 0
		default:
			if javaWordStarts(content, i) {
				if length := javaComplexityToken(content[i:]); length != 0 {
					javaCountComplexity(fileJob)
					i += length - 1
				}
			}
		}
	}

	// The generic loop leaves the cursor on the last byte it looked at, which
	// is the one before endPoint when it got that far.
	if index < endPoint {
		return endPoint - 1, SCode, 0
	}

	return index, SCode, 0
}

// javaStringState runs to the closing quote, which a run of backslashes of odd
// length in front of it does not count as.
func javaStringState(fileJob *FileJob, index, endPoint int, endQuote byte) (int, int64) {
	content := fileJob.Content

	for i := index; i < endPoint; i++ {
		index = i

		if content[i] == '\n' {
			return i, SString
		}

		isEscaped := false
		if content[i-1] == '\\' {
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

		if !isEscaped && content[i] == endQuote {
			return i, SCode
		}
	}

	return index, SString
}

// javaCommentState runs to the closer of a block comment, which does not nest
// in Java.
func javaCommentState(fileJob *FileJob, index, endPoint int, currentState int64) (int, int64) {
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

	return index, currentState
}

// countLoopJava stands in for countLoopGeneric where the language is Java and
// none of the extra outputs are wanted. It returns false when it ended the
// count early, the same way the generic loop does.
func countLoopJava(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	currentState := SBlank
	var endQuote byte

	for index := bomSkip; index < int(fileJob.Bytes); index++ {
		if !isWhitespace(content[index]) {
			switch currentState {
			case SCode:
				index, currentState, endQuote = javaCodeState(fileJob, index, endPoint)
			case SString:
				index, currentState = javaStringState(fileJob, index, endPoint, endQuote)
			case SMulticomment, SMulticommentCode:
				index, currentState = javaCommentState(fileJob, index, endPoint, currentState)
			case SBlank, SMulticommentBlank:
				index, currentState, endQuote = javaBlankState(fileJob, index)
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

			switch currentState {
			case SCode, SString, SCommentCode, SMulticommentCode:
				fileJob.Code++
				currentState = resetState(currentState)
			case SComment, SMulticomment, SMulticommentBlank:
				fileJob.Comment++
				currentState = resetState(currentState)
			case SBlank:
				fileJob.Blank++
			}
		}
	}

	return true
}

// DisableSpecialisedCounters turns every counter written for one language off
// and sends all of them through the generic loop. It is what the differential
// test runs the same file through both ways with, and an escape hatch should a
// counter ever disagree with the generic loop in the wild.
var DisableSpecialisedCounters bool

// useJavaCounter reports whether the Java counter can answer for this file. It
// produces the four line counts and the complexity count and nothing else, so
// anything that asks for more is left to the generic loop.
func useJavaCounter(fileJob *FileJob) bool {
	return !DisableSpecialisedCounters &&
		fileJob.Language == "Java" &&
		!Duplicates &&
		!Cognitive &&
		!Trace &&
		!NoLarge &&
		!fileJob.ClassifyContent &&
		!fileJob.TrackComplexityLines &&
		fileJob.Callback == nil
}
