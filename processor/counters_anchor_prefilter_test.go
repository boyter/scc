// SPDX-License-Identifier: MIT

package processor

import (
	"math/rand"
	"testing"
)

// The anchor prefilter is the first test of <lang>ComplexityAnchored - the byte
// in front of the anchor - lifted out of the function and into the caller, which
// already holds that byte and would otherwise pay for a call to learn it says
// no. <lang>AnchorPrev[prev] & <lang>AnchorBit[anchor] == 0 means skip.
//
// The tables are only allowed to be wrong in one direction. A bit that is set
// where it need not be costs a call that would have been made anyway, and no
// count changes. A bit that is clear where the function could have said yes
// loses a complexity count silently, on some file nobody is looking at. So the
// tables must be a conservative superset, and these tests are what holds them to
// it:
//
//   - TestAnchorPrefilterCoversEveryAnchor: every anchor byte of the language
//     has a bit. An anchor with no bit ANDs to zero against everything, which
//     would skip the call always rather than never - the same silent loss
//     written the other way round.
//   - TestAnchorPrefilterKeepsEveryCheck: every complexity check the language
//     database gives the counter is still reached, spelled as real code.
//   - TestAnchorPrefilterIsConservative: over all 256 previous bytes and every
//     anchor, every pair the prefilter clears is a pair the function rejects.
//     The contents are built out of the language's own check spellings, laid
//     over a buffer at every offset that puts an anchor under the cursor, and
//     then read at five floors and with the byte in front forced to each of the
//     256. Every path to true in these functions ends in a literal match against
//     one of those spellings, so a content that holds no corrupted spelling in
//     the window has nothing to say; the random pass covers what the templates
//     do not.

// anchorPrefilter is one counter's prefilter, the function it stands in front
// of, and the check list both are derived from.
type anchorPrefilter struct {
	Language string
	Bit      [256]uint16
	Prev     [256]uint16
	Anchored func(content []byte, index, floor int) bool
	Checks   map[string]byte
	// CasedEarly are anchor bytes the code scan answers in a case of their own
	// before it reaches default, so the prefilter never sees them and a bit for
	// them would mean nothing. Rust's ? is the only one: rustCodeState counts it
	// where it stands and rustComplexityAnchored has no arm for it at all.
	CasedEarly []byte
}

// widen carries a counter's [256]uint8 tables into the uint16 the table below
// is written in. Swift needs sixteen bits because it anchors nine checks; every
// other language fits in eight, and this is what lets one test hold them all.
func widen(table [256]uint8) [256]uint16 {
	var out [256]uint16
	for index, value := range table {
		out[index] = uint16(value)
	}

	return out
}

func anchorPrefilters() []anchorPrefilter {
	return []anchorPrefilter{
		{"C", widen(cAnchorBit), widen(cAnchorPrev), cComplexityAnchored, cComplexityAnchors, nil},
		{"C++", widen(cppAnchorBit), widen(cppAnchorPrev), cppComplexityAnchored, cppComplexityAnchors, nil},
		{"Java", widen(javaAnchorBit), widen(javaAnchorPrev), javaComplexityAnchored, javaComplexityAnchors, nil},
		{"C#", widen(csharpAnchorBit), widen(csharpAnchorPrev), csharpComplexityAnchored, csharpComplexityAnchors, nil},
		{"Kotlin", widen(kotlinAnchorBit), widen(kotlinAnchorPrev), kotlinComplexityAnchored, kotlinComplexityAnchors, nil},
		{"Go", widen(goAnchorBit), widen(goAnchorPrev), goComplexityAnchored, goComplexityAnchors, nil},
		{"Rust", widen(rustAnchorBit), widen(rustAnchorPrev), rustComplexityAnchored, rustComplexityAnchors, []byte{'?'}},
		{"Scala", widen(scalaAnchorBit), widen(scalaAnchorPrev), scalaComplexityAnchored, scalaComplexityAnchors, nil},
		{"Swift", swiftAnchorBit, swiftAnchorPrev, swiftComplexityAnchored, swiftComplexityAnchors, nil},
		{"Python", widen(pythonAnchorBit), widen(pythonAnchorPrev), pythonComplexityAnchored, pythonComplexityAnchors, nil},
		{"LLVM IR", widen(llvmAnchorBit), widen(llvmAnchorPrev), llvmComplexityAnchored, llvmComplexityAnchors, nil},
		{"Assembly", widen(asmAnchorBit), widen(asmAnchorPrev), asmComplexityAnchored, asmComplexityAnchors, nil},
	}
}

