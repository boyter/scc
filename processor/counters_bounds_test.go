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

var countersBoundsLanguages = counterLanguages()

// TestSpecialisedCountersShortContent walks every string of up to three bytes
// over the bytes that mean something to these counters, with and without a byte
// order mark, since the mark moves the index the loop starts from.
func TestSpecialisedCountersShortContent(t *testing.T) {
	ProcessConstants()
	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })
	SpecialisedCounters = true

	// h and y are the anchors Java and Kotlin read furthest back from: four
	// bytes for the catch behind an h, two for the try behind a y. A string of
	// three bytes cannot hold either keyword, which is the point: it puts the
	// anchor where the read runs off the front of the file.
	//
	// ? is Swift's whole complexity check and JavaScript's postfix three, and
	// < and > are Scala's four bracket checks. Each is a stop byte that is not
	// a letter, so a short string of them reaches the matcher with nothing in
	// front of it and nothing behind.
	// R and ( are C++: a raw string is recognised by reading R back from its
	// quote and its closer is read out of the file between the two, so a bare
	// R" or R"( at the end of a file is where that read can run off the end.
	alphabet := []byte{'"', '\'', '\\', '/', '*', '\n', '\r', ' ', '\t', 0, 'a', '{', '#', 'h', 'y', '?', '<', '>', '@', '`', 'R', '(', '=', 'b', 'e', 'r', 'f'}

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
	previous := SpecialisedCounters
	t.Cleanup(func() { SpecialisedCounters = previous })
	SpecialisedCounters = true

	// Every letter the complexity checks of these languages are spelled with, so
	// a random string can assemble a keyword, a near miss of one, and an anchor
	// with nothing behind it.
	alphabet := []byte(`"'\/*()` + "\n\r\t {}#=!|&?<>@`" + "abcdefghilnopRrstuUwxyL8")
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
