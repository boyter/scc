// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"math/rand"
	"testing"
)

// anchoredMatchSlow is what the line walk replaced for an anchored heuristic: a
// search of the whole file with the original pattern. The rewrite is only
// allowed to be faster, never to answer differently, and that is what these
// tests hold it to.
func anchoredMatchSlow(re interface{ Match([]byte) bool }, content []byte) bool {
	return re.Match(content)
}

// planAnchoredResults runs a plan over content and returns, per language, which
// of its heuristics the line walk decided.
func planAnchoredResults(plan *heuristicPlan, content []byte) []bool {
	found := make([]bool, plan.nlits)
	hit := make([]bool, len(plan.anchRe))
	plan.present(content, found, hit)
	return hit
}

// TestAnchoredFormAgreesWithTheSearch is the property the rewrite rests on. An
// anchored pattern can only match where a line's indentation ends, so trying it
// there has to give the same answer as searching the file for it. A disagreement
// would silently change which language a file is counted as.
func TestAnchoredFormAgreesWithTheSearch(t *testing.T) {
	ProcessConstants()

	corpus := [][]byte{
		[]byte(""),
		[]byte("\n"),
		[]byte("\n\n\n   \t\n"),
		[]byte("template <typename T>\n"),
		[]byte("   \t template\t<T>\n"),
		[]byte("template\n<T> x;\n"), // \s* across the newline
		[]byte("\n\n\n\ttemplate <T>"),
		[]byte("xtemplate <T>\n"),         // not at a line start
		[]byte("int template_arg <x>;\n"), // literal present, pattern not
		[]byte("class Foo {\npublic:\n};\n"),
		[]byte("  class\n"), // literal at a line start, pattern short of a match
		[]byte("using namespace std;\n"),
		[]byte("using\tnamespace  std;\n"),
		[]byte("using namespace\n"),
		[]byte("namespace {\n"),
		[]byte("public:\n"),
		[]byte("public: int x;\n"), // the $ has to fail here
		[]byte("\tprotected:"),
		[]byte("try {\n} catch (...) {}\n"),
		[]byte("constexpr int x = 1;\n"),
		[]byte("  catch\t(...)"),
		[]byte("@interface Thing : NSObject\n@end\n"),
		[]byte("@interfacex\n"),
		[]byte("#import <Foundation/Foundation.h>\n"),
		[]byte("#import \"Thing.h\"\n"),
		[]byte("#import <Foundation/Foundation>\n"), // no .h, so no match
		[]byte("#include <vector>\nclass Foo {\npublic:\n};\n"),
		[]byte("no markers of any kind in this plain text at all"),
		bytes.Repeat([]byte("int x; /* filler */\n"), 500),
		append(bytes.Repeat([]byte("int x;\n"), 500), []byte("class Foo {\n")...),
		// A line start whose literal matches over and over while the pattern
		// never does, which is the case the walk has to keep cheap.
		bytes.Repeat([]byte("class\n"), 500),
		bytes.Repeat([]byte("using x;\n"), 500),
	}

	sets := [][]string{
		{"C Header", "C++ Header", "Objective C"},
		{"C++ Header", "Objective C"},
	}
	for _, langs := range ExtensionToLanguage {
		if len(langs) > 1 {
			sets = append(sets, langs)
		}
	}

	// Random content built from the plans' own literals lands on partial matches
	// far more often than random bytes would.
	rng := rand.New(rand.NewSource(7))
	pieces := [][]byte{
		[]byte("\n"), []byte(" "), []byte("\t"), []byte("\r\n"), []byte(":"), []byte(";"),
		[]byte("class"), []byte("namespace"), []byte("using"), []byte("template"),
		[]byte("try"), []byte("constexpr"), []byte("catch"), []byte("public"),
		[]byte("private"), []byte("protected"), []byte("@interface"), []byte("@end"),
		[]byte("#import"), []byte(".h"), []byte("\""), []byte(">"), []byte("<"),
		[]byte("x"), []byte("("),
		// The bytes the walk steps over but a lead may not accept. Without
		// these the corpus cannot tell an over-skip from a legal one, and the
		// walk settling a heuristic at a position its pattern could not have
		// started at goes unnoticed: a lone carriage return, a form feed and a
		// vertical tab in front of a keyword each reclassified a C header as a
		// C++ one. A vertical tab is in neither lead, since Go's \s leaves it
		// out; a carriage return and a form feed are in \s but not in [ \t].
		[]byte("\r"), []byte("\n\r"), []byte("\f"), []byte("\v"),
	}
	for i := 0; i < 400; i++ {
		var b []byte
		for j := 0; j < 1+rng.Intn(40); j++ {
			b = append(b, pieces[rng.Intn(len(pieces))]...)
		}
		corpus = append(corpus, b)
	}

	checked := 0
	for _, langs := range sets {
		for _, l := range langs {
			LoadLanguageFeature(l)
		}
		plan := planFor(langs)
		if len(plan.anchRe) == 0 {
			continue
		}

		for _, content := range corpus {
			hit := planAnchoredResults(plan, content)
			for _, pl := range plan.langs {
				for i, h := range pl.heuristics {
					if h.slot < 0 {
						continue
					}
					checked++
					want := anchoredMatchSlow(h.re, content)
					if got := hit[h.slot]; got != want {
						t.Errorf("%s heuristic %d on %q: walk said %v, search said %v",
							pl.name, i, content, got, want)
					}
				}
			}
		}
	}

	if checked == 0 {
		t.Fatal("no anchored heuristic was exercised")
	}
	t.Logf("checked %d anchored heuristic answers", checked)
}

// TestAnchoredFormOnlyFiresOnShapesItUnderstands keeps the rewrite from
// quietly claiming a pattern it would answer differently.
func TestAnchoredFormOnlyFiresOnShapesItUnderstands(t *testing.T) {
	for _, pat := range []string{
		`^\s*class`,         // no (?m), so ^ is not a line start
		`(?m)^class`,        // no indentation to skip, nothing to gain
		`(?m)^\s*`,          // nothing after the lead
		`(?m)^\s*\bclass`,   // a word boundary against the text cut away
		`(?m)^\s*\Aclass`,   // its own text anchor
		`class`,             // not anchored at all
		`(?m)^[ ]*class`,    // a lead shape that is not one of the two written
		`(?m)^\s*(unclosed`, // does not compile once rewritten
	} {
		if re, _ := anchoredForm(pat); re != nil {
			t.Errorf("anchoredForm(%q) returned %q, want nil", pat, re)
		}
	}

	for _, pat := range []string{
		`(?m)^\s*template\s*<`,
		`(?m)^[ \t]*(private|public|protected):$`,
	} {
		if re, _ := anchoredForm(pat); re == nil {
			t.Errorf("anchoredForm(%q) returned nil, want a rewrite", pat)
		}
	}
}

// TestAnchoredLiteralsRejectsWhatTheWalkCannotFind guards the other half of the
// rewrite: the walk finds a pattern by its literals, so a literal it cannot
// land on has to keep the heuristic on the ordinary search.
func TestAnchoredLiteralsRejectsWhatTheWalkCannotFind(t *testing.T) {
	if anchoredLiterals(nil) {
		t.Error("no literals should not be usable")
	}
	if anchoredLiterals([][]byte{[]byte("class"), {}}) {
		t.Error("an empty literal should not be usable")
	}
	if anchoredLiterals([][]byte{[]byte(" class")}) {
		t.Error("a literal starting with skipped whitespace should not be usable")
	}
	if !anchoredLiterals([][]byte{[]byte("class"), []byte("namespace")}) {
		t.Error("plain literals should be usable")
	}
}
