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

// languagesTimelineSparkCells is the fixed width of the Trend sparkline cell
// in the 79-column tabular report. The fixed-resolution per-bucket series is
// downsampled to this many cells.
const languagesTimelineSparkCells = 26

// languagesTimelineWideSparkCells is the sparkline width for --wide (109 cols).
const languagesTimelineWideSparkCells = 56

// languagesTimelineTopN caps tabular rows. CSV/JSON are uncapped.
const languagesTimelineTopN = 12

// languagesTimelineRow is the materialised per-language result.
type languagesTimelineRow struct {
	Language      string
	StartingLines int64
	CodeNow       int64
	Change        int64
	SharePercent  float64
	// Deltas is the per-bucket net code delta for this language.
	Deltas []int64
	// Trajectory is the absolute code count at the end of each bucket, i.e.
	// StartingLines + cumulative sum of Deltas. Used for the sparkline.
	Trajectory []int64
}

// languagesTimelineEvent records one observed file change so the observer
// can bin it under the real window's Bucketing in Finalise (the engine
// doesn't expose the window until after the walk).
type languagesTimelineEvent struct {
	Language  string
	When      time.Time
	CodeDelta int64
}

// historyLanguagesObserver collects per-commit per-language code deltas and
// the baseline snapshot, then materialises per-language trajectories in
// Finalise. Implements BaselineObserver so the engine seeds it with the
// pre-window tree before the walk.
type historyLanguagesObserver struct {
	starting    map[string]int64
	events      []languagesTimelineEvent
	bucketCount int

	bucket Bucketing
	window HistoryWindow
	rows   []languagesTimelineRow
}

func newHistoryLanguagesObserver(buckets int) *historyLanguagesObserver {
	if buckets <= 0 {
		buckets = 60
	}
	return &historyLanguagesObserver{
		starting:    map[string]int64{},
		bucketCount: buckets,
	}
}

// Seed sums baseline code lines per language so each language gets an
// absolute starting line count.
func (o *historyLanguagesObserver) Seed(baseline BaselineSnapshot) {
	for _, bf := range baseline.Files {
		var code int64
		for _, lt := range bf.LineTypes {
			if lt == LINE_CODE {
				code++
			}
		}
		if code == 0 {
			continue
		}
		o.starting[bf.Language] += code
	}
}

func (o *historyLanguagesObserver) Observe(c CommitInfo, changes []FileChange) {
	for _, fc := range changes {
		added := splitAddedCodeLines(fc.AddedRanges, fc.LineTypes)
		removed := countRangeLines(fc.RemovedRanges)
		delta := int64(added) - int64(removed)
		if delta == 0 {
			continue
		}
		o.events = append(o.events, languagesTimelineEvent{
			Language:  fc.Language,
			When:      c.When,
			CodeDelta: delta,
		})
	}
}

func (o *historyLanguagesObserver) Finalise(window HistoryWindow, head HeadSnapshot) {
	o.window = window
	o.bucket = NewBucketing(window.From, window.To, o.bucketCount)

	deltas := map[string][]int64{}
	for _, ev := range o.events {
		s := deltas[ev.Language]
		if s == nil {
			s = make([]int64, o.bucket.N)
			deltas[ev.Language] = s
		}
		idx := o.bucket.Index(ev.When)
		s[idx] += ev.CodeDelta
	}

	// Union of languages — those with a starting count and those touched in
	// the window.
	langSet := map[string]struct{}{}
	for lang := range o.starting {
		langSet[lang] = struct{}{}
	}
	for lang := range deltas {
		langSet[lang] = struct{}{}
	}

	rows := make([]languagesTimelineRow, 0, len(langSet))
	for lang := range langSet {
		start := o.starting[lang]
		series := deltas[lang]
		if series == nil {
			series = make([]int64, o.bucket.N)
		}
		traj := make([]int64, o.bucket.N)
		running := start
		for i, d := range series {
			running += d
			if running < 0 {
				running = 0
			}
			traj[i] = running
		}
		codeNow := start
		if len(traj) > 0 {
			codeNow = traj[len(traj)-1]
		}
		rows = append(rows, languagesTimelineRow{
			Language:      lang,
			StartingLines: start,
			CodeNow:       codeNow,
			Change:        codeNow - start,
			Deltas:        series,
			Trajectory:    traj,
		})
	}

	var grand int64
	for _, r := range rows {
		grand += r.CodeNow
	}
	if grand > 0 {
		for i := range rows {
			rows[i].SharePercent = float64(rows[i].CodeNow) / float64(grand) * 100.0
		}
	}

	slices.SortFunc(rows, func(a, b languagesTimelineRow) int {
		if a.CodeNow != b.CodeNow {
			if a.CodeNow < b.CodeNow {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Language, b.Language)
	})
	o.rows = rows
}

// runLanguagesTimelineReport is the dispatch entry point called from
// Process() when --timeline is set without --by-author. Opens the repo,
// walks the window with the configured bucket count, and writes the chosen
// format.
func runLanguagesTimelineReport(repoPath string) error {
	observer := newHistoryLanguagesObserver(HistoryBuckets)
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	out, err := renderLanguagesTimeline(observer)
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

func renderLanguagesTimeline(o *historyLanguagesObserver) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderLanguagesTimelineTabular(o), nil
	case "csv":
		return renderLanguagesTimelineCSV(o)
	case "json":
		return renderLanguagesTimelineJSON(o)
	default:
		return "", fmt.Errorf("unsupported --format %q for --timeline (supported: tabular, csv, json)", Format)
	}
}

