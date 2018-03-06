package processor

import (
	"testing"
)

func TestCountStatsLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	countStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One lines expected %d", fileJob.Lines)
	}

	fileJob.Content = []byte("import this")
	countStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One lines expected %d", fileJob.Lines)
	}

	fileJob.Content = []byte("\n")
	countStats(&fileJob)
	if fileJob.Lines != 2 {
		t.Errorf("Two lines expected %d", fileJob.Lines)
	}

	fileJob.Content = []byte("1\n2\n3")
	countStats(&fileJob)
	if fileJob.Lines != 3 {
		t.Errorf("Three lines expected %d", fileJob.Lines)
	}
}
