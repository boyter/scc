// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// A line comment that is also the start of a longer token, such as Lua's --
// in front of --[[, was counted as code when the rest of that token did not
// follow it, because the token match only fell back to tokens with a closing
// delimiter.
func TestCountStatsLineCommentPrefixOfBlockComment(t *testing.T) {
	ProcessConstants()

	cases := []struct {
		language string
		content  string
	}{
		{"Lua", "--[ not a block comment\nlocal x = 1\n"},
		{"Lua", "--[=x not a block comment\nlocal x = 1\n"},
		{"CMake", "#[ not a bracket comment\nset(X 1)\n"},
	}

	for _, c := range cases {
		fileJob := FileJob{
			Language: c.language,
		}
		fileJob.SetContent(c.content)

		CountStats(&fileJob)

		if fileJob.Comment != 1 {
			t.Errorf("%s %q: expected 1 comment got %d", c.language, c.content, fileJob.Comment)
		}
		if fileJob.Code != 1 {
			t.Errorf("%s %q: expected 1 code got %d", c.language, c.content, fileJob.Code)
		}
	}
}
