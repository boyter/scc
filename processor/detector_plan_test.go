// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"math/rand"
	"testing"
)

// literalPresentSlow is what the plan replaced: a scan of the whole file per
// literal. It is kept here as the thing the plan is checked against.
func literalPresentSlow(content, lit []byte, anchored bool) bool {
	if !anchored {
		return bytes.Contains(content, lit)
	}

	offset := 0
	for {
		i := bytes.Index(content[offset:], lit)
		if i < 0 {
			return false
		}
		pos := offset + i
		j := pos
		for j > 0 && (content[j-1] == ' ' || content[j-1] == '\t') {
			j--
		}
		if j == 0 || content[j-1] == '\n' {
			return true
		}
		offset = pos + 1
	}
}

// TestHeuristicPlanNeverMissesALiteral is the property that keeps the plan
// honest. The plan only decides whether a heuristic's regex is worth running,
// so it may say "possible" where the old per-literal scan said no — that only
// costs a regex run. It must never say "impossible" where the old scan found
// the literal, because that would silently disable a heuristic.
func TestHeuristicPlanNeverMissesALiteral(t *testing.T) {
	ProcessConstants()

	corpus := [][]byte{
		[]byte(""),
		[]byte("\n"),
		[]byte("#include <stdio.h>\n"),
		[]byte("#include <vector>\nclass Foo {\npublic:\n};\n"),
		[]byte("  \ttemplate <typename T>\nstruct S {};\n"),
		[]byte("@interface Thing : NSObject\n@end\n"),
		[]byte("#import <Foundation/Foundation.h>\n"),
		[]byte("struct class_attribute { int classy; };\nentry try trying\n"),
		[]byte("std::vector<int> v;\n#if __cplusplus > 201703L\n#endif\n"),
		[]byte("namespace n { using namespace std; }\n"),
		[]byte("\r\n\ttry {\n} catch (...) {\n}\n"),
		[]byte("private:\nprotected:\npublic:\n"),
		[]byte("no markers of any kind in this plain text at all"),
		[]byte("__has_cpp_attribute(x)"),
		bytes.Repeat([]byte("int x; /* filler */\n"), 500),
		// Near misses for the substrings the singles are actually scanned for:
		// "bute" for __has_cpp_attribute, "plusplus" for __cplusplus and "::"
		// for std::. Each of these holds the substring and not the literal.
		[]byte("__attribute__((packed)) int x;\ndistribute\n"),
		[]byte("int plusplus; /* not __cplusplus */\n"),
		[]byte("label::\nns::thing\nnotstd::x\n"),
		[]byte("bute plusplus ::"),
		// And the literals sitting where the screen match is not the first one.
		[]byte("attribute distribute __has_cpp_attribute(x)\n"),
		[]byte("plusplus\nplusplus\n#if __cplusplus\n"),
		[]byte("a::b c::d std::vector\n"),
	}

	sets := [][]string{
		{"C Header", "C++ Header", "Objective C"},
		{"C++ Header", "Objective C"},
		{"C Header"},
	}
	// Every extension that maps to more than one language, so the property is
	// checked over the whole database rather than the handful above.
	for _, langs := range ExtensionToLanguage {
		if len(langs) > 1 {
			sets = append(sets, langs)
		}
	}

	for _, langs := range sets {
		for _, l := range langs {
			LoadLanguageFeature(l)
		}

		plan := planFor(langs)
		if plan.nlits > planFoundStack {
			t.Logf("language set %v has %d literals, over the stack size", langs, plan.nlits)
		}

		for _, content := range corpus {
			found := make([]bool, plan.nlits)
			plan.present(content, found)

			// Walk the same heuristics the plan was built from and check that
			// anything the old scan found the plan found too.
			for _, pl := range plan.langs {
				LanguageFeaturesMutex.Lock()
				features := LanguageFeatures[pl.name]
				LanguageFeaturesMutex.Unlock()

				for i, h := range features.Heuristics {
					ph := pl.heuristics[i]
					if len(ph.ids) == 0 {
						continue // always runs, nothing to miss
					}
					for j, lit := range h.Literals {
						want := literalPresentSlow(content, lit, h.Anchored)
						got := found[ph.ids[j]]
						if want && !got {
							t.Errorf("%s heuristic %d: plan missed literal %q (anchored=%v) in %q",
								pl.name, i, lit, h.Anchored, content)
						}
						// An unanchored literal is answered exactly, so a plan
						// claiming one the file does not hold is a bug in the
						// substring screen rather than a harmless over-report.
						if !h.Anchored && got && !want {
							t.Errorf("%s heuristic %d: plan invented literal %q in %q",
								pl.name, i, lit, content)
						}
					}
				}
			}
		}
	}
}

