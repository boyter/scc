// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// Visual Basic has no backslash escape, so "C:\" is a whole string and the
// backslash is just a character of the path. Read as an escape it carried the
// string on to the next quote in the file, which turned the comment under it
// into code.
func TestCountStatsVisualBasicBackslashBeforeClosingQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Visual Basic",
	}

	fileJob.SetContent(`Dim p = "C:\"
' a comment
Dim s = "say ""hi"""
`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 comments got %d", fileJob.Comment)
	}
}

// A quote inside a string is written twice, so a doubled quote does not end
// it, and the backslash in front of its real closer is not read as escaping
// that.
func TestCountStatsVisualBasicDoubledQuoteThenBackslash(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Visual Basic",
	}

	fileJob.SetContent(`Dim s = "say ""C:\"""
' a comment

Dim q = """"
' another comment
`)

	CountStats(&fileJob)

	if fileJob.Lines != 5 {
		t.Errorf("Expected 5 lines got %d", fileJob.Lines)
	}
	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 blank got %d", fileJob.Blank)
	}
}

// VBA and VB6 read strings the same way, in class modules and in the .bas
// standard modules counted as Basic.
func TestCountStatsVisualBasicForApplicationsAndBasicStrings(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"Visual Basic for Applications", "Basic"} {
		fileJob := FileJob{
			Language: language,
		}

		fileJob.SetContent(`Const Root As String = "C:\Temp\"
' a comment
Debug.Print "say ""hi"""
`)

		CountStats(&fileJob)

		if fileJob.Code != 2 {
			t.Errorf("%s: Expected 2 code got %d", language, fileJob.Code)
		}
		if fileJob.Comment != 1 {
			t.Errorf("%s: Expected 1 comments got %d", language, fileJob.Comment)
		}
	}
}

// FreeBASIC, which also uses .bas, has an escaped string written !"..." in
// which a backslash does escape the quote after it.
func TestCountStatsBasicEscapedString(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Basic",
	}

	fileJob.SetContent(`print !"say \"hi\" from C:\\"
' a comment
print !"\""
' another comment
`)

	CountStats(&fileJob)

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}
