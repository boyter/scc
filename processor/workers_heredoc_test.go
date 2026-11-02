// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// A heredoc's body runs to a line holding nothing but the word named after the
// <<, so the closer is chosen where it is written and no fixed list of quotes
// can hold it. Read as anything else, an apostrophe in the body opens a string
// that swallows every comment below it. Linejudge case 2120.
func TestCountStatsShellHeredoc(t *testing.T) {
	for _, test := range []struct {
		name          string
		content       string
		code, comment int64
	}{
		{"apostrophe in the body", "cat <<TEXT\nit's fine in here\nTEXT\n# a real comment\n# a second real comment\n", 3, 2},
		{"space before the word", "cat << TEXT\nit's\nTEXT\n# c\n", 3, 1},
		{"single quoted word", "cat <<'TEXT'\nit's $x\nTEXT\n# c\n", 3, 1},
		{"double quoted word", "cat <<\"TEXT\"\nit's\nTEXT\n# c\n", 3, 1},
		{"backslashed word", "cat <<\\TEXT\nit's\nTEXT\n# c\n", 3, 1},
		{"dash form closes on a tab indented word", "cat <<-TEXT\n\tit's\n\tTEXT\n# c\n", 3, 1},
		{"plain form does not close on an indented word", "cat <<TEXT\nit's\n  TEXT\nTEXT\n# c\n", 4, 1},
		{"the word inside a line does not close", "cat <<TEXT\nnot TEXT here, it's\nTEXT\n# c\n", 3, 1},
		{"a longer word does not close", "cat <<TEXT\nTEXTS it's\nTEXT\n# c\n", 3, 1},
		{"comment marker in the body is text", "cat <<TEXT\n# not a comment\nTEXT\n# c\n", 3, 1},
		{"rest of the opening line is not scanned", "cat <<TEXT | sed 's/a/b/'\nit's\nTEXT\n# c\n", 3, 1},
		{"closer ends the file", "cat <<TEXT\nit's\nTEXT", 3, 0},
		{"closer before a carriage return", "cat <<TEXT\r\nit's\r\nTEXT\r\n# c\r\n", 3, 1},
		{"two heredocs in a row", "cat <<A\nit's\nA\n# c\ncat <<B\nit's\nB\n# d\n", 6, 2},
		{"here-string is not a heredoc", "cat <<< word\n# c\nword\n", 2, 1},
		{"shift by a number is not a heredoc", "echo $((1<<2))\n# c\n", 1, 1},
		{"shift by a name with no closer below is not a heredoc", "echo $((1 << n))\n# c\n", 1, 1},
		{"heredoc at the start of a line", "<<TEXT cat\nit's\nTEXT\n# c\n", 3, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, code, comment, _ := countOne(t, "Shell", test.content)
			if code != test.code || comment != test.comment {
				t.Errorf("got code=%d comment=%d, want code=%d comment=%d", code, comment, test.code, test.comment)
			}

			// Classifying the content walks every byte, which takes the slow
			// code scan and the byte marking in the heredoc body.
			fileJob := &FileJob{Language: "Shell", Content: []byte(test.content), Bytes: int64(len(test.content)), ClassifyContent: true}
			CountStats(fileJob)
			if fileJob.Code != test.code || fileJob.Comment != test.comment {
				t.Errorf("classified: got code=%d comment=%d, want code=%d comment=%d", fileJob.Code, fileJob.Comment, test.code, test.comment)
			}
		})
	}
}

// The shells that share the POSIX heredoc all read it.
func TestCountStatsHeredocShells(t *testing.T) {
	for _, language := range []string{"Shell", "BASH", "Korn Shell", "Zsh"} {
		_, code, comment, _ := countOne(t, language, "cat <<TEXT\nit's\nTEXT\n# c\n")
		if code != 3 || comment != 1 {
			t.Errorf("%s: got code=%d comment=%d, want code=3 comment=1", language, code, comment)
		}
	}
}

// C3 opens a doc comment with <*, but writes a vector of unknown length as
// int[<*>], and its lexer reads [< as one token there, so that <* opens
// nothing. Linejudge case 1240.
func TestCountStatsC3VectorTypeOpensNoComment(t *testing.T) {
	for _, test := range []struct {
		name          string
		content       string
		code, comment int64
	}{
		{"vector type under a doc comment", "<*\n Rotate every lane of the vector.\n*>\nmacro rotate(int[<*>] x, uint shift)\n{\n\tuint lanes = shift;\n\treturn $$fshl(x, x, lanes);\n}\n", 5, 3},
		{"vector type then a comment", "fn void f(int[<*>] x) {}\n// c\n", 1, 1},
		{"doc comment still opens", "<* doc *>\nfn void f() {}\n", 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, code, comment, _ := countOne(t, "C3", test.content)
			if code != test.code || comment != test.comment {
				t.Errorf("got code=%d comment=%d, want code=%d comment=%d", code, comment, test.code, test.comment)
			}
		})
	}
}
