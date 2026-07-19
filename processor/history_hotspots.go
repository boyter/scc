// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"fmt"
	"os"
	"slices"
	"strings"

	jsoniter "github.com/json-iterator/go"
	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
)

// HotspotsJSONReport walks the git history at repoPath and returns the
// hotspots report as a JSON string. It is the programmatic entry point used by
// the MCP server, which needs the rendered data rather than the stdout/file
// side effects of runHotspotsReport. A limit > 0 caps the number of files in
// the output (highest-scoring first); limit <= 0 returns every scored file.
// HistoryDepth and the mailmap folding behave exactly as on the CLI path.
func HotspotsJSONReport(repoPath string, limit int) (string, error) {
	observer := newHotspotsObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return "", err
	}
	return renderHotspotsJSONLimited(observer, limit)
}

// runHotspotsReport is the dispatch entry point called from Process() when
// --hotspots is set. Opens the repo at repoPath, walks history, and writes
// the chosen format to stdout or FileOutput.
func runHotspotsReport(repoPath string) error {
	observer := newHotspotsObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	out, err := renderHotspots(observer)
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

// hotspotsRecord is the per-file accumulator and the final row.
type hotspotsRecord struct {
	File         string
	Language     string
	Complexity   int64
	Commits      int
	LinesChanged int64
	Authors      map[authorID]struct{}
	CodeChurn    int64
	CommentChurn int64
	Score        float64
}

// hotspotsObserver accumulates per-file commit / churn / author stats during
// the walk, then materialises the table at Finalise using the HEAD snapshot
// for current language and complexity. Implements MailmapObserver so the
// Authrs column folds identities the same way the author rollup does,
// without paying for the full baseline tree classification.
type hotspotsObserver struct {
	files    map[string]*hotspotsRecord
	registry *authorRegistry
	window   HistoryWindow
	snapshot HeadSnapshot
	records  []hotspotsRecord
	totalRaw int // total files seen across the window (for the "X of Y" footer)
}

func newHotspotsObserver() *hotspotsObserver {
	return &hotspotsObserver{
		files:    map[string]*hotspotsRecord{},
		registry: newAuthorRegistry(nil),
	}
}

// SetMailmap satisfies MailmapObserver — rebuilds the registry with the
// repo's .mailmap so Authrs folds identities the same way the author rollup
// does.
func (o *hotspotsObserver) SetMailmap(mm *mailmap) {
	o.registry = newAuthorRegistry(mm)
}

func (o *hotspotsObserver) Observe(c CommitInfo, changes []FileChange) {
	aid := o.registry.intern(c.Author, c.Email)
	for _, fc := range changes {
		// Rename: migrate the old path's accumulator so churn history is
		// continuous across the rename.
		if fc.FromPath != "" && fc.FromPath != fc.Path {
			if old, ok := o.files[fc.FromPath]; ok {
				old.File = fc.Path
				o.files[fc.Path] = old
				delete(o.files, fc.FromPath)
			}
		}
		rec := o.files[fc.Path]
		if rec == nil {
			rec = &hotspotsRecord{
				File:    fc.Path,
				Authors: map[authorID]struct{}{},
			}
			o.files[fc.Path] = rec
		}
		rec.Commits++
		added := countRangeLines(fc.AddedRanges)
		removed := countRangeLines(fc.RemovedRanges)
		rec.LinesChanged += int64(added + removed)
		rec.Authors[aid] = struct{}{}

		code, comment := splitChurnByType(fc.AddedRanges, fc.LineTypes)
		rec.CodeChurn += int64(code)
		rec.CommentChurn += int64(comment)
	}
}

func (o *hotspotsObserver) Finalise(window HistoryWindow, head HeadSnapshot) {
	o.window = window
	o.snapshot = head
	o.totalRaw = 0

	for path, rec := range o.files {
		hf, alive := head.Files[path]
		if !alive {
			continue
		}
		rec.Language = hf.Language
		rec.Complexity = hf.Complexity
		o.totalRaw++

		score := float64(rec.Complexity) * float64(rec.Commits)
		rec.Score = score
	}

	records := make([]hotspotsRecord, 0, o.totalRaw)
	for path, rec := range o.files {
		if _, alive := head.Files[path]; !alive {
			continue
		}
		records = append(records, *rec)
	}

	// Normalise 0–100 across the surviving set.
	maxScore := 0.0
	for _, r := range records {
		if r.Score > maxScore {
			maxScore = r.Score
		}
	}
	for i := range records {
		if maxScore > 0 {
			records[i].Score = records[i].Score / maxScore * 100.0
		}
	}

	slices.SortFunc(records, func(a, b hotspotsRecord) int {
		if a.Score == b.Score {
			return strings.Compare(a.File, b.File)
		}
		if a.Score < b.Score {
			return 1
		}
		return -1
	})
	o.records = records
}

// countRangeLines sums the line counts across a slice of line ranges.
func countRangeLines(ranges []LineRange) int {
	total := 0
	for _, r := range ranges {
		total += r.Count
	}
	return total
}

// splitChurnByType classifies *added* lines only into code vs comment buckets
// using the per-line LineType vector for the new blob. Removed lines aren't
// classified — the old blob isn't fetched on the churn path. Reported as
// +Code% in the tabular output. Blank lines don't count toward either bucket.
func splitChurnByType(added []LineRange, lineTypes []LineType) (code, comment int) {
	for _, r := range added {
		for i := 0; i < r.Count; i++ {
			ln := r.Start - 1 + i // 0-based index into lineTypes
			if ln < 0 || ln >= len(lineTypes) {
				continue
			}
			switch lineTypes[ln] {
			case LINE_CODE:
				code++
			case LINE_COMMENT:
				comment++
			}
		}
	}
	return
}

// renderHotspots returns the formatted output for the chosen --format.
func renderHotspots(o *hotspotsObserver) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderHotspotsTabular(o), nil
	case "csv":
		return renderHotspotsCSV(o)
	case "json":
		return renderHotspotsJSON(o)
	default:
		return "", fmt.Errorf("unsupported --format %q for --hotspots (supported: tabular, csv, json)", Format)
	}
}

