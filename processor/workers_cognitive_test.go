// SPDX-License-Identifier: MIT

package processor

import "testing"

// countCognitive runs CountStats over content for the given language with the
// Cognitive global enabled and returns the resulting FileJob so callers can
// assert on both Complexity and Cognitive.
func countCognitive(t *testing.T, language, content string) FileJob {
	t.Helper()
	ProcessConstants()

	fileJob := FileJob{Language: language}
	fileJob.SetContent(content)
	CountStats(&fileJob)
	return fileJob
}

// countCognitiveLines runs CountStats with both Cognitive and per-line tracking
// on, so callers can assert on the CognitiveLine array as well as the whole-file
// tally.
func countCognitiveLines(t *testing.T, language, content string) FileJob {
	t.Helper()
	ProcessConstants()

	fileJob := FileJob{Language: language, TrackComplexityLines: true}
	fileJob.SetContent(content)
	CountStats(&fileJob)
	return fileJob
}

func sumInt64(xs []int64) int64 {
	var total int64
	for _, x := range xs {
		total += x
	}
	return total
}

// The per-line array must total to the whole-file Cognitive value, for flat and
// nested files alike.
func TestCognitiveLineSumEqualsCognitive(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	cases := map[string]string{
		"flat": "func main() {\n" +
			"    if a {\n    }\n    if b {\n    }\n    if c {\n    }\n}\n",
		"nested": "func main() {\n" +
			"    if a {\n        if b {\n            if c {\n            }\n        }\n    }\n}\n",
		"no trailing newline": "func main() {\n    if a {\n        if b {\n        }\n    }\n}",
	}

	for name, content := range cases {
		job := countCognitiveLines(t, "Go", content)
		if got := sumInt64(job.CognitiveLine); got != job.Cognitive {
			t.Errorf("case %q: sum(CognitiveLine)=%d, want Cognitive=%d (line array %v)", name, got, job.Cognitive, job.CognitiveLine)
		}
		if job.Cognitive == 0 {
			t.Errorf("case %q: expected non-zero Cognitive", name)
		}
	}
}

// CognitiveLine is trimmed to exactly Lines entries, the same invariant
// ComplexityLine holds, with and without a trailing newline.
func TestCognitiveLineLengthEqualsLines(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	cases := map[string]string{
		"trailing newline":    "func main() {\n    if a {\n    }\n}\n",
		"no trailing newline": "func main() {\n    if a {\n    }\n}",
	}

	for name, content := range cases {
		job := countCognitiveLines(t, "Go", content)
		if int64(len(job.CognitiveLine)) != job.Lines {
			t.Errorf("case %q: len(CognitiveLine)=%d, want Lines=%d", name, len(job.CognitiveLine), job.Lines)
		}
		// Must stay in lock-step with ComplexityLine's length.
		if len(job.CognitiveLine) != len(job.ComplexityLine) {
			t.Errorf("case %q: len(CognitiveLine)=%d != len(ComplexityLine)=%d", name, len(job.CognitiveLine), len(job.ComplexityLine))
		}
	}
}

// When Cognitive is off, CognitiveLine stays nil even with per-line tracking on,
// exactly as ComplexityLine stays empty when its own tracking is off.
func TestCognitiveLineEmptyWhenDisabled(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	job := countCognitiveLines(t, "Go", "func main() {\n    if a {\n    }\n}\n")

	if job.CognitiveLine != nil {
		t.Errorf("CognitiveLine should be nil when Cognitive disabled, got %v", job.CognitiveLine)
	}
	// ComplexityLine is still populated because TrackComplexityLines is on.
	if len(job.ComplexityLine) == 0 {
		t.Errorf("ComplexityLine should still be populated when only Cognitive is off")
	}
}

func TestCognitiveFlatVsNested(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	flat := "func main() {\n" +
		"    if a {\n" +
		"    }\n" +
		"    if b {\n" +
		"    }\n" +
		"    if c {\n" +
		"    }\n" +
		"}\n"

	nested := "func main() {\n" +
		"    if a {\n" +
		"        if b {\n" +
		"            if c {\n" +
		"            }\n" +
		"        }\n" +
		"    }\n" +
		"}\n"

	flatJob := countCognitive(t, "Go", flat)
	nestedJob := countCognitive(t, "Go", nested)

	if flatJob.Complexity != 3 {
		t.Errorf("flat Complexity: expected 3 got %d", flatJob.Complexity)
	}
	if nestedJob.Complexity != 3 {
		t.Errorf("nested Complexity: expected 3 got %d", nestedJob.Complexity)
	}
	if flatJob.Complexity != nestedJob.Complexity {
		t.Errorf("flat and nested should have equal Complexity, got %d vs %d", flatJob.Complexity, nestedJob.Complexity)
	}

	if flatJob.Cognitive != 6 {
		t.Errorf("flat Cognitive: expected 6 got %d", flatJob.Cognitive)
	}
	if nestedJob.Cognitive != 9 {
		t.Errorf("nested Cognitive: expected 9 got %d", nestedJob.Cognitive)
	}
	if nestedJob.Cognitive <= flatJob.Cognitive {
		t.Errorf("nested Cognitive (%d) should be strictly greater than flat (%d)", nestedJob.Cognitive, flatJob.Cognitive)
	}
}

