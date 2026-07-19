// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	jsoniter "github.com/json-iterator/go"
)

// couplingOverviewTopN caps the rows in the tabular all-pairs (--coupling)
// overview. This is deliberately capped where the other history reports are not:
// the all-pairs view is a repo-wide glance, and a flat dump of every pair is
// unreadable. The per-file --coupling-for view and the CSV/JSON output stay
// uncapped — the full pair set is one --format away.
const couplingOverviewTopN = 15

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

// CouplingPartner is one file that co-changes with a chosen target file.
//
// Couple answers "if I change the target, how likely am I to touch this too".
// On its own it is confounded by the partner's base rate: a file that changes
// in most commits scores a near-perfect Couple against ANY target, purely
// because it is always there. Such a hub shows HIGH Couple and LOW Reverse —
// e.g. a target touched 3 times, always alongside a file touched 158 times,
// gives Couple 100% / Reverse 1.9%.
//
// Degree is the base-rate-corrected view and is what rows are ranked by; the
// two directional numbers are kept as supporting detail.
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

// Degree is the symmetric coupling ratio Shared/(target+partner−Shared) as a
// 0–100 percentage — the same measure the pairwise --coupling report ranks by,
// so the two views agree on what "strongly coupled" means.
//
// Unlike Couple it is not fooled by a busy partner: a hub present in every one
// of the target's commits still scores low here, because its own large commit
// total sits in the denominator.
func (p CouplingPartner) Degree() float64 {
	union := p.TargetCommit + p.PartnerCommit - p.Shared
	if union <= 0 {
		return 0
	}
	return float64(p.Shared) / float64(union) * 100.0
}

