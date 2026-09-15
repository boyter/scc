// SPDX-License-Identifier: MIT

package processor

import (
	"math/rand"
	"testing"
)

// stopMask reads a [256]bool as bytes and shifts the result, which is only
// meaningful while a bool is one byte holding 0 or 1. The language does not
// promise that; gc does it, and this is what notices if that ever stops being
// true or if a table is ever written by something other than the builders.
func TestStopMaskMatchesTheTable(t *testing.T) {
	var table [256]bool
	for _, b := range []byte{0, '\n', '"', '/', '&', '|', '=', 'f', 'l', 'w', 0xff} {
		table[b] = true
	}

	mask := stopMask(&table)
	for b := 0; b < 256; b++ {
		want := uint8(0)
		if table[b] {
			want = 1
		}
		if mask[b] != want {
			t.Fatalf("byte %d: mask says %d, table says %t", b, mask[b], table[b])
		}
	}
}

// scanToStop has to answer exactly what walking the table one byte at a time
// answers, including at the boundaries where the eight wide step gives way to
// the one at a time tail.
func TestScanToStopAgreesWithTheWalk(t *testing.T) {
	var table [256]bool
	for _, b := range []byte{0, '\n', '"', '/'} {
		table[b] = true
	}
	mask := stopMask(&table)

	walk := func(content []byte, i, endPoint int) int {
		for ; i < endPoint; i++ {
			if table[content[i]] {
				return i
			}
		}

		return -1
	}

	random := rand.New(rand.NewSource(1))
	alphabet := []byte{'a', 'b', ' ', '\n', '"', '/', 0, 'x'}

	for length := 0; length <= 40; length++ {
		for trial := 0; trial < 200; trial++ {
			content := make([]byte, length)
			for i := range content {
				content[i] = alphabet[random.Intn(len(alphabet))]
			}
			for start := 0; start <= length; start++ {
				for end := start; end <= length; end++ {
					got := scanToStop(content, start, end, mask)
					want := walk(content, start, end)
					if got != want {
						t.Fatalf("content=%q start=%d end=%d: wide says %d, walk says %d",
							content, start, end, got, want)
					}
				}
			}
		}
	}
}