// skipped is the production test, written once so the tests cannot drift from
// the counters.
// casedEarly reports whether the code scan answers this byte before the default
// arm, so the prefilter is never consulted for it.
func (p *anchorPrefilter) casedEarly(b byte) bool {
	for _, value := range p.CasedEarly {
		if value == b {
			return true
		}
	}

	return false
}

func (p *anchorPrefilter) skipped(content []byte, index, floor int) bool {
	return p.Prev[byteBefore(content, index, floor)]&p.Bit[content[index]] == 0
}

// TestAnchorPrefilterCoversEveryAnchor holds the bit table to the check list:
// every byte some check is anchored on has a bit, and no other byte has one.
func TestAnchorPrefilterCoversEveryAnchor(t *testing.T) {
	for _, prefilter := range anchorPrefilters() {
		var wanted [256]bool
		for _, anchor := range prefilter.Checks {
			if prefilter.casedEarly(anchor) {
				continue
			}
			wanted[anchor] = true
		}

		for value := range 256 {
			has := prefilter.Bit[value] != 0
			if has != wanted[value] {
				t.Errorf("%s: byte %q has a bit %v, is an anchor %v",
					prefilter.Language, byte(value), has, wanted[value])
			}
		}

		// Two anchors sharing a bit would let one anchor's previous byte admit
		// the other's, which is only wasteful, but it is never what was meant.
		bits := map[uint16]byte{}
		for value := range 256 {
			bit := prefilter.Bit[value]
			if bit == 0 {
				continue
			}
			if other, ok := bits[bit]; ok {
				t.Errorf("%s: %q and %q share a bit", prefilter.Language, other, byte(value))
			}
			bits[bit] = byte(value)
		}
	}
}

// TestAnchorPrefilterKeepsEveryCheck writes each complexity check of the
// language as real code and asserts the counter still reaches it: the function
// says yes somewhere in the line, and the prefilter does not skip that byte.
// This is the direction that loses counts.
func TestAnchorPrefilterKeepsEveryCheck(t *testing.T) {
	// Every context a check can appear in that the code scan reaches, which is
	// anywhere but the first byte of code on the line - that one is
	// <lang>ComplexityAtLineStart's and has its own tests.
	leads := []string{" ", "\t", ") ", "; ", "} ", "x) ", "0 "}

	for _, prefilter := range anchorPrefilters() {
		for spelling, anchor := range prefilter.Checks {
			if prefilter.casedEarly(anchor) {
				// Exempt because the scan answers it earlier - which is only
				// true while the matcher has no arm for it. If one is ever
				// added, the exemption is wrong and this says so.
				content := []byte(" " + spelling + "x;")
				for index := range content {
					if content[index] == anchor && prefilter.Anchored(content, index, 0) {
						t.Errorf("%s: %q is listed as answered before the default arm, but the matcher counts it",
							prefilter.Language, spelling)
					}
				}

				continue
			}

			kept := false
			for _, lead := range leads {
				content := []byte(lead + spelling + "x;")
				for index := range content {
					if content[index] != anchor {
						continue
					}
					if !prefilter.Anchored(content, index, 0) {
						continue
					}
					if prefilter.skipped(content, index, 0) {
						t.Errorf("%s: %q in %q: the prefilter skips a check the function counts",
							prefilter.Language, spelling, content)
					}
					kept = true
				}
			}

			if !kept {
				t.Errorf("%s: %q was not counted in any context, so this test proves nothing about it",
					prefilter.Language, spelling)
			}
		}
	}
}

// TestAnchorPrefilterIsConservative is the whole of the argument: for every one
// of the 256 possible previous bytes against every anchor of every language,
// each pair the prefilter clears is asserted to be a pair the unfiltered
// function rejects, over contents built from the language's own check spellings.
func TestAnchorPrefilterIsConservative(t *testing.T) {
	const (
		index  = 10
		length = 24
	)

	fillers := []byte{' ', 'z'}
	tails := []byte{'(', '{', ';'}

	for _, prefilter := range anchorPrefilters() {
		var covered [256][256]bool
		buffer := make([]byte, length)

		for spelling := range prefilter.Checks {
			for offset := range len(spelling) {
				if prefilter.Bit[spelling[offset]] == 0 {
					// Nothing under the cursor to prefilter.
					continue
				}

				for _, filler := range fillers {
					for _, tail := range tails {
						for previous := range 256 {
							for value := range buffer {
								buffer[value] = filler
							}
							copy(buffer[index-offset:], spelling)
							if end := index - offset + len(spelling); end < length {
								buffer[end] = tail
							}
							buffer[index-1] = byte(previous)

							// Five floors: the file start, and each of the
							// positions a backwards read is clamped at, which
							// is where wordStartsAt turns into a yes and
							// hasPrefixAt into a no.
							for _, floor := range []int{0, index - 5, index - 4, index - 2, index - 1} {
								front := byteBefore(buffer, index, floor)
								covered[front][buffer[index]] = true

								if prefilter.skipped(buffer, index, floor) &&
									prefilter.Anchored(buffer, index, floor) {
									t.Fatalf("%s: prefilter clears %q in front of %q but the function counts %q at %d floor %d",
										prefilter.Language, front, buffer[index], buffer, index, floor)
								}
							}
						}
					}
				}
			}
		}

		// The claim is over all 256 previous bytes and every anchor, so the
		// contents above have to have reached all of them.
		for value := range 256 {
			if prefilter.Bit[value] == 0 {
				continue
			}
			for previous := range 256 {
				if !covered[previous][value] {
					t.Errorf("%s: the pair %q in front of %q was never tested",
						prefilter.Language, byte(previous), byte(value))
				}
			}
		}
	}
}

