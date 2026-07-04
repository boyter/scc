// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
)

// cognitiveFixtureChan returns a closed channel with two Go files whose
// Cognitive values are known so callers can assert on aggregation.
func cognitiveFixtureChan() chan *FileJob {
	inputChan := make(chan *FileJob, 8)
	inputChan <- &FileJob{
		Language:   "Go",
		Filename:   "aaaa.go",
		Location:   "aaaa.go",
		Bytes:      100,
		Lines:      100,
		Code:       80,
		Comment:    10,
		Blank:      10,
		Complexity: 5,
		Cognitive:  11,
	}
	inputChan <- &FileJob{
		Language:   "Go",
		Filename:   "bbbb.go",
		Location:   "bbbb.go",
		Bytes:      100,
		Lines:      100,
		Code:       80,
		Comment:    10,
		Blank:      10,
		Complexity: 5,
		Cognitive:  20,
	}
	close(inputChan)
	return inputChan
}

func TestFileSummarizeLongCognitive(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	res := fileSummarizeLong(cognitiveFixtureChan())

	if !strings.Contains(res, "Cognitive") {
		t.Errorf("wide cognitive output missing Cognitive header:\n%s", res)
	}
	// language total and grand total are both 11 + 20 = 31
	if !strings.Contains(res, "31") {
		t.Errorf("wide cognitive output missing summed cognitive 31:\n%s", res)
	}
}

func TestFileSummarizeLongCognitiveDisabledParity(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	res := fileSummarizeLong(cognitiveFixtureChan())

	if strings.Contains(res, "Cognitive") {
		t.Errorf("wide output leaked Cognitive column while disabled:\n%s", res)
	}
	// Break lines must stay at the original 109-rune width when disabled.
	for _, line := range strings.Split(res, "\n") {
		if strings.HasPrefix(line, "─") && len([]rune(line)) != 109 {
			t.Errorf("disabled wide break width changed: got %d want 109", len([]rune(line)))
		}
	}
}

// TestFileSummarizeShortCognitiveOverride asserts that in the default (non-wide)
// view --cognitive overrides the Complexity column value in place (header stays
// "Complexity") rather than adding a column: the printed magnitude is the summed
// Cognitive (31), not the summed Complexity (10).
func TestFileSummarizeShortCognitiveOverride(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	res := fileSummarizeShort(cognitiveFixtureChan())

	if !strings.Contains(res, "Complexity") {
		t.Errorf("short view lost the Complexity header:\n%s", res)
	}
	if strings.Contains(res, "Cognitive") {
		t.Errorf("short view should not add a Cognitive column:\n%s", res)
	}
	// The Go language row and Total both carry the cognitive magnitude 31,
	// never the cyclomatic 10.
	if !strings.Contains(res, "31") {
		t.Errorf("short view Complexity column did not show cognitive 31:\n%s", res)
	}
	if strings.Contains(res, " 10\n") {
		t.Errorf("short view still shows cyclomatic complexity 10 instead of cognitive:\n%s", res)
	}
}

// TestFileSummarizeShortCognitiveDisabledParity asserts the default view is
// byte-identical to before when --cognitive is off: it shows the cyclomatic
// value and no cognitive anywhere.
func TestFileSummarizeShortCognitiveDisabledParity(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	res := fileSummarizeShort(cognitiveFixtureChan())

	if strings.Contains(res, "Cognitive") {
		t.Errorf("short view leaked Cognitive while disabled:\n%s", res)
	}
	// summed cyclomatic complexity is 10; cognitive 31 must not appear
	if !strings.Contains(res, " 10\n") {
		t.Errorf("short view lost cyclomatic complexity 10 while disabled:\n%s", res)
	}
}

