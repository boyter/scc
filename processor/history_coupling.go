// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

// CouplingTopN is the cap on rows shown in the tabular coupling report.
// CSV / JSON output is not capped.
const CouplingTopN = 20

// CouplingMinShared is the floor on co-change count for a pair to appear in
// any output. A pair that changed together only once is almost always a
// coincidence, not coupling, so the noise is dropped at the source. Raw counts
// below this are still accumulated; they're just not reported.
const CouplingMinShared = 2

// CouplingMaxFilesPerCommit is the default size cap: commits touching more than
// this many files are excluded from PAIR counting (each file still counts
// toward its own commit total). A commit touching hundreds of files is a sweep
// — initial import, vendored dump, gofmt, a license-header change — that
// carries no logical-coupling signal yet costs O(k²) pairs. 0 disables the cap.
const CouplingMaxFilesPerCommit = 30

// CouplingCount is the raw, unopinionated co-change record for one unordered
// file pair: how often each file changed across the window, and how often they
// changed in the same commit. scc emits these integers; any ratio a consumer
// wants — symmetric degree, or the directional P(B changes | A changed) that
// answers "blast radius" — is a division the consumer chooses, not scc.
type CouplingCount struct {
	A        string // lexicographically smaller surviving path
	B        string // lexicographically larger surviving path
	Shared   int    // commits in which BOTH changed
	CommitsA int    // commits in which A changed (window total)
	CommitsB int    // commits in which B changed (window total)
}

// Degree is the symmetric coupling ratio shared/(a+b−shared) as a 0–100
// percentage — the standard temporal-coupling "degree". It is a convenience
// for the human-facing table only; the raw counts sit beside it so the number
// is never a black box. Returns 0 when the union is empty.
func (c CouplingCount) Degree() float64 {
	union := c.CommitsA + c.CommitsB - c.Shared
	if union <= 0 {
		return 0
	}
	return float64(c.Shared) / float64(union) * 100.0
}

// couplingObserver accumulates temporal (change) coupling from the commit
// stream: which files keep changing together in the same commit. It implements
// only CommitObserver — coupling needs neither the start-tree baseline nor the
// mailmap, so it is the cheapest observer the history engine carries.
type couplingObserver struct {
	maxFilesPerCommit int

	fileCommits map[string]int    // path -> commits touching it
	pairShared  map[pairKey]int   // unordered pair -> co-change count
	alias       map[string]string // renamed-from path -> renamed-to path

	// Resolved-and-filtered state, materialised at Finalise. fc and ps have had
	// renames folded; head is the survivor set. Both the pair-list view and the
	// file-oriented "blast radius" query read from these.
	fc   map[string]int
	ps   map[pairKey]int
	head HeadSnapshot

	window     HistoryWindow
	pairs      []CouplingCount // materialised at Finalise, strongest first
	totalPairs int             // pairs meeting the floor (for the footer)
	skipped    int             // commits dropped from pair counting by the cap
}

type pairKey struct{ a, b string }

func newCouplingObserver() *couplingObserver {
	return &couplingObserver{
		maxFilesPerCommit: CouplingMaxFilesPerCommit,
		fileCommits:       map[string]int{},
		pairShared:        map[pairKey]int{},
		alias:             map[string]string{},
	}
}

