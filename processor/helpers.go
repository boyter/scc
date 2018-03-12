package processor

import (
	"time"
)

func makeTimestampMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