// Tabular column format. 20+1+26+1+11+1+8+1+10 = 79.
var tabularShortLanguagesTimelineFormatHead = "%-20s %-26s %11s %8s %10s\n"

// Wide tabular: same columns, wider sparkline. 20+1+56+1+11+1+8+1+10 = 109.
var tabularWideLanguagesTimelineFormatHead = "%-20s %-56s %11s %8s %10s\n"

func renderLanguagesTimelineTabular(o *historyLanguagesObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Languages", o.window, wide))

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	format := tabularShortLanguagesTimelineFormatHead
	cells := languagesTimelineSparkCells
	if wide {
		format = tabularWideLanguagesTimelineFormatHead
		cells = languagesTimelineWideSparkCells
	}

	_, _ = fmt.Fprintf(&sb, format, "Language", "Trend", "Code", "Share", "Change")
	sb.WriteString(brk)

	limit := len(o.rows)
	if limit > languagesTimelineTopN {
		limit = languagesTimelineTopN
	}

	for i := 0; i < limit; i++ {
		r := o.rows[i]
		langCol := unicodeAwareTrim(r.Language, 19)
		langCol = unicodeAwareRightPad(langCol, 20)
		spark := renderLanguagesTrajectorySparkline(r.Trajectory, cells)
		codeStr := formatWithCommas(p, r.CodeNow)
		shareStr := fmt.Sprintf("%6.1f%%", r.SharePercent)
		changeStr := formatCodeDelta(p, r.Change)
		_, _ = fmt.Fprintf(&sb, format, langCol, spark, codeStr, shareStr, changeStr)
	}

	sb.WriteString(brk)
	return sb.String()
}

// renderLanguagesTrajectorySparkline downsamples the absolute trajectory to
// a sparkline. Each line is normalised to its own min/max for shape clarity
// (the Share column carries cross-language comparison).
func renderLanguagesTrajectorySparkline(traj []int64, cells int) string {
	if len(traj) == 0 {
		if asciiOutput() {
			return strings.Repeat(".", cells)
		}
		return strings.Repeat("▁", cells)
	}
	values := make([]float64, len(traj))
	for i, v := range traj {
		values[i] = float64(v)
	}
	return renderSparkline(values, cells)
}

func renderLanguagesTimelineCSV(o *historyLanguagesObserver) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("# buckets: %d\n", o.bucket.N))

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{
		"Language", "BucketStart", "Code", "CodeNow", "SharePercent", "Change",
	})

	for _, r := range o.rows {
		for i, code := range r.Trajectory {
			bucketStart := o.bucket.Start(i).UTC().Format(historyDateLayout)
			_ = w.Write([]string{
				r.Language,
				bucketStart,
				fmt.Sprintf("%d", code),
				fmt.Sprintf("%d", r.CodeNow),
				fmt.Sprintf("%.1f", r.SharePercent),
				fmt.Sprintf("%d", r.Change),
			})
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type languagesTimelineJSONBucket struct {
	BucketStart string `json:"bucketStart"`
	Code        int64  `json:"code"`
}

type languagesTimelineJSONLang struct {
	Language     string                        `json:"language"`
	CodeNow      int64                         `json:"codeNow"`
	SharePercent float64                       `json:"sharePercent"`
	Change       int64                         `json:"change"`
	Series       []languagesTimelineJSONBucket `json:"series"`
}

type languagesTimelineJSONWindow struct {
	Depth   int    `json:"depth"`
	Commits int    `json:"commits"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type languagesTimelineJSONDoc struct {
	Report    string                      `json:"report"`
	Window    languagesTimelineJSONWindow `json:"window"`
	Buckets   int                         `json:"buckets"`
	Languages []languagesTimelineJSONLang `json:"languages"`
}

func renderLanguagesTimelineJSON(o *historyLanguagesObserver) (string, error) {
	doc := languagesTimelineJSONDoc{
		Report: "languages-timeline",
		Window: languagesTimelineJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		Buckets:   o.bucket.N,
		Languages: make([]languagesTimelineJSONLang, 0, len(o.rows)),
	}
	for _, r := range o.rows {
		jl := languagesTimelineJSONLang{
			Language:     r.Language,
			CodeNow:      r.CodeNow,
			SharePercent: round1(r.SharePercent),
			Change:       r.Change,
			Series:       make([]languagesTimelineJSONBucket, 0, len(r.Trajectory)),
		}
		for i, code := range r.Trajectory {
			jl.Series = append(jl.Series, languagesTimelineJSONBucket{
				BucketStart: o.bucket.Start(i).UTC().Format(historyDateLayout),
				Code:        code,
			})
		}
		doc.Languages = append(doc.Languages, jl)
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
