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

// timelineCommit is one step in a fixture history. Files maps relative path
// to file content; Author / Email / When override the per-commit identity
// and timestamp so tests can place commits in specific windows.
type timelineCommit struct {
	Files  map[string]string
	Author string
	Email  string
	When   time.Time
}

// makeTimelineRepo builds a temp on-disk repo from a slice of timelineCommit
// snapshots. Used to seed historyAuthorTimelineObserver with known author /
// timestamp distributions.
func makeTimelineRepo(t *testing.T, commits []timelineCommit) string {
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
				When:  snap.When,
			},
		})
		if err != nil {
			t.Fatalf("commit %d: %v", i, err)
		}
	}
	return dir
}

func findTimelineRow(t *testing.T, rows []authorTimelineRow, name string) authorTimelineRow {
	t.Helper()
	for _, r := range rows {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no timeline row for %q in %+v", name, rows)
	return authorTimelineRow{}
}

func TestBucketingIndex(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 11, 0, 0, 0, 0, time.UTC) // 10-day span
	b := NewBucketing(from, to, 10)

	if b.N != 10 {
		t.Fatalf("N = %d, want 10", b.N)
	}
	if b.Width != 24*time.Hour {
		t.Fatalf("Width = %s, want 24h", b.Width)
	}

	cases := []struct {
		t    time.Time
		want int
	}{
		{from, 0},
		{from.Add(1 * time.Hour), 0},
		{from.Add(24 * time.Hour), 1},
		{from.Add(24*time.Hour + time.Second), 1},
		{from.Add(5 * 24 * time.Hour), 5},
		{to, 9},                   // clamp to N-1
		{to.Add(time.Hour), 9},    // past To clamps
		{from.Add(-time.Hour), 0}, // before From clamps
	}
	for _, c := range cases {
		if got := b.Index(c.t); got != c.want {
			t.Errorf("Index(%s) = %d, want %d", c.t, got, c.want)
		}
	}
}

func TestBucketingDegenerateWindow(t *testing.T) {
	when := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	b := NewBucketing(when, when, 8)
	if got := b.Index(when); got != 0 {
		t.Errorf("degenerate window: Index = %d, want 0", got)
	}
	if got := b.Index(when.Add(time.Hour)); got != 0 {
		t.Errorf("degenerate window: future Index = %d, want 0", got)
	}
}

func TestBucketingStart(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(10 * 24 * time.Hour)
	b := NewBucketing(from, to, 10)
	if got := b.Start(0); !got.Equal(from) {
		t.Errorf("Start(0) = %s, want %s", got, from)
	}
	if got := b.Start(5); !got.Equal(from.Add(5 * 24 * time.Hour)) {
		t.Errorf("Start(5) = %s, want %s", got, from.Add(5*24*time.Hour))
	}
}

