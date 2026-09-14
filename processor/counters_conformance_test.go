// SPDX-License-Identifier: MIT

package processor

import (
	"sort"
	"strings"
	"testing"
)

// The anti-drift layer, and the reason hand-writing a counter is safe rather
// than reckless. It runs with no corpus, no env var and no file counted, and it
// fails at go test time the moment someone edits languages.json without
// touching the counter that answers for that language.
//
// Four things are asserted, and the fourth is the one that catches the drift
// that matters: a token set can go on matching perfectly while a newly added
// check quietly invalidates a backwards read. See spec 07 04-testing §2.

// sortedBytes is a set of bytes written out as a string, which is the form the
// expected answers are recorded in and the form a failure is readable in.
func sortedBytes(set map[byte]struct{}) string {
	out := make([]byte, 0, len(set))
	for b := range set {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return string(out)
}

func bytesOf(tokens []string) map[byte]struct{} {
	set := map[byte]struct{}{}
	for _, token := range tokens {
		for i := 0; i < len(token); i++ {
			set[token[i]] = struct{}{}
		}
	}

	return set
}

// languageChecks is every complexity check of a language, prefix and postfix
// together, which is the whole of what a counter's anchor map has to cover.
func languageChecks(language Language) []string {
	return append(append([]string{}, language.ComplexityChecks...), language.ComplexityChecksPostfix...)
}

// languageDelimiters is every token that opens or closes a quote, a line
// comment or a block comment. These are the bytes a backwards read from an
// anchor must never be able to cross.
func languageDelimiters(language Language) []string {
	delimiters := append([]string{}, language.LineComment...)
	for _, pair := range language.MultiLine {
		delimiters = append(delimiters, pair...)
	}
	for _, quote := range language.Quotes {
		delimiters = append(delimiters, quote.Start, quote.End)
	}

	return delimiters
}

// languageOpeners is every token that can take the scan out of code: a line
// comment, the opener of a block comment, the start of a quote. A closer is
// deliberately not here. The state that looks for one is already inside the
// region it closes and finds it with bytes.Index rather than with the stop
// table, so a counter has no reason to stop on it while scanning code.
func languageOpeners(language Language) []string {
	openers := append([]string{}, language.LineComment...)
	for _, pair := range language.MultiLine {
		openers = append(openers, pair[0])
	}
	for _, quote := range language.Quotes {
		openers = append(openers, quote.Start)
	}

	return openers
}

func sortedStrings(in []string) []string {
	out := append([]string{}, in...)
	sort.Strings(out)

	return out
}

// Assertion 1. Every token the counter says it handles is a token the language
// database says the language has, and the other way about. A check added to
// languages.json that the counter knows nothing of fails here.
func TestCounterTokensMatchTheLanguageDatabase(t *testing.T) {
	ProcessConstants()

	for _, spec := range counterSpecs() {
		language, ok := languageDatabase[spec.Language]
		if !ok {
			t.Errorf("%s: counter declares a language the database does not have", spec.Language)
			continue
		}

		declared := make([]string, 0, len(spec.Anchors))
		for check := range spec.Anchors {
			declared = append(declared, check)
		}

		if got, want := sortedStrings(declared), sortedStrings(languageChecks(language)); !equalStrings(got, want) {
			t.Errorf("%s: complexity checks drifted\n  counter : %q\n  database: %q", spec.Language, got, want)
		}

		if got, want := sortedStrings(spec.LineComments), sortedStrings(language.LineComment); !equalStrings(got, want) {
			t.Errorf("%s: line comments drifted\n  counter : %q\n  database: %q", spec.Language, got, want)
		}

		var quotes []string
		for _, quote := range language.Quotes {
			quotes = append(quotes, quote.Start, quote.End)
		}
		if got, want := sortedStrings(spec.Quotes), sortedStrings(quotes); !equalStrings(got, want) {
			t.Errorf("%s: quotes drifted\n  counter : %q\n  database: %q", spec.Language, got, want)
		}

		var blocks, wantBlocks []string
		for _, pair := range spec.BlockComments {
			blocks = append(blocks, strings.Join(pair, " "))
		}
		for _, pair := range language.MultiLine {
			wantBlocks = append(wantBlocks, strings.Join(pair, " "))
		}
		if got, want := sortedStrings(blocks), sortedStrings(wantBlocks); !equalStrings(got, want) {
			t.Errorf("%s: block comments drifted\n  counter : %q\n  database: %q", spec.Language, got, want)
		}
	}
}

// Assertion 2. The counter stops wherever the generic loop stops, apart from
// the bytes anchoring deliberately drops.
//
// It is not the literal superset of TokenFirst that a naive counter would be,
// and it must not be: anchoring is exactly the trick of not stopping on the
// first byte of a keyword. So the assertion is that every byte TokenFirst holds
// is either the first byte of a complexity check, which assertion 3 covers
// instead, or a byte the counter also stops on.
func TestCounterStopTableCoversTheGenericLoop(t *testing.T) {
	ProcessConstants()

	for _, spec := range counterSpecs() {
		language := languageDatabase[spec.Language]

		LanguageFeaturesMutex.Lock()
		features := LanguageFeatures[spec.Language]
		LanguageFeaturesMutex.Unlock()

		if features.TokenFirst == nil {
			t.Fatalf("%s: no token table was built for the language", spec.Language)
		}

		checkFirst := map[byte]struct{}{}
		for _, check := range languageChecks(language) {
			checkFirst[check[0]] = struct{}{}
		}

		for b := range 256 {
			if !features.TokenFirst[byte(b)] {
				continue
			}
			if _, anchored := checkFirst[byte(b)]; anchored {
				continue
			}
			if !spec.Stop[byte(b)] {
				t.Errorf("%s: the generic loop stops on %q and the counter does not", spec.Language, byte(b))
			}
		}

		// The smaller table is what --no-complexity runs with, and the same
		// argument holds for it minus the checks, which are not in the trie at
		// all under that flag.
		for _, token := range languageOpeners(language) {
			if token == "" {
				continue
			}
			if !spec.StopNoComplexity[token[0]] {
				t.Errorf("%s: --no-complexity does not stop on %q, the first byte of %q", spec.Language, token[0], token)
			}
		}

		for _, b := range []byte{'\n', 0} {
			if !spec.Stop[b] || !spec.StopNoComplexity[b] {
				t.Errorf("%s: does not stop on %q, which ends a line or marks the file binary", spec.Language, b)
			}
		}
	}
}

// Assertion 3. Every anchor is a byte the scan stops on and a byte the check is
// actually spelled with. An anchor outside the stop table means the matcher is
// never called; an anchor that is not in the check means it is called on the
// wrong byte.
func TestCounterAnchorsAreInTheStopTable(t *testing.T) {
	ProcessConstants()

	for _, spec := range counterSpecs() {
		for check, anchor := range spec.Anchors {
			if !spec.Stop[anchor] {
				t.Errorf("%s: %q is anchored on %q, which is not in the stop table", spec.Language, check, anchor)
			}
			if !strings.ContainsRune(check, rune(anchor)) {
				t.Errorf("%s: %q is anchored on %q, which the check is not spelled with", spec.Language, check, anchor)
			}
		}
	}
}

// Assertion 4, and the one that earns its keep. Anchoring means reading
// backwards from a byte in the middle of a keyword, and that is only sound
// while no byte a check is spelled with can also open or close a quote or a
// comment. Where one can, the read may cross out of the code the scan is in and
// the counter has to carry a written argument for why it does not.
//
// The expected answers are recorded per language. A check added to Ruby that
// happens to hold a b would otherwise pass every other assertion here while
// silently invalidating the backwards read.
func TestCounterAnchoringCollisions(t *testing.T) {
	ProcessConstants()

	for _, spec := range counterSpecs() {
		language := languageDatabase[spec.Language]

		delimiters := bytesOf(languageDelimiters(language))
		collisions := map[byte]struct{}{}
		for b := range bytesOf(languageChecks(language)) {
			if _, ok := delimiters[b]; ok {
				collisions[b] = struct{}{}
			}
		}

		if got := sortedBytes(collisions); got != spec.Collisions {
			t.Errorf("%s: anchoring collisions changed, got %q want %q\n"+
				"a byte a check is spelled with now also opens or closes a quote or a comment, "+
				"so the backwards read from an anchor can cross a token boundary. Either prove it "+
				"cannot for this language and record the argument in the counter, or anchor the "+
				"colliding checks on their first byte.",
				spec.Language, got, spec.Collisions)
		}
	}
}

// The same collision check run over all sixteen languages a counter is planned
// for, so the answer is recorded before the counter that depends on it is
// written rather than after.
//
// Thirteen are empty and anchoring is sound for them as written. The three that
// are not each need an argument in their own counter, per spec 07
// 03-architecture §2.1: Python's r and f open the raw and formatted string
// prefixes and are also spelled in for, if, finally and elif; Ruby's = is
// spelled in != and == and opens =begin and =end, and its e and i are in both
// the block comment delimiters and else, if and while; Rust's r opens r" and
// br" and is spelled in for.
func TestAnchoringCollisionsForEveryPlannedCounter(t *testing.T) {
	ProcessConstants()

	for _, want := range []struct {
		language   string
		collisions string
	}{
		{"C", ""},
		{"C Header", ""},
		{"C#", ""},
		{"C++", ""},
		{"C++ Header", ""},
		{"Go", ""},
		{"Java", ""},
		{"JavaScript", ""},
		{"Kotlin", ""},
		{"PHP", ""},
		{"Scala", ""},
		{"Swift", ""},
		{"TypeScript", ""},
		{"Python", "fr"},
		{"Ruby", "=ei"},
		{"Rust", "r"},
	} {
		language, ok := languageDatabase[want.language]
		if !ok {
			t.Errorf("%s: not in the language database", want.language)
			continue
		}

		delimiters := bytesOf(languageDelimiters(language))
		collisions := map[byte]struct{}{}
		for b := range bytesOf(languageChecks(language)) {
			if _, ok := delimiters[b]; ok {
				collisions[b] = struct{}{}
			}
		}

		if got := sortedBytes(collisions); got != want.collisions {
			t.Errorf("%s: anchoring collisions are now %q, recorded as %q", want.language, got, want.collisions)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}
