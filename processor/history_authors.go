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
	"github.com/mattn/go-runewidth"
	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
)

// authorsTopN is the cap on tabular rows for the author rollup. CSV/JSON
// output is not capped. The remainder collapses into a single "others (N)"
// row in the tabular table.
const authorsTopN = 15

// authorNameColWidth / authorNameTrim are the trim/pad widths for the
// "Author" column in the 79-col tabular report. Both wide and short use the
// same name column.
const (
	authorNameTrim     = 30
	authorNameColWidth = 31
)

// authorRow is one materialised row in the report. Sentinel is true for the
// "(before window)" pseudo-author whose lines pre-date the walk window.
type authorRow struct {
	Name        string
	Email       string
	Code        int64
	Comment     int64
	Complexity  int64
	Files       int
	OwnsPercent float64
	LastCommit  time.Time
	Sentinel    bool
}

// historyAuthorsObserver accumulates per-file forward-replay blame during
// the walk, then collapses it into per-author totals on Finalise. It
// implements both CommitObserver and BaselineObserver, so the engine seeds
// it with the pre-window tree state (and the .mailmap) before the walk.
type historyAuthorsObserver struct {
	blame      map[string][]authorID
	lineTypes  map[string][]LineType
	complexity map[string][]int

	registry *authorRegistry
	lastSeen map[authorID]time.Time

	window   HistoryWindow
	snapshot HeadSnapshot

	rows       []authorRow
	busFactor  int
	busAuthors []string
	busCovered float64
}

func newHistoryAuthorsObserver() *historyAuthorsObserver {
	return &historyAuthorsObserver{
		blame:      map[string][]authorID{},
		lineTypes:  map[string][]LineType{},
		complexity: map[string][]int{},
		lastSeen:   map[authorID]time.Time{},
		registry:   newAuthorRegistry(nil),
	}
}

// Seed installs the mailmap and seeds the per-file blame maps from the
// baseline snapshot — every pre-window line maps to sentinelAuthorID so
// surviving untouched lines are correctly attributed to "(before window)".
func (o *historyAuthorsObserver) Seed(baseline BaselineSnapshot) {
	o.registry = newAuthorRegistry(baseline.Mailmap)
	for path, bf := range baseline.Files {
		n := len(bf.LineTypes)
		if n == 0 {
			continue
		}
		o.blame[path] = make([]authorID, n) // zero value = sentinelAuthorID
		o.lineTypes[path] = bf.LineTypes
		o.complexity[path] = bf.Complexity
	}
}

func (o *historyAuthorsObserver) Observe(c CommitInfo, changes []FileChange) {
	aid := o.registry.intern(c.Author, c.Email)
	if prev, ok := o.lastSeen[aid]; !ok || c.When.After(prev) {
		o.lastSeen[aid] = c.When
	}
	for _, fc := range changes {
		prev := o.blame[fc.Path]
		newN := len(fc.LineTypes)
		o.blame[fc.Path] = applyDiffToBlame(prev, newN, fc.AddedRanges, fc.RemovedRanges, aid)
		o.lineTypes[fc.Path] = fc.LineTypes
		o.complexity[fc.Path] = fc.Complexity
	}
}