func TestCognitiveDisabledByDefault(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	ProcessConstants()

	nested := "func main() {\n" +
		"    if a {\n" +
		"        if b {\n" +
		"            if c {\n" +
		"            }\n" +
		"        }\n" +
		"    }\n" +
		"}\n"

	fileJob := FileJob{Language: "Go"}
	fileJob.SetContent(nested)
	CountStats(&fileJob)

	if fileJob.Cognitive != 0 {
		t.Errorf("Cognitive should be 0 when disabled, got %d", fileJob.Cognitive)
	}
	// Complexity must be unaffected by the cognitive machinery.
	if fileJob.Complexity != 3 {
		t.Errorf("Complexity should be 3 when Cognitive disabled, got %d", fileJob.Complexity)
	}
}

func TestCognitiveTabsAndSpacesEquivalent(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	spaces := "func main() {\n" +
		"    if a {\n" +
		"        if b {\n" +
		"            if c {\n" +
		"            }\n" +
		"        }\n" +
		"    }\n" +
		"}\n"

	tabs := "func main() {\n" +
		"\tif a {\n" +
		"\t\tif b {\n" +
		"\t\t\tif c {\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n"

	spacesJob := countCognitive(t, "Go", spaces)
	tabsJob := countCognitive(t, "Go", tabs)

	if spacesJob.Cognitive != tabsJob.Cognitive {
		t.Errorf("tabs and spaces should yield equal Cognitive, got spaces=%d tabs=%d", spacesJob.Cognitive, tabsJob.Cognitive)
	}
	if tabsJob.Cognitive != 9 {
		t.Errorf("tabs Cognitive: expected 9 got %d", tabsJob.Cognitive)
	}
}

func TestCognitiveCommentAndBlankDoNotAffectNesting(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	plain := "func main() {\n" +
		"    if a {\n" +
		"    }\n" +
		"    if b {\n" +
		"    }\n" +
		"}\n"

	withNoise := "func main() {\n" +
		"    if a {\n" +
		"    }\n" +
		"\n" +
		"            // deeply indented comment\n" +
		"    if b {\n" +
		"    }\n" +
		"}\n"

	plainJob := countCognitive(t, "Go", plain)
	noiseJob := countCognitive(t, "Go", withNoise)

	if plainJob.Cognitive != 4 {
		t.Errorf("plain Cognitive: expected 4 got %d", plainJob.Cognitive)
	}
	if noiseJob.Cognitive != plainJob.Cognitive {
		t.Errorf("comment/blank lines changed Cognitive: plain=%d noise=%d", plainJob.Cognitive, noiseJob.Cognitive)
	}
	if noiseJob.Complexity != plainJob.Complexity {
		t.Errorf("comment/blank lines changed Complexity: plain=%d noise=%d", plainJob.Complexity, noiseJob.Complexity)
	}
}

func TestCognitiveStringsDoNotAffectNesting(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	// The `if fake` sits inside a multiline raw string: it must contribute no
	// complexity and its deep indent must not push the indent stack.
	content := "func main() {\n" +
		"    s := `\n" +
		"            if fake {\n" +
		"    `\n" +
		"    if a {\n" +
		"    }\n" +
		"}\n"

	job := countCognitive(t, "Go", content)

	if job.Complexity != 1 {
		t.Errorf("Complexity: expected 1 (string contents ignored) got %d", job.Complexity)
	}
	if job.Cognitive != 2 {
		t.Errorf("Cognitive: expected 2 (real if at nesting 1) got %d", job.Cognitive)
	}
}

func TestCognitivePostfixLanguage(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	// Rust uses `?` as a postfix complexity token, routed via
	// countComplexityPostfix. Each ? and the if must accrue weighted cognitive.
	content := "fn main() {\n" +
		"    let x = foo()?;\n" +
		"    if a {\n" +
		"        let y = bar()?;\n" +
		"    }\n" +
		"}\n"

	job := countCognitive(t, "Rust", content)

	// two `?` postfix tokens + one `if`
	if job.Complexity != 3 {
		t.Errorf("Complexity: expected 3 got %d", job.Complexity)
	}
	// foo()? at nesting 1 (+2), if at nesting 1 (+2), bar()? at nesting 2 (+3)
	if job.Cognitive != 7 {
		t.Errorf("Cognitive: expected 7 got %d", job.Cognitive)
	}
	if job.Cognitive == 0 {
		t.Errorf("postfix language should accrue Cognitive")
	}
}

func TestCognitiveNoPanicPathological(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	cases := map[string]string{
		"empty":            "",
		"single newline":   "\n",
		"all whitespace":   "   \t  \n   \n\t\t",
		"no trailing nl":   "func main() {\n    if a {\n    }\n}",
		"starts indented":  "        if a {\n        }\n",
		"only indentation": "\t\t\t\t",
	}

	for name, content := range cases {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("case %q panicked: %v", name, r)
				}
			}()
			job := countCognitive(t, "Go", content)
			if job.Cognitive < 0 {
				t.Errorf("case %q produced negative Cognitive %d", name, job.Cognitive)
			}
		}()
	}
}