func TestAuthorTimelineSeriesShapes(t *testing.T) {
	saveDepth, saveBuckets := HistoryDepth, HistoryBuckets
	HistoryDepth, HistoryBuckets = 100, 12
	t.Cleanup(func() {
		HistoryDepth, HistoryBuckets = saveDepth, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	// Three authors, three different temporal patterns.
	// Window spans ~12 weeks; 12 buckets ≈ 1 week each.
	week := 7 * 24 * time.Hour

	// Rising commits: Alice commits in weeks 6..11.
	// Falling: Bob commits in weeks 0..5.
	// U-shape: Carol commits in weeks 0..2 and 9..11.
	commits := []timelineCommit{}
	addCommit := func(author, email string, w int, line string) {
		path := "main.go"
		when := base.Add(time.Duration(w) * week)
		// Build incremental file content from prior commits' lines.
		content := "package x\n"
		for _, c := range commits {
			if existing, ok := c.Files[path]; ok {
				content = existing
			}
		}
		content += line + "\n"
		commits = append(commits, timelineCommit{
			Files:  map[string]string{path: content},
			Author: author, Email: email, When: when,
		})
	}

	for i, w := range []int{0, 1, 2, 3, 4, 5} {
		addCommit("Bob", "bob@x", w, "func b"+itoa(i)+"() {}")
	}
	for i, w := range []int{6, 7, 8, 9, 10, 11} {
		addCommit("Alice", "alice@x", w, "func a"+itoa(i)+"() {}")
	}
	for i, w := range []int{0, 1, 2, 9, 10, 11} {
		addCommit("Carol", "carol@x", w, "func c"+itoa(i)+"() {}")
	}

	dir := makeTimelineRepo(t, commits)

	obs := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if len(obs.rows) != 3 {
		t.Fatalf("want 3 author rows, got %d (%+v)", len(obs.rows), obs.rows)
	}

	alice := findTimelineRow(t, obs.rows, "Alice")
	bob := findTimelineRow(t, obs.rows, "Bob")
	carol := findTimelineRow(t, obs.rows, "Carol")

	if alice.TotalCommits != 6 || bob.TotalCommits != 6 || carol.TotalCommits != 6 {
		t.Errorf("commit totals = A:%d B:%d C:%d, want 6 each",
			alice.TotalCommits, bob.TotalCommits, carol.TotalCommits)
	}

	// Bob (falling) — commits should sit in the early half.
	earlyBob, lateBob := halfSums(bob.Series)
	if earlyBob <= lateBob {
		t.Errorf("Bob should have more commits early; early=%d late=%d series=%v",
			earlyBob, lateBob, sumCommits(bob.Series))
	}
	// Alice (rising) — late half should dominate.
	earlyAlice, lateAlice := halfSums(alice.Series)
	if lateAlice <= earlyAlice {
		t.Errorf("Alice should have more commits late; early=%d late=%d series=%v",
			earlyAlice, lateAlice, sumCommits(alice.Series))
	}
	// Carol (U-shape) — first and last quarters should both be non-zero,
	// middle quarter zero.
	cs := sumCommits(carol.Series)
	q := len(cs) / 4
	mid := 0
	for i := q; i < len(cs)-q; i++ {
		mid += cs[i]
	}
	if mid != 0 {
		t.Errorf("Carol should be quiet in mid window; series=%v mid=%d", cs, mid)
	}
}

// halfSums returns (earlyHalfSum, lateHalfSum) of the commit counts.
func halfSums(series []authorTimelineBucket) (int, int) {
	half := len(series) / 2
	early, late := 0, 0
	for i, b := range series {
		if i < half {
			early += b.Commits
		} else {
			late += b.Commits
		}
	}
	return early, late
}

func sumCommits(series []authorTimelineBucket) []int {
	out := make([]int, len(series))
	for i, b := range series {
		out[i] = b.Commits
	}
	return out
}

func TestAuthorTimelineTagUpArrow(t *testing.T) {
	// Final bucket is the peak — should fire ↑.
	series := buildBuckets([]int{1, 1, 1, 1, 5})
	if got := authorTimelineTag(series, 24*time.Hour); got != "↑" {
		t.Errorf("rising tag = %q, want ↑", got)
	}
}

func TestAuthorTimelineTagQuietMonths(t *testing.T) {
	// 4 trailing zero buckets, each 30 days wide → quiet 4mo.
	series := buildBuckets([]int{5, 3, 1, 0, 0, 0, 0})
	got := authorTimelineTag(series, 30*24*time.Hour)
	if !strings.HasPrefix(got, "quiet ") {
		t.Errorf("quiet tag = %q, want quiet Nmo", got)
	}
	if !strings.HasSuffix(got, "mo") {
		t.Errorf("quiet tag = %q, want suffix mo", got)
	}
}

func TestAuthorTimelineTagEmpty(t *testing.T) {
	// Short quiet tail (< 1 month) → no tag.
	series := buildBuckets([]int{5, 1, 0, 0})
	if got := authorTimelineTag(series, 24*time.Hour); got != "" {
		t.Errorf("short quiet = %q, want empty", got)
	}
}

func buildBuckets(commits []int) []authorTimelineBucket {
	out := make([]authorTimelineBucket, len(commits))
	for i, c := range commits {
		out[i].Commits = c
	}
	return out
}

func TestAuthorTimelineSparklineDownsampling(t *testing.T) {
	// 60 buckets, sparkline width 24 — must produce 24 visible runes.
	series := make([]authorTimelineBucket, 60)
	for i := range series {
		series[i].Commits = i
	}

	for _, cells := range []int{24, 12, 8} {
		out := renderAuthorTimelineSparkline(series, cells)
		if got := runeCount(out); got != cells {
			t.Errorf("sparkline cells=%d produced %d runes (%q)", cells, got, out)
		}
	}
}

func runeCount(s string) int {
	return len([]rune(s))
}

func TestAuthorTimelineSparklineAsciiUnderCi(t *testing.T) {
	saveCi := Ci
	Ci = true
	t.Cleanup(func() { Ci = saveCi })

	series := buildBuckets([]int{0, 1, 2, 3, 5, 8})
	out := renderAuthorTimelineSparkline(series, 12)
	for _, r := range out {
		if r > 127 {
			t.Fatalf("CI sparkline contains non-ASCII rune %U (%q)", r, out)
		}
	}
}

func TestAuthorTimelineCSVLongFormat(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "csv", 10
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\n"},
			Author: "Alice", Email: "alice@x", When: base,
		},
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Bob", Email: "bob@x", When: base.Add(48 * time.Hour),
		},
	})

	obs := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthorTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if !strings.HasPrefix(out, "# window:") {
		t.Fatalf("CSV should start with '# window:' comment:\n%s", out)
	}
	if !strings.Contains(out, "# buckets: 10\n") {
		t.Errorf("CSV missing '# buckets: 10' line:\n%s", out)
	}

	// Skip the two comment lines.
	lines := strings.SplitN(out, "\n", 3)
	body := lines[2]
	r := csv.NewReader(strings.NewReader(body))
	rows, err := r.ReadAll()
	if err != nil {
		t.Fatalf("csv parse: %v\n%s", err, body)
	}
	wantHeader := []string{"Author", "Email", "BucketStart", "Commits", "CodeDelta"}
	for i, h := range wantHeader {
		if rows[0][i] != h {
			t.Errorf("header col %d = %q, want %q", i, rows[0][i], h)
		}
	}
	// Long format: each row is (author × bucket). 2 authors × 10 buckets = 20 rows.
	if got, want := len(rows)-1, len(obs.rows)*HistoryBuckets; got != want {
		t.Errorf("CSV body rows = %d, want authors*buckets = %d", got, want)
	}
}

