// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// TestCollectReportDataOnFixtureRepo runs the orchestrator against a small
// git fixture repo and asserts every section is populated. It exercises the
// happy path of the spec: GitAvailable=true, all four git pointers non-nil,
// per-file table and cost results present.
func TestCollectReportDataOnFixtureRepo(t *testing.T) {
	ProcessConstants()

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\n\nfunc A() {}\n"},
		{"a.go": "package a\n\nfunc A() {}\n\nfunc B() {}\n", "b.go": "package a\n\nfunc C() {}\n"},
		{"a.go": "package a\n\nfunc A() {}\n\nfunc B() {}\n\nfunc D() {}\n"},
	})

	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	if !data.GitAvailable {
		t.Errorf("expected GitAvailable=true for fixture repo, got false")
	}
	if data.RepoName == "" {
		t.Errorf("expected RepoName to be derived from path, got empty")
	}
	if data.SccVersion == "" {
		t.Errorf("expected SccVersion to be set")
	}
	if data.GeneratedAt.IsZero() {
		t.Errorf("expected GeneratedAt to be set")
	}

	if data.Totals.Files == 0 {
		t.Errorf("expected Totals.Files > 0, got 0")
	}
	if len(data.Summary) == 0 {
		t.Errorf("expected Summary to contain at least one language")
	}
	if len(data.Files) == 0 {
		t.Errorf("expected Files slice to be populated")
	}

	if data.ULOC == nil {
		t.Errorf("expected ULOC section, got nil")
	} else if data.ULOC.Global == 0 {
		t.Errorf("expected ULOC.Global > 0, got 0")
	}

	if data.LineLength == nil {
		t.Errorf("expected LineLength section, got nil")
	} else {
		if len(data.LineLength.Buckets) == 0 {
			t.Errorf("expected LineLength.Buckets populated")
		}
		if data.LineLength.Max == 0 {
			t.Errorf("expected LineLength.Max > 0")
		}
	}

	if data.Hotspots == nil {
		t.Errorf("expected Hotspots section to be populated when git available")
	}
	if data.Authors == nil {
		t.Errorf("expected Authors section to be populated when git available")
	}
	if data.LanguageTimeline == nil {
		t.Errorf("expected LanguageTimeline section to be populated when git available")
	}
	if data.AuthorTimeline == nil {
		t.Errorf("expected AuthorTimeline section to be populated when git available")
	}

	if data.Cocomo == nil {
		t.Errorf("expected Cocomo result, got nil")
	}
	if data.Locomo == nil {
		t.Errorf("expected Locomo result, got nil")
	}
}

// TestCollectReportDataOnNonGitDir verifies the git-less path: detect=false,
// all four git pointers nil, but the language/file/cost sections still
// populated.
func TestCollectReportDataOnNonGitDir(t *testing.T) {
	ProcessConstants()

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	if data.GitAvailable {
		t.Errorf("expected GitAvailable=false for non-git tempdir, got true")
	}
	if data.Hotspots != nil {
		t.Errorf("expected Hotspots=nil when git unavailable, got %+v", data.Hotspots)
	}
	if data.Authors != nil {
		t.Errorf("expected Authors=nil when git unavailable, got %+v", data.Authors)
	}
	if data.LanguageTimeline != nil {
		t.Errorf("expected LanguageTimeline=nil when git unavailable")
	}
	if data.AuthorTimeline != nil {
		t.Errorf("expected AuthorTimeline=nil when git unavailable")
	}

	if data.Totals.Files == 0 {
		t.Errorf("expected Totals.Files > 0 even without git, got 0")
	}
	if data.Cocomo == nil {
		t.Errorf("expected Cocomo result even without git, got nil")
	}
}

// TestCollectReportDataRestoresFlags asserts that the package-level flag
// vars mutated inside CollectReportData are restored to their on-entry
// values, even when the report mode flipped them on. Critical for
// in-process re-entrancy.
func TestCollectReportDataRestoresFlags(t *testing.T) {
	ProcessConstants()

	prevUloc, prevMaxMean, prevFiles := UlocMode, MaxMean, Files
	UlocMode, MaxMean, Files = false, false, false
	t.Cleanup(func() {
		UlocMode = prevUloc
		MaxMean = prevMaxMean
		Files = prevFiles
	})

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := CollectReportData(dir); err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	if UlocMode {
		t.Errorf("expected UlocMode restored to false, got true")
	}
	if MaxMean {
		t.Errorf("expected MaxMean restored to false, got true")
	}
	if Files {
		t.Errorf("expected Files restored to false, got true")
	}
}

