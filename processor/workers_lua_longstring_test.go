// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Lua, Luau and XMake declared their long bracket string literal with start and
// end swapped ("start": "]]", "end": "[["). Because the opening [[ never
// matched, a -- inside a long string was treated as a line comment, and a
// stray ]] in ordinary code (such as the nested index a[b[1]]) opened a string
// that ran until the next [[ or EOF, hiding the comments and code after it.
// Declaring the quote as start "[[" / end "]]" makes the scanner see the real
// string so the embedded -- is ignored.

func TestCountStatsLuaLongStringCommentMarkerNotComment(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"Lua", "Luau", "XMake"} {
		fileJob := FileJob{
			Language: language,
		}

		fileJob.SetContent(`local s = [[
-- not a comment, just string content
]]
print(s)`)

		CountStats(&fileJob)

		if fileJob.Lines != 4 {
			t.Errorf("%s: Expected 4 lines got %d", language, fileJob.Lines)
		}
		if fileJob.Code != 4 {
			t.Errorf("%s: Expected 4 code got %d", language, fileJob.Code)
		}
		if fileJob.Comment != 0 {
			t.Errorf("%s: Expected 0 comments got %d", language, fileJob.Comment)
		}
	}
}

func TestCountStatsLuaNestedIndexDoesNotOpenString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Lua",
	}

	fileJob.SetContent(`local x = a[b[1]]
-- a real comment
local y = 2`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comment got %d", fileJob.Comment)
	}
}

// Guard the fix does not break real Lua block comments, which use --[[ ... ]].
func TestCountStatsLuaBlockCommentStillCounted(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Lua",
	}

	fileJob.SetContent(`--[[
block comment body
]]
local x = 1`)

	CountStats(&fileJob)

	if fileJob.Comment != 3 {
		t.Errorf("Expected 3 comments got %d", fileJob.Comment)
	}
	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code got %d", fileJob.Code)
	}
}
