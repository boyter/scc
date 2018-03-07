package processor

import (
	"testing"
)

func TestCountStatsLines(t *testing.T) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	// Both tokei and sloccount count this as 0 so lets follow suit
	// cloc ignores the file itself because it is empty
	countStats(&fileJob)
	if fileJob.Lines != 0 {
		t.Errorf("Zero lines expected got %d", fileJob.Lines)
	}

	// Interestingly this file would be 0 lines in "wc -l" because it only counts newlines
	// all others count this as 1
	fileJob.Content = []byte("import this")
	countStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One line expected got %d", fileJob.Lines)
	}

	// tokei counts this as 1 because its still on a single line unless something follows
	// the newline its still 1 line
	fileJob.Content = []byte("1\n")
	countStats(&fileJob)
	if fileJob.Lines != 1 {
		t.Errorf("One lines expected got %d", fileJob.Lines)
	}

	fileJob.Content = []byte("1\n2\n")
	countStats(&fileJob)
	if fileJob.Lines != 2 {
		t.Errorf("Two lines expected got %d", fileJob.Lines)
	}

	fileJob.Content = []byte("1\n2\n3")
	countStats(&fileJob)
	if fileJob.Lines != 3 {
		t.Errorf("Three lines expected got %d", fileJob.Lines)
	}

	content := ""
	for i := 0; i < 5000; i++ {
		content += "a\n"
		fileJob.Content = []byte(content)
		countStats(&fileJob)
		if fileJob.Lines != int64(i+1) {
			t.Errorf("Expected %d got %d", i+1, fileJob.Lines)
		}
	}
}

func BenchmarkCountStatsLinesEmpty(b *testing.B) {
	fileJob := FileJob{
		Content: []byte(""),
	}

	for i := 0; i < b.N; i++ {
		countStats(&fileJob)
	}
}

func BenchmarkCountStatsLinesSomething(b *testing.B) {
	fileJob := FileJob{
		Content: []byte("this is a test\nof some stuff\n to see how fast things go"),
	}

	for i := 0; i < b.N; i++ {
		countStats(&fileJob)
	}
}
