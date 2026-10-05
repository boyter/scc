// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Tests verifying line counting, comments, and dialect rules corresponding
// to LineJudge conformance heuristics and multi-counter accuracy comparisons (#769).

// TestLineJudgeBatchCommentHeuristics tests Batch script comment counting heuristics.
// In LineJudge 1230, scc's prefix-based trie scanner identifies comment tokens
// at line head (REM, ::). This test pins the exact counting behavior.
func TestLineJudgeBatchCommentHeuristics(t *testing.T) {
	ProcessConstants()

	tests := []struct {
		name    string
		content string
		lines   int64
		code    int64
		comment int64
		blank   int64
	}{
		{
			name:    "standard REM comment",
			content: "REM a comment\necho done\n",
			lines:   2, code: 1, comment: 1, blank: 0,
		},
		{
			name:    "double colon comment",
			content: ":: this is a comment\necho hello\n",
			lines:   2, code: 1, comment: 1, blank: 0,
		},
		{
			name: "LineJudge 1230 case: REM comment distinguished from command starting with REM",
			// scc correctly identifies REM as a comment and REMOVE as a code command
			content: "REM a comment\nREMOVE /Q file.txt\necho done\n",
			lines:   3, code: 2, comment: 1, blank: 0,
		},
		{
			name:    "command with prefix before REM is code",
			content: "XREMOVE /Q file.txt\necho done\n",
			lines:   2, code: 2, comment: 0, blank: 0,
		},
		{
			name:    "REM keyword inside arguments is code",
			content: "echo REM inside\n",
			lines:   1, code: 1, comment: 0, blank: 0,
		},
		{
			name:    "mixed comments and blank lines",
			content: "@echo off\n\nREM build step\n:: clean step\n\ngoto end\n",
			lines:   6, code: 2, comment: 2, blank: 2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			job := FileJob{Language: "Batch"}
			job.SetContent(tc.content)
			CountStats(&job)

			if job.Lines != tc.lines {
				t.Errorf("%s: expected %d lines, got %d", tc.name, tc.lines, job.Lines)
			}
			if job.Code != tc.code {
				t.Errorf("%s: expected %d code, got %d", tc.name, tc.code, job.Code)
			}
			if job.Comment != tc.comment {
				t.Errorf("%s: expected %d comment, got %d", tc.name, tc.comment, job.Comment)
			}
			if job.Blank != tc.blank {
				t.Errorf("%s: expected %d blank, got %d", tc.name, tc.blank, job.Blank)
			}
		})
	}
}

// TestLineJudgeDocstringCountingHeuristics verifies Python docstring classification
// against LineJudge rules: docstrings standing alone count as comments, whereas
// docstrings adjacent to code count as code.
func TestLineJudgeDocstringCountingHeuristics(t *testing.T) {
	ProcessConstants()

	tests := []struct {
		name    string
		content string
		lines   int64
		code    int64
		comment int64
		blank   int64
	}{
		{
			name:    "standalone module docstring",
			content: "\"\"\"\nModule docstring\nSecond line\n\"\"\"\nx = 1\n",
			lines:   5, code: 1, comment: 4, blank: 0,
		},
		{
			name:    "single line standalone docstring",
			content: "\"\"\"Single line docstring\"\"\"\nx = 1\n",
			lines:   2, code: 1, comment: 1, blank: 0,
		},
		{
			name:    "docstring adjacent to code on same line",
			content: "x = 1; \"\"\"inline doc\"\"\"\ny = 2\n",
			lines:   2, code: 2, comment: 0, blank: 0,
		},
		{
			name:    "docstring with embedded blank line",
			content: "\"\"\"\nSummary.\n\nDescription.\n\"\"\"\npass\n",
			lines:   6, code: 1, comment: 5, blank: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fast, generic := countBothWays(t, "Python", []byte(tc.content))

			if fast.Lines != tc.lines || generic.Lines != tc.lines {
				t.Errorf("%s: lines mismatch (fast=%d, generic=%d, want=%d)", tc.name, fast.Lines, generic.Lines, tc.lines)
			}
			if fast.Code != tc.code || generic.Code != tc.code {
				t.Errorf("%s: code mismatch (fast=%d, generic=%d, want=%d)", tc.name, fast.Code, generic.Code, tc.code)
			}
			if fast.Comment != tc.comment || generic.Comment != tc.comment {
				t.Errorf("%s: comment mismatch (fast=%d, generic=%d, want=%d)", tc.name, fast.Comment, generic.Comment, tc.comment)
			}
			if fast.Blank != tc.blank || generic.Blank != tc.blank {
				t.Errorf("%s: blank mismatch (fast=%d, generic=%d, want=%d)", tc.name, fast.Blank, generic.Blank, tc.blank)
			}
		})
	}
}