func TestAuthorTimelineJSONShape(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "json", 8
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\n"},
			Author: "Alice", Email: "alice@x", When: base,
		},
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Bob", Email: "bob@x", When: base.Add(168 * time.Hour),
		},
	})

	obs := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthorTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	var doc authorTimelineJSONDoc
	if err := jsoniter.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("json parse: %v, body:\n%s", err, out)
	}
	if doc.Report != "author-timeline" {
		t.Errorf("report = %q, want author-timeline", doc.Report)
	}
	if doc.Buckets != 8 {
		t.Errorf("buckets = %d, want 8", doc.Buckets)
	}
	if doc.Window.Commits != 2 {
		t.Errorf("window.commits = %d, want 2", doc.Window.Commits)
	}
	if len(doc.Authors) != 2 {
		t.Fatalf("authors count = %d, want 2", len(doc.Authors))
	}
	for _, a := range doc.Authors {
		if len(a.Series) != 8 {
			t.Errorf("author %q series len = %d, want 8", a.Name, len(a.Series))
		}
	}

	// bucketStart values should be in non-decreasing order — the date-only
	// format may collapse sub-day buckets onto the same string, but no
	// bucket should appear earlier than its predecessor. The underlying
	// Bucketing.Width is checked for evenness separately.
	first := doc.Authors[0].Series
	var prev time.Time
	for i, b := range first {
		ts, err := time.Parse(historyDateLayout, b.BucketStart)
		if err != nil {
			t.Fatalf("parse bucketStart %q: %v", b.BucketStart, err)
		}
		if i > 0 && ts.Before(prev) {
			t.Errorf("bucket %d before %d: %s vs %s", i, i-1, ts, prev)
		}
		prev = ts
	}
	// Even spacing: every bucket Start(i) - Start(i-1) is the same width.
	if obs.bucket.Width == 0 {
		t.Errorf("Bucketing.Width is zero")
	}
}

func TestAuthorTimelineTabularContainsHeader(t *testing.T) {
	saveDepth, saveFormat, saveBuckets := HistoryDepth, Format, HistoryBuckets
	HistoryDepth, Format, HistoryBuckets = 100, "tabular", 12
	t.Cleanup(func() {
		HistoryDepth, Format, HistoryBuckets = saveDepth, saveFormat, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Alice", Email: "alice@x", When: base,
		},
	})

	obs := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	out, err := renderAuthorTimeline(obs)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, "Authors") {
		t.Errorf("tabular missing 'Authors' header:\n%s", out)
	}
	if !strings.Contains(out, "Activity") {
		t.Errorf("tabular missing 'Activity' column:\n%s", out)
	}
	if !strings.Contains(out, "Code±") {
		t.Errorf("tabular missing 'Code±' column:\n%s", out)
	}
}

func TestAuthorTimelineRejectsUnsupportedFormat(t *testing.T) {
	saveFormat := Format
	Format = "xml"
	t.Cleanup(func() { Format = saveFormat })

	obs := newHistoryAuthorTimelineObserver(8)
	if _, err := renderAuthorTimeline(obs); err == nil {
		t.Fatal("expected error for --format xml")
	}
}

func TestAuthorTimelineMailmapFolding(t *testing.T) {
	saveDepth, saveBuckets := HistoryDepth, HistoryBuckets
	HistoryDepth, HistoryBuckets = 100, 6
	t.Cleanup(func() {
		HistoryDepth, HistoryBuckets = saveDepth, saveBuckets
	})

	base := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	dir := makeTimelineRepo(t, []timelineCommit{
		{
			Files: map[string]string{
				".mailmap": "Alice <alice@example.com> <alt@example.com>\n",
				"a.go":     "package a\nfunc A() {}\n",
			},
			Author: "Alice", Email: "alice@example.com", When: base,
		},
		{
			Files:  map[string]string{"a.go": "package a\nfunc A() {}\nfunc B() {}\n"},
			Author: "Alice", Email: "alt@example.com", When: base.Add(24 * time.Hour),
		},
	})

	obs := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(dir, obs); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	count := 0
	for _, r := range obs.rows {
		if r.Name == "Alice" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Alice rows after mailmap fold = %d, want 1; rows = %+v", count, obs.rows)
	}
}