func (o *couplingObserver) Observe(_ CommitInfo, changes []FileChange) {
	// Record renames so paths counted under an old name before the rename can be
	// folded into the new name at Finalise. Cheap to note here, resolved once at
	// the end rather than migrated eagerly per commit.
	for _, fc := range changes {
		if fc.FromPath != "" && fc.FromPath != fc.Path {
			o.alias[fc.FromPath] = fc.Path
		}
	}

	// Distinct paths only — a rename can surface the same logical file twice.
	// FileChange already excludes deletes, binaries, submodules, ignored and
	// unclassifiable paths, so every Path here is a real counted source file.
	paths := make([]string, 0, len(changes))
	seen := make(map[string]struct{}, len(changes))
	for _, fc := range changes {
		if _, dup := seen[fc.Path]; dup {
			continue
		}
		seen[fc.Path] = struct{}{}
		paths = append(paths, fc.Path)
		o.fileCommits[fc.Path]++ // every file's own total — the ratio denominator
	}

	if len(paths) < 2 {
		return // nothing can couple in a single-file commit
	}
	if o.maxFilesPerCommit > 0 && len(paths) > o.maxFilesPerCommit {
		o.skipped++
		return // totals already counted; skip the O(k²) pair explosion
	}

	sort.Strings(paths) // canonical order so the pair key is stable: a < b
	for i := 0; i < len(paths); i++ {
		for j := i + 1; j < len(paths); j++ {
			o.pairShared[pairKey{paths[i], paths[j]}]++
		}
	}
}

func (o *couplingObserver) Finalise(window HistoryWindow, head HeadSnapshot) {
	o.window = window

	// Fold rename history: collapse every path to its final name, so a file that
	// lived under an old path before a rename shares one set of counts with its
	// current name. Done once here — O(total) — rather than migrated per commit.
	fileCommits := make(map[string]int, len(o.fileCommits))
	for path, n := range o.fileCommits {
		fileCommits[o.resolve(path)] += n
	}
	pairShared := make(map[pairKey]int, len(o.pairShared))
	for k, shared := range o.pairShared {
		a, b := o.resolve(k.a), o.resolve(k.b)
		if a == b {
			continue // both sides renamed to the same file — no longer a pair
		}
		if a > b {
			a, b = b, a
		}
		pairShared[pairKey{a, b}] += shared
	}

	o.fc = fileCommits
	o.ps = pairShared
	o.head = head

	pairs := make([]CouplingCount, 0, len(pairShared))
	for k, shared := range pairShared {
		if shared < CouplingMinShared {
			continue
		}
		// Keep only pairs whose BOTH files still exist in HEAD — same convention
		// as the rest of the engine, which never reports files that are gone.
		if _, ok := head.Files[k.a]; !ok {
			continue
		}
		if _, ok := head.Files[k.b]; !ok {
			continue
		}
		pairs = append(pairs, CouplingCount{
			A: k.a, B: k.b, Shared: shared,
			CommitsA: fileCommits[k.a],
			CommitsB: fileCommits[k.b],
		})
	}

	// Strongest absolute co-change first, then strongest degree, then path so the
	// order is deterministic. Volume first surfaces the heavyweight couplings;
	// the Degree column lets the reader tell genuine coupling from two busy files
	// that merely co-change by chance.
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Shared != pairs[j].Shared {
			return pairs[i].Shared > pairs[j].Shared
		}
		di, dj := pairs[i].Degree(), pairs[j].Degree()
		if di != dj {
			return di > dj
		}
		if pairs[i].A != pairs[j].A {
			return pairs[i].A < pairs[j].A
		}
		return pairs[i].B < pairs[j].B
	})

	o.pairs = pairs
	o.totalPairs = len(pairs)
}

// resolve follows the rename chain from path to its final name. Guards against
// a pathological cycle in the alias map by capping the walk.
func (o *couplingObserver) resolve(path string) string {
	for i := 0; i < 64; i++ {
		next, ok := o.alias[path]
		if !ok || next == path {
			return path
		}
		path = next
	}
	return path
}

// CouplingPartner is one file that co-changes with a chosen target file, with
// the two directional probabilities a "blast radius" answer needs. Couple is
// the one that answers "if I change the target, how likely am I to touch this
// too"; Reverse is the opposite direction, which exposes hubs (a header that
// everything touches has a low Couple from any given file but a high Reverse).
type CouplingPartner struct {
	Path          string
	Shared        int // commits changing BOTH target and this partner
	PartnerCommit int // partner's window commit total
	TargetCommit  int // target's window commit total
}