// TestCollectReportDataHonoursReportSkip verifies --report-skip nilling out
// the matching *Result pointers. Uses a tempdir (no git) so the only
// sections under test are ULOC, line-length, and cost.
func TestCollectReportDataHonoursReportSkip(t *testing.T) {
	ProcessConstants()

	prevSkip := ReportSkipNames
	ReportSkipNames = map[string]bool{
		"uloc":       true,
		"linelength": true,
		"cocomo":     true,
		"locomo":     true,
	}
	t.Cleanup(func() { ReportSkipNames = prevSkip })

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	if data.ULOC != nil {
		t.Errorf("expected ULOC=nil under --report-skip uloc, got %+v", data.ULOC)
	}
	if data.LineLength != nil {
		t.Errorf("expected LineLength=nil under --report-skip linelength, got %+v", data.LineLength)
	}
	if data.Cocomo != nil {
		t.Errorf("expected Cocomo=nil under --report-skip cocomo, got %+v", data.Cocomo)
	}
	if data.Locomo != nil {
		t.Errorf("expected Locomo=nil under --report-skip locomo, got %+v", data.Locomo)
	}
}

// TestRenderReportEmbedsShareCardMeta renders a full report and asserts the
// OpenGraph / Twitter Card meta tags from spec 04 are present with the
// embedded data: URL share card. Locks in the unfurl contract.
func TestRenderReportEmbedsShareCardMeta(t *testing.T) {
	ProcessConstants()

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\n\nfunc A() {}\n"},
		{"a.go": "package a\n\nfunc A() {}\n\nfunc B() {}\n"},
	})

	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	out := filepath.Join(t.TempDir(), "report.html")
	if err := RenderReport(data, out); err != nil {
		t.Fatalf("RenderReport: %v", err)
	}

	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read rendered report: %v", err)
	}
	html := string(body)

	wants := []string{
		`<meta property="og:type" content="website">`,
		`<meta property="og:title" content="scc analysed `,
		`<meta property="og:description" content="`,
		`<meta property="og:image" content="data:image/svg`,
		`<meta name="twitter:card" content="summary_large_image">`,
		`<meta name="twitter:title" content="scc analysed `,
		`<meta name="twitter:description" content="`,
		`<meta name="twitter:image" content="data:image/svg`,
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("rendered report missing %q", w)
		}
	}

	// The description should follow the spec-04 headline shape: files · SLOC.
	if !strings.Contains(html, " files · ") || !strings.Contains(html, " SLOC") {
		t.Errorf("rendered report description doesn't match headline shape; got HTML:\n%s", html)
	}
}

// TestRenderReportSkipCardDropsImageTags asserts that --report-skip card
// suppresses both image meta tags but leaves the text ones intact.
func TestRenderReportSkipCardDropsImageTags(t *testing.T) {
	ProcessConstants()

	prevSkip := ReportSkipNames
	ReportSkipNames = map[string]bool{"card": true}
	t.Cleanup(func() { ReportSkipNames = prevSkip })

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}

	out := filepath.Join(t.TempDir(), "report.html")
	if err := RenderReport(data, out); err != nil {
		t.Fatalf("RenderReport: %v", err)
	}

	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read rendered report: %v", err)
	}
	html := string(body)

	if strings.Contains(html, `property="og:image"`) {
		t.Errorf("expected no og:image when card is skipped")
	}
	if strings.Contains(html, `name="twitter:image"`) {
		t.Errorf("expected no twitter:image when card is skipped")
	}
	// Text tags must still be present.
	for _, w := range []string{`property="og:title"`, `name="twitter:description"`} {
		if !strings.Contains(html, w) {
			t.Errorf("expected text meta tag %q to remain when card is skipped", w)
		}
	}
}

// TestCocomoPrettyCost locks in the headline cost format ("$1,234,567" style).
func TestCocomoPrettyCost(t *testing.T) {
	r := CocomoResult{CurrencySymbol: "$", EstimatedCost: 1234567.89}
	got := r.PrettyCost()
	if got != "$1,234,567" {
		t.Errorf("PrettyCost = %q, want %q", got, "$1,234,567")
	}
}

