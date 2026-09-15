// SPDX-License-Identifier: MIT

package processor

import "bytes"

// Thirty-two of the 366 languages declare no comment, no string and no
// complexity check: Plain Text, Markdown, JSON, CSV, ReStructuredText, AsciiDoc,
// Patch, the lock and manifest formats. Nothing in one of their files can change
// the state of the scan, so counting one is deciding, for each line, whether it
// holds a byte that is not whitespace. The generic loop arrives at the same
// answer the long way, walking every byte through a state machine, a blank-run
// skip and a trie that has nothing in it to match.
//
// This is not a counter in the sense the other eighteen are. There is no
// language in it and nothing to get wrong per language, so one function answers
// for all thirty-two and for any language that later declares nothing either.
// It is reached on ProcessMask being zero, which is the same thing said in the
// form the loop already holds.
func noTokensAtAll(langFeatures LanguageFeature) bool {
	return langFeatures.ProcessMask == 0
}

// countLoopNoTokens counts a file that has no tokens in it.
//
// IndexByte finds the end of each line a vector at a time, and the line is then
// read only as far as its first byte that is not whitespace, which on a line of
// prose is the first byte. A blank line is read to its end, but a blank line is
// short.
//
// The nul check is the one thing that cannot be skipped: the generic loop marks
// a file binary on a nul inside the first ten thousand bytes, so up to there the
// line is walked in full and past there it is not.
func countLoopNoTokens(fileJob *FileJob, bomSkip, endPoint int) bool {
	content := fileJob.Content
	total := int(fileJob.Bytes)
	var tally counterTally

	for index := bomSkip; index < total; {
		line := content[index:total]
		if next := bytes.IndexByte(line, '\n'); next >= 0 {
			line = line[:next]
		}

		// Where the line's first byte that is not whitespace sits answers both
		// questions. Its presence is the whole of whether the line is code,
		// and its position is where the nul search has to start.
		first := -1
		for offset := 0; offset < len(line); offset++ {
			if !isWhitespace(line[offset]) {
				first = offset

				break
			}
		}

		if first >= 0 {
			// A nul marks the file binary, but only where the generic loop
			// would see one, and it sees less than it looks. blankState carries
			// no binary check, so the first byte of a line that is not
			// whitespace never gets tested however it is spelled; only
			// codeState tests, and it is entered on the bytes after that one.
			// Its walk also stops at endPoint, so the last byte of the file is
			// never tested either - cpython carries a csv fuzz corpus ending
			// \n\n\0 that is counted rather than dropped. Both limits are
			// applied here rather than reasoned about at the call site.
			start := index + first + 1
			end := min(index+len(line), endPoint, 10000)
			if start < end {
				if at := bytes.IndexByte(content[start:end], 0); at >= 0 && isBinary(start+at, 0) {
					tally.Binary = true
					tally.addTo(fileJob)

					return false
				}
			}
		}

		tally.Lines++
		if first >= 0 {
			tally.Code++
		} else {
			tally.Blank++
		}

		index += len(line) + 1
	}

	tally.addTo(fileJob)

	return true
}
