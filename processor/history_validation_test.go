// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"strings"
	"testing"
)

// resetHistoryFlagState snapshots and restores every global flag the
// validator inspects. Each test sets only the fields it cares about; the
// cleanup callback puts the world back exactly how it was found so other
// tests in the package are not disturbed.
func resetHistoryFlagState(t *testing.T) {
	t.Helper()

	saved := struct {
		Hotspots               bool
		ByAuthor               bool
		Timeline               bool
		HistoryDepth           int
		HistoryBuckets         int
		DirFilePaths           []string
		Files                  bool
		UlocMode               bool
		Dryness                bool
		MaxMean                bool
		Duplicates             bool
		MinifiedGenerated      bool
		Minified               bool
		Generated              bool
		IgnoreMinifiedGenerate bool
		IgnoreMinified         bool
		IgnoreGenerated        bool
		Cocomo                 bool
		Locomo                 bool
		CostComparison         bool
		SLOCCountFormat        bool
		NoLarge                bool
		SortBy                 string
	}{
		Hotspots, ByAuthor, Timeline,
		HistoryDepth, HistoryBuckets,
		append([]string(nil), DirFilePaths...),
		Files, UlocMode, Dryness, MaxMean, Duplicates,
		MinifiedGenerated, Minified, Generated,
		IgnoreMinifiedGenerate, IgnoreMinified, IgnoreGenerated,
		Cocomo, Locomo, CostComparison, SLOCCountFormat, NoLarge,
		SortBy,
	}

	Hotspots, ByAuthor, Timeline = false, false, false
	HistoryDepth, HistoryBuckets = 1000, 60
	DirFilePaths = []string{}
	Files, UlocMode, Dryness, MaxMean, Duplicates = false, false, false, false, false
	MinifiedGenerated, Minified, Generated = false, false, false
	IgnoreMinifiedGenerate, IgnoreMinified, IgnoreGenerated = false, false, false
	Cocomo, Locomo, CostComparison, SLOCCountFormat, NoLarge = false, false, false, false, false
	SortBy = "files"

	t.Cleanup(func() {
		Hotspots, ByAuthor, Timeline = saved.Hotspots, saved.ByAuthor, saved.Timeline
		HistoryDepth, HistoryBuckets = saved.HistoryDepth, saved.HistoryBuckets
		DirFilePaths = saved.DirFilePaths
		Files, UlocMode, Dryness, MaxMean, Duplicates = saved.Files, saved.UlocMode, saved.Dryness, saved.MaxMean, saved.Duplicates
		MinifiedGenerated, Minified, Generated = saved.MinifiedGenerated, saved.Minified, saved.Generated
		IgnoreMinifiedGenerate, IgnoreMinified, IgnoreGenerated = saved.IgnoreMinifiedGenerate, saved.IgnoreMinified, saved.IgnoreGenerated
		Cocomo, Locomo, CostComparison, SLOCCountFormat, NoLarge = saved.Cocomo, saved.Locomo, saved.CostComparison, saved.SLOCCountFormat, saved.NoLarge
		SortBy = saved.SortBy
	})
}

func TestValidateRejectsNegativeDepth(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	HistoryDepth = -1

	var buf bytes.Buffer
	err := validateHistoryFlags(&buf)
	if err == nil {
		t.Fatal("expected error for negative depth, got nil")
	}
	if !strings.Contains(err.Error(), "--depth") {
		t.Errorf("expected error to mention --depth, got: %s", err.Error())
	}
}

func TestValidateRejectsZeroBuckets(t *testing.T) {
	resetHistoryFlagState(t)
	Timeline = true
	HistoryBuckets = 0

	var buf bytes.Buffer
	err := validateHistoryFlags(&buf)
	if err == nil {
		t.Fatal("expected error for zero buckets, got nil")
	}
	if !strings.Contains(err.Error(), "--buckets") {
		t.Errorf("expected error to mention --buckets, got: %s", err.Error())
	}
}

func TestValidateAllowsZeroBucketsWithoutTimeline(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	HistoryBuckets = 0

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("buckets only matters for --timeline; got error: %s", err.Error())
	}
}

func TestValidateAllowsZeroDepth(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	HistoryDepth = 0

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("depth=0 means entire history; expected nil, got: %s", err.Error())
	}
}

func TestValidateWarnsOnMultiplePaths(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	DirFilePaths = []string{".", "processor", "vendor"}

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	out := buf.String()
	if !strings.Contains(out, "single repository") {
		t.Errorf("expected warning to mention single repository, got: %q", out)
	}
	if !strings.Contains(out, "processor") || !strings.Contains(out, "vendor") {
		t.Errorf("expected warning to list the dropped paths, got: %q", out)
	}
}

func TestValidateWarnsOnIgnoredFlags(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	Files = true

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	out := buf.String()
	if !strings.Contains(out, "--by-file") {
		t.Errorf("expected warning to list --by-file, got: %q", out)
	}
	if !strings.Contains(out, "ignore these flags") {
		t.Errorf("expected combined ignored-flags warning, got: %q", out)
	}
}

func TestValidateWarnsOnNonDefaultSort(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	SortBy = "complexity"

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if !strings.Contains(buf.String(), "--sort") {
		t.Errorf("expected --sort in ignored list, got: %q", buf.String())
	}
}

func TestValidateDefaultSortNotWarned(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	SortBy = "files"

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if buf.Len() != 0 {
		t.Errorf("default sort should not warn, got: %q", buf.String())
	}
}

func TestValidateNoOpWithoutHistoryFlag(t *testing.T) {
	resetHistoryFlagState(t)
	// Deliberately set a state that would otherwise trip every branch:
	// a hard error condition plus a warning condition. Without a history
	// flag set the helper must return nil and emit nothing.
	HistoryDepth = -1
	HistoryBuckets = 0
	DirFilePaths = []string{".", "extra"}
	Files = true

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("expected nil when no history flag set, got: %s", err.Error())
	}
	if buf.Len() != 0 {
		t.Errorf("expected no warnings when no history flag set, got: %q", buf.String())
	}
}

func TestValidateCleanInputProducesNoWarnings(t *testing.T) {
	resetHistoryFlagState(t)
	Hotspots = true
	DirFilePaths = []string{"."}

	var buf bytes.Buffer
	if err := validateHistoryFlags(&buf); err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	if buf.Len() != 0 {
		t.Errorf("expected no warnings for clean input, got: %q", buf.String())
	}
}
