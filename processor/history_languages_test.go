// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
)

func findLanguagesRow(t *testing.T, rows []languagesTimelineRow, language string) languagesTimelineRow {
	t.Helper()
	for _, r := range rows {
		if r.Language == language {
			return r
		}
	}
	t.Fatalf("no row for language %q in %+v", language, rows)
	return languagesTimelineRow{}
}

// TestLanguagesTimelineTSRisesJSFalls verifies the trajectory shape — when
// TypeScript code is added over time while JavaScript code is steadily
// removed, the trajectory and change sign should reflect that.
func TestLanguagesTimelineTSRisesJSFalls(t *testing.T) {
	// Set HistoryDepth so the JS-baseline commit (commit 0) sits OUTSIDE
	// the window — the engine then seeds JavaScript with that file's lines
	// via BaselineSnapshot. Subsequent commits inside the window remove
	// JS and add TS.
	saveDepth, saveBuckets := HistoryDepth, HistoryBuckets
	HistoryDepth, HistoryBuckets = 9, 10
	t.Cleanup(func() {
		HistoryDepth, HistoryBuckets = saveDepth, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	// Commit 0 (outside window when depth=9): app.js has 20 lines.
	bigJS := "var x = 1;\n"
	for i := range 19 {
		bigJS += "var y" + itoa(i) + " = " + itoa(i) + ";\n"
	}

	commits := []timelineCommit{
		{
			Files:  map[string]string{"app.js": bigJS},
			Author: "Alice", Email: "a@x", When: base,
		},
	}

	// Commits 1..9 (inside window): each removes 2 JS lines and adds 4 TS lines.
	currentJS := bigJS
	currentTS := ""
	for d := 1; d <= 9; d++ {
		jsLines := strings.Split(strings.TrimRight(currentJS, "\n"), "\n")
		if len(jsLines) > 2 {
			jsLines = jsLines[:len(jsLines)-2]
		}
		currentJS = strings.Join(jsLines, "\n") + "\n"
		for k := range 4 {
			currentTS += "const z" + itoa(d) + "_" + itoa(k) + ": number = " + itoa(k) + ";\n"
		}
		commits = append(commits, timelineCommit{
			Files: map[string]string{
				"app.js": currentJS,
				"app.ts": currentTS,
			},
			Author: "Bob", Email: "b@x", When: base.Add(time.Duration(d) * day),
		})
	}

	dir := makeTimelineRepo(t, commits)

	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	ts := findLanguagesRow(t, obs.rows, "TypeScript")
	js := findLanguagesRow(t, obs.rows, "JavaScript")

	if ts.Change <= 0 {
		t.Errorf("TypeScript change = %d, want positive", ts.Change)
	}
	if js.Change >= 0 {
		t.Errorf("JavaScript change = %d, want negative", js.Change)
	}
	// Trajectory: TS should end higher than its lowest point, JS should
	// end lower than its starting baseline.
	tsLast := ts.Trajectory[len(ts.Trajectory)-1]
	if tsLast == 0 {
		t.Errorf("TS trajectory should be non-zero at end; traj=%v", ts.Trajectory)
	}
	if js.CodeNow >= js.StartingLines {
		t.Errorf("JS codeNow %d should be below starting %d; traj=%v",
			js.CodeNow, js.StartingLines, js.Trajectory)
	}
}

// TestLanguagesTimelineLanguageRemoval — a language that exists at the start
// of the window and is then wholly removed should show codeNow == 0 and a
// negative change.
func TestLanguagesTimelineLanguageRemoval(t *testing.T) {
	// depth=2 → only the last 2 commits are in the window. The first
	// commit (which establishes the JS file) sits OUTSIDE the window and
	// becomes the baseline — so JavaScript starts with code, then drops
	// to zero across the in-window commits.
	saveDepth, saveBuckets := HistoryDepth, HistoryBuckets
	HistoryDepth, HistoryBuckets = 2, 6
	t.Cleanup(func() {
		HistoryDepth, HistoryBuckets = saveDepth, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	jsContent := "var a = 1;\nvar b = 2;\nvar c = 3;\nvar d = 4;\n"
	commits := []timelineCommit{
		// Outside the window: JS exists.
		{
			Files:  map[string]string{"old.js": jsContent, "keep.go": "package k\nfunc K() {}\n"},
			Author: "Alice", Email: "a@x", When: base,
		},
		// In-window: reduce JS.
		{
			Files:  map[string]string{"old.js": "var a = 1;\nvar b = 2;\n"},
			Author: "Alice", Email: "a@x", When: base.Add(24 * time.Hour),
		},
		// In-window: remove all JS lines.
		{
			Files:  map[string]string{"old.js": "\n"},
			Author: "Alice", Email: "a@x", When: base.Add(48 * time.Hour),
		},
	}

	dir := makeTimelineRepo(t, commits)
	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	js := findLanguagesRow(t, obs.rows, "JavaScript")
	if js.CodeNow != 0 {
		t.Errorf("JavaScript codeNow = %d, want 0; traj=%v", js.CodeNow, js.Trajectory)
	}
	if js.Change >= 0 {
		t.Errorf("JavaScript change = %d, want negative", js.Change)
	}
}

// TestLanguagesTimelineSharesSumToHundred — sum of share percentages across
// all surviving languages should round to ~100.
func TestLanguagesTimelineSharesSumToHundred(t *testing.T) {
	saveDepth, saveBuckets := HistoryDepth, HistoryBuckets
	HistoryDepth, HistoryBuckets = 100, 8
	t.Cleanup(func() {
		HistoryDepth, HistoryBuckets = saveDepth, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files: map[string]string{
				"a.go": "package a\nfunc A() {}\nfunc B() {}\n",
				"b.py": "def f():\n    return 1\n",
				"c.rs": "fn main() {}\n",
			},
			Author: "Alice", Email: "a@x", When: base,
		},
		{
			Files: map[string]string{
				"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n",
			},
			Author: "Bob", Email: "b@x", When: base.Add(24 * time.Hour),
		},
	})

	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if len(obs.rows) == 0 {
		t.Fatal("no language rows produced")
	}

	var total float64
	for _, r := range obs.rows {
		total += r.SharePercent
	}
	if total < 99.5 || total > 100.5 {
		t.Errorf("share total = %.2f, want ~100", total)
	}
}

// TestLanguagesTimelineCSVLongFormat — long format has one row per
// (language × bucket).
func TestLanguagesTimelineCSVLongFormat(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "csv", 10
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\n"},
			Author: "Alice", Email: "a@x", When: base,
		},
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Bob", Email: "b@x", When: base.Add(48 * time.Hour),
		},
	})

	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderLanguagesTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.HasPrefix(out, "# window:") {
		t.Fatalf("CSV should start with '# window:' comment:\n%s", out)
	}
	if !strings.Contains(out, "# buckets: 10\n") {
		t.Errorf("CSV missing '# buckets: 10' line:\n%s", out)
	}

	lines := strings.SplitN(out, "\n", 3)
	body := lines[2]
	r := csv.NewReader(strings.NewReader(body))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v\n%s", err, body)
	}
	wantHeader := []string{"Language", "BucketStart", "Code", "CodeNow", "SharePercent", "Change"}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Errorf("header col %d = %q, want %q", i, rows[0][i], h)
		}
	}
	// Long format: nLanguages * buckets body rows.
	if got, want := len(rows)-1, len(obs.rows)*HistoryBuckets; got != want {
		t.Errorf("CSV body rows = %d, want languages*buckets = %d", got, want)
	}
}

