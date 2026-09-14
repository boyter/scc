// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// How an anchor is chosen, written down and made reproducible.
//
// The stop rate table in spec 07 00-goal.md drives the phase ordering and the
// ship-or-not criterion for every counter, and the method that produced it was
// not recorded anywhere. This is that method. See 03-architecture §5.
//
// The rule, in one line: for each complexity check, anchor it on the byte of
// that check which is rarest in real code of the language, subject to two
// constraints that are not negotiable.
//
//	1. A check may not be anchored on a byte that opens or closes a quote, a
//	   line comment or a block comment. Reading backwards from such a byte can
//	   cross out of the code the scan is in. TestCounterAnchoringCollisions
//	   computes the colliding set per language; where it is not empty the
//	   counter must either anchor those checks on their first byte or carry a
//	   written argument for why the read is still sound.
//
//	2. Two checks may share an anchor only where the byte behind the anchor
//	   tells them apart, and no check may hold the anchor of another check in a
//	   position where reading back from it would match. The stop table comment
//	   in each counter carries that argument, and
//	   TestCounterAnchorsAreInTheStopTable holds the anchors to being bytes the
//	   checks are actually spelled with.
//
// The stop rate that comes out of it is the fraction of corpus bytes the scan
// has to stop on, which is what the table in 00-goal.md reports. countStopRate
// below computes it for an arbitrary table, so a phase arguing about whether it
// met its target has a number rather than a recollection.

// countStopRate reports the fraction of content the scan stops on with a given
// table. It is the measurement the anchor choice is made against.
func countStopRate(content []byte, stop *[256]bool) (stops, total int) {
	for _, b := range content {
		if stop[b] {
			stops++
		}
	}

	return stops, len(content)
}

// firstByteStop is the naive table: stop on the first byte of every complexity
// check, which is what the Java counter did before anchoring and what the
// generic loop's TokenFirst does. It is the baseline an anchored table is
// measured against.
func firstByteStop(language Language, base *[256]bool) [256]bool {
	table := *base
	for _, check := range languageChecks(language) {
		if check != "" {
			table[check[0]] = true
		}
	}

	return table
}

// TestAnchorSelectionIsRecorded is the method above run against whatever corpus
// is to hand, so the numbers are reproducible rather than remembered. It never
// fails on a rate: a corpus is a sample and a threshold on one would be a test
// that fails on somebody else's machine. It fails only where anchoring made the
// scan stop on more bytes rather than fewer, which is a mistake and not a
// sample.
//
// Point SCC_ANCHOR_CORPUS at a tree of real code to measure against it;
// examples/language is used otherwise, which is small but is checked in and so
// always gives a number.
func TestAnchorSelectionIsRecorded(t *testing.T) {
	ProcessConstants()

	corpus := os.Getenv("SCC_ANCHOR_CORPUS")
	if corpus == "" {
		corpus = filepath.Join("..", "examples", "language")
	}

	// The extensions each counted language is sampled by. A real tree is walked
	// with scc's own detection in the layer 3 differential; here the extension
	// is enough and keeps the test free of it.
	for _, sample := range []struct {
		language  string
		extension string
	}{
		{"C", ".c"},
		{"C Header", ".h"},
		{"Java", ".java"},
		{"JavaScript", ".js"},
	} {
		var spec counterSpec
		for _, candidate := range counterSpecs() {
			if candidate.Language == sample.language {
				spec = candidate
			}
		}
		if spec.Language == "" {
			t.Errorf("%s has no counter spec", sample.language)
			continue
		}

		var content []byte
		_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !strings.HasSuffix(path, sample.extension) {
				return nil
			}
			if len(content) > 32<<20 {
				return filepath.SkipAll
			}
			read, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			content = append(content, read...)

			return nil
		})

		if len(content) == 0 {
			t.Logf("%s: no %s files under %s, nothing measured", sample.language, sample.extension, corpus)
			continue
		}

		naiveTable := firstByteStop(languageDatabase[sample.language], spec.StopNoComplexity)
		naive, total := countStopRate(content, &naiveTable)
		anchored, _ := countStopRate(content, spec.Stop)

		naiveRate := float64(naive) * 100 / float64(total)
		anchoredRate := float64(anchored) * 100 / float64(total)

		t.Logf("%-9s %7d bytes  naive %5.1f%%  anchored %5.1f%%  %.2fx fewer stops",
			sample.language, total, naiveRate, anchoredRate, naiveRate/anchoredRate)

		if anchored > naive {
			t.Errorf("%s: anchoring made the scan stop on more bytes, %.1f%% against %.1f%%",
				sample.language, anchoredRate, naiveRate)
		}
	}
}

// The byte frequencies the anchor choice is made from, printed rather than
// asserted. Run with -v against a corpus to pick anchors for a new language:
// the rarest byte of each check that is not in the collision set is the anchor.
func TestByteFrequencyForAnchorChoice(t *testing.T) {
	corpus := os.Getenv("SCC_ANCHOR_CORPUS")
	if corpus == "" {
		t.Skip("set SCC_ANCHOR_CORPUS to a tree of real code to measure byte frequencies")
	}

	extension := os.Getenv("SCC_ANCHOR_EXTENSION")
	if extension == "" {
		t.Skip("set SCC_ANCHOR_EXTENSION to the extension to sample, such as .rs")
	}

	var counts [256]int
	total := 0
	_ = filepath.Walk(corpus, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, extension) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, b := range content {
			counts[b]++
		}
		total += len(content)

		return nil
	})

	if total == 0 {
		t.Skipf("no %s files under %s", extension, corpus)
	}

	type frequency struct {
		b     byte
		share float64
	}
	var frequencies []frequency
	for b := range 256 {
		if counts[b] == 0 {
			continue
		}
		frequencies = append(frequencies, frequency{byte(b), float64(counts[b]) * 100 / float64(total)})
	}
	sort.Slice(frequencies, func(i, j int) bool { return frequencies[i].share < frequencies[j].share })

	t.Logf("%d bytes of %s, rarest first", total, extension)
	for _, f := range frequencies {
		if f.share > 5 {
			break
		}
		t.Logf("  %q %.2f%%", f.b, f.share)
	}
}
