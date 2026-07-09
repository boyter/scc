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
	Name string
	// Display is the name as rendered in the tabular report and bus-factor
	// footer. It equals Name unless two or more in-window identities share
	// the same Name, in which case each colliding row is suffixed with a
	// distinguishing marker (see disambiguateNames) so the reader can tell
	// the identities apart. CSV/JSON emit Name and Email raw and ignore this.
	Display         string
	Email           string
	Code            int64
	Comment         int64
	Complexity      int64
	Files           int
	OwnsPercent     float64
	InWindowPercent float64
	LastCommit      time.Time
	Sentinel        bool
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

	rows         []authorRow
	busFactor    int
	busAuthors   []string
	busCovered   float64
	inWindowCode int64
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
		// Rename: carry the old path's per-line blame forward as the prior
		// state, then drop the stale key so it is not double-counted. A
		// pure rename has no Added/Removed ranges, so applyDiffToBlame just
		// copies the carried-forward blame — every line keeps its original
		// author. A rename with edits attributes only the edited lines to
		// the renaming commit.
		if fc.FromPath != "" && fc.FromPath != fc.Path {
			if oldBlame, ok := o.blame[fc.FromPath]; ok {
				prev = oldBlame
				delete(o.blame, fc.FromPath)
				delete(o.lineTypes, fc.FromPath)
				delete(o.complexity, fc.FromPath)
			}
		}
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

		// Plurality of code lines: who has the most code in this file. A
		// real author always outranks the sentinel — the sentinel only owns
		// the file when no real author has any code here. Tie-break on
		// smaller authorID for determinism.
		var plur authorID
		var plurCount int64
		for aid, c := range perFile {
			if aid == sentinelAuthorID {
				continue
			}
			if c > plurCount || (c == plurCount && aid < plur) {
				plur = aid
				plurCount = c
			}
		}
		if plurCount == 0 {
			// No real author has code here; fall back to the sentinel.
			if c, ok := perFile[sentinelAuthorID]; ok {
				plur = sentinelAuthorID
				plurCount = c
			}
		}
		if plurCount > 0 {
			totals[plur].Files++
		}
	}

	var sentinelCode int64
	if s, ok := totals[sentinelAuthorID]; ok {
		sentinelCode = s.Code
	}
	inWindowCode := grandCode - sentinelCode
	o.inWindowCode = inWindowCode

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
		} else {
			if inWindowCode > 0 {
				row.InWindowPercent = float64(a.Code) / float64(inWindowCode) * 100.0
			}
			if when, ok := o.lastSeen[aid]; ok {
				row.LastCommit = when
			}
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
	disambiguateNames(rows)
	o.rows = rows

	cumPercent := 0.0
	for _, r := range rows {
		if r.Sentinel {
			continue
		}
		if r.Code == 0 {
			break
		}
		cumPercent += r.InWindowPercent
		o.busAuthors = append(o.busAuthors, r.Display)
		if cumPercent > 50 {
			break
		}
	}
	o.busFactor = len(o.busAuthors)
	o.busCovered = cumPercent
}

// disambiguateNames sets Display on every row. When two or more in-window
// identities share the same display Name (e.g. one contributor committing
// under both a work and a noreply email — kept as distinct identities because
// no .mailmap merges them), each colliding row is suffixed with a short
// marker so the reader — and the bus-factor footer, which reuses Display —
// can tell them apart. Non-colliding names, the "others" roll-up and the
// sentinel are left bare.
func disambiguateNames(rows []authorRow) {
	groups := map[string][]int{}
	for i := range rows {
		if rows[i].Sentinel {
			continue
		}
		groups[rows[i].Name] = append(groups[rows[i].Name], i)
	}
	for _, idx := range groups {
		if len(idx) < 2 {
			continue
		}
		markers := disambiguationMarkers(rows, idx)
		for k, i := range idx {
			rows[i].Display = rows[i].Name + " (" + markers[k] + ")"
		}
	}
	for i := range rows {
		if rows[i].Display == "" {
			rows[i].Display = rows[i].Name
		}
	}
}

// disambiguationMarkers returns one marker per row in idx (all sharing a
// display Name), picking the shortest candidate form that is distinct across
// the whole group: registrable domain first (the tidiest, e.g. "github.com"),
// then the full domain, then the full email. Because two identities with the
// same name and same email intern to one authorID, a collision group always
// has distinct emails, so the final form is guaranteed to separate them.
func disambiguationMarkers(rows []authorRow, idx []int) []string {
	forms := []func(string) string{
		registrableDomain,
		emailDomain,
		func(email string) string { return email },
	}
	for _, form := range forms {
		out := make([]string, len(idx))
		seen := map[string]struct{}{}
		distinct := true
		for k, i := range idx {
			m := form(rows[i].Email)
			if m == "" {
				distinct = false
				break
			}
			if _, dup := seen[m]; dup {
				distinct = false
				break
			}
			seen[m] = struct{}{}
			out[k] = m
		}
		if distinct {
			return out
		}
	}
	// Unreachable in practice (see doc comment); fall back to raw email.
	out := make([]string, len(idx))
	for k, i := range idx {
		out[k] = rows[i].Email
	}
	return out
}

// registrableDomain returns the last two labels of the email's domain
// (e.g. "users.noreply.github.com" -> "github.com"), a short human-readable
// marker for the common single-TLD case. It is a display heuristic, not a
// public-suffix-correct computation — disambiguationMarkers falls back to the
// full domain when this form fails to separate a collision group.
func registrableDomain(email string) string {
	d := emailDomain(email)
	if d == "" {
		return ""
	}
	parts := strings.Split(d, ".")
	if len(parts) <= 2 {
		return d
	}
	return strings.Join(parts[len(parts)-2:], ".")
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

	limit := min(len(realRows), authorsTopN)

	for i := range limit {
		r := realRows[i]
		writeAuthorRow(&sb, p, wide, r.Display, r.Code, r.Comment, r.Complexity,
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
	if o.inWindowCode == 0 {
		return "Bus factor 0 · no code touched in window"
	}
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
		suffix = fmt.Sprintf(" last-touched %.0f%% of in-window code (single point of failure)", covered)
	} else {
		suffix = fmt.Sprintf(" last-touched %.0f%% of in-window code", covered)
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
	Name            *string `json:"name"`
	Email           *string `json:"email"`
	Code            int64   `json:"code"`
	Complexity      int64   `json:"complexity"`
	Comment         int64   `json:"comment"`
	Files           int     `json:"files"`
	OwnsPercent     float64 `json:"ownsPercent"`
	InWindowPercent float64 `json:"inWindowPercent"`
	LastCommit      string  `json:"lastCommit,omitempty"`
	BeforeWindow    bool    `json:"beforeWindow"`
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
			Code:            r.Code,
			Complexity:      r.Complexity,
			Comment:         r.Comment,
			Files:           r.Files,
			OwnsPercent:     round1(r.OwnsPercent),
			InWindowPercent: round1(r.InWindowPercent),
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
