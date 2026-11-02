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

// REM opens a comment as well as the apostrophe, in the spellings it is
// written in: the VB editors case it as Rem and older code has it as REM or
// rem. It has to be the whole word, so REMARK and Remove are code.
func TestCountStatsVisualBasicRemComment(t *testing.T) {
	ProcessConstants()

	for _, language := range []string{"Visual Basic", "Visual Basic for Applications", "Basic"} {
		fileJob := FileJob{
			Language: language,
		}

		fileJob.SetContent(`Rem a comment
REM another comment
rem and another
Rem
REMARK = 1
Remove()
x = 1 : Rem code first
`)

		CountStats(&fileJob)

		if fileJob.Code != 3 {
			t.Errorf("%s: Expected 3 code got %d", language, fileJob.Code)
		}
		if fileJob.Comment != 4 {
			t.Errorf("%s: Expected 4 comments got %d", language, fileJob.Comment)
		}
	}
}

// A quote in a REM comment is part of the comment, so it does not open a
// string that runs on and turns the comment under it into code.
func TestCountStatsVisualBasicRemCommentWithQuote(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Visual Basic",
	}

	fileJob.SetContent(`Rem the path ends in "
' a comment
Dim s = "x"
`)

	CountStats(&fileJob)

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 code got %d", fileJob.Code)
	}
	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 comments got %d", fileJob.Comment)
	}
}

// Visual Basic keywords are written capitalised. Each If, ElseIf, For, While,
// Until, Case and AndAlso or OrElse counts once, while the End If, Exit For and
// End While that close or leave a block do not, even with a comment after
// them, and neither does the Select Case in front of the Case lines.
func TestCountStatsVisualBasicComplexity(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Visual Basic",
	}

	fileJob.SetContent(`If a AndAlso b Then
    For i = 1 To 10
        If i = 5 Then Exit For ' found it
        Continue For
    Next
ElseIf c OrElse d Then
    While x < 3
        x += 1
    End While ' x is 3
Else
    Select Case x
        Case 1
        Case Else
    End Select
End If ' done
Do
Loop Until x > 5
`)

	CountStats(&fileJob)

	if fileJob.Complexity != 10 {
		t.Errorf("Expected complexity 10 got %d", fileJob.Complexity)
	}
}

// VBA has no AndAlso or OrElse, so its And and Or count instead.
func TestCountStatsVisualBasicForApplicationsComplexity(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Visual Basic for Applications",
	}

	fileJob.SetContent(`If a And b Then
    For Each c In d
        If c Or e Then Exit For ' found it
    Next
ElseIf f Then
    Do While g
    Loop
End If ' done
Select Case h
    Case 1
    Case Else
End Select
`)

	CountStats(&fileJob)

	if fileJob.Complexity != 9 {
		t.Errorf("Expected complexity 9 got %d", fileJob.Complexity)
	}
}

// The Rem inside x.Remove(0) is not a comment, so the rest of the line is
// still read: the string opened after it runs on to the next line, and the
// OrElse after it counts. Classifying the content takes the slower scan, which
// has to agree.
func TestCountStatsVisualBasicRemInsideWord(t *testing.T) {
	ProcessConstants()

	for _, classify := range []bool{false, true} {
		fileJob := FileJob{
			Language:        "Visual Basic",
			ClassifyContent: classify,
		}

		fileJob.SetContent(`Dim s = x.Remove(0) & "first
' still in the string"
' a comment
If x.Remove(1) OrElse y Then
End If
`)

		CountStats(&fileJob)

		if fileJob.Code != 4 {
			t.Errorf("classify %v: Expected 4 code got %d", classify, fileJob.Code)
		}
		if fileJob.Comment != 1 {
			t.Errorf("classify %v: Expected 1 comments got %d", classify, fileJob.Comment)
		}
		if fileJob.Complexity != 2 {
			t.Errorf("classify %v: Expected complexity 2 got %d", classify, fileJob.Complexity)
		}
	}
}
