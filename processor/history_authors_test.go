// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	jsoniter "github.com/json-iterator/go"
)

// authoredCommit is one step in a fixture history. Files maps relative path
// to the full file content at this commit. Author/Email override the default
// per-commit identity.
type authoredCommit struct {
	Files  map[string]string
	Author string
	Email  string
}

// makeAuthoredRepo builds a temp on-disk repo with a sequence of commits each
// using the caller's named author. Used to exercise per-author attribution
// in the authors observer.
func makeAuthoredRepo(t *testing.T, commits []authoredCommit) string {
	t.Helper()
	ProcessConstants()
	dir := t.TempDir()

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("init repo: %v", err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatalf("worktree: %v", err)
	}

	when := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	for i, snap := range commits {
		for path, content := range snap.Files {
			full := filepath.Join(dir, path)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", full, err)
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatalf("write %s: %v", full, err)
			}
			if _, err := wt.Add(path); err != nil {
				t.Fatalf("add %s: %v", path, err)
			}
		}
		_, err := wt.Commit("commit "+itoa(i), &git.CommitOptions{
			Author: &object.Signature{
				Name:  snap.Author,
				Email: snap.Email,
				When:  when.Add(time.Duration(i) * time.Hour),
			},
		})
		if err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	return dir
}

// findAuthorRow returns the row whose canonical name matches `name`, or
// fails the test.
func findAuthorRow(t *testing.T, rows []authorRow, name string) authorRow {
	t.Helper()
	for _, r := range rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no row found for %q in %+v", name, rows)
	return authorRow{}
}

func TestAuthorsLastToucherAttribution(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Alice introduces 7 lines; Bob rewrites lines 4–7 (4 lines).
	// Net: Alice owns 3, Bob owns 4. No (before window) — full history walked.
	first := "package x\nfunc A() {}\nfunc B() {}\nfunc C() {}\nfunc D() {}\nfunc E() {}\nfunc F() {}\n"
	// 7 lines: line1=package, line2=A, line3=B, line4=C, line5=D, line6=E, line7=F.
	// We want Bob to rewrite lines 4-7 (4 lines).
	second := "package x\nfunc A() {}\nfunc B() {}\nfunc CC() {}\nfunc DD() {}\nfunc EE() {}\nfunc FF() {}\n"

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"main.go": first}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"main.go": second}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	alice := findAuthorRow(t, obs.rows, "Alice")
	bob := findAuthorRow(t, obs.rows, "Bob")

	if alice.Code != 3 {
		t.Errorf("Alice Code = %d, want 3", alice.Code)
	}
	if bob.Code != 4 {
		t.Errorf("Bob Code = %d, want 4", bob.Code)
	}
}

func TestAuthorsPercentagesSumTo100(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	sum := 0.0
	for _, r := range obs.rows {
		sum += r.OwnsPercent
	}
	if sum < 99.99 || sum > 100.01 {
		t.Errorf("OwnsPercent sum = %f, want ~100", sum)
	}
}

func TestAuthorsBaselineSentinel(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 1
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Two commits, depth=1 keeps only the second; the first becomes baseline.
	first := "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"               // 4 lines, all "before window"
	second := "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\nfunc D() {}\n" // adds 1

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": first}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"a.go": second}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	// Bob is the only real author in the window; expect (before window) sentinel
	// to hold Alice's surviving lines.
	var sentinel *authorRow
	for i := range obs.rows {
		if obs.rows[i].Sentinel {
			sentinel = &obs.rows[i]
		}
	}
	if sentinel == nil {
		t.Fatalf("no sentinel row; got rows %+v", obs.rows)
	}
	if sentinel.Code == 0 {
		t.Errorf("sentinel Code = 0, want surviving baseline lines")
	}
	if sentinel.OwnsPercent < 50 {
		t.Errorf("sentinel OwnsPercent = %f, want > 50 (only 1 line added in window)", sentinel.OwnsPercent)
	}
}

func TestAuthorsMailmapFolding(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Same person under two emails — mailmap folds them.
	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{
			".mailmap": "Alice <alice@example.com> <alt@example.com>\n",
			"a.go":     "package a\nfunc A() {}\nfunc B() {}\n",
		}, Author: "Alice", Email: "alice@example.com"},
		{Files: map[string]string{
			"a.go": "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n",
		}, Author: "Alice", Email: "alt@example.com"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	// Should collapse to a single Alice row.
	aliceCount := 0
	for _, r := range obs.rows {
		if r.Name == "Alice" {
			aliceCount++
		}
	}
	if aliceCount != 1 {
		t.Errorf("Alice rows after mailmap fold = %d, want 1; rows = %+v", aliceCount, obs.rows)
	}
}