// Tabular column formats.
//
//	%-27s %8s %7s %8s %8s %7s %8s
//	27 + 1 + 8 + 1 + 7 + 1 + 8 + 1 + 8 + 1 + 7 + 1 + 8 = 79
var tabularShortHotspotsFormatHead = "%-27s %8s %7s %8s %8s %7s %8s\n"
var tabularShortHotspotsFormatBody = "%-27s %8s %7d %8d %8s %7d %8.1f\n"

// Wide variant — 109 columns, adds a 16-char hotspot bar and a +Code% column
// (%-share of *added* lines that were code; removed lines aren't classified).
// The bar carries the wide layout's extra width so it reads as a real chart;
// the File column stays modest rather than padding dead space to the rule.
//
//	%-32s %8s %7s %8s %8s %7s %8s %7s %-16s
//	32 + 1 + 8 + 1 + 7 + 1 + 8 + 1 + 8 + 1 + 7 + 1 + 8 + 1 + 7 + 1 + 16 = 109
var tabularWideHotspotsFormatHead = "%-32s %8s %7s %8s %8s %7s %8s %7s %-16s\n"
var tabularWideHotspotsFormatBody = "%-32s %8s %7d %8d %8s %7d %8.1f %6.1f%% %-16s\n"

func renderHotspotsTabular(o *hotspotsObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Hotspots", o.window, wide))

	printer := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))
	if wide {
		_, _ = fmt.Fprintf(&sb, tabularWideHotspotsFormatHead,
			"File", "Lang", "Cmplx", "Commits", "Lines±", "Authrs", "Hotspot", "+Code%", "Bar")
	} else {
		_, _ = fmt.Fprintf(&sb, tabularShortHotspotsFormatHead,
			"File", "Lang", "Cmplx", "Commits", "Lines±", "Authrs", "Hotspot")
	}
	sb.WriteString(brk)

	shown := 0
	for _, r := range o.records {
		// Score 0 means no complexity signal — not a hotspot. The CSV and JSON
		// renderers skip these too, so the tabular view matches rather than
		// trailing the list with zero-score noise.
		if r.Score <= 0 {
			continue
		}
		shown++
		fileTrim, fileWidth := 26, 27
		if wide {
			fileTrim, fileWidth = 31, 32
		}
		fileCol := unicodeAwareTrim(r.File, fileTrim)
		fileCol = unicodeAwareRightPad(fileCol, fileWidth)
		langCol := trimLanguageShort(r.Language, 8)
		linesCol := formatWithCommas(printer, r.LinesChanged)
		if wide {
			codeShare := 0.0
			totalChurn := r.CodeChurn + r.CommentChurn
			if totalChurn > 0 {
				codeShare = float64(r.CodeChurn) / float64(totalChurn) * 100.0
			}
			bar := renderBar(r.Score/100.0, 16)
			_, _ = fmt.Fprintf(&sb, tabularWideHotspotsFormatBody,
				fileCol, langCol, r.Complexity, r.Commits, linesCol,
				len(r.Authors), r.Score, codeShare, bar)
		} else {
			_, _ = fmt.Fprintf(&sb, tabularShortHotspotsFormatBody,
				fileCol, langCol, r.Complexity, r.Commits, linesCol,
				len(r.Authors), r.Score)
		}
	}

	sb.WriteString(brk)
	if shown > 0 {
		footer := fmt.Sprintf("complexity × change-frequency, normalised · %d files", shown)
		sb.WriteString(footer)
		sb.WriteByte('\n')
		sb.WriteString(brk)
	}
	return sb.String()
}

