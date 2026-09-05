// SPDX-License-Identifier: MIT

package processor

import (
	"math/rand"
	"testing"
)

// The specialised counters look at the byte in front of a quote to decide
// whether it is escaped. The state machine starts blank, so the code and string
// states cannot be entered at the first byte and there is always a byte in
// front, but that is an argument about the whole loop rather than anything the
// read itself enforces. These pin the behaviour so a future rearrangement of
// the states fails here rather than panicking on a file that opens with a
// quote.

func countWithoutPanic(t *testing.T, language string, content []byte) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic counting %s content %q: %v", language, content, r)
		}
	}()

	fileJob := &FileJob{Language: language, Content: content, Bytes: int64(len(content))}
	CountStats(fileJob)
}

var countersBoundsLanguages = []string{"C", "C Header", "Java"}

// TestSpecialisedCountersShortContent walks every string of up to three bytes
// over the bytes that mean something to these counters, with and without a byte
// order mark, since the mark moves the index the loop starts from.
func TestSpecialisedCountersShortContent(t *testing.T) {
	ProcessConstants()
	SpecialisedCounters = true
	defer func() { SpecialisedCounters = false }()

	alphabet := []byte{'"', '\'', '\\', '/', '*', '\n', '\r', ' ', '\t', 0, 'a', '{', '#'}

	var contents [][]byte
	for _, a := range alphabet {
		contents = append(contents, []byte{a})
		for _, b := range alphabet {
			contents = append(contents, []byte{a, b})
			for _, c := range alphabet {
				contents = append(contents, []byte{a, b, c})
			}
		}
	}

	bom := []byte{0xEF, 0xBB, 0xBF}
	for _, content := range append([][]byte{}, contents...) {
		contents = append(contents, append(append([]byte{}, bom...), content...))
	}
	contents = append(contents, []byte{}, nil)

	for _, language := range countersBoundsLanguages {
		for _, content := range contents {
			countWithoutPanic(t, language, append([]byte{}, content...))
		}
	}
}

// TestSpecialisedCountersRandomContent covers the arrangements the exhaustive
// walk above is too short to reach, such as a quote reached after a run of
// backslashes.
func TestSpecialisedCountersRandomContent(t *testing.T) {
	ProcessConstants()
	SpecialisedCounters = true
	defer func() { SpecialisedCounters = false }()

	alphabet := []byte(`"'\/*` + "\n\r\t {}#abc=!|&")
	random := rand.New(rand.NewSource(1))

	for _, language := range countersBoundsLanguages {
		for i := 0; i < 20000; i++ {
			content := make([]byte, random.Intn(24))
			for j := range content {
				content[j] = alphabet[random.Intn(len(alphabet))]
			}
			countWithoutPanic(t, language, content)
		}
	}
}
