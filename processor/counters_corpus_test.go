// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The corpus differential, which every counter has been checked against and
// which until now lived in somebody's shell history rather than here. It is the
// layer the anchor and fuzz tests refer to and the one thing nobody could
// re-run: the whole of a tree counted twice, once with the counter written for
// each file's language and once with the generic loop, with the two required to
// agree on every file.
//
// The generic loop is the oracle. Where the two disagree the generic one is
// right by definition, save for the two places a counter is meant to know
// better, which are turned off here so that the oracle is one.
//
// Point SCC_CORPUS at a tree of real code to run it against that; examples/ is
// used otherwise, which is small but is checked in and so always gives an
// answer. A run over llvm-project or the kernel is worth more than anything
// here and costs nothing but time:
//
//	SCC_CORPUS=/path/to/llvm-project go test ./processor/ -run Corpus -v

// corpusFileLimit keeps a stray database dump or vendored blob from turning the
// differential into a memory test. Nothing scc counts for real is near it.
const corpusFileLimit = 8 << 20

func corpusRoot() string {
	if root := os.Getenv("SCC_CORPUS"); root != "" {
		return root
	}

	return filepath.Join("..", "examples")
}

// walkCorpus hands fn every file under the corpus that a counter answers for,
// with the language scc's own detection gives it rather than one guessed from
// the extension. It returns how many files were handed over, per language, so a
// test can say what it actually covered rather than claiming a tree it skipped.
func walkCorpus(t *testing.T, root string, fn func(language, path string, content []byte)) map[string]int {
	t.Helper()

	seen := map[string]int{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if name := info.Name(); name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}

			return nil
		}
		if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > corpusFileLimit {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		name := filepath.Base(path)
		possible, extension := DetectLanguage(name)
		language := DetermineLanguage(name, extension, possible, content)
		if counterDispatch[language] == nil {
			return nil
		}

		seen[language]++
		fn(language, path, content)

		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}

	return seen
}

// logCoverage writes what the run actually reached, which is the part of a
// corpus result that is easy to leave out and easy to be misled by. A language
// with no files under the corpus is not a pass.
func logCoverage(t *testing.T, root string, seen map[string]int) {
	t.Helper()

	languages := make([]string, 0, len(seen))
	total := 0
	for language, count := range seen {
		languages = append(languages, language)
		total += count
	}
	sort.Strings(languages)

	t.Logf("%d files under %s", total, root)
	for _, language := range languages {
		t.Logf("  %-12s %6d files", language, seen[language])
	}

	var missing []string
	for _, language := range counterLanguages() {
		if seen[language] == 0 {
			missing = append(missing, language)
		}
	}
	if len(missing) != 0 {
		t.Logf("no files under %s for: %s", root, strings.Join(missing, ", "))
	}
}

// withoutDeliberateDivergences turns off the two places a counter knows better
// than the generic loop, so that for the length of the test the generic loop is
// an oracle rather than merely the older answer. Both are restored whatever the
// test does.
func withoutDeliberateDivergences(t *testing.T) {
	t.Helper()

	previousRegex := ecmaRegexLiterals
	ecmaRegexLiterals = false
	t.Cleanup(func() { ecmaRegexLiterals = previousRegex })

	previousChar := rustCharLiterals
	rustCharLiterals = false
	t.Cleanup(func() { rustCharLiterals = previousChar })
}

func TestCounterCorpusAgreesWithTheGenericLoop(t *testing.T) {
	ProcessConstants()

	root := corpusRoot()
	withoutDeliberateDivergences(t)

	// Complexity off is a different stop table and so a different scan over the
	// same bytes, which is half the surface and none of it covered by counting
	// with complexity on. Complexity is the flag that turns it off.
	for _, mode := range []struct {
		name         string
		noComplexity bool
	}{
		{name: "complexity", noComplexity: false},
		{name: "no-complexity", noComplexity: true},
	} {
		t.Run(mode.name, func(t *testing.T) {
			previous := Complexity
			Complexity = mode.noComplexity
			ProcessConstants()
			t.Cleanup(func() {
				Complexity = previous
				ProcessConstants()
			})

			failures := 0
			seen := walkCorpus(t, root, func(language, path string, content []byte) {
				if failures > 20 {
					return
				}

				fast, generic := countBothWays(t, language, content)
				if countsDiffer(fast, generic) {
					failures++
					t.Errorf("%s disagrees on %s\n  counter: lines=%d code=%d comment=%d blank=%d complexity=%d binary=%t\n  generic: lines=%d code=%d comment=%d blank=%d complexity=%d binary=%t",
						language, path,
						fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity, fast.Binary,
						generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Complexity, generic.Binary)
				}
			})

			if mode.name == "complexity" {
				logCoverage(t, root, seen)
			}
		})
	}
}

// The other half of the differential, and the one that would have spoken up
// about the minified JavaScript under examples/ long before now: with the two
// divergences left on, a counter may disagree with the generic loop only where
// its language has declared that it will. A counter that starts disagreeing in
// a language with nothing declared is a bug wherever it shows up, and a corpus
// is the only place it is going to show up.
func TestCounterCorpusDivergesOnlyWhereDeclared(t *testing.T) {
	ProcessConstants()

	declared := map[string]bool{}
	for _, divergence := range counterDivergences() {
		declared[divergence.Language] = true
	}

	root := corpusRoot()
	diverged := map[string]int{}

	walkCorpus(t, root, func(language, path string, content []byte) {
		fast, generic := countBothWays(t, language, content)
		if !countsDiffer(fast, generic) {
			return
		}

		diverged[language]++
		if !declared[language] {
			t.Errorf("%s diverges on %s but declares no divergence\n  counter: lines=%d code=%d comment=%d blank=%d complexity=%d\n  generic: lines=%d code=%d comment=%d blank=%d complexity=%d",
				language, path,
				fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity,
				generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Complexity)
		}
	})

	languages := make([]string, 0, len(diverged))
	for language := range diverged {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	for _, language := range languages {
		t.Logf("%s diverges on %d files under %s, as declared", language, diverged[language], root)
	}
}