// TestLanguagesTimelineJSONShape — the JSON schema matches the plan.
func TestLanguagesTimelineJSONShape(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "json", 8
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\n"},
			Author: "Alice", Email: "a@x", When: base,
		},
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Bob", Email: "b@x", When: base.Add(168 * time.Hour),
		},
	})

	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderLanguagesTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var doc languagesTimelineJSONDoc
	if err := jsoniter.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json parse: %v, body:\n%s", err, out)
	}
	if doc.Report != "languages-timeline" {
		t.Errorf("report = %q, want languages-timeline", doc.Report)
	}
	if doc.Buckets != 8 {
		t.Errorf("buckets = %d, want 8", doc.Buckets)
	}
	if doc.Window.Commits != 2 {
		t.Errorf("window.commits = %d, want 2", doc.Window.Commits)
	}
	if len(doc.Languages) == 0 {
		t.Fatal("expected at least one language")
	}
	for _, l := range doc.Languages {
		if len(l.Series) != 8 {
			t.Errorf("language %q series len = %d, want 8", l.Language, len(l.Series))
		}
	}
}

// TestLanguagesTimelineSparklinePeakNotFullBlock — the unicode tick set
// excludes U+2588 (full block) so peak cells leave a 1-pixel gap at top.
// Adjacent tall cells would otherwise merge into a solid wall when a
// trajectory rises monotonically (as language line counts typically do).
func TestLanguagesTimelineSparklinePeakNotFullBlock(t *testing.T) {
	saveCi := Ci
	Ci = false
	t.Cleanup(func() { Ci = saveCi })
	// FileOutput non-empty triggers ascii fallback; ensure it's clear.
	saveFile := FileOutput
	FileOutput = ""
	t.Cleanup(func() { FileOutput = saveFile })

	// asciiOutput() also short-circuits to true when stdout is not a TTY,
	// which is the case in `go test`. Drive renderSparkline directly via
	// the branch we care about by setting Ci=false and exercising the
	// helper through its rune output: we just verify the helper never
	// emits U+2588.
	out := renderSparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 8)
	if strings.ContainsRune(out, '█') {
		t.Errorf("sparkline contains U+2588 full block: %q", out)
	}
}

