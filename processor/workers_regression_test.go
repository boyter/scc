package processor

import "testing"

// https://github.com/boyter/scc/issues/72
// Turns out the above is due to BOM being present for that file
func TestCountStatsIssue72(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "C#",
	}

	fileJob.SetContent(`   // Comment 1
namespace Baz
{
    using System;

    public class FooClass
    {
        public void Test(string report)
        {
          // Comment 2
          throw new NotImplementedException();
        }
    }
}`)

	// Set the BOM
	fileJob.Content[0] = 239
	fileJob.Content[1] = 187
	fileJob.Content[2] = 191

	CountStats(&fileJob)

	if fileJob.Lines != 14 {
		t.Errorf("Expected 14 lines")
	}

	if fileJob.Code != 11 {
		t.Errorf("Expected 11 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsPr76(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Go",
	}

	fileJob.SetContent(`package main
var MyString = ` + "`\\`" + `
// Comment`)

	CountStats(&fileJob)

	if fileJob.Lines != 3 {
		t.Errorf("Expected 3 lines")
	}

	if fileJob.Code != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}

// https://github.com/boyter/scc/issues/62
func TestCountStatsIssue62(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Python",
	}

	fileJob.SetContent(`def f():
	"""
	This is a docstring
	"""
	# A normal comment

	hello_world = "Some string declaration"
	print(hello_world)
	pass

	def g():
	'''
	This is a not PEP-8 conform docstring'''
	pass
`)

	CountStats(&fileJob)

	if fileJob.Lines != 14 {
		t.Errorf("Expected 14 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 6 {
		t.Errorf("Expected 6 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Blank)
	}
}

func TestCountStatsIssue123(t *testing.T) {
	ProcessConstants()
	fileJob := FileJob{
		Language: "Python",
	}

	fileJob.SetContent(`"""
hello there! how's it going?
"""`)

	CountStats(&fileJob)

	if fileJob.Lines != 2 {
		t.Errorf("Expected 2 lines got %d", fileJob.Lines)
	}

	if fileJob.Code != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Code)
	}

	if fileJob.Comment != 1 {
		t.Errorf("Expected 1 lines got %d", fileJob.Comment)
	}

	if fileJob.Blank != 0 {
		t.Errorf("Expected 0 lines got %d", fileJob.Blank)
	}
}