// withReportSkipReset snapshots ReportSkipNames around a test body and
// restores it afterward. Spec 05 tests mutate the global heavily and we want
// the test order not to matter.
func withReportSkipReset(t *testing.T) {
	t.Helper()
	prev := ReportSkipNames
	t.Cleanup(func() { ReportSkipNames = prev })
}

// TestParseReportSkipKnownNames exercises the happy path: every recognised
// name is parsed, lower-cased, and the warning channel stays silent.
func TestParseReportSkipKnownNames(t *testing.T) {
	withReportSkipReset(t)

	var warn bytes.Buffer
	parseReportSkipTo("Cocomo, Locomo, hotspots, AUTHORS, timeline, files, uloc, linelength, card", &warn)

	for _, name := range []string{"cocomo", "locomo", "hotspots", "authors", "timeline", "files", "uloc", "linelength", "card"} {
		if !ReportSkipped(name) {
			t.Errorf("ReportSkipped(%q) = false, want true after parseReportSkip", name)
		}
	}
	if got := warn.String(); got != "" {
		t.Errorf("expected no warning output for recognised names, got %q", got)
	}
}

// TestParseReportSkipUnknownNameWarns covers spec 05's "unknown names emit a
// warning on stderr, continue" rule. The unknown name is still recorded so
// future template helpers can surface "ignored" hints.
func TestParseReportSkipUnknownNameWarns(t *testing.T) {
	withReportSkipReset(t)

	var warn bytes.Buffer
	parseReportSkipTo("cocomo, bogus, also-bogus", &warn)

	if !ReportSkipped("cocomo") {
		t.Errorf("recognised name should be marked skipped")
	}
	out := warn.String()
	if !strings.Contains(out, "unknown section \"bogus\"") {
		t.Errorf("expected warning to name the bogus section, got %q", out)
	}
	if !strings.Contains(out, "unknown section \"also-bogus\"") {
		t.Errorf("expected a warning per unknown section, got %q", out)
	}
}

// TestParseReportSkipEmptyClearsMap asserts that parsing an empty string
// resets the map. Important because the var is package-level and a previous
// in-process run could otherwise leak state.
func TestParseReportSkipEmptyClearsMap(t *testing.T) {
	withReportSkipReset(t)
	ReportSkipNames = map[string]bool{"cocomo": true}

	var warn bytes.Buffer
	parseReportSkipTo("", &warn)

	if len(ReportSkipNames) != 0 {
		t.Errorf("expected ReportSkipNames cleared by empty input, got %v", ReportSkipNames)
	}
	if warn.Len() != 0 {
		t.Errorf("expected no warning for empty input, got %q", warn.String())
	}
}

// TestReportSkippedCaseInsensitive locks in the spec 05 contract that
// callers can pass either case.
func TestReportSkippedCaseInsensitive(t *testing.T) {
	withReportSkipReset(t)
	ReportSkipNames = map[string]bool{"cocomo": true}

	for _, name := range []string{"cocomo", "Cocomo", "COCOMO"} {
		if !ReportSkipped(name) {
			t.Errorf("ReportSkipped(%q) = false, want true", name)
		}
	}
	if ReportSkipped("hotspots") {
		t.Errorf("ReportSkipped(\"hotspots\") = true, want false")
	}
}

// TestSkippedTemplateHelper renders a tiny template that uses the `skipped`
// helper to gate a block. Locks in the spec 05 contract that the helper is
// registered and reads ReportSkipNames.
func TestSkippedTemplateHelper(t *testing.T) {
	withReportSkipReset(t)
	ReportSkipNames = map[string]bool{"cocomo": true}

	tmpl, err := template.New("t").Funcs(reportFuncs).Parse(
		`{{ if not (skipped "cocomo") }}cocomo-on{{ else }}cocomo-off{{ end }};{{ if not (skipped "hotspots") }}hotspots-on{{ end }}`,
	)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	got := out.String()
	want := "cocomo-off;hotspots-on"
	if got != want {
		t.Errorf("template output = %q, want %q", got, want)
	}
}

// TestDetectRepoNameHonoursReportTitle covers step 1 of the spec 05
// auto-detection chain — an explicit --report-title wins over everything.
func TestDetectRepoNameHonoursReportTitle(t *testing.T) {
	prev := ReportTitle
	ReportTitle = "My Custom Repo"
	t.Cleanup(func() { ReportTitle = prev })

	if got := detectRepoName(t.TempDir()); got != "My Custom Repo" {
		t.Errorf("detectRepoName = %q, want %q", got, "My Custom Repo")
	}
}

