// SPDX-License-Identifier: MIT

package processor

import "testing"

// TestOneByteTokenAtEndOfFile pins the crash a file ending in a one byte
// complexity check used to cause.
//
// Trie.Match reports the depth its walk reached. Where the token runs to the
// very end of the slice it was handed the walk stops for want of bytes rather
// than on a byte the token does not carry, so the depth it reports is one short.
// For a token of one byte that is zero, and every caller stepping back by one
// from it drove the index to -1 and panicked.
//
// Eight languages carry a one byte complexity check: APL, Alchemist, Brainfuck,
// Factor, K, Stata, Swift and jq. A Brainfuck file of a single + , or a Swift
// file of a single ? , crashed scc outright.
func TestOneByteTokenAtEndOfFile(t *testing.T) {
	ProcessConstants()

	for _, test := range []struct {
		language string
		content  string
	}{
		{"Swift", "?"},
		{"Swift", "let a = b?"},
		{"Swift", "let a = b ?"},
		{"Brainfuck", "+"},
		{"Brainfuck", "["},
		{"Brainfuck", "]"},
		{"Brainfuck", "<"},
		{"Brainfuck", ">"},
		{"Brainfuck", "-"},
		{"Brainfuck", "."},
		{"Brainfuck", ","},
		{"Brainfuck", "[+]"},
		{"Brainfuck", ",[.,]"},
		{"jq", "."},
		{"Factor", "?"},
		{"Factor", "="},
		{"Stata", "|"},
		{"Stata", "&"},
		{"Alchemist", "+"},
		{"Alchemist", "!"},
		{"APL", "~"},
		{"APL", "="},
		{"APL", ":"},
		{"K", "'"},
		{"K", "/"},
		{"K", "\\"},
		{"K", "|"},
		{"K", "&"},
		{"K", "!"},
		{"K", "="},
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic counting %s content %q: %v", test.language, test.content, r)
				}
			}()

			fileJob := &FileJob{
				Language: test.language,
				Content:  []byte(test.content),
				Bytes:    int64(len(test.content)),
			}
			CountStats(fileJob)

			if fileJob.Lines != 1 {
				t.Errorf("%s content %q counted %d lines, want 1", test.language, test.content, fileJob.Lines)
			}
		}()
	}
}