// partnersFor returns every surviving file coupled to target, ranked by Degree
// descending. Returns nil when the target never changed in the window.
//
// Ranking is by Degree, not Couple: Couple alone puts every busy file at 100%
// (it was present for all of a rarely-touched target's commits by base rate
// alone), which buries the genuine peer couplings under hubs.
//
// target must be a current (HEAD) path: Finalise has already folded every
// pre-rename name into its final one, so the counts keyed here are complete,
// but an old path that no longer exists in HEAD will not match. Callers
// validate against HEAD via resolveCouplingTarget before the walk.
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
	// Degree first (base-rate corrected), then raw co-change volume, then path so
	// the order is deterministic.
	sort.Slice(out, func(i, j int) bool {
		di, dj := out[i].Degree(), out[j].Degree()
		if di != dj {
			return di > dj
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
//
// target accepts the same forms as --coupling-for and is validated against HEAD
// before the walk, so a caller passing a bad path gets an immediate error rather
// than paying for a full traversal first.
func CouplingForJSONReport(repoPath, target string, limit int) (string, error) {
	resolved, err := resolveCouplingTarget(repoPath, target)
	if err != nil {
		return "", err
	}
	observer := newCouplingObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return "", err
	}
	return renderCouplingForJSONLimited(observer, resolved, limit)
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

// couplingSuggestionLimit caps the "did you mean" candidates offered when a
// --coupling-for path misses.
const couplingSuggestionLimit = 5

// errStopTreeWalk halts a tree walk once enough suggestions are collected.
// object.Tree.Files().ForEach surfaces whatever the callback returns, so it is
// compared back at the call site and discarded.
var errStopTreeWalk = errors.New("stop tree walk")

// couplingTargetCandidates returns the repo-relative forms to try for a
// user-supplied --coupling-for path, most likely first. Git keys every path
// from the repository root with forward slashes and no "./" prefix, so the
// string a user naturally types ("./processor/x.go", an absolute path, or a
// path relative to a subdirectory they are standing in) rarely matches as-is.
func couplingTargetCandidates(repoRoot, target string) []string {
	var out []string
	seen := map[string]struct{}{}
	add := func(p string) {
		if p == "" {
			return
		}
		p = path.Clean(filepath.ToSlash(p))
		if p == "." || p == ".." || strings.HasPrefix(p, "../") {
			return // escapes the repository, or names no file
		}
		if _, dup := seen[p]; dup {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}

	if filepath.IsAbs(target) {
		// Absolute → relative to the repository root.
		if rel, err := filepath.Rel(repoRoot, target); err == nil {
			add(rel)
		}
		return out
	}

	// As typed, cleaned. Handles both "processor/x.go" and "./processor/x.go".
	add(filepath.ToSlash(target))

	// Relative to the working directory, for running scc from a subdirectory:
	// `cd processor && scc --coupling-for constants.go .` means processor/constants.go.
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(repoRoot, filepath.Join(cwd, target)); err == nil {
			add(rel)
		}
	}
	return out
}

// couplingTargetMiss builds the error for a --coupling-for path that matched
// nothing in HEAD, offering same-basename files as suggestions. Only ever runs
// on the failure path, so walking the tree here costs nothing in the happy case.
func couplingTargetMiss(tree *object.Tree, target string) error {
	base := path.Base(path.Clean(filepath.ToSlash(target)))
	var matches []string
	err := tree.Files().ForEach(func(f *object.File) error {
		if path.Base(f.Name) != base {
			return nil
		}
		matches = append(matches, f.Name)
		if len(matches) >= couplingSuggestionLimit {
			return errStopTreeWalk
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopTreeWalk) {
		return fmt.Errorf("--coupling-for %q is not in HEAD (deleted, ignored, or path typo)", target)
	}
	if len(matches) > 0 {
		return fmt.Errorf("--coupling-for %q is not in HEAD; did you mean:\n  %s",
			target, strings.Join(matches, "\n  "))
	}
	return fmt.Errorf("--coupling-for %q is not in HEAD (deleted, ignored, or path typo)", target)
}

// resolveCouplingTarget maps a user-supplied --coupling-for path onto the
// git-style path the history engine keys on, verifying it against HEAD *before*
// the caller pays for a full history walk. A typo previously cost a complete
// walk (seconds to minutes) before reporting the miss.
//
// A repository with no HEAD (freshly initialised, no commits) is not an error
// here: the path is normalised and the walk reports the empty window as usual.
func resolveCouplingTarget(repoPath, target string) (string, error) {
	fallback := path.Clean(filepath.ToSlash(target))

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return "", fmt.Errorf("open git repository: %w", err)
	}
	ref, err := repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			return fallback, nil // empty repo — let the walk report the empty window
		}
		return "", fmt.Errorf("read HEAD: %w", err)
	}
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("read HEAD commit: %w", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		return "", fmt.Errorf("read HEAD tree: %w", err)
	}

	repoRoot := repoPath
	if wt, err := repo.Worktree(); err == nil {
		repoRoot = wt.Filesystem.Root()
	}

	for _, candidate := range couplingTargetCandidates(repoRoot, target) {
		if _, err := tree.File(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", couplingTargetMiss(tree, target)
}

// runCouplingReport is the dispatch entry point called from Process() when
// --coupling is set. Walks history and writes the chosen format to stdout or
// FileOutput.
func runCouplingReport(repoPath string) error {
	// Resolve and validate the target before the walk — a bad path should fail
	// in milliseconds, not after a full history traversal.
	target := ""
	if CouplingFor != "" {
		resolved, err := resolveCouplingTarget(repoPath, CouplingFor)
		if err != nil {
			return err
		}
		target = resolved
	}

	observer := newCouplingObserver()
	if _, err := runHistory(repoPath, observer); err != nil {
		return err
	}
	var out string
	var err error
	if target != "" {
		out, err = renderCouplingFor(observer, target)
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

// %-27s %-26s %15s %8s
// 27 + 1 + 26 + 1 + 15 + 1 + 8 = 79
// Mirrors the --coupling-for view: plain "Shared Commits" / "Coupling" headers,
// no jargon and no explanatory footer. The per-file A/B commit counts drop from
// the table (they remain in the CSV / JSON output); dropping them also frees the
// width the two file paths need.
var tabularCouplingFormatHead = "%-27s %-26s %15s %8s\n"
var tabularCouplingFormatBody = "%-27s %-26s %15d %7.1f%%\n"

// Wide tabular: same columns, both file paths widened to fill the 109-col rule.
// 42+1+41+1+15+1+8 = 109.
var tabularWideCouplingFormatHead = "%-42s %-41s %15s %8s\n"
var tabularWideCouplingFormatBody = "%-42s %-41s %15d %7.1f%%\n"

func renderCouplingTabular(o *couplingObserver) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	sb.WriteString(historyHeader("Change Coupling", o.window, wide))

	headFmt, bodyFmt := tabularCouplingFormatHead, tabularCouplingFormatBody
	aTrim, aWidth, bTrim, bWidth := 26, 27, 25, 26
	if wide {
		headFmt, bodyFmt = tabularWideCouplingFormatHead, tabularWideCouplingFormatBody
		aTrim, aWidth, bTrim, bWidth = 41, 42, 40, 41
	}

	_, _ = fmt.Fprintf(&sb, headFmt,
		"File A", "File B", "Shared Commits", "Coupling")
	sb.WriteString(brk)

	// The all-pairs view is a repo-wide overview, not a per-file answer: a flat
	// dump of every pair is unreadable, so the tabular form shows only the
	// strongest couplings. The full set is always available via --format csv/json
	// (e.g. to build a coupling graph). --coupling-for is the per-file drill-down.
	limit := min(len(o.pairs), couplingOverviewTopN)
	for _, p := range o.pairs[:limit] {
		aCol := unicodeAwareRightPad(unicodeAwareTrim(p.A, aTrim), aWidth)
		bCol := unicodeAwareRightPad(unicodeAwareTrim(p.B, bTrim), bWidth)
		_, _ = fmt.Fprintf(&sb, bodyFmt,
			aCol, bCol, p.Shared, p.Degree())
	}

	sb.WriteString(brk)
	if limit > 0 {
		var footer string
		if len(o.pairs) > limit {
			footer = fmt.Sprintf("top %d of %d pairs · sharing ≥%d commits", limit, len(o.pairs), CouplingMinShared)
		} else {
			footer = fmt.Sprintf("%d pairs · sharing ≥%d commits", len(o.pairs), CouplingMinShared)
		}
		sb.WriteString(footer)
		sb.WriteByte('\n')
		sb.WriteString(brk)
	} else {
		footer := "no file pairs met the coupling threshold"
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

// %-51s %16s %10s
// 51 + 1 + 16 + 1 + 10 = 79, matching the tabular break rule. The middle column
// is widened to spell out "Shared Commits" rather than a bare "Shared".
// The human view keeps three columns: the related file, how many commits
// touched both, and the symmetric coupling score it is ranked by. The
// directional Couple / Reverse ratios stay in the CSV and JSON output for tools
// that want them — on screen they were the source of the base-rate confusion.
var tabularCouplingForFormatHead = "%-51s %16s %10s\n"
var tabularCouplingForFormatBody = "%-51s %16d %9.1f%%\n"

// Wide tabular: same columns, Related File widened to fill the 109-col rule.
// 81 + 1 + 16 + 1 + 10 = 109.
var tabularWideCouplingForFormatHead = "%-81s %16s %10s\n"
var tabularWideCouplingForFormatBody = "%-81s %16d %9.1f%%\n"

func renderCouplingForTabular(o *couplingObserver, target string) string {
	wide := More || strings.EqualFold(Format, "wide")
	brk := tabularBreakFor(wide)

	var sb strings.Builder
	// The columns speak for themselves, so no descriptive sentence sits between
	// the banner and the table. The target file name is carried by the thin-target
	// warning when it matters, and always by the CSV / JSON output.
	sb.WriteString(historyHeader("Change Coupling", o.window, wide))

	partners := o.partnersFor(target)

	if _, alive := o.head.Files[target]; !alive {
		sb.WriteString(fmt.Sprintf("%s is not in HEAD (deleted, ignored, or path typo)\n", target))
		sb.WriteString(brk)
		return sb.String()
	}

	// A low target commit count makes the ratios coarse, but the Shared Commits
	// column already shows that directly, so the tabular view carries no extra
	// warning sentence. historyHeader already ends with a break, so the table
	// head follows straight on.
	headFmt, bodyFmt := tabularCouplingForFormatHead, tabularCouplingForFormatBody
	nameTrim, nameWidth := 50, 51
	if wide {
		headFmt, bodyFmt = tabularWideCouplingForFormatHead, tabularWideCouplingForFormatBody
		nameTrim, nameWidth = 80, 81
	}

	_, _ = fmt.Fprintf(&sb, headFmt, "Related File", "Shared Commits", "Coupling")
	sb.WriteString(brk)

	for _, p := range partners {
		nameCol := unicodeAwareRightPad(unicodeAwareTrim(p.Path, nameTrim), nameWidth)
		_, _ = fmt.Fprintf(&sb, bodyFmt, nameCol, p.Shared, p.Degree())
	}

	sb.WriteString(brk)
	if len(partners) > 0 {
		_, _ = fmt.Fprintf(&sb, "%d coupled files · pairs sharing ≥%d commits\n",
			len(partners), CouplingMinShared)
	} else {
		sb.WriteString("no file shares enough commits with this target to couple\n")
	}
	sb.WriteString(brk)
	return sb.String()
}

func renderCouplingForCSV(o *couplingObserver, target string) (string, error) {
	var sb strings.Builder
	sb.WriteString(formatWindowComment(o.window))
	sb.WriteByte('\n')

	w := csv.NewWriter(&sb)
	_ = w.Write([]string{"Target", "Partner", "Shared", "TargetCommits", "PartnerCommits", "Degree", "Couple", "Reverse"})
	for _, p := range o.partnersFor(target) {
		_ = w.Write([]string{
			target,
			p.Path,
			fmt.Sprintf("%d", p.Shared),
			fmt.Sprintf("%d", p.TargetCommit),
			fmt.Sprintf("%d", p.PartnerCommit),
			fmt.Sprintf("%.1f", p.Degree()),
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
	Degree         float64 `json:"degree"`
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
			Degree:         round1(p.Degree()),
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
