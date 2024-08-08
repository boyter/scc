// SPDX-License-Identifier: MIT

package processor

import (
	"time"
)

// Returns the current time as a millisecond timestamp
func makeTimestampMilli() int64 {
	return time.Now().UnixMilli()
}

// Returns the current time as a nanosecond timestamp as some things
// are far too fast to measure using nanoseconds
func makeTimestampNano() int64 {
	return time.Now().UnixNano()
}