// TestAnchorPrefilterIsConservativeOnRandomContent is the same claim made
// without the templates, which are built from a reading of the functions and so
// could share a blind spot with them. The alphabet is every byte the checks are
// spelled with plus the openers, the separators and the two bytes that end a
// line and a file, which is the whole of what these functions look at.
func TestAnchorPrefilterIsConservativeOnRandomContent(t *testing.T) {
	if testing.Short() {
		t.Skip("six million calls")
	}

	// Derived rather than written down: the bytes every check of every language
	// is spelled with, plus the openers, separators and terminators the
	// matchers look at. A hand written alphabet covered the C family and left
	// out Go's g, LLVM IR's b k m and Python's y d, so those anchors were never
	// put under the cursor.
	seen := map[byte]bool{}
	for _, prefilter := range anchorPrefilters() {
		for spelling := range prefilter.Checks {
			for index := range len(spelling) {
				seen[spelling[index]] = true
			}
		}
	}
	for _, b := range []byte("_09|&=!<>(){}[] ;:\t\n\x00\\\"/") {
		seen[b] = true
	}
	alphabet := make([]byte, 0, len(seen))
	for b := range 256 {
		if seen[byte(b)] {
			alphabet = append(alphabet, byte(b))
		}
	}
	random := rand.New(rand.NewSource(1))
	buffer := make([]byte, 20)

	for _, prefilter := range anchorPrefilters() {
		for range 200000 {
			for value := range buffer {
				buffer[value] = alphabet[random.Intn(len(alphabet))]
			}

			for index := range buffer {
				if prefilter.Bit[buffer[index]] == 0 {
					continue
				}
				for _, floor := range []int{0, index} {
					if prefilter.skipped(buffer, index, floor) &&
						prefilter.Anchored(buffer, index, floor) {
						t.Fatalf("%s: prefilter clears %q in front of %q but the function counts %q at %d floor %d",
							prefilter.Language, byteBefore(buffer, index, floor), buffer[index],
							buffer, index, floor)
					}
				}
			}
		}
	}
}

// TestAnchorPrefilterIsMinimal is the other half of
// TestAnchorPrefilterIsConservative, and together they pin the tables exactly
// rather than only safely: every bit that is set has a content the function
// really does count. A bit set for no reason is only a wasted call, so this one
// is not the invariant, but a bit nobody can find a witness for is a misreading
// of the function and worth knowing about.
func TestAnchorPrefilterIsMinimal(t *testing.T) {
	// Every byte a check can be written behind, which is what the previous byte
	// is drawn from, plus the identifier bytes the backwards reads want.
	leads := []string{" ", "\t", ";", ")", "}", "\n", "=", "!", "e", "i", "s", "r", "c", "\x00"}

	for _, prefilter := range anchorPrefilters() {
		var witnessed [256]uint16
		for spelling := range prefilter.Checks {
			for _, lead := range leads {
				content := []byte(lead + spelling + "x;")
				for index := range content {
					if prefilter.Bit[content[index]] == 0 || !prefilter.Anchored(content, index, 0) {
						continue
					}
					witnessed[byteBefore(content, index, 0)] |= prefilter.Bit[content[index]]
				}
			}
		}

		for previous := range 256 {
			unwitnessed := prefilter.Prev[previous] &^ witnessed[previous]
			// A word boundary is any byte that does not carry an identifier on,
			// and one witness per anchor stands for all of them.
			if !isIdentifierContinue(byte(previous)) {
				unwitnessed &^= witnessed[' ']
			}
			if unwitnessed != 0 {
				t.Errorf("%s: the table keeps the call for %q in front of anchor bits %016b, and nothing counts there",
					prefilter.Language, byte(previous), unwitnessed)
			}
		}
	}
}
