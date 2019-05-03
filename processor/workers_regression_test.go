package processor

import "testing"

// https://github.com/boyter/scc/issues/72
// Turns out the above is due to BOM being present for that file
func TestCountStatsIssue72(t *testing.T) {
	ProcessConstants()

	// Try out every since BOM that we are aware of
	for _, v := range ByteOrderMarks {
		fileJob := FileJob{
			Language: "C#",
		}

		fileJob.Content = append(v, []byte(`// Comment 1
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
}`)...)

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
}