// TestPickScreenIsASubstring is the property the substring screen rests on: a
// file that does not hold the screen cannot hold the literal, so a screen that
// is not a substring of its literal at the recorded offset would silently
// disable a heuristic.
func TestPickScreenIsASubstring(t *testing.T) {
	for _, lit := range []string{
		"", "a", "ab", "std::", "__cplusplus", "__has_cpp_attribute",
		"<unordered_map>", "@implementation", "#import", "\x00\xff\xfe",
		"aaaa", "zzzz", "ezez",
	} {
		off, screen := pickScreen([]byte(lit))
		if screen == nil {
			continue
		}
		if off <= 0 || off+len(screen) > len(lit) {
			t.Fatalf("%q: screen %q at offset %d is out of range", lit, screen, off)
		}
		if lit[off:off+len(screen)] != string(screen) {
			t.Fatalf("%q: screen %q is not the literal at offset %d", lit, screen, off)
		}
	}
}

// TestScanShortLiterals drives the unanchored scans over literals shorter and
// more awkward than any language ships, including one that is a single byte and
// ones that only fit at the very end of the content, where the group scan has no
// second byte to sort by.
func TestScanShortLiterals(t *testing.T) {
	lits := [][]byte{
		[]byte("<a"), []byte("<b"), []byte("<ab"), []byte("<"), // a group on '<'
		[]byte("qz"), // a single, screened on its rarest byte
	}

	plan := &heuristicPlan{lits: lits, nlits: len(lits)}
	plan.groups = []planGroup{{first: '<', ids: []int32{0, 1, 2, 3}}}
	for _, id := range plan.groups[0].ids {
		if lit := lits[id]; len(lit) == 1 {
			plan.groups[0].short = append(plan.groups[0].short, id)
		} else {
			plan.groups[0].second[lit[1]] = append(plan.groups[0].second[lit[1]], id)
		}
	}
	off, screen := pickScreen(lits[4])
	plan.singles = []planSingle{{id: 4, screen: screen, off: off}}

	for _, content := range []string{
		"", "<", "x<", "<a", "x<a", "<ab", "<b", "<a<b", "a<", "q", "qz", "xqz", "qqz", "zq", "qzq",
		"<<<", "<a<ab<b", "the quick brown fox", "<z",
	} {
		found := make([]bool, plan.nlits)
		plan.present([]byte(content), found)
		for id, lit := range lits {
			if want := bytes.Contains([]byte(content), lit); want != found[id] {
				t.Errorf("content %q literal %q: plan said %v, search said %v", content, lit, found[id], want)
			}
		}
	}
}

// TestPlanAnswersUnanchoredExactly checks the unanchored half of every plan
// against a plain search, over content built to land the substring screens on
// text that is not the literal.
func TestPlanAnswersUnanchoredExactly(t *testing.T) {
	ProcessConstants()

	sets := [][]string{{"C Header", "C++ Header", "Objective C"}}
	for _, langs := range ExtensionToLanguage {
		if len(langs) > 1 {
			sets = append(sets, langs)
		}
	}

	rng := rand.New(rand.NewSource(1))

	for _, langs := range sets {
		for _, l := range langs {
			LoadLanguageFeature(l)
		}
		plan := planFor(langs)
		if plan.nlits == 0 {
			continue
		}

		// Content is built out of pieces of the plan's own literals so the
		// scans land on partial matches far more often than random bytes would.
		var pieces [][]byte
		for _, lit := range plan.lits {
			pieces = append(pieces, lit)
			for cut := 1; cut < len(lit); cut++ {
				pieces = append(pieces, lit[cut:], lit[:cut])
			}
		}
		pieces = append(pieces, []byte(" "), []byte("\n"), []byte("\t"), []byte("x"))

		for range 400 {
			var content []byte
			for range rng.Intn(40) {
				content = append(content, pieces[rng.Intn(len(pieces))]...)
			}

			found := make([]bool, plan.nlits)
			plan.present(content, found)

			for _, pl := range plan.langs {
				LanguageFeaturesMutex.Lock()
				features := LanguageFeatures[pl.name]
				LanguageFeaturesMutex.Unlock()

				for i, h := range features.Heuristics {
					if h.Anchored || len(pl.heuristics[i].ids) == 0 {
						continue
					}
					for j, lit := range h.Literals {
						want := bytes.Contains(content, lit)
						got := found[pl.heuristics[i].ids[j]]
						if want != got {
							t.Fatalf("%s literal %q: plan said %v, search said %v, in %q",
								pl.name, lit, got, want, content)
						}
					}
				}
			}
		}
	}
}