// Couple is P(partner changes | target changed) = Shared / TargetCommit, the
// directional blast-radius probability: edit the target, expect to edit this.
func (p CouplingPartner) Couple() float64 {
	if p.TargetCommit <= 0 {
		return 0
	}
	return float64(p.Shared) / float64(p.TargetCommit) * 100.0
}

// Reverse is P(target changes | partner changed) = Shared / PartnerCommit. A
// large gap between Reverse and Couple marks an asymmetric (hub-style) link
// rather than a true peer coupling.
func (p CouplingPartner) Reverse() float64 {
	if p.PartnerCommit <= 0 {
		return 0
	}
	return float64(p.Shared) / float64(p.PartnerCommit) * 100.0
}

// partnersFor returns every surviving file coupled to target, ranked by Couple
// (the blast-radius direction) descending. target is matched after rename
// folding. Returns nil when the target never changed in the window.
func (o *couplingObserver) partnersFor(target string) []CouplingPartner {
	tc := o.fc[target]
	if tc == 0 {
		return nil
	}
	out := make([]CouplingPartner, 0)
	for k, shared := range o.ps {
		if shared < CouplingMinShared {
			continue
		}
		var partner string
		switch target {
		case k.a:
			partner = k.b
		case k.b:
			partner = k.a
		default:
			continue
		}
		if _, ok := o.head.Files[partner]; !ok {
			continue
		}
		out = append(out, CouplingPartner{
			Path:          partner,
			Shared:        shared,
			PartnerCommit: o.fc[partner],
			TargetCommit:  tc,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		ci, cj := out[i].Couple(), out[j].Couple()
		if ci != cj {
			return ci > cj
		}
		if out[i].Shared != out[j].Shared {
			return out[i].Shared > out[j].Shared
		}
		return out[i].Path < out[j].Path
	})
	return out
}

// CouplingForJSONReport walks history and returns the directional coupling for
// a single target file as JSON — the MCP entry point. limit > 0 caps the
// partner list (strongest Couple first); limit <= 0 returns every partner.
func CouplingForJSONReport(repoPath, target string, limit int) (string, error) {
	observer := newCouplingObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return "", err
	}
	return renderCouplingForJSONLimited(observer, target, limit)
}

// CouplingJSONReport walks the git history at repoPath and returns the coupling
// report as a JSON string — the programmatic entry point for the MCP server,
// which needs the rendered data rather than stdout side effects. A limit > 0
// caps the pair list (strongest first); limit <= 0 returns every pair.
func CouplingJSONReport(repoPath string, limit int) (string, error) {
	observer := newCouplingObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return "", err
	}
	return renderCouplingJSONLimited(observer, limit)
}

// runCouplingReport is the dispatch entry point called from Process() when
// --coupling is set. Walks history and writes the chosen format to stdout or
// FileOutput.
func runCouplingReport(repoPath string) error {
	observer := newCouplingObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	var out string
	var err error
	if CouplingFor != "" {
		out, err = renderCouplingFor(observer, CouplingFor)
	} else {
		out, err = renderCoupling(observer)
	}
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

func renderCoupling(o *couplingObserver) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderCouplingTabular(o), nil
	case "csv":
		return renderCouplingCSV(o)
	case "json":
		return renderCouplingJSON(o)
	default:
		return "", fmt.Errorf("unsupported --format %q for --coupling (supported: tabular, csv, json)", Format)
	}
}

// %-23s %-23s %8s %6s %6s %8s
// 23 + 1 + 23 + 1 + 8 + 1 + 6 + 1 + 6 + 1 + 8 = 79
var tabularCouplingFormatHead = "%-23s %-23s %8s %6s %6s %8s\n"
var tabularCouplingFormatBody = "%-23s %-23s %8d %6d %6d %7.1f%%\n"