func (o *historyAuthorsObserver) Finalise(window HistoryWindow, head HeadSnapshot) {
	o.window = window
	o.snapshot = head

	type acc struct {
		Code       int64
		Comment    int64
		Complexity int64
		Files      int
	}
	totals := map[authorID]*acc{}
	var grandCode int64

	for path, blame := range o.blame {
		if _, alive := head.Files[path]; !alive {
			continue
		}
		types := o.lineTypes[path]
		perFile := map[authorID]int64{}

		for i := 0; i < len(blame) && i < len(types); i++ {
			aid := blame[i]
			a := totals[aid]
			if a == nil {
				a = &acc{}
				totals[aid] = a
			}
			switch types[i] {
			case LINE_CODE:
				a.Code++
				perFile[aid]++
				grandCode++
			case LINE_COMMENT:
				a.Comment++
			}
		}
		for _, lineNo := range o.complexity[path] {
			idx := lineNo - 1
			if idx < 0 || idx >= len(blame) {
				continue
			}
			aid := blame[idx]
			a := totals[aid]
			if a == nil {
				a = &acc{}
				totals[aid] = a
			}
			a.Complexity++
		}

		// Plurality of code lines: who has the most code in this file.
		// Tie-break on smaller authorID for determinism (sentinel wins
		// only when no real author has code).
		var plur authorID
		var plurCount int64
		for aid, c := range perFile {
			if c > plurCount || (c == plurCount && aid < plur) {
				plur = aid
				plurCount = c
			}
		}
		if plurCount > 0 {
			totals[plur].Files++
		}
	}

	rows := make([]authorRow, 0, len(totals))
	for aid, a := range totals {
		rec := o.registry.record(aid)
		row := authorRow{
			Name:       rec.Name,
			Email:      rec.Email,
			Code:       a.Code,
			Comment:    a.Comment,
			Complexity: a.Complexity,
			Files:      a.Files,
		}
		if grandCode > 0 {
			row.OwnsPercent = float64(a.Code) / float64(grandCode) * 100.0
		}
		if aid == sentinelAuthorID {
			row.Sentinel = true
		} else if when, ok := o.lastSeen[aid]; ok {
			row.LastCommit = when
		}
		rows = append(rows, row)
	}

	// Sentinel sorted to the end; real authors by Code desc, then Name.
	slices.SortFunc(rows, func(a, b authorRow) int {
		if a.Sentinel != b.Sentinel {
			if a.Sentinel {
				return 1
			}
			return -1
		}
		if a.Code != b.Code {
			if a.Code < b.Code {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Name, b.Name)
	})
	o.rows = rows

	cumPercent := 0.0
	for _, r := range rows {
		if r.Sentinel {
			continue
		}
		if r.Code == 0 {
			break
		}
		cumPercent += r.OwnsPercent
		o.busAuthors = append(o.busAuthors, r.Name)
		if cumPercent > 50 {
			break
		}
	}
	o.busFactor = len(o.busAuthors)
	o.busCovered = cumPercent
}

// runAuthorsReport is the dispatch entry point called from Process() when
// --by-author is set (and --timeline is not). Opens the repo at repoPath,
// walks history with baseline seeding, and writes the chosen format to
// stdout or FileOutput.
func runAuthorsReport(repoPath string) error {
	observer := newHistoryAuthorsObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	out, err := renderAuthors(observer)
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

func renderAuthors(o *historyAuthorsObserver) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderAuthorsTabular(o), nil
	case "csv":
		return renderAuthorsCSV(o)
	case "json":
		return renderAuthorsJSON(o)
	default:
		return "", fmt.Errorf("unsupported --format %q for --by-author (supported: tabular, csv, json)", Format)
	}
}

// Short tabular: %-31s %9s %9s %7s %8s %10s = 79.
var tabularShortAuthorsFormatHead = "%-31s %9s %9s %7s %8s %10s\n"

// Wide tabular: inserts the Comment column. %-31s %9s %9s %9s %7s %8s %10s = 88.
var tabularWideAuthorsFormatHead = "%-31s %9s %9s %9s %7s %8s %10s\n"