// TestLineJudgeRustCharLiteralHeuristics verifies Rust character literal parsing
// heuristics corresponding to LineJudge cases 4010 and 4020.
func TestLineJudgeRustCharLiteralHeuristics(t *testing.T) {
	ProcessConstants()

	tests := []struct {
		name             string
		content          string
		expectedCode     int64
		expectedComment  int64
		expectedGenericC int64
		expectedGenericM int64
	}{
		{
			name:             "LineJudge 4010: char literal holding quote",
			content:          "let c = '\"';\n// comment 1\n// comment 2\n",
			expectedCode:     1,
			expectedComment:  2,
			expectedGenericC: 3,
			expectedGenericM: 0,
		},
		{
			name:             "LineJudge 4020: char literals with escapes",
			content:          "let a = '\\'';\nlet b = '\"';\nlet c = 1;\n// comment 1\n// comment 2\n",
			expectedCode:     3,
			expectedComment:  2,
			expectedGenericC: 5,
			expectedGenericM: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fast, generic := countBothWays(t, "Rust", []byte(tc.content))

			if fast.Code != tc.expectedCode || fast.Comment != tc.expectedComment {
				t.Errorf("%s fast counter: got code=%d comment=%d, want code=%d comment=%d",
					tc.name, fast.Code, fast.Comment, tc.expectedCode, tc.expectedComment)
			}

			if generic.Code != tc.expectedGenericC || generic.Comment != tc.expectedGenericM {
				t.Errorf("%s generic loop: got code=%d comment=%d, want code=%d comment=%d",
					tc.name, generic.Code, generic.Comment, tc.expectedGenericC, tc.expectedGenericM)
			}
		})
	}
}

// TestLineJudgeEcmaRegexLiteralHeuristics verifies ECMAScript regular expression
// parsing heuristics corresponding to LineJudge cases 7010 and 7020.
func TestLineJudgeEcmaRegexLiteralHeuristics(t *testing.T) {
	ProcessConstants()

	tests := []struct {
		language         string
		name             string
		content          string
		expectedCode     int64
		expectedComment  int64
		expectedGenericC int64
		expectedGenericM int64
	}{
		{
			language:         "JavaScript",
			name:             "LineJudge 7010 JS: regex holding quote",
			content:          "const pattern = /\";/;\n// comment 1\n// comment 2\n",
			expectedCode:     1,
			expectedComment:  2,
			expectedGenericC: 3,
			expectedGenericM: 0,
		},
		{
			language:         "JavaScript",
			name:             "LineJudge 7020 JS: regex holding comment opener",
			content:          "const pattern = /\\/\\//;\n// comment 1\n",
			expectedCode:     1,
			expectedComment:  1,
			expectedGenericC: 1,
			expectedGenericM: 1,
		},
		{
			language:         "TypeScript",
			name:             "LineJudge 7010 TS: regex holding quote",
			content:          "const pattern: RegExp = /\";/;\n// comment 1\n// comment 2\n",
			expectedCode:     1,
			expectedComment:  2,
			expectedGenericC: 3,
			expectedGenericM: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fast, generic := countBothWays(t, tc.language, []byte(tc.content))

			if fast.Code != tc.expectedCode || fast.Comment != tc.expectedComment {
				t.Errorf("%s fast counter: got code=%d comment=%d, want code=%d comment=%d",
					tc.name, fast.Code, fast.Comment, tc.expectedCode, tc.expectedComment)
			}

			if generic.Code != tc.expectedGenericC || generic.Comment != tc.expectedGenericM {
				t.Errorf("%s generic loop: got code=%d comment=%d, want code=%d comment=%d",
					tc.name, generic.Code, generic.Comment, tc.expectedGenericC, tc.expectedGenericM)
			}
		})
	}
}