func renderCouplingTabular(o *couplingObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Change Coupling", o.window, wide))

	_, _ = fmt.Fprintf(&sb, tabularCouplingFormatHead,
		"File A", "File B", "Both", "A", "B", "Degree")
	sb.WriteString(brk)

	limit := min(len(o.pairs), CouplingTopN)
	for i := range limit {
		p := o.pairs[i]
		aCol := unicodeAwareRightPad(unicodeAwareTrim(p.A, 22), 23)
		bCol := unicodeAwareRightPad(unicodeAwareTrim(p.B, 22), 23)
		_, _ = fmt.Fprintf(&sb, tabularCouplingFormatBody,
			aCol, bCol, p.Shared, p.CommitsA, p.CommitsB, p.Degree())
	}

	sb.WriteString(brk)
	if limit > 0 {
		footer := fmt.Sprintf("   Both = commits changing both · A/B = each file's commits · Degree = Both/(A+B−Both)")
		sb.WriteString(footer)
		sb.WriteByte('\n')
		footer2 := fmt.Sprintf("   pairs sharing ≥%d commits · %d of %d shown · commits touching >%d files excluded",
			CouplingMinShared, limit, o.totalPairs, o.maxFilesPerCommit)
		sb.WriteString(footer2)
		sb.WriteByte('\n')
		sb.WriteString(brk)
	} else {
		footer := "   no file pairs met the coupling threshold"
		sb.WriteString(footer)
		sb.WriteByte('\n')
		sb.WriteString(brk)
	}
	return sb.String()
}