func renderAuthorsTabular(o *historyAuthorsObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Authors", o.window, wide))

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	if wide {
		_, _ = fmt.Fprintf(&sb, tabularWideAuthorsFormatHead,
			"Author", "Code", "Comment", "Cmplx", "Files", "Owns", "Last seen")
	} else {
		_, _ = fmt.Fprintf(&sb, tabularShortAuthorsFormatHead,
			"Author", "Code", "Cmplx", "Files", "Owns", "Last seen")
	}
	sb.WriteString(brk)

	realRows := make([]authorRow, 0, len(o.rows))
	var sentinel *authorRow
	for i := range o.rows {
		r := o.rows[i]
		if r.Sentinel {
			s := r
			sentinel = &s
		} else {
			realRows = append(realRows, r)
		}
	}

	limit := len(realRows)
	if limit > authorsTopN {
		limit = authorsTopN
	}

	for i := 0; i < limit; i++ {
		r := realRows[i]
		writeAuthorRow(&sb, p, wide, r.Name, r.Code, r.Comment, r.Complexity,
			fmt.Sprintf("%d", r.Files), r.OwnsPercent, lastSeenString(r))
	}

	if limit < len(realRows) {
		var (
			count      int
			code       int64
			comment    int64
			complexity int64
			owns       float64
		)
		for i := limit; i < len(realRows); i++ {
			r := realRows[i]
			count++
			code += r.Code
			comment += r.Comment
			complexity += r.Complexity
			owns += r.OwnsPercent
		}
		writeAuthorRow(&sb, p, wide,
			fmt.Sprintf("others (%d)", count), code, comment, complexity,
			"—", owns, "—")
	}

	if sentinel != nil && (sentinel.Code+sentinel.Comment+sentinel.Complexity) > 0 {
		writeAuthorRow(&sb, p, wide,
			"(before window)", sentinel.Code, sentinel.Comment, sentinel.Complexity,
			fmt.Sprintf("%d", sentinel.Files), sentinel.OwnsPercent, "—")
	}

	sb.WriteString(brk)

	footerWidth := runewidth.StringWidth(strings.TrimRight(brk, "\n"))
	footer := formatAuthorsFooter(o, footerWidth)
	sb.WriteString(footer)
	sb.WriteByte('\n')
	sb.WriteString(brk)

	return sb.String()
}

func lastSeenString(r authorRow) string {
	if r.LastCommit.IsZero() {
		return "—"
	}
	return r.LastCommit.UTC().Format(historyDateLayout)
}

func writeAuthorRow(sb *strings.Builder, p *gmessage.Printer, wide bool,
	name string, code, comment, complexity int64,
	files string, owns float64, lastSeen string) {

	nameCol := unicodeAwareTrim(name, authorNameTrim)
	nameCol = unicodeAwareRightPad(nameCol, authorNameColWidth)
	codeStr := formatWithCommas(p, code)
	cmplxStr := formatWithCommas(p, complexity)
	ownsStr := fmt.Sprintf("%6.1f%%", owns)

	if wide {
		commentStr := formatWithCommas(p, comment)
		_, _ = fmt.Fprintf(sb, tabularWideAuthorsFormatHead,
			nameCol, codeStr, commentStr, cmplxStr, files, ownsStr, lastSeen)
	} else {
		_, _ = fmt.Fprintf(sb, tabularShortAuthorsFormatHead,
			nameCol, codeStr, cmplxStr, files, ownsStr, lastSeen)
	}
}

func formatAuthorsFooter(o *historyAuthorsObserver, width int) string {
	if o.busFactor == 0 {
		return "Bus factor 0 · no authored code in window"
	}
	covered := o.busCovered
	if covered > 100 {
		covered = 100
	}

	prefix := fmt.Sprintf("Bus factor %d · ", o.busFactor)
	var suffix string
	if o.busFactor == 1 {
		suffix = fmt.Sprintf(" last-touched %.0f%% of code (single point of failure)", covered)
	} else {
		suffix = fmt.Sprintf(" last-touched %.0f%% of code", covered)
	}

	single := prefix + strings.Join(o.busAuthors, " + ") + suffix
	if width <= 0 || runewidth.StringWidth(single) <= width {
		return single
	}
	return wrapBusFactorFooter(prefix, o.busAuthors, suffix, width)
}

