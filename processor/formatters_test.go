package processor

import (
	"testing"
)

// When using columise  ~28726 ns/op
// When using optimised ~11293 ns/op
func BenchmarkFileSummerize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		fileSummaryJobQueue := make(chan *FileJob, 1000)

		fileSummaryJobQueue <- &FileJob{
			Blank:      1,
			Bytes:      1,
			Code:       1,
			Comment:    1,
			Complexity: 1,
			Language:   "Go",
			Lines:      10,
		}
		close(fileSummaryJobQueue)
		b.StartTimer()

		fileSummerize(&fileSummaryJobQueue)
	}
}