// TestSortSummaryFilesCognitive verifies the sort-key resolution: with
// --cognitive the "complexity" key ranks by cognitive weight, an explicit
// "cognitive" key always ranks by cognitive, and without the flag "complexity"
// still ranks by cyclomatic complexity.
func TestSortSummaryFilesCognitive(t *testing.T) {
	// a has higher cyclomatic, b has higher cognitive — so the orderings differ.
	newSummary := func() *LanguageSummary {
		return &LanguageSummary{Files: []*FileJob{
			{Location: "a", Complexity: 9, Cognitive: 1},
			{Location: "b", Complexity: 1, Cognitive: 9},
		}}
	}
	prevSort := SortBy
	defer func() { SortBy = prevSort }()

	// complexity key, cognitive off → cyclomatic order (a first)
	Cognitive = false
	SortBy = "complexity"
	s := newSummary()
	sortSummaryFiles(s)
	if s.Files[0].Location != "a" {
		t.Errorf("complexity/off: expected a first, got %s", s.Files[0].Location)
	}

	// complexity key, cognitive on → cognitive order (b first)
	Cognitive = true
	defer func() { Cognitive = false }()
	s = newSummary()
	sortSummaryFiles(s)
	if s.Files[0].Location != "b" {
		t.Errorf("complexity/cognitive: expected b first, got %s", s.Files[0].Location)
	}

	// explicit cognitive key → cognitive order (b first) regardless
	SortBy = "cognitive"
	s = newSummary()
	sortSummaryFiles(s)
	if s.Files[0].Location != "b" {
		t.Errorf("cognitive key: expected b first, got %s", s.Files[0].Location)
	}
}

func TestToJSONCognitive(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	res := toJSON(cognitiveFixtureChan())

	var langs []LanguageSummary
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	if err := json.Unmarshal([]byte(res), &langs); err != nil {
		t.Fatalf("failed to unmarshal cognitive JSON: %v", err)
	}
	if len(langs) != 1 {
		t.Fatalf("expected 1 language, got %d", len(langs))
	}
	// round-trips to the summed cognitive value
	if langs[0].Cognitive != 31 {
		t.Errorf("expected language Cognitive 31, got %d", langs[0].Cognitive)
	}
}

func TestToJSONCognitiveDisabledParity(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	// When the flag is off, bumpCognitive never runs so FileJob.Cognitive stays
	// 0; JSON parity relies on omitempty dropping the zero value. Model that here
	// with a zero-cognitive fixture rather than the non-zero shared one.
	inputChan := make(chan *FileJob, 2)
	inputChan <- &FileJob{Language: "Go", Filename: "a.go", Location: "a.go", Lines: 100, Code: 80, Complexity: 5}
	close(inputChan)

	res := toJSON(inputChan)

	if strings.Contains(res, "Cognitive") {
		t.Errorf("JSON leaked Cognitive key while disabled:\n%s", res)
	}
}

func TestToCSVCognitive(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	res := toCSV(cognitiveFixtureChan())

	lines := strings.Split(strings.TrimSpace(res), "\n")
	if !strings.HasSuffix(lines[0], "Cognitive") {
		t.Errorf("CSV header missing trailing Cognitive column: %q", lines[0])
	}
	// summed cognitive for the single Go row is 31
	if !strings.HasSuffix(strings.TrimSpace(lines[1]), ",31") {
		t.Errorf("CSV row missing cognitive value 31: %q", lines[1])
	}
}

func TestToCSVCognitiveDisabledParity(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	res := toCSV(cognitiveFixtureChan())

	if strings.Contains(res, "Cognitive") {
		t.Errorf("CSV leaked Cognitive column while disabled:\n%s", res)
	}
}

func TestToSQLCognitive(t *testing.T) {
	Cognitive = true
	defer func() { Cognitive = false }()

	res := toSql(cognitiveFixtureChan())

	if !strings.Contains(res, "nCognitive") {
		t.Errorf("SQL schema missing nCognitive column:\n%s", res)
	}
	// each per-file insert carries its cognitive value as the final column
	if !strings.Contains(res, ", 11);") || !strings.Contains(res, ", 20);") {
		t.Errorf("SQL inserts missing per-file cognitive values:\n%s", res)
	}
}

func TestToSQLCognitiveDisabledParity(t *testing.T) {
	if Cognitive {
		t.Fatalf("Cognitive should default to false")
	}
	res := toSql(cognitiveFixtureChan())

	if strings.Contains(res, "Cognitive") || strings.Contains(res, "nCognitive") {
		t.Errorf("SQL leaked Cognitive column while disabled:\n%s", res)
	}
}