func formatWithCommas(p *gmessage.Printer, n int64) string {
	return p.Sprintf("%d", n)
}

func trimLanguageShort(lang string, size int) string {
	if len(lang) <= size {
		return lang
	}
	// keep most informative bit
	return lang[:size-1] + "…"
}

func renderHotspotsCSV(o *hotspotsObserver) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{
		"File", "Language", "Complexity", "Commits",
		"LinesChanged", "Authors", "CodeChurn", "CommentChurn", "Score",
	})

	for _, r := range o.records {
		if r.Score <= 0 {
			continue
		}
		_ = w.Write([]string{
			r.File,
			r.Language,
			fmt.Sprintf("%d", r.Complexity),
			fmt.Sprintf("%d", r.Commits),
			fmt.Sprintf("%d", r.LinesChanged),
			fmt.Sprintf("%d", len(r.Authors)),
			fmt.Sprintf("%d", r.CodeChurn),
			fmt.Sprintf("%d", r.CommentChurn),
			fmt.Sprintf("%.1f", r.Score),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type hotspotsJSONFile struct {
	File         string  `json:"file"`
	Language     string  `json:"language"`
	Complexity   int64   `json:"complexity"`
	Commits      int     `json:"commits"`
	LinesChanged int64   `json:"linesChanged"`
	Authors      int     `json:"authors"`
	CodeChurn    int64   `json:"codeChurn"`
	CommentChurn int64   `json:"commentChurn"`
	Score        float64 `json:"score"`
}

type hotspotsJSONWindow struct {
	Depth   int    `json:"depth"`
	Commits int    `json:"commits"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type hotspotsJSONDoc struct {
	Report string             `json:"report"`
	Window hotspotsJSONWindow `json:"window"`
	Files  []hotspotsJSONFile `json:"files"`
}

func renderHotspotsJSON(o *hotspotsObserver) (string, error) {
	return renderHotspotsJSONLimited(o, 0)
}

// renderHotspotsJSONLimited renders the JSON document, capping the file list at
// limit rows (highest-scoring first, matching the sort applied in Finalise). A
// limit <= 0 includes every scored file, preserving the uncapped CLI behaviour.
func renderHotspotsJSONLimited(o *hotspotsObserver, limit int) (string, error) {
	doc := hotspotsJSONDoc{
		Report: "hotspots",
		Window: hotspotsJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		Files: make([]hotspotsJSONFile, 0, len(o.records)),
	}
	for _, r := range o.records {
		if r.Score <= 0 {
			continue
		}
		if limit > 0 && len(doc.Files) >= limit {
			break
		}
		doc.Files = append(doc.Files, hotspotsJSONFile{
			File:         r.File,
			Language:     r.Language,
			Complexity:   r.Complexity,
			Commits:      r.Commits,
			LinesChanged: r.LinesChanged,
			Authors:      len(r.Authors),
			CodeChurn:    r.CodeChurn,
			CommentChurn: r.CommentChurn,
			Score:        round1(r.Score),
		})
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func round1(f float64) float64 {
	return float64(int64(f*10+0.5)) / 10
}