// TestDetectRepoNameFallsBackToBasename covers step 3 of the chain: with no
// override and no git remote, the basename of the analysed path is used.
func TestDetectRepoNameFallsBackToBasename(t *testing.T) {
	prev := ReportTitle
	ReportTitle = ""
	t.Cleanup(func() { ReportTitle = prev })

	dir := filepath.Join(t.TempDir(), "myproj")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if got := detectRepoName(dir); got != "myproj" {
		t.Errorf("detectRepoName = %q, want %q", got, "myproj")
	}
}

// TestNoCocomoEquivalentToReportSkipCocomo verifies the spec 05 row that
// --no-cocomo and --report-skip cocomo produce the same outcome (Cocomo
// section nil). Uses a non-git tempdir to keep the run fast.
func TestNoCocomoEquivalentToReportSkipCocomo(t *testing.T) {
	ProcessConstants()
	withReportSkipReset(t)

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Path 1: --no-cocomo (the Cocomo flag is "skip" when true).
	prevCocomo := Cocomo
	Cocomo = true
	t.Cleanup(func() { Cocomo = prevCocomo })

	ReportSkipNames = map[string]bool{}
	dataA, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData (no-cocomo): %v", err)
	}
	if dataA.Cocomo != nil {
		t.Errorf("--no-cocomo: expected Cocomo=nil, got %+v", dataA.Cocomo)
	}

	// Path 2: --report-skip cocomo, with the Cocomo skip flag off.
	Cocomo = false
	ReportSkipNames = map[string]bool{"cocomo": true}
	dataB, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData (skip cocomo): %v", err)
	}
	if dataB.Cocomo != nil {
		t.Errorf("--report-skip cocomo: expected Cocomo=nil, got %+v", dataB.Cocomo)
	}
}

