// SPDX-License-Identifier: MIT

package processor

import (
	"reflect"
	"testing"
)

// cognitiveLineNumbers must report the 1-based lines that accrued cognitive
// weight, mirroring complexityLineNumbers for a fixture with known nesting.
func TestCognitiveLineNumbers(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	// if tokens sit on lines 4, 5 and 6.
	content := "package nested\n" + // 1
		"\n" + // 2
		"func N() {\n" + // 3
		"\tif a {\n" + // 4
		"\t\tif b {\n" + // 5
		"\t\t\tif c {\n" + // 6
		"\t\t\t}\n" + // 7
		"\t\t}\n" + // 8
		"\t}\n" + // 9
		"}\n" // 10

	job := countCognitiveLines(t, "Go", content)

	got := cognitiveLineNumbers(&job)
	want := []int{4, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("cognitiveLineNumbers = %v, want %v (CognitiveLine %v)", got, want, job.CognitiveLine)
	}

	// With Cognitive off the array is nil, so the helper returns an empty slice.
	Cognitive = false
	off := countCognitiveLines(t, "Go", content)
	if n := len(cognitiveLineNumbers(&off)); n != 0 {
		t.Errorf("cognitiveLineNumbers should be empty when Cognitive off, got %d entries", n)
	}
	Cognitive = true
}

// A nested-heavy file must outrank a flat file with higher cyclomatic
// complexity and equal change frequency once cognitive weighting is on; the
// default (flat) weighting ranks them the other way around.
func TestHotspotsCognitiveRankingFlips(t *testing.T) {
	saveDepth, saveFormat := HistoryDepth, Format
	HistoryDepth, Format = 100, "tabular"
	t.Cleanup(func() { HistoryDepth, Format = saveDepth, saveFormat })

	// flat.go: 4 flat ifs   -> Complexity 4, Cognitive 8 (each if at nesting 1)
	// nested.go: 3 nested ifs -> Complexity 3, Cognitive 9 (2+3+4)
	// Both files are touched in every commit, so change frequency is identical.
	flatFinal := "package flat\n\nfunc F() {\n" +
		"\tif a {\n\t}\n\tif b {\n\t}\n\tif c {\n\t}\n\tif d {\n\t}\n}\n"
	nestedFinal := "package nested\n\nfunc N() {\n" +
		"\tif a {\n\t\tif b {\n\t\t\tif c {\n\t\t\t}\n\t\t}\n\t}\n}\n"

	dir := makeFixtureRepo(t, []map[string]string{
		{
			"flat.go":   "package flat\n\nfunc F() {}\n",
			"nested.go": "package nested\n\nfunc N() {}\n",
		},
		{
			"flat.go":   "package flat\n\nfunc F() {\n\tif a {\n\t}\n}\n",
			"nested.go": "package nested\n\nfunc N() {\n\tif a {\n\t\tif b {\n\t\t}\n\t}\n}\n",
		},
		{
			"flat.go":   flatFinal,
			"nested.go": nestedFinal,
		},
	})

	rankOf := func(recs []hotspotsRecord, file string) int {
		for i, r := range recs {
			if r.File == file {
				return i
			}
		}
		return -1
	}

	// Default (flat cyclomatic) weighting: flat.go (Complexity 4) outranks
	// nested.go (Complexity 3) at equal change frequency.
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	flatObs := newHotspotsObserver()
	if _, err := runHistory(dir, flatObs); err != nil {
		t.Fatalf("runHistory (flat): %v", err)
	}
	if got := rankOf(flatObs.records, "flat.go"); got != 0 {
		t.Fatalf("under flat weighting flat.go rank = %d, want 0 (records %v)", got, flatObs.records)
	}
	if rankOf(flatObs.records, "flat.go") >= rankOf(flatObs.records, "nested.go") {
		t.Fatalf("under flat weighting flat.go should outrank nested.go")
	}

	// Cognitive weighting: nested.go (Cognitive 9) outranks flat.go (Cognitive 8).
	Cognitive = true
	defer func() { Cognitive = false }()
	cogObs := newHotspotsObserver()
	if _, err := runHistory(dir, cogObs); err != nil {
		t.Fatalf("runHistory (cognitive): %v", err)
	}
	if got := rankOf(cogObs.records, "nested.go"); got != 0 {
		t.Fatalf("under cognitive weighting nested.go rank = %d, want 0 (records %v)", got, cogObs.records)
	}
	if rankOf(cogObs.records, "nested.go") >= rankOf(cogObs.records, "flat.go") {
		t.Fatalf("under cognitive weighting nested.go should outrank flat.go")
	}

	// Sanity: the cognitive magnitudes are what we expect, and the Complexity
	// column still reports cyclomatic complexity (unchanged output).
	nested := findRecord(cogObs.records, "nested.go")
	flat := findRecord(cogObs.records, "flat.go")
	if nested.Cognitive != 9 || flat.Cognitive != 8 {
		t.Errorf("cognitive magnitudes: nested=%d (want 9) flat=%d (want 8)", nested.Cognitive, flat.Cognitive)
	}
	if nested.Complexity != 3 || flat.Complexity != 4 {
		t.Errorf("complexity column changed: nested=%d (want 3) flat=%d (want 4)", nested.Complexity, flat.Complexity)
	}
}

func findRecord(recs []hotspotsRecord, file string) hotspotsRecord {
	for _, r := range recs {
		if r.File == file {
			return r
		}
	}
	return hotspotsRecord{}
}
