package processor

import (
	"time"
)

func makeTimestampMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

func makeTimestampNano() int64 {
	return time.Now().UnixNano()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