// TestReportImplicitlyEnablesULOCAndLineLength verifies that --report flips
// the ULOC and line-length analysis modes on, and --report-skip suppresses
// them — the spec 05 rows for `--uloc` / `-m` / `--character`.
func TestReportImplicitlyEnablesULOCAndLineLength(t *testing.T) {
	ProcessConstants()
	withReportSkipReset(t)

	prevUloc, prevMaxMean, prevFiles := UlocMode, MaxMean, Files
	UlocMode, MaxMean, Files = false, false, false
	t.Cleanup(func() {
		UlocMode = prevUloc
		MaxMean = prevMaxMean
		Files = prevFiles
	})

	dir := t.TempDir()
	if err := writeTestFile(filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	ReportSkipNames = map[string]bool{}
	data, err := CollectReportData(dir)
	if err != nil {
		t.Fatalf("CollectReportData: %v", err)
	}
	if data.ULOC == nil {
		t.Errorf("expected ULOC populated when --report runs without --report-skip uloc")
	}
	if data.LineLength == nil {
		t.Errorf("expected LineLength populated when --report runs without --report-skip linelength")
	}
	if len(data.Files) == 0 {
		t.Errorf("expected per-file table populated when --report runs without --report-skip files")
	}
}

// TestReportSkipRecognisedListMatchesSpec locks in the exact set of names
// spec 05 enumerates as recognised so a future code change can't silently
// add or drop one without updating the spec.
func TestReportSkipRecognisedListMatchesSpec(t *testing.T) {
	want := []string{"cocomo", "locomo", "hotspots", "authors", "timeline", "files", "uloc", "linelength", "card"}
	if len(reportSkipRecognised) != len(want) {
		t.Errorf("reportSkipRecognised size = %d, want %d", len(reportSkipRecognised), len(want))
	}
	for _, name := range want {
		if !reportSkipRecognised[name] {
			t.Errorf("reportSkipRecognised missing %q", name)
		}
	}
}

// TestCommaFmt locks in thousands-separator rendering for the headline
// numbers (`{{ comma .Totals.Code }}`). Covers boundaries the template hits:
// zero, < 1k, the 4- and 7-digit thresholds, and a negative.
func TestCommaFmt(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{12, "12"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{1000000000, "1,000,000,000"},
	}
	for _, tc := range cases {
		if got := commaFmt(tc.in); got != tc.want {
			t.Errorf("commaFmt(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}

	// The `comma` template helper wraps commaFmt with a negative-sign guard.
	helper := reportFuncs["comma"].(func(int64) string)
	if got := helper(-1234); got != "-1,234" {
		t.Errorf("comma(-1234) = %q, want %q", got, "-1,234")
	}
	if got := helper(0); got != "0" {
		t.Errorf("comma(0) = %q, want %q", got, "0")
	}
}

// TestPctHelper locks in the "{{ pct n d }}" format and the zero-denominator
// guard the template relies on (e.g. when a section is skipped).
func TestPctHelper(t *testing.T) {
	pct := reportFuncs["pct"].(func(int64, int64) string)
	cases := []struct {
		num, denom int64
		want       string
	}{
		{0, 0, "0.0%"},
		{1, 0, "0.0%"},
		{0, 100, "0.0%"},
		{1, 4, "25.0%"},
		{1, 3, "33.3%"},
		{2, 3, "66.7%"},
		{100, 100, "100.0%"},
	}
	for _, tc := range cases {
		if got := pct(tc.num, tc.denom); got != tc.want {
			t.Errorf("pct(%d, %d) = %q, want %q", tc.num, tc.denom, got, tc.want)
		}
	}
}

// TestHumanBytes locks in SI-style byte rendering used by the share card and
// per-file table.
func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0B"},
		{1, "1B"},
		{999, "999B"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{12000, "12.0K"},
		{100000, "100K"},
		{1500000, "1.5M"},
		{1500000000, "1.5G"},
		{1500000000000, "1.5T"},
		{1500000000000000, "1.5P"},
	}
	for _, tc := range cases {
		if got := humanBytes(tc.in); got != tc.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestDonutArcsMath asserts the geometry of the donut: arc dasharrays sum to
// the canonical circumference, offsets are negative cumulative segments, and
// colours flow through langColor.
func TestDonutArcsMath(t *testing.T) {
	if got := donutArcs(nil); got != nil {
		t.Errorf("donutArcs(nil) = %+v, want nil", got)
	}
	if got := donutArcs([]LanguageSummary{{Name: "Go", Code: 0}}); got != nil {
		t.Errorf("donutArcs(zero-total) = %+v, want nil", got)
	}

	arcs := donutArcs([]LanguageSummary{
		{Name: "Go", Code: 75},
		{Name: "JavaScript", Code: 25},
	})
	if len(arcs) != 2 {
		t.Fatalf("donutArcs returned %d arcs, want 2", len(arcs))
	}
	if arcs[0].Color != "#00ADD8" {
		t.Errorf("arc[0].Color = %q, want #00ADD8 (Go)", arcs[0].Color)
	}
	if arcs[1].Color != "#f1e05a" {
		t.Errorf("arc[1].Color = %q, want #f1e05a (JavaScript)", arcs[1].Color)
	}
	if arcs[0].Dashoffset != 0 {
		t.Errorf("arc[0].Dashoffset = %f, want 0", arcs[0].Dashoffset)
	}
	// Second arc's offset is -1*first-arc-segment = -75.
	if arcs[1].Dashoffset != -75 {
		t.Errorf("arc[1].Dashoffset = %f, want -75", arcs[1].Dashoffset)
	}
	if arcs[0].Dasharray != "75.000 25.000" {
		t.Errorf("arc[0].Dasharray = %q, want %q", arcs[0].Dasharray, "75.000 25.000")
	}
	if arcs[1].Dasharray != "25.000 75.000" {
		t.Errorf("arc[1].Dasharray = %q, want %q", arcs[1].Dasharray, "25.000 75.000")
	}
}

// TestSparklinePath covers the SVG-path generator for the timeline tables:
// empty input is "", a single point is a degenerate "M", two-point series
// renders an "M…L…" pair, fill mode closes the path back to the baseline,
// and a flat series doesn't divide by zero.
func TestSparklinePath(t *testing.T) {
	if got := sparklinePath(nil, 100, 20); got != "" {
		t.Errorf("sparklinePath(nil) = %q, want \"\"", got)
	}
	if got := sparklinePath([]int{1, 2, 3}, 0, 20); got != "" {
		t.Errorf("sparklinePath with w=0 should be empty, got %q", got)
	}

	// Single point: x=0, span=1 → y=h (baseline).
	if got := sparklinePath([]int{5}, 100, 20); got != "M0.0,20.0" {
		t.Errorf("sparklinePath([5]) = %q, want %q", got, "M0.0,20.0")
	}

	// Two-point ascending series spans the full box.
	got := sparklinePath([]int{0, 10}, 100, 20)
	if got != "M0.0,20.0 L100.0,0.0" {
		t.Errorf("sparklinePath([0,10]) = %q, want %q", got, "M0.0,20.0 L100.0,0.0")
	}

	// Flat series should not divide by zero — span is forced to 1, so every
	// point lands on the baseline.
	flat := sparklinePath([]int{4, 4, 4}, 100, 20)
	if !strings.Contains(flat, "M0.0,20.0") || !strings.Contains(flat, "L100.0,20.0") {
		t.Errorf("sparklinePath(flat) = %q, want degenerate baseline path", flat)
	}

	// Fill mode closes the path back to the baseline and back to x=0.
	fill := sparklineFill([]int{0, 10}, 100, 20)
	if !strings.HasSuffix(fill, " L100.0,20.0 L0.0,20.0 Z") {
		t.Errorf("sparklineFill should close to baseline, got %q", fill)
	}
}

// TestDataURLCard locks in the og:image data: URL prefix and the space
// escaping the unfurl scrapers require.
func TestDataURLCard(t *testing.T) {
	got := dataURLCard(template.HTML("  <svg>hello world</svg>  "))
	want := "data:image/svg+xml;utf8,%3Csvg%3Ehello%20world%3C%2Fsvg%3E"
	if got != want {
		t.Errorf("dataURLCard = %q, want %q", got, want)
	}

	// Empty input still yields the prefix so the template can embed it
	// unconditionally; the og:image gate is in the template, not here.
	if got := dataURLCard(""); got != "data:image/svg+xml;utf8," {
		t.Errorf("dataURLCard(\"\") = %q, want bare prefix", got)
	}
}

// goldenReportFixture returns a hand-curated ReportData with every
// section populated and every timestamp / duration / floating-point value
// fixed. Used by TestRenderReport_Golden to catch unintended template
// changes. Keep this deterministic — never call time.Now(), never inject
// random IDs, never rely on map iteration order.
func goldenReportFixture() ReportData {
	generated := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	lastCommit := time.Date(2026, 1, 14, 18, 0, 0, 0, time.UTC)

	return ReportData{
		RepoName:     "golden-fixture",
		SccVersion:   "3.8.0-test",
		GeneratedAt:  generated,
		Duration:     1500 * time.Millisecond,
		GitAvailable: true,
		Summary: []LanguageSummary{
			{Name: "Go", Count: 12, Code: 8000, Comment: 600, Blank: 1200, Complexity: 420, ULOC: 5400},
			{Name: "JavaScript", Count: 6, Code: 2400, Comment: 180, Blank: 360, Complexity: 90, ULOC: 1800},
			{Name: "Markdown", Count: 3, Code: 240, Comment: 0, Blank: 60, Complexity: 0, ULOC: 220},
		},
		Totals: Totals{
			Files: 21, Lines: 13040, Code: 10640, Comment: 780, Blank: 1620, Complexity: 510, Bytes: 524288,
		},
		ULOC: &ULOCResult{
			Global: 7420, TotalLines: 13040, Dryness: 0.569,
			PerLanguage: []ULOCLanguage{
				{Language: "Go", ULOC: 5400},
				{Language: "JavaScript", ULOC: 1800},
				{Language: "Markdown", ULOC: 220},
			},
		},
		LineLength: &LineLengthResult{
			Mean: 38.2, Max: 220, TotalLines: 10640,
			Buckets: []LineLengthBucket{
				{Start: 0, End: 20, Label: "0–20", Count: 4200},
				{Start: 20, End: 40, Label: "20–40", Count: 3800},
				{Start: 40, End: 60, Label: "40–60", Count: 1600},
				{Start: 60, End: 80, Label: "60–80", Count: 700},
				{Start: 80, End: 100, Label: "80–100", Count: 220},
				{Start: 100, End: 120, Label: "100–120", Count: 100},
				{Start: 120, End: 0, Label: "120+", Count: 20},
			},
			Outliers: []LineLengthOutlier{
				{File: "internal/wide.go", Language: "Go", LineLength: 220},
				{File: "web/styles.js", Language: "JavaScript", LineLength: 180},
			},
		},
		Hotspots: &HotspotsResult{
			Available: true,
			TotalRaw:  3,
			Records: []HotspotRow{
				{File: "internal/server.go", Language: "Go", Complexity: 240, Commits: 64, LinesChanged: 1200, Authors: 4, Score: 92.5},
				{File: "internal/cache.go", Language: "Go", Complexity: 120, Commits: 28, LinesChanged: 420, Authors: 2, Score: 41.2},
				{File: "web/app.js", Language: "JavaScript", Complexity: 60, Commits: 14, LinesChanged: 180, Authors: 2, Score: 17.8},
			},
		},
		Authors: &AuthorsResult{
			BusFactor:    1,
			BusAuthors:   []string{"Alice"},
			BusCovered:   0.62,
			InWindowCode: 9800,
			Rows: []AuthorRow{
				{Name: "Alice", Email: "alice@example.com", Code: 6200, Files: 14, OwnsPercent: 62.4, InWindowPercent: 64.0, LastCommit: lastCommit},
				{Name: "Bob", Email: "bob@example.com", Code: 2400, Files: 8, OwnsPercent: 24.1, InWindowPercent: 22.0, LastCommit: lastCommit.Add(-48 * time.Hour)},
				{Name: "(before window)", Sentinel: true},
			},
		},
		LanguageTimeline: &LangTimelineResult{
			Buckets: 4,
			Rows: []LangTimelineRow{
				{Language: "Go", StartingLines: 4000, CodeNow: 8000, Change: 4000, SharePercent: 75.2, Trajectory: []int64{4000, 5200, 6800, 8000}},
				{Language: "JavaScript", StartingLines: 2200, CodeNow: 2400, Change: 200, SharePercent: 22.6, Trajectory: []int64{2200, 2300, 2350, 2400}},
				{Language: "Markdown", StartingLines: 200, CodeNow: 240, Change: 40, SharePercent: 2.2, Trajectory: []int64{200, 210, 220, 240}},
			},
		},
		AuthorTimeline: &AuthorTimelineResult{
			Buckets: 4,
			Rows: []AuthorTimelineRow{
				{Name: "Alice", Email: "alice@example.com", TotalCommits: 48, CodeDelta: 3800, Series: []AuthorTimelineBucket{
					{Commits: 8, CodeDelta: 500},
					{Commits: 14, CodeDelta: 1200},
					{Commits: 12, CodeDelta: 900},
					{Commits: 14, CodeDelta: 1200},
				}},
				{Name: "Bob", Email: "bob@example.com", TotalCommits: 22, CodeDelta: 400, Series: []AuthorTimelineBucket{
					{Commits: 6, CodeDelta: 200},
					{Commits: 6, CodeDelta: 100},
					{Commits: 5, CodeDelta: 50},
					{Commits: 5, CodeDelta: 50},
				}},
			},
		},
		Cocomo: &CocomoResult{
			ProjectType: "organic", SumCode: 10640,
			EstimatedEffort: 26.4, EstimatedCost: 297500, ScheduleMonths: 8.4, PeopleRequired: 3.1,
			AverageWage: 56286, Overhead: 2.4, EAF: 1.0, CurrencySymbol: "$",
		},
		Locomo: &LocomoResult{
			InputTokens: 320000, OutputTokens: 96000, Cost: 18.25,
			GenerationSeconds: 1800, ReviewHours: 12, AverageComplexityMult: 1.4,
			IterationFactor: 1.2, Preset: "medium",
		},
		Files: []*FileJob{
			{Location: "internal/server.go", Language: "Go", Lines: 3200, Code: 2600, Comment: 200, Blank: 400, Complexity: 240, Bytes: 96000, Uloc: 2200},
			{Location: "internal/cache.go", Language: "Go", Lines: 1400, Code: 1100, Comment: 120, Blank: 180, Complexity: 120, Bytes: 42000, Uloc: 900},
			{Location: "web/app.js", Language: "JavaScript", Lines: 2000, Code: 1600, Comment: 80, Blank: 320, Complexity: 60, Bytes: 60000, Uloc: 1200},
			{Location: "README.md", Language: "Markdown", Lines: 120, Code: 100, Comment: 0, Blank: 20, Complexity: 0, Bytes: 4096, Uloc: 90},
		},
	}
}

// TestRenderReport_Golden snapshots the full HTML report output against
// testdata/golden-report.html. Run with UPDATE_GOLDEN=1 to refresh after
// an intentional template change. The fixture is hand-curated and
// deterministic — no time.Now, no random colour assignment, no map
// iteration in the input — so a stable template produces stable bytes.
func TestRenderReport_Golden(t *testing.T) {
	ProcessConstants()
	withReportSkipReset(t)
	ReportSkipNames = map[string]bool{}

	d := goldenReportFixture()

	var buf bytes.Buffer
	if err := renderReportTo(&buf, d); err != nil {
		t.Fatalf("renderReportTo: %v", err)
	}
	got := buf.Bytes()

	goldenPath := filepath.Join("testdata", "golden-report.html")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden mismatch (run UPDATE_GOLDEN=1 to refresh)\n--- want (%d bytes)\n--- got  (%d bytes)\nfirst diff at byte %d",
			len(want), len(got), firstByteDiff(want, got))
	}
}

// firstByteDiff returns the index of the first byte at which a and b
// differ, or -1 if they're equal. Used by the golden test's failure
// message so a reviewer can jump straight to the divergence.
func firstByteDiff(a, b []byte) int {
	n := min(len(b), len(a))
	for i := range n {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// TestConfirmReportOverwriteExplicitPathSilent verifies the "explicit
// path = consent" branch: when the user typed --report=foo.html, we
// should not stat, not prompt, not produce a warning. The function must
// succeed even with a nil reader and writer.
func TestConfirmReportOverwriteExplicitPathSilent(t *testing.T) {
	if err := confirmReportOverwrite("anywhere.html", false, true, nil, nil); err != nil {
		t.Errorf("explicit path: expected nil, got %v", err)
	}
	if err := confirmReportOverwrite("anywhere.html", false, false, nil, nil); err != nil {
		t.Errorf("explicit path non-TTY: expected nil, got %v", err)
	}
}

// TestConfirmReportOverwriteMissingFileProceeds covers the bare-flag
// case where the file doesn't exist yet — no prompt, no error.
func TestConfirmReportOverwriteMissingFileProceeds(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "scc-report.html")
	var out bytes.Buffer
	if err := confirmReportOverwrite(missing, true, true, strings.NewReader(""), &out); err != nil {
		t.Errorf("missing file: expected nil, got %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("missing file: expected no prompt, got %q", out.String())
	}
}

// TestConfirmReportOverwritePromptAccepts walks the happy interactive
// path: file exists, stdin attached, user types "y" — proceed.
func TestConfirmReportOverwritePromptAccepts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scc-report.html")
	if err := os.WriteFile(path, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	for _, ans := range []string{"y\n", "Y\n", "yes\n", "YES\n", "  y \n"} {
		var out bytes.Buffer
		err := confirmReportOverwrite(path, true, true, strings.NewReader(ans), &out)
		if err != nil {
			t.Errorf("answer %q: expected nil, got %v", ans, err)
		}
		if !strings.Contains(out.String(), "Overwrite?") {
			t.Errorf("answer %q: expected prompt in stderr, got %q", ans, out.String())
		}
	}
}

// TestConfirmReportOverwritePromptRejects covers the "default to no"
// rule the prompt advertises in its `[y/N]` suffix. Anything that
// isn't an affirmative aborts.
func TestConfirmReportOverwritePromptRejects(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scc-report.html")
	if err := os.WriteFile(path, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	for _, ans := range []string{"n\n", "N\n", "no\n", "\n", "garbage\n"} {
		var out bytes.Buffer
		err := confirmReportOverwrite(path, true, true, strings.NewReader(ans), &out)
		if err == nil {
			t.Errorf("answer %q: expected abort, got nil", ans)
			continue
		}
		if !strings.Contains(err.Error(), "not overwritten") {
			t.Errorf("answer %q: expected 'not overwritten' in error, got %v", ans, err)
		}
	}
}

// TestConfirmReportOverwriteNonTTYRefuses locks in the CI / piped-stdin
// guard: with the default name and an existing file we refuse to
// silently clobber, and the error tells the user how to opt in.
func TestConfirmReportOverwriteNonTTYRefuses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scc-report.html")
	if err := os.WriteFile(path, []byte("<html></html>"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	err := confirmReportOverwrite(path, true, false, strings.NewReader(""), io.Discard)
	if err == nil {
		t.Fatalf("non-TTY: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--report="+path) {
		t.Errorf("non-TTY error should point at explicit-path form, got %v", err)
	}
}

// TestDefaultReportNameMatchesNoOptDefVal locks in the contract between
// main.go's NoOptDefVal wiring and runReport's "did the user supply a
// path" check. If these drift, the prompt logic silently breaks.
func TestDefaultReportNameMatchesNoOptDefVal(t *testing.T) {
	if DefaultReportName != "scc-report.html" {
		t.Errorf("DefaultReportName = %q, want %q (main.go wires this as NoOptDefVal)",
			DefaultReportName, "scc-report.html")
	}
}