// TestCountingHeuristicsSpecializedVsGenericEquivalence verifies that the fast
// specialized counters and generic loop produce identical counts for standard
// code constructs across multiple languages.
func TestCountingHeuristicsSpecializedVsGenericEquivalence(t *testing.T) {
	ProcessConstants()

	testMatrix := []struct {
		language string
		content  string
	}{
		{
			language: "Go",
			content:  "package main\n\n// line comment\nfunc main() {\n\t/* block comment */\n\tprintln(\"hello /* world */ // test\")\n}\n",
		},
		{
			language: "Java",
			content:  "public class App {\n\t// single line\n\t/* multi\n\t * line */\n\tString s = \"// not comment\";\n}\n",
		},
		{
			language: "C",
			content:  "#include <stdio.h>\n\n// main comment\nint main() {\n\t/* block */ int x = 1;\n\tchar *s = \"/* not comment */\";\n\treturn 0;\n}\n",
		},
		{
			language: "C++",
			content:  "#include <iostream>\n\n// header comment\nint main() {\n\tint a = 1; // code then comment\n\t/* block\n\t   span */\n\treturn 0;\n}\n",
		},
		{
			language: "Python",
			content:  "# module comment\ndef foo():\n\tx = 1\n\t# inner comment\n\ts = '# not a comment'\n\treturn x\n",
		},
		{
			language: "Ruby",
			content:  "# frozen_string_literal: true\n\ndef hello\n\t# greeting\n\tputs \"# not comment\"\nend\n",
		},
		{
			language: "PHP",
			content:  "<?php\n// single line\n# perl style\n/* multi\n   line */\necho 'hello';\n",
		},
	}

	for _, tc := range testMatrix {
		t.Run(tc.language, func(t *testing.T) {
			fast, generic := countBothWays(t, tc.language, []byte(tc.content))

			if countsDiffer(fast, generic) {
				t.Errorf("%s: specialized counter diverged from generic loop (fast=%+v, generic=%+v)",
					tc.language, fast, generic)
			}
			if fast.Lines != fast.Code+fast.Comment+fast.Blank {
				t.Errorf("%s: total lines (%d) does not match code(%d)+comment(%d)+blank(%d)",
					tc.language, fast.Lines, fast.Code, fast.Comment, fast.Blank)
			}
		})
	}
}

// TestLineJudgeDialectBucketInvariant asserts that total lines always strictly
// partition into code, comment, and blank buckets without overlap or loss.
func TestLineJudgeDialectBucketInvariant(t *testing.T) {
	ProcessConstants()

	samples := []struct {
		language string
		content  string
	}{
		{"Go", "\n\nfunc f() {}\n\n"},
		{"Python", "\n# comment only\n\n"},
		{"C", "/* open\n\nclose */\n"},
		{"JavaScript", "const a = 1;\n\n// comment\n\nconst b = 2;\n"},
		{"Batch", "\nREM comment\n\ncall cmd\n\n"},
	}

	for _, s := range samples {
		t.Run(s.language, func(t *testing.T) {
			job := FileJob{Language: s.language}
			job.SetContent(s.content)
			CountStats(&job)

			sum := job.Code + job.Comment + job.Blank
			if sum != job.Lines {
				t.Errorf("%s: bucket invariant violated: Lines=%d != Code(%d) + Comment(%d) + Blank(%d)",
					s.language, job.Lines, job.Code, job.Comment, job.Blank)
			}
		})
	}
}