// TestLanguagesTimelineSparklineAsciiUnderCi — under --ci the sparkline
// must be glyph-free ASCII.
func TestLanguagesTimelineSparklineAsciiUnderCi(t *testing.T) {
	saveCi := Ci
	Ci = true
	t.Cleanup(func() { Ci = saveCi })

	traj := []int64{0, 1, 2, 3, 5, 8, 13}
	out := renderLanguagesTrajectorySparkline(traj, 12)
	for _, r := range out {
		if r > 127 {
			t.Fatalf("CI sparkline contains non-ASCII rune %U (%q)", r, out)
		}
	}
}

// TestLanguagesTimelineSparklineDownsampling — sparkline produces exactly the
// number of cells requested regardless of input size.
func TestLanguagesTimelineSparklineDownsampling(t *testing.T) {
	traj := make([]int64, 60)
	for i := range traj {
		traj[i] = int64(i)
	}
	for _, cells := range []int{26, 56, 8} {
		out := renderLanguagesTrajectorySparkline(traj, cells)
		if got := runeCount(out); got != cells {
			t.Errorf("sparkline cells=%d produced %d runes (%q)", cells, got, out)
		}
	}
}

// TestLanguagesTimelineTabularHeader — tabular header contains expected
// column labels.
func TestLanguagesTimelineTabularHeader(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "tabular", 8
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Alice", Email: "a@x", When: base,
		},
	})

	obs := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderLanguagesTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Languages", "Trend", "Code", "Share", "Change"} {
		if !strings.Contains(out, want) {
			t.Errorf("tabular missing %q column:\n%s", want, out)
		}
	}
}

// TestLanguagesTimelineRejectsUnsupportedFormat — unknown --format is an
// error.
func TestLanguagesTimelineRejectsUnsupportedFormat(t *testing.T) {
	saveFormat := Format
	Format = "xml"
	t.Cleanup(func() { Format = saveFormat })

	obs := newHistoryLanguagesObserver(8)
	if _, err := renderLanguagesTimeline(obs); err == nil {
		t.Fatal("expected error for --format xml")
	}
}

// TestLanguagesTimelineDropsAbsentLanguages — a language that never appears
// in the window (no starting lines, no observed changes) should not produce
// a row.
func TestLanguagesTimelineDropsAbsentLanguages(t *testing.T) {
	obs := newHistoryLanguagesObserver(4)
	obs.Finalise(HistoryWindow{
		Depth:   10,
		Commits: 0,
		From:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		To:      time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}, emptySnapshot())

	if len(obs.rows) != 0 {
		t.Errorf("expected no rows for empty window, got %+v", obs.rows)
	}
}