func renderCouplingCSV(o *couplingObserver) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"FileA", "FileB", "Shared", "CommitsA", "CommitsB", "Degree"})
	for _, p := range o.pairs {
		_ = w.Write([]string{
			p.A,
			p.B,
			fmt.Sprintf("%d", p.Shared),
			fmt.Sprintf("%d", p.CommitsA),
			fmt.Sprintf("%d", p.CommitsB),
			fmt.Sprintf("%.1f", p.Degree()),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type couplingJSONPair struct {
	FileA    string  `json:"fileA"`
	FileB    string  `json:"fileB"`
	Shared   int     `json:"shared"`
	CommitsA int     `json:"commitsA"`
	CommitsB int     `json:"commitsB"`
	Degree   float64 `json:"degree"`
}

type couplingJSONDoc struct {
	Report string             `json:"report"`
	Window hotspotsJSONWindow `json:"window"`
	Pairs  []couplingJSONPair `json:"pairs"`
}

func renderCouplingJSON(o *couplingObserver) (string, error) {
	return renderCouplingJSONLimited(o, 0)
}

func renderCouplingJSONLimited(o *couplingObserver, limit int) (string, error) {
	doc := couplingJSONDoc{
		Report: "coupling",
		Window: hotspotsJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		Pairs: make([]couplingJSONPair, 0, len(o.pairs)),
	}
	for _, p := range o.pairs {
		if limit > 0 && len(doc.Pairs) >= limit {
			break
		}
		doc.Pairs = append(doc.Pairs, couplingJSONPair{
			FileA:    p.A,
			FileB:    p.B,
			Shared:   p.Shared,
			CommitsA: p.CommitsA,
			CommitsB: p.CommitsB,
			Degree:   round1(p.Degree()),
		})
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// --- file-oriented "blast radius" view ---------------------------------------

func renderCouplingFor(o *couplingObserver, target string) (string, error) {
	switch strings.ToLower(Format) {
	case "", "tabular", "wide":
		return renderCouplingForTabular(o, target), nil
	case "csv":
		return renderCouplingForCSV(o, target)
	case "json":
		return renderCouplingForJSON(o, target)
	default:
		return "", fmt.Errorf("unsupported --format %q for --coupling-for (supported: tabular, csv, json)", Format)
	}
}

// %-50s %7s %9s %9s
// 50 + 1 + 7 + 1 + 9 + 1 + 9 = 78
var tabularCouplingForFormatHead = "%-50s %7s %9s %9s\n"
var tabularCouplingForFormatBody = "%-50s %7d %8.1f%% %8.1f%%\n"

func renderCouplingForTabular(o *couplingObserver, target string) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	// Reuse the standard window header, then a second line naming the target.
	sb.WriteString(historyHeader("Change Coupling", o.window, wide))

	partners := o.partnersFor(target)
	tc := o.fc[target]

	if _, alive := o.head.Files[target]; !alive {
		sb.WriteString(fmt.Sprintf("   %s is not in HEAD (deleted, ignored, or path typo)\n", target))
		sb.WriteString(brk)
		return sb.String()
	}

	_, _ = fmt.Fprintf(&sb, "   if you change %s — what tends to change with it (%d commits in window)\n", target, tc)
	sb.WriteString(brk)
	_, _ = fmt.Fprintf(&sb, tabularCouplingForFormatHead, "Partner", "Both", "Couple", "Reverse")
	sb.WriteString(brk)

	limit := min(len(partners), CouplingTopN)
	for i := range limit {
		p := partners[i]
		nameCol := unicodeAwareRightPad(unicodeAwareTrim(p.Path, 49), 50)
		_, _ = fmt.Fprintf(&sb, tabularCouplingForFormatBody, nameCol, p.Shared, p.Couple(), p.Reverse())
	}

	sb.WriteString(brk)
	if limit > 0 {
		sb.WriteString("   Couple = P(partner changes | you changed the target) · Reverse = the opposite\n")
		_, _ = fmt.Fprintf(&sb, "   %d of %d coupled files shown · pairs sharing ≥%d commits\n",
			limit, len(partners), CouplingMinShared)
	} else {
		sb.WriteString("   no file shares enough commits with this target to couple\n")
	}
	sb.WriteString(brk)
	return sb.String()
}

func renderCouplingForCSV(o *couplingObserver, target string) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"Target", "Partner", "Shared", "TargetCommits", "PartnerCommits", "Couple", "Reverse"})
	for _, p := range o.partnersFor(target) {
		_ = w.Write([]string{
			target,
			p.Path,
			fmt.Sprintf("%d", p.Shared),
			fmt.Sprintf("%d", p.TargetCommit),
			fmt.Sprintf("%d", p.PartnerCommit),
			fmt.Sprintf("%.1f", p.Couple()),
			fmt.Sprintf("%.1f", p.Reverse()),
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return sb.String(), nil
}

type couplingForJSONPartner struct {
	File           string  `json:"file"`
	Shared         int     `json:"shared"`
	PartnerCommits int     `json:"partnerCommits"`
	Couple         float64 `json:"couple"`
	Reverse        float64 `json:"reverse"`
}

type couplingForJSONDoc struct {
	Report        string                   `json:"report"`
	Target        string                   `json:"target"`
	TargetCommits int                      `json:"targetCommits"`
	Window        hotspotsJSONWindow       `json:"window"`
	Partners      []couplingForJSONPartner `json:"partners"`
}

func renderCouplingForJSON(o *couplingObserver, target string) (string, error) {
	return renderCouplingForJSONLimited(o, target, 0)
}

func renderCouplingForJSONLimited(o *couplingObserver, target string, limit int) (string, error) {
	doc := couplingForJSONDoc{
		Report:        "coupling-for",
		Target:        target,
		TargetCommits: o.fc[target],
		Window: hotspotsJSONWindow{
			Depth:   o.window.Depth,
			Commits: o.window.Commits,
			From:    formatWindowDate(o.window.From),
			To:      formatWindowDate(o.window.To),
		},
		Partners: make([]couplingForJSONPartner, 0),
	}
	for _, p := range o.partnersFor(target) {
		if limit > 0 && len(doc.Partners) >= limit {
			break
		}
		doc.Partners = append(doc.Partners, couplingForJSONPartner{
			File:           p.Path,
			Shared:         p.Shared,
			PartnerCommits: p.PartnerCommit,
			Couple:         round1(p.Couple()),
			Reverse:        round1(p.Reverse()),
		})
	}
	b, err := jsoniter.Marshal(doc)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
