// SPDX-License-Identifier: MIT

package processor

import "testing"

// Nix writes shell snippets in indented strings, opened and closed by two
// single quotes. Read as code, a glob like dist/* in one opened a block comment
// that ran on to the next */ in the file.
func TestCountStatsNixIndentedString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Nix"}
	fileJob.SetContent(`{ stdenv }:

stdenv.mkDerivation {
  installPhase = ''
    mkdir -p $out/share
    cp -r dist/* $out/share
  '';

  # Run the tests.
  doCheck = true;
}
`)
	CountStats(&fileJob)

	if fileJob.Lines != 11 || fileJob.Code != 8 || fileJob.Comment != 1 || fileJob.Blank != 2 {
		t.Errorf("Expected 11 lines 8 code 1 comment 2 blank got %d lines %d code %d comment %d blank",
			fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank)
	}
}

// Inside an indented string two single quotes followed by $, a third single
// quote or a backslash are an escape, for a literal $, a pair of quotes and a
// backslash escape, so none of them ends the string. Each case holds a single
// escape, so reading it as the end would turn the shell comment under it into
// a comment.
func TestCountStatsNixIndentedStringEscapes(t *testing.T) {
	ProcessConstants()

	for _, escape := range []string{`''${HOME}`, `'''quoted`, `''\n`} {
		fileJob := FileJob{Language: "Nix"}
		fileJob.SetContent("{\n" +
			"  script = ''\n" +
			"    echo " + escape + "\n" +
			"    # still the script\n" +
			"  '';\n" +
			"}\n")
		CountStats(&fileJob)

		if fileJob.Lines != 6 || fileJob.Code != 6 || fileJob.Comment != 0 || fileJob.Blank != 0 {
			t.Errorf("%s: expected 6 lines 6 code 0 comment 0 blank got %d lines %d code %d comment %d blank",
				escape, fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank)
		}
	}
}

// A double quoted string is unchanged, and two single quotes inside one are
// just part of its text.
func TestCountStatsNixDoubleQuotedString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{Language: "Nix"}
	fileJob.SetContent(`{
  a = "it's ''not'' indented /* or a comment";
  # A comment.
  b = "\"/*";
  # Another comment.
}
`)
	CountStats(&fileJob)

	if fileJob.Lines != 6 || fileJob.Code != 4 || fileJob.Comment != 2 || fileJob.Blank != 0 {
		t.Errorf("Expected 6 lines 4 code 2 comment 0 blank got %d lines %d code %d comment %d blank",
			fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank)
	}
}