// wrapBusFactorFooter word-wraps the bus-factor line on " + " token
// boundaries when it would otherwise exceed width. Continuation lines are
// indented to align under the first name so the structure stays readable.
// Width is measured in display columns (runewidth), not bytes, so non-ASCII
// author names and CI-mode ASCII breaks both produce the right wrap point.
func wrapBusFactorFooter(prefix string, names []string, suffix string, width int) string {
	indent := strings.Repeat(" ", runewidth.StringWidth(prefix))
	var sb strings.Builder
	line := prefix
	lineWidth := runewidth.StringWidth(line)

	for i, name := range names {
		token := name
		if i < len(names)-1 {
			token += " + "
		}
		tokenWidth := runewidth.StringWidth(token)
		if line != prefix && line != indent && lineWidth+tokenWidth > width {
			sb.WriteString(strings.TrimRight(line, " "))
			sb.WriteByte('\n')
			line = indent
			lineWidth = runewidth.StringWidth(indent)
		}
		line += token
		lineWidth += tokenWidth
	}

	suffixWidth := runewidth.StringWidth(suffix)
	if lineWidth+suffixWidth > width && line != indent {
		sb.WriteString(strings.TrimRight(line, " "))
		sb.WriteByte('\n')
		line = indent + strings.TrimLeft(suffix, " ")
	} else {
		line += suffix
	}
	sb.WriteString(line)
	return sb.String()
}

func renderAuthorsCSV(o *historyAuthorsObserver) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{
		"Author", "Email", "Code", "Complexity", "Comment", "Files",
		"OwnsPercent", "LastCommit", "BeforeWindow",
	})
	for _, r := range o.rows {
		name, email := r.Name, r.Email
		lastCommit := ""
		beforeWindow := "false"
		if r.Sentinel {
			name, email = "", ""
			beforeWindow = "true"
		} else if !r.LastCommit.IsZero() {
			lastCommit = r.LastCommit.UTC().Format(historyDateLayout)
		}
		_ = w.Write([]string{
			name,
			email,
			fmt.Sprintf("%d", r.Code),
			fmt.Sprintf("%d", r.Complexity),
			fmt.Sprintf("%d", r.Comment),
			fmt.Sprintf("%d", r.Files),
			fmt.Sprintf("%.1f", r.OwnsPercent),
			lastCommit,
			beforeWindow,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type authorsJSONAuthor struct {
	Name         *string `json:"name"`
	Email        *string `json:"email"`
	Code         int64   `json:"code"`
	Complexity   int64   `json:"complexity"`
	Comment      int64   `json:"comment"`
	Files        int     `json:"files"`
	OwnsPercent  float64 `json:"ownsPercent"`
	LastCommit   string  `json:"lastCommit,omitempty"`
	BeforeWindow bool    `json:"beforeWindow"`
}

type authorsJSONWindow struct {
	Depth   int    `json:"depth"`
	Commits int    `json:"commits"`
	From    string `json:"from"`
	To      string `json:"to"`
}

type authorsJSONDoc struct {
	Report    string              `json:"report"`
	Window    authorsJSONWindow   `json:"window"`
	BusFactor int                 `json:"busFactor"`
	Authors   []authorsJSONAuthor `json:"authors"`
}

func renderAuthorsJSON(o *historyAuthorsObserver) (string, error) {
	doc := authorsJSONDoc{
		Report: "authors",
		Window: authorsJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		BusFactor: o.busFactor,
		Authors:   make([]authorsJSONAuthor, 0, len(o.rows)),
	}
	for _, r := range o.rows {
		a := authorsJSONAuthor{
			Code:        r.Code,
			Complexity:  r.Complexity,
			Comment:     r.Comment,
			Files:       r.Files,
			OwnsPercent: round1(r.OwnsPercent),
		}
		if r.Sentinel {
			a.BeforeWindow = true
		} else {
			name, email := r.Name, r.Email
			a.Name = &name
			a.Email = &email
			if !r.LastCommit.IsZero() {
				a.LastCommit = r.LastCommit.UTC().Format(historyDateLayout)
			}
		}
		doc.Authors = append(doc.Authors, a)
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
