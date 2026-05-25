// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
)

// authorTimelineSparkCells is the fixed width of the Activity sparkline cell
// in the 79-column tabular report. The fixed-resolution per-bucket series is
// downsampled to this many cells.
const authorTimelineSparkCells = 24

// authorTimelineTopN caps tabular rows. CSV/JSON are uncapped.
const authorTimelineTopN = 15

// authorTimelineBucket is one bucket's worth of accumulator state for one
// author.
type authorTimelineBucket struct {
	Commits   int
	CodeDelta int64
}

// authorTimelineRow is the materialised per-author result.
type authorTimelineRow struct {
	Name         string
	Email        string
	TotalCommits int
	CodeDelta    int64
	Series       []authorTimelineBucket
}

// authorTimelineEvent records one observed commit's contribution so the
// observer can bin it under the real window's Bucketing in Finalise (the
// engine doesn't expose the window until after the walk).
type authorTimelineEvent struct {
	Author    authorID
	When      time.Time
	CodeDelta int64
}

// historyAuthorTimelineObserver collects per-commit events during the walk
// and bins them into per-(author, bucket) totals on Finalise. It implements
// BaselineObserver only to pick up the mailmap; per-line baseline state is
// not needed since the report tracks deltas, not last-toucher attribution.
type historyAuthorTimelineObserver struct {
	registry    *authorRegistry
	events      []authorTimelineEvent
	bucketCount int

	bucket Bucketing
	window HistoryWindow
	rows   []authorTimelineRow
}

func newHistoryAuthorTimelineObserver(buckets int) *historyAuthorTimelineObserver {
	if buckets <= 0 {
		buckets = 60
	}
	return &historyAuthorTimelineObserver{
		registry:    newAuthorRegistry(nil),
		bucketCount: buckets,
	}
}

func (o *historyAuthorTimelineObserver) Seed(baseline BaselineSnapshot) {
	o.registry = newAuthorRegistry(baseline.Mailmap)
}

func (o *historyAuthorTimelineObserver) Observe(c CommitInfo, changes []FileChange) {
	aid := o.registry.intern(c.Author, c.Email)

	var delta int64
	for _, fc := range changes {
		added := splitAddedCodeLines(fc.AddedRanges, fc.LineTypes)
		removed := countRangeLines(fc.RemovedRanges)
		delta += int64(added) - int64(removed)
	}
	o.events = append(o.events, authorTimelineEvent{
		Author:    aid,
		When:      c.When,
		CodeDelta: delta,
	})
}

func (o *historyAuthorTimelineObserver) Finalise(window HistoryWindow, head HeadSnapshot) {
	o.window = window
	o.bucket = NewBucketing(window.From, window.To, o.bucketCount)

	series := map[authorID][]authorTimelineBucket{}
	for _, ev := range o.events {
		s := series[ev.Author]
		if s == nil {
			s = make([]authorTimelineBucket, o.bucket.N)
			series[ev.Author] = s
		}
		idx := o.bucket.Index(ev.When)
		s[idx].Commits++
		s[idx].CodeDelta += ev.CodeDelta
	}

	rows := make([]authorTimelineRow, 0, len(series))
	for aid, s := range series {
		if aid == sentinelAuthorID {
			continue
		}
		rec := o.registry.record(aid)
		row := authorTimelineRow{
			Name:   rec.Name,
			Email:  rec.Email,
			Series: s,
		}
		for _, b := range s {
			row.TotalCommits += b.Commits
			row.CodeDelta += b.CodeDelta
		}
		rows = append(rows, row)
	}

	slices.SortFunc(rows, func(a, b authorTimelineRow) int {
		if a.TotalCommits != b.TotalCommits {
			if a.TotalCommits < b.TotalCommits {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Name, b.Name)
	})
	o.rows = rows
}

// splitAddedCodeLines returns the count of added lines classified as code by
// the new blob's LineTypes vector. Mirrors splitChurnByType but only returns
// the code component.
func splitAddedCodeLines(added []LineRange, lineTypes []LineType) int {
	code := 0
	for _, r := range added {
		for i := 0; i < r.Count; i++ {
			ln := r.Start - 1 + i
			if ln < 0 || ln >= len(lineTypes) {
				continue
			}
			if lineTypes[ln] == LINE_CODE {
				code++
			}
		}
	}
	return code
}

// runAuthorTimelineReport is the dispatch entry point called from Process()
// when --by-author --timeline is set. Opens the repo, walks the window with
// the configured bucket count, and writes the chosen format.
func runAuthorTimelineReport(repoPath string) error {
	observer := newHistoryAuthorTimelineObserver(HistoryBuckets)
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	out, err := renderAuthorTimeline(observer)
	if err != nil {
		return err
	}
	if FileOutput == "" {
		fmt.Print(out)
	} else {
		if err := os.WriteFile(FileOutput, []byte(out), 0644); err != nil {
			return err
		}
		fmt.Println("results written to " + FileOutput)
	}
	return nil
}

func renderAuthorTimeline(o *historyAuthorTimelineObserver) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderAuthorTimelineTabular(o), nil
	case "csv":
		return renderAuthorTimelineCSV(o)
	case "json":
		return renderAuthorTimelineJSON(o)
	default:
		return "", fmt.Errorf("unsupported --format %q for --by-author --timeline (supported: tabular, csv, json)", Format)
	}
}

// Tabular column format. 24+1+24+1+8+1+9+1+10 = 79.
var tabularShortAuthorTimelineFormatHead = "%-24s %-24s %8s %9s %-10s\n"