func TestAuthorsBusFactorDominant(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Alice writes most code; Bob adds a tiny bit. Bus factor should be 1.
	bigAlice := "package x\n" + strings.Repeat("func F() {}\n", 20)
	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": bigAlice}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"b.go": "package y\nfunc B() {}\n"}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if obs.busFactor != 1 {
		t.Errorf("busFactor = %d, want 1", obs.busFactor)
	}
}

func TestAuthorsBusFactorBalanced(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 100
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Three roughly-equal contributors; bus factor should be >= 2.
	chunk := func(name string) string {
		return "package " + name + "\n" + strings.Repeat("func F() {}\n", 5)
	}
	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": chunk("a")}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"b.go": chunk("b")}, Author: "Bob", Email: "bob@x"},
		{Files: map[string]string{"c.go": chunk("c")}, Author: "Carol", Email: "carol@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if obs.busFactor < 2 {
		t.Errorf("busFactor = %d, want >= 2", obs.busFactor)
	}
}

func TestAuthorsBusFactorIgnoresSentinel(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 2
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Big baseline commit predates the window (depth=2 keeps only the last
	// two commits). Alice and Bob each add a small amount in-window. The
	// surviving HEAD is dominated by the baseline (sentinel), but bus factor
	// must reflect Alice/Bob, not be diluted by the sentinel.
	baseline := "package x\n" + strings.Repeat("func B() {}\n", 50)
	aliceAdd := baseline + "func A1() {}\nfunc A2() {}\nfunc A3() {}\n"
	bobAdd := aliceAdd + "func Bo1() {}\nfunc Bo2() {}\n"

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": baseline}, Author: "Zed", Email: "zed@x"},
		{Files: map[string]string{"a.go": aliceAdd}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"a.go": bobAdd}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if obs.busFactor < 1 || obs.busFactor > 2 {
		t.Errorf("busFactor = %d, want 1 or 2", obs.busFactor)
	}

	// Sentinel must exist and dominate share-of-all, but must not appear in
	// the bus-factor walk.
	var sentinelFound bool
	for _, r := range obs.rows {
		if r.Sentinel {
			sentinelFound = true
			if r.OwnsPercent < 50 {
				t.Errorf("sentinel OwnsPercent = %.1f, want > 50 (most code is pre-window)", r.OwnsPercent)
			}
			if r.InWindowPercent != 0 {
				t.Errorf("sentinel InWindowPercent = %.1f, want 0", r.InWindowPercent)
			}
		}
	}
	if !sentinelFound {
		t.Fatalf("no sentinel row; got rows %+v", obs.rows)
	}

	// busCovered is over in-window code; with only Alice+Bob in-window it
	// must reach > 50 within the (1 or 2) walked authors. The old behavior
	// would have left busCovered ~ a few percent, diluted by the sentinel.
	if obs.busCovered <= 50 {
		t.Errorf("busCovered = %.1f, want > 50 (in-window denominator)", obs.busCovered)
	}
}

func TestAuthorsBusFactorAllPreWindow(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 1
	t.Cleanup(func() { HistoryDepth = saveDepth })

	// Two commits: depth=1 keeps only the second. The second commit just
	// deletes a file, so every surviving line at HEAD came from the baseline
	// — there is no in-window code at all.
	first := "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"
	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{
			"a.go": first,
			"b.go": "package b\nfunc B() {}\n",
		}, Author: "Alice", Email: "alice@x"},
		// Second commit: rewrite b.go to be empty — does not modify a.go,
		// so the only HEAD code is Alice's a.go, all pre-window.
		{Files: map[string]string{"b.go": ""}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if obs.inWindowCode != 0 {
		t.Errorf("inWindowCode = %d, want 0", obs.inWindowCode)
	}
	if obs.busFactor != 0 {
		t.Errorf("busFactor = %d, want 0", obs.busFactor)
	}

	footer := formatAuthorsFooter(obs, 79)
	if !strings.Contains(footer, "no code touched in window") {
		t.Errorf("footer = %q, want 'no code touched in window'", footer)
	}
}

func TestAuthorsCSVIncludesEveryAuthorAndSentinel(t *testing.T) {
	saveDepth, saveFormat := HistoryDepth, Format
	HistoryDepth, Format = 1, "csv"
	t.Cleanup(func() { HistoryDepth, Format = saveDepth, saveFormat })

	first := "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\n"
	second := "package a\nfunc A() {}\nfunc B() {}\nfunc C() {}\nfunc D() {}\n"
	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": first}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"a.go": second}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthors(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.HasPrefix(out, "# window:") {
		t.Fatalf("CSV should start with '# window:' comment, got:\n%s", out)
	}

	body := strings.SplitN(out, "\n", 2)[1]
	r := csv.NewReader(strings.NewReader(body))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v", err)
	}
	wantHeader := []string{"Author", "Email", "Code", "Complexity", "Comment", "Files", "OwnsPercent", "LastCommit", "BeforeWindow"}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Errorf("header col %d = %q, want %q", i, rows[0][i], h)
		}
	}

	// Find Bob and sentinel row.
	var sawBob, sawSentinel bool
	for _, row := range rows[1:] {
		if row[0] == "Bob" {
			sawBob = true
		}
		if row[len(row)-1] == "true" {
			sawSentinel = true
		}
	}
	if !sawBob {
		t.Errorf("CSV missing Bob row")
	}
	if !sawSentinel {
		t.Errorf("CSV missing (before window) sentinel row")
	}
}

