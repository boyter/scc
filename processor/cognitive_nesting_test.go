// SPDX-License-Identifier: MIT

package processor

import "testing"

// These tests pin the core promise of cognitive complexity: the SAME number of
// branch points (identical cyclomatic Complexity) must score HIGHER the more
// deeply they are nested. Cyclomatic counts branches flatly; cognitive weights
// each branch by 1+nesting, so shape — not just count — drives the number.
//
// All fixtures below contain exactly four `if`s (Complexity == 4) arranged from
// maximally nested to maximally flat, and assert both the exact cognitive values
// (derived from the 1+nesting rule) and the strict ordering between them.

// deepChain: four ifs each nested one level deeper than the last.
//
//	if          nesting 1 -> +2
//	  if        nesting 2 -> +3
//	    if      nesting 3 -> +4
//	      if    nesting 4 -> +5      total 14
const deepChain = "func main() {\n" +
	"    if a {\n" +
	"        if b {\n" +
	"            if c {\n" +
	"                if d {\n" +
	"                }\n" +
	"            }\n" +
	"        }\n" +
	"    }\n" +
	"}\n"

// deNested: two independent 2-deep pairs (the user's "de-nested" example).
//
//	if          nesting 1 -> +2
//	  if        nesting 2 -> +3
//	if          nesting 1 -> +2
//	  if        nesting 2 -> +3      total 10
const deNested = "func main() {\n" +
	"    if a {\n" +
	"        if b {\n" +
	"        }\n" +
	"    }\n" +
	"    if c {\n" +
	"        if d {\n" +
	"        }\n" +
	"    }\n" +
	"}\n"

// flat: all four ifs as siblings at the same level.
//
//	if  if  if  if   each nesting 1 -> +2 each   total 8
const flat = "func main() {\n" +
	"    if a {\n    }\n" +
	"    if b {\n    }\n" +
	"    if c {\n    }\n" +
	"    if d {\n    }\n" +
	"}\n"

// TestCognitiveDeepVsDeNestedVsFlat is the headline test: same four branches,
// three shapes, strictly decreasing cognitive as they flatten — while the flat
// cyclomatic Complexity stays 4 throughout.
func TestCognitiveDeepVsDeNestedVsFlat(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	deep := countCognitive(t, "Go", deepChain)
	mid := countCognitive(t, "Go", deNested)
	shallow := countCognitive(t, "Go", flat)

	// Same number of branch points: cyclomatic cannot tell these apart.
	for name, job := range map[string]FileJob{"deep": deep, "deNested": mid, "flat": shallow} {
		if job.Complexity != 4 {
			t.Errorf("%s: expected Complexity 4 (four ifs), got %d", name, job.Complexity)
		}
	}

	// Exact cognitive values from the 1+nesting rule.
	if deep.Cognitive != 14 {
		t.Errorf("deepChain Cognitive: expected 14, got %d", deep.Cognitive)
	}
	if mid.Cognitive != 10 {
		t.Errorf("deNested Cognitive: expected 10, got %d", mid.Cognitive)
	}
	if shallow.Cognitive != 8 {
		t.Errorf("flat Cognitive: expected 8, got %d", shallow.Cognitive)
	}

	// The relationship the metric exists to express: deeper nesting scores
	// strictly higher, even at identical cyclomatic complexity.
	if !(deep.Cognitive > mid.Cognitive && mid.Cognitive > shallow.Cognitive) {
		t.Errorf("expected deep > deNested > flat, got %d, %d, %d",
			deep.Cognitive, mid.Cognitive, shallow.Cognitive)
	}
}

// TestCognitiveNestedHigherThanDeNested is the user's literal example, written
// with top-level (column 0) ifs and 2-space indentation to show the ranking is
// independent of the wrapping function and the indent unit.
//
//	nested:            de-nested:
//	if                 if
//	  if                 if
//	    if             if
//	      if             if
func TestCognitiveNestedHigherThanDeNested(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	nested := "if a {\n" +
		"  if b {\n" +
		"    if c {\n" +
		"      if d {\n" +
		"      }\n" +
		"    }\n" +
		"  }\n" +
		"}\n"

	splitNested := "if a {\n" +
		"  if b {\n" +
		"  }\n" +
		"}\n" +
		"if c {\n" +
		"  if d {\n" +
		"  }\n" +
		"}\n"

	nestedJob := countCognitive(t, "Go", nested)
	splitJob := countCognitive(t, "Go", splitNested)

	if nestedJob.Complexity != splitJob.Complexity {
		t.Fatalf("fixtures must have equal cyclomatic Complexity, got nested=%d split=%d",
			nestedJob.Complexity, splitJob.Complexity)
	}
	// nested: 1+2+3+4 = 10 ; de-nested: 1+2+1+2 = 6
	if nestedJob.Cognitive != 10 {
		t.Errorf("nested Cognitive: expected 10, got %d", nestedJob.Cognitive)
	}
	if splitJob.Cognitive != 6 {
		t.Errorf("de-nested Cognitive: expected 6, got %d", splitJob.Cognitive)
	}
	if nestedJob.Cognitive <= splitJob.Cognitive {
		t.Errorf("nested (%d) should score strictly higher than de-nested (%d)",
			nestedJob.Cognitive, splitJob.Cognitive)
	}
}

// TestCognitiveMonotonicWithDepth: a single chain of N nested ifs. Each added
// level of depth adds strictly more cognitive weight than the previous level did
// (the increments grow 2,3,4,... as nesting deepens), unlike cyclomatic which
// would add a flat 1 per branch.
func TestCognitiveMonotonicWithDepth(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	// Build chains of depth 1..5 and record cognitive at each depth.
	chain := func(depth int) string {
		var b string
		b = "func main() {\n"
		indent := "    "
		pad := ""
		for i := 0; i < depth; i++ {
			pad += indent
			b += pad + "if x {\n"
		}
		for i := 0; i < depth; i++ {
			b += pad + "}\n"
			pad = pad[:len(pad)-len(indent)]
		}
		b += "}\n"
		return b
	}

	prev := int64(-1)
	prevDelta := int64(-1)
	for depth := 1; depth <= 5; depth++ {
		job := countCognitive(t, "Go", chain(depth))
		if int64(job.Complexity) != int64(depth) {
			t.Errorf("depth %d: expected Complexity %d, got %d", depth, depth, job.Complexity)
		}
		if prev >= 0 {
			delta := job.Cognitive - prev
			// Each deeper level contributes more than the one before it.
			if delta <= prevDelta {
				t.Errorf("depth %d: cognitive delta %d should exceed previous delta %d (values grow super-linearly with depth)", depth, delta, prevDelta)
			}
			prevDelta = delta
		}
		if job.Cognitive <= prev {
			t.Errorf("depth %d: cognitive %d should exceed shallower depth's %d", depth, job.Cognitive, prev)
		}
		prev = job.Cognitive
	}
}
