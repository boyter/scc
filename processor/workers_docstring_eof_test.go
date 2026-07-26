// SPDX-License-Identifier: MIT

package processor

import "testing"

// Reported downstream by cs (github.com/boyter/cs), which vendors this package:
//
//	panic: runtime error: index out of range [1411] with length 1411
//	  processor.blankState(...) workers.go:377
//
// The trigger is a file whose final bytes are a docstring/ignore-escape quote
// start token (Python """ or ''', C# @") with no trailing newline.
//
// verifyIgnoreEscape advances index past the matched start token
// (index += len(Quotes[i].Start)) without bounding it against len(Content).
// When the token ends the file, index lands exactly on len(Content), and
// blankState then writes ContentByteType[index] out of bounds.
//
// The out-of-bounds write only fires when ClassifyContent is set (cs sets it;
// scc's own CLI does not), but the same over-advance corrupts line counting for
// every caller: the final line is skipped entirely.

var docstringEOFCases = []struct {
	name     string
	language string
	content  string
	lines    int
}{
	{"python triple double quote at eof", "Python", "def greet():\n    return \"hi\"\n\"\"\"", 3},
	{"python triple single quote at eof", "Python", "x = 1\n'''", 2},
	{"python docstring is whole file", "Python", "\"\"\"", 1},
	{"csharp ignore escape quote at eof", "C#", "class A {\n@\"", 2},
	// Control: the same token followed by a newline has always worked.
	{"python docstring with trailing newline", "Python", "x = 1\n\"\"\"\n", 2},
}

// A quote start token at EOF must not write past the end of ContentByteType.
func TestCountStatsDocStringAtEOFNoPanic(t *testing.T) {
	ProcessConstants()

	for _, tc := range docstringEOFCases {
		t.Run(tc.name, func(t *testing.T) {
			fileJob := FileJob{Language: tc.language, ClassifyContent: true}
			fileJob.SetContent(tc.content)

			CountStats(&fileJob)

			if len(fileJob.ContentByteType) != len(fileJob.Content) {
				t.Errorf("Expected ContentByteType of length %d got %d", len(fileJob.Content), len(fileJob.ContentByteType))
			}
		})
	}
}

// The same over-advance silently drops the last line for every caller, whether
// or not content classification is enabled.
func TestCountStatsDocStringAtEOFLineCount(t *testing.T) {
	ProcessConstants()

	for _, tc := range docstringEOFCases {
		t.Run(tc.name, func(t *testing.T) {
			fileJob := FileJob{Language: tc.language}
			fileJob.SetContent(tc.content)

			CountStats(&fileJob)

			if fileJob.Lines != int64(tc.lines) {
				t.Errorf("Expected %d lines got %d", tc.lines, fileJob.Lines)
			}
			if fileJob.Lines != fileJob.Code+fileJob.Comment+fileJob.Blank {
				t.Errorf("Expected lines %d to equal code %d + comment %d + blank %d",
					fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank)
			}
		})
	}
}