func TestAuthorsJSONShape(t *testing.T) {
	saveDepth, saveFormat := HistoryDepth, Format
	HistoryDepth, Format = 100, "json"
	t.Cleanup(func() { HistoryDepth, Format = saveDepth, saveFormat })

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": "package a\nfunc A() {}\n"}, Author: "Alice", Email: "alice@x"},
		{Files: map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"}, Author: "Bob", Email: "bob@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthors(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var doc authorsJSONDoc
	if err := jsoniter.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json parse: %v, body:\n%s", err, out)
	}
	if doc.Report != "authors" {
		t.Errorf("report = %q, want authors", doc.Report)
	}
	if doc.Window.Commits != 2 {
		t.Errorf("window.commits = %d, want 2", doc.Window.Commits)
	}
	if len(doc.Authors) == 0 {
		t.Fatalf("no authors in JSON output")
	}
	// A real author has Name/Email set.
	var foundReal bool
	for _, a := range doc.Authors {
		if a.Name != nil && *a.Name != "" {
			foundReal = true
		}
	}
	if !foundReal {
		t.Errorf("no real author with name field in JSON: %+v", doc.Authors)
	}
}

func TestRenderAuthorsRejectsUnsupportedFormat(t *testing.T) {
	saveFormat := Format
	Format = "xml"
	t.Cleanup(func() { Format = saveFormat })
	obs := newHistoryAuthorsObserver()
	if _, err := renderAuthors(obs); err == nil {
		t.Fatal("expected error for --format xml")
	}
}

func TestWrapBusFactorFooterFitsSingleLine(t *testing.T) {
	got := wrapBusFactorFooter("Bus factor 2 · ", []string{"Alice", "Bob"},
		" last-touched 80% of code", 79)
	if strings.Contains(got, "\n") {
		t.Errorf("short footer should not wrap: %q", got)
	}
}

func TestWrapBusFactorFooterBreaksOnTokenBoundary(t *testing.T) {
	names := []string{}
	for i := range 20 {
		names = append(names, "Author"+itoa(i))
	}
	got := wrapBusFactorFooter("Bus factor 20 · ", names,
		" last-touched 60% of code", 79)
	if !strings.Contains(got, "\n") {
		t.Errorf("long footer should wrap: %q", got)
	}
	for line := range strings.SplitSeq(got, "\n") {
		if runewidthStringWidthForTest(line) > 79 {
			t.Errorf("wrapped line exceeds 79 cols (%d): %q",
				runewidthStringWidthForTest(line), line)
		}
	}
}

func runewidthStringWidthForTest(s string) int {
	w := 0
	for _, r := range s {
		if r == '\t' {
			w++
			continue
		}
		w++
	}
	return w
}

func TestAuthorsTabularContainsBusFactorFooter(t *testing.T) {
	saveDepth, saveFormat := HistoryDepth, Format
	HistoryDepth, Format = 100, "tabular"
	t.Cleanup(func() { HistoryDepth, Format = saveDepth, saveFormat })

	dir := makeAuthoredRepo(t, []authoredCommit{
		{Files: map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"}, Author: "Alice", Email: "alice@x"},
	})

	obs := newHistoryAuthorsObserver()
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthors(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "Bus factor") {
		t.Errorf("tabular output missing 'Bus factor' footer:\n%s", out)
	}
	if !strings.Contains(out, "Authors") {
		t.Errorf("tabular output missing 'Authors' header:\n%s", out)
	}
}