func renderAuthorTimelineTabular(o *historyAuthorTimelineObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Authors", o.window, wide))

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	_, _ = fmt.Fprintf(&sb, tabularShortAuthorTimelineFormatHead,
		"Author", "Activity", "Commits", "Code±", "")
	sb.WriteString(brk)

	limit := len(o.rows)
	if limit > authorTimelineTopN {
		limit = authorTimelineTopN
	}

	for i := 0; i < limit; i++ {
		r := o.rows[i]
		nameCol := unicodeAwareTrim(r.Name, 23)
		nameCol = unicodeAwareRightPad(nameCol, 24)
		spark := renderAuthorTimelineSparkline(r.Series, authorTimelineSparkCells)
		tag := authorTimelineTag(r.Series, o.bucket.Width)
		commitsStr := formatWithCommas(p, int64(r.TotalCommits))
		codeStr := formatCodeDelta(p, r.CodeDelta)
		_, _ = fmt.Fprintf(&sb, tabularShortAuthorTimelineFormatHead,
			nameCol, spark, commitsStr, codeStr, tag)
	}

	sb.WriteString(brk)
	return sb.String()
}

// renderAuthorTimelineSparkline projects per-bucket commit counts to a
// sparkline using the shared helper from history_render.go.
func renderAuthorTimelineSparkline(series []authorTimelineBucket, cells int) string {
	if len(series) == 0 {
		if asciiOutput() {
			return strings.Repeat(".", cells)
		}
		return strings.Repeat("▁", cells)
	}
	values := make([]float64, len(series))
	for i, b := range series {
		values[i] = float64(b.Commits)
	}
	return renderSparkline(values, cells)
}

// authorTimelineTag derives the trailing presentation tag. Returns:
//   - "↑"        — final bucket is >= 80% of the row's peak (still active).
//   - "quiet Nmo" — trailing zero buckets cover >= 1 month of wall clock.
//   - ""         — no notable trend.
//
// Tags are tabular-only; CSV/JSON do not carry them.
func authorTimelineTag(series []authorTimelineBucket, width time.Duration) string {
	if len(series) == 0 {
		return ""
	}
	maxCommits := 0
	for _, b := range series {
		if b.Commits > maxCommits {
			maxCommits = b.Commits
		}
	}
	if maxCommits == 0 {
		return ""
	}

	last := series[len(series)-1].Commits
	if last > 0 && float64(last) >= 0.8*float64(maxCommits) {
		return "↑"
	}

	zeroTail := 0
	for i := len(series) - 1; i >= 0; i-- {
		if series[i].Commits != 0 {
			break
		}
		zeroTail++
	}
	if zeroTail == 0 || width <= 0 {
		return ""
	}
	totalQuiet := time.Duration(zeroTail) * width
	const month = 30 * 24 * time.Hour
	if totalQuiet < month {
		return ""
	}
	months := int((totalQuiet + month/2) / month)
	if months < 1 {
		months = 1
	}
	return fmt.Sprintf("quiet %dmo", months)
}

// formatCodeDelta renders a signed code delta with a leading sign and
// thousands separators, e.g. "+38,000" or "-21".
func formatCodeDelta(p *gmessage.Printer, delta int64) string {
	if delta >= 0 {
		return "+" + formatWithCommas(p, delta)
	}
	return "-" + formatWithCommas(p, -delta)
}

func renderAuthorTimelineCSV(o *historyAuthorTimelineObserver) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("# buckets: %d\n", o.bucket.N))

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"Author", "Email", "BucketStart", "Commits", "CodeDelta"})

	for _, r := range o.rows {
		for i, b := range r.Series {
			bucketStart := o.bucket.Start(i).UTC().Format(historyDateLayout)
			_ = w.Write([]string{
				r.Name,
				r.Email,
				bucketStart,
				fmt.Sprintf("%d", b.Commits),
				fmt.Sprintf("%d", b.CodeDelta),
			})
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type authorTimelineJSONBucket struct {
	BucketStart string `json:"bucketStart"`
	Commits     int    `json:"commits"`
	CodeDelta   int64  `json:"codeDelta"`
}

type authorTimelineJSONAuthor struct {
	Name         string                     `json:"name"`
	Email        string                     `json:"email"`
	TotalCommits int                        `json:"totalCommits"`
	CodeDelta    int64                      `json:"codeDelta"`
	Series       []authorTimelineJSONBucket `json:"series"`
}

type authorTimelineJSONWindow struct {
	Depth   int    `json:"depth"`
	Commits int    `json:"commits"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type authorTimelineJSONDoc struct {
	Report  string                     `json:"report"`
	Window  authorTimelineJSONWindow   `json:"window"`
	Buckets int                        `json:"buckets"`
	Authors []authorTimelineJSONAuthor `json:"authors"`
}

func renderAuthorTimelineJSON(o *historyAuthorTimelineObserver) (string, error) {
	doc := authorTimelineJSONDoc{
		Report: "author-timeline",
		Window: authorTimelineJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		Buckets: o.bucket.N,
		Authors: make([]authorTimelineJSONAuthor, 0, len(o.rows)),
	}
	for _, r := range o.rows {
		ja := authorTimelineJSONAuthor{
			Name:         r.Name,
			Email:        r.Email,
			TotalCommits: r.TotalCommits,
			CodeDelta:    r.CodeDelta,
			Series:       make([]authorTimelineJSONBucket, 0, len(r.Series)),
		}
		for i, b := range r.Series {
			ja.Series = append(ja.Series, authorTimelineJSONBucket{
				BucketStart: o.bucket.Start(i).UTC().Format(historyDateLayout),
				Commits:     b.Commits,
				CodeDelta:   b.CodeDelta,
			})
		}
		doc.Authors = append(doc.Authors, ja)
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
