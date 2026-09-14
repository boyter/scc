// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"testing"
)

// The differential test with the corpus taken out of it.
//
// Eight of the thirteen languages a counter is planned for have no corpus on
// any machine this is likely to run on, so for those the fuzzer is the only
// differential coverage there will be. It is also the only layer that reaches
// the shapes nobody writes on purpose: a quote opened and never closed at the
// last byte, a comment opener cut in half by the end of the file, a run of
// backslashes in front of a terminator.
//
// countLoopGeneric is the oracle. Where the two disagree the generic one is
// right by definition, so any disagreement at all is a failure.

// fuzzCounterLanguages is every language a counter answers for, derived from
// the one registry so a new counter is fuzzed without anyone remembering to add
// it here.
var fuzzCounterLanguages = counterLanguages()

// seedFromExamples reads the sample files scc keeps for language detection and
// hands them to the fuzzer as seeds, which is a far better starting corpus than
// anything written by hand here. Failing to read one is not an error: the
// fuzzer works from the literal seeds below either way.
func seedFromExamples(f *testing.F) {
	f.Helper()

	matches, err := filepath.Glob(filepath.Join("..", "examples", "language", "*"))
	if err != nil {
		return
	}

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() > 64*1024 {
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		f.Add(content)
	}
}

func FuzzSpecialisedCounter(f *testing.F) {
	ProcessConstants()

	seedFromExamples(f)

	// The shapes the corpus is unlikely to hold, each of which has broken one
	// loop or the other at some point.
	for _, seed := range []string{
		"",
		"\n",
		"/",
		"/*",
		"*/",
		"//",
		`"`,
		`\"`,
		"'",
		"/* \n */",
		"/*/",
		"/**/",
		"//\\\n still a comment\n",
		"char *s = \"a \\\\\";\n",
		"if (a) { for (;;) { while (b) {} } }\n",
		"else { switch (c) { case 1: break; } }\n",
		"try {} catch (E e) {} finally {}\n",
		"\x00",
		"a\x00b\n",
		"\xEF\xBB\xBFif (a) {}\n",
		"\t\t\t\t\t\t\t\t   \t \t if (a) {}\n",
		"/* a\n\n\n\n b */\n",
		// C++ raw strings, whose closer is read out of the file. The delimiter
		// is attacker controlled and bounded only by maxRawStringDelimiter, so
		// the shapes that matter are the ones that end before it does.
		`R"`,
		`R"(`,
		`R"()"`,
		`R"tag(x)tag"`,
		`R"(a "b" c)";`,
		`R"(` + "\n\n" + `)";`,
		`u8R"(x)";`,
		`R"aaaaaaaaaaaaaaaa(x)aaaaaaaaaaaaaaaa";`,
		`R"aaaaaaaaaaaaaaaaa(x)aaaaaaaaaaaaaaaaa";`,
		`R" (`,
		`R")"`,
		"const char *s = R\"(a\\\n b)\";\n",
	} {
		f.Add([]byte(seed))
	}

	// The oracle is the generic loop, so the one place a counter is meant to
	// disagree with it has to come out, exactly as it does for the corpus
	// differential. M16 fires wherever a regular expression literal holds a
	// quote or a comment opener, which is a shape the fuzzer reaches constantly
	// and which says nothing about whether the rest of the counter is right.
	// The divergence itself is pinned by the fixtures of counterDivergences.
	previousRegexLiterals := ecmaRegexLiterals
	ecmaRegexLiterals = false
	f.Cleanup(func() { ecmaRegexLiterals = previousRegexLiterals })

	f.Fuzz(func(t *testing.T, content []byte) {
		// A file scc would never reach: the loops are bounded by fileJob.Bytes
		// and the caller sets that from the content it read.
		if len(content) > 1<<20 {
			t.Skip("larger than anything worth fuzzing")
		}

		for _, language := range fuzzCounterLanguages {
			fast, generic := countBothWays(t, language, content)
			if countsDiffer(fast, generic) {
				t.Fatalf("%s disagrees on %q\n  counter: lines=%d code=%d comment=%d blank=%d complexity=%d binary=%t\n  generic: lines=%d code=%d comment=%d blank=%d complexity=%d binary=%t",
					language, content,
					fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Complexity, fast.Binary,
					generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Complexity, generic.Binary)
			}
		}
	})
}

// The same target with complexity off, which is a different stop table and so a
// different scan over the same bytes. Kept separate because the globals it
// moves are not safe to move inside the fuzz function.
func FuzzSpecialisedCounterNoComplexity(f *testing.F) {
	Complexity = true
	ProcessConstants()
	f.Cleanup(func() {
		Complexity = false
		ProcessConstants()
	})

	seedFromExamples(f)
	f.Add([]byte("if (a) { for (;;) {} }\n// a comment\n"))
	f.Add([]byte("/* a\n b */ \"a string\"\n"))

	previousRegexLiterals := ecmaRegexLiterals
	ecmaRegexLiterals = false
	f.Cleanup(func() { ecmaRegexLiterals = previousRegexLiterals })

	f.Fuzz(func(t *testing.T, content []byte) {
		if len(content) > 1<<20 {
			t.Skip("larger than anything worth fuzzing")
		}

		for _, language := range fuzzCounterLanguages {
			fast, generic := countBothWays(t, language, content)
			if countsDiffer(fast, generic) {
				t.Fatalf("%s disagrees with complexity off on %q\n  counter: lines=%d code=%d comment=%d blank=%d binary=%t\n  generic: lines=%d code=%d comment=%d blank=%d binary=%t",
					language, content,
					fast.Lines, fast.Code, fast.Comment, fast.Blank, fast.Binary,
					generic.Lines, generic.Code, generic.Comment, generic.Blank, generic.Binary)
			}
		}
	})
}
