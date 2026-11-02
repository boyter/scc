// SPDX-License-Identifier: MIT

package processor

import "testing"

// TestCaseInsensitiveLineComment covers the languages that read their own
// keywords in any case. Batch and ASP open a comment with REM however it is
// typed, and the spellings are expanded when the language is compiled rather
// than the match being folded, so this is really a test that the expansion
// reached the trie.
func TestCaseInsensitiveLineComment(t *testing.T) {
	tests := []struct {
		name          string
		language      string
		content       string
		code, comment int64
	}{
		{
			name:     "batch every spelling",
			language: "Batch",
			content:  "REM upper\nrEm mixed\nrem lower\nReM other\nECHO hi\n",
			code:     1, comment: 4,
		},
		{
			name:     "asp every spelling",
			language: "ASP",
			content:  "' apostrophe\nREM upper\nrEm mixed\nResponse.Write 1\n",
			code:     1, comment: 3,
		},
		{
			// The word boundary still applies: REMOVE is a command, not a
			// comment, whatever the case of it.
			name:     "batch longer word is not a comment",
			language: "Batch",
			content:  "REMOVE this is code\nrEmOvE also code\nREM this is comment\n",
			code:     2, comment: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, code, comment, _ := countOne(t, tc.language, tc.content)
			if code != tc.code || comment != tc.comment {
				t.Errorf("got code=%d comment=%d, want code=%d comment=%d", code, comment, tc.code, tc.comment)
			}
		})
	}
}

// TestCommentIsWord covers a comment token that is a word in its own right.
// Forth writes its line comment as a backslash, which is a word like any
// other, so it opens a comment only where it stands alone.
func TestCommentIsWord(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		code, comment int64
	}{
		{"backslash then space", "\\ a comment\n1 2 + .\n", 1, 1},
		{"backslash joined to a word", "\\foo not a comment\n1 2 + .\n", 2, 0},
		{"backslash alone on the line", "\\\n1 2 + .\n", 1, 1},
		{"backslash after code", "1 2 + . \\ trailing comment\n", 1, 0},
		{"backslash joined to preceding word", "word\\ still code\n", 1, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, code, comment, _ := countOne(t, "Forth", tc.content)
			if code != tc.code || comment != tc.comment {
				t.Errorf("got code=%d comment=%d, want code=%d comment=%d", code, comment, tc.code, tc.comment)
			}
		})
	}
}

// TestCaseSpellingsExpansion pins the expansion itself, including the refusal
// to expand something long enough to blow up.
func TestCaseSpellingsExpansion(t *testing.T) {
	if got := caseSpellings([]string{"REM"}, false); len(got) != 1 {
		t.Errorf("case sensitive language expanded to %v", got)
	}
	if got := caseSpellings([]string{"REM"}, true); len(got) != 8 {
		t.Errorf("REM expanded to %d spellings, want 8: %v", len(got), got)
	}
	if got := caseSpellings([]string{"::"}, true); len(got) != 1 || got[0] != "::" {
		t.Errorf("a token with no letters expanded to %v", got)
	}
	// Nine letters is past the cap, so it is left as written rather than
	// becoming 512 entries.
	long := "abcdefghi"
	if got := caseSpellings([]string{long}, true); len(got) != 1 || got[0] != long {
		t.Errorf("an over long token expanded to %d entries", len(got))
	}
}
