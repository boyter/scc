// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hasCandidate reports whether want is among the candidate forms.
func hasCandidate(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

// A "./"-prefixed path is what a shell tab-completes to, and it must resolve to
// the bare git path. This was the original --coupling-for bug: the prefix was
// passed through and never matched, after a full history walk.
func TestCouplingTargetCandidatesStripsDotSlash(t *testing.T) {
	got := couplingTargetCandidates("/repo", "./processor/constants.go")
	if !hasCandidate(got, "processor/constants.go") {
		t.Errorf("expected ./ prefix to be stripped, got %v", got)
	}
}

func TestCouplingTargetCandidatesBarePathUnchanged(t *testing.T) {
	got := couplingTargetCandidates("/repo", "processor/constants.go")
	if !hasCandidate(got, "processor/constants.go") {
		t.Errorf("expected bare path preserved, got %v", got)
	}
}

func TestCouplingTargetCandidatesAbsoluteBecomesRepoRelative(t *testing.T) {
	got := couplingTargetCandidates("/repo", filepath.FromSlash("/repo/processor/constants.go"))
	if !hasCandidate(got, "processor/constants.go") {
		t.Errorf("expected absolute path made repo-relative, got %v", got)
	}
}

// A path outside the repository can never be a git path, so it must yield no
// candidates rather than a "../" form that would be looked up and miss.
func TestCouplingTargetCandidatesRejectsEscapingPaths(t *testing.T) {
	if got := couplingTargetCandidates("/repo", filepath.FromSlash("/etc/passwd")); len(got) != 0 {
		t.Errorf("expected no candidates for a path outside the repo, got %v", got)
	}
	if got := couplingTargetCandidates("/repo", "../../../etc/passwd"); len(got) != 0 {
		t.Errorf("expected no candidates for an escaping relative path, got %v", got)
	}
}

// When the typed form and the cwd-relative form agree, only one candidate should
// survive — the lookup is against HEAD and duplicates just cost tree reads.
func TestCouplingTargetCandidatesDedupes(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot determine working directory")
	}
	got := couplingTargetCandidates(cwd, "./x/y.go")
	if len(got) != 1 || got[0] != "x/y.go" {
		t.Errorf("expected exactly one deduped candidate x/y.go, got %v", got)
	}
}

// Running scc from inside a subdirectory: `cd processor && scc --coupling-for
// constants.go` must offer processor/constants.go, since git keys from the root.
func TestCouplingTargetCandidatesResolvesFromSubdirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Skip("cannot determine working directory")
	}
	repoRoot := filepath.Dir(cwd) // tests run in ./processor, so this is the repo root
	got := couplingTargetCandidates(repoRoot, "constants.go")
	want := filepath.Base(cwd) + "/constants.go"
	if !hasCandidate(got, want) {
		t.Errorf("expected %q among candidates for a subdirectory-relative path, got %v", want, got)
	}
}

// headWith builds a HeadSnapshot whose Files map contains every named path, so
// the Finalise survivor filter keeps them.
func headWith(paths ...string) HeadSnapshot {
	h := HeadSnapshot{Files: map[string]HeadFile{}}
	for _, p := range paths {
		h.Files[p] = HeadFile{Path: p}
	}
	return h
}

// commit is a tiny helper turning a list of paths into the []FileChange the
// observer consumes (only Path matters for coupling).
func commit(paths ...string) []FileChange {
	out := make([]FileChange, 0, len(paths))
	for _, p := range paths {
		out = append(out, FileChange{Path: p})
	}
	return out
}

func findPair(pairs []CouplingCount, a, b string) (CouplingCount, bool) {
	if a > b {
		a, b = b, a
	}
	for _, p := range pairs {
		if p.A == a && p.B == b {
			return p, true
		}
	}
	return CouplingCount{}, false
}

func TestCouplingBasicCounts(t *testing.T) {
	o := newCouplingObserver()
	// a+b change together twice; a alone once; b+c together once.
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Observe(CommitInfo{}, commit("a.go"))
	o.Observe(CommitInfo{}, commit("b.go", "c.go"))
	o.Finalise(HistoryWindow{}, headWith("a.go", "b.go", "c.go"))

	ab, ok := findPair(o.pairs, "a.go", "b.go")
	if !ok {
		t.Fatalf("expected a.go↔b.go pair, got %+v", o.pairs)
	}
	if ab.Shared != 2 {
		t.Errorf("a↔b Shared = %d, want 2", ab.Shared)
	}
	if ab.CommitsA != 3 { // a.go changed in 3 commits
		t.Errorf("a.go CommitsA = %d, want 3", ab.CommitsA)
	}
	if ab.CommitsB != 3 { // b.go changed in 3 commits
		t.Errorf("b.go CommitsB = %d, want 3", ab.CommitsB)
	}

	// b↔c shared only once, below CouplingMinShared (2) → must be filtered out.
	if _, ok := findPair(o.pairs, "b.go", "c.go"); ok {
		t.Errorf("b.go↔c.go shares 1 commit; should be below the min-shared floor")
	}
}

func TestCouplingDegree(t *testing.T) {
	// a and b each change in exactly the same 2 commits → union 2, degree 100%.
	o := newCouplingObserver()
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Finalise(HistoryWindow{}, headWith("a.go", "b.go"))

	ab, _ := findPair(o.pairs, "a.go", "b.go")
	if got := ab.Degree(); got != 100.0 {
		t.Errorf("degree = %.1f, want 100.0", got)
	}
}

func TestCouplingLargeCommitSkipped(t *testing.T) {
	o := newCouplingObserver()
	o.maxFilesPerCommit = 3

	// A 4-file commit exceeds the cap: no pairs counted from it, but each file's
	// own commit total still increments.
	o.Observe(CommitInfo{}, commit("a.go", "b.go", "c.go", "d.go"))
	// A normal 2-file commit still produces a pair.
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Finalise(HistoryWindow{}, headWith("a.go", "b.go", "c.go", "d.go"))

	if o.skipped != 1 {
		t.Errorf("skipped = %d, want 1", o.skipped)
	}
	ab, ok := findPair(o.pairs, "a.go", "b.go")
	if !ok {
		t.Fatalf("expected a↔b pair from the small commits")
	}
	// Shared from the two small commits only — the big commit did not add one.
	if ab.Shared != 2 {
		t.Errorf("a↔b Shared = %d, want 2 (big commit excluded)", ab.Shared)
	}
	// But a.go's own total includes the big commit: 3 commits.
	if ab.CommitsA != 3 {
		t.Errorf("a.go CommitsA = %d, want 3", ab.CommitsA)
	}
	// c.go↔d.go only ever co-changed in the skipped big commit → no pair.
	if _, ok := findPair(o.pairs, "c.go", "d.go"); ok {
		t.Errorf("c↔d should not exist; their only co-change was the skipped commit")
	}
}

func TestCouplingRenameFolds(t *testing.T) {
	o := newCouplingObserver()
	// old.go couples with b.go twice, then old.go is renamed to new.go and
	// couples with b.go once more under the new name.
	o.Observe(CommitInfo{}, commit("old.go", "b.go"))
	o.Observe(CommitInfo{}, commit("old.go", "b.go"))
	o.Observe(CommitInfo{}, []FileChange{{Path: "new.go", FromPath: "old.go"}, {Path: "b.go"}})
	o.Finalise(HistoryWindow{}, headWith("new.go", "b.go"))

	// old.go is gone from HEAD; its history must fold into new.go.
	if _, ok := findPair(o.pairs, "old.go", "b.go"); ok {
		t.Errorf("old.go should have folded into new.go, not survive as a pair")
	}
	nb, ok := findPair(o.pairs, "new.go", "b.go")
	if !ok {
		t.Fatalf("expected new.go↔b.go after rename fold, got %+v", o.pairs)
	}
	if nb.Shared != 3 {
		t.Errorf("new.go↔b.go Shared = %d, want 3 (2 pre-rename + 1 post)", nb.Shared)
	}
	if nb.CommitsA != 3 {
		t.Errorf("new.go CommitsA = %d, want 3 (old.go's history folded in)", nb.CommitsA)
	}
}

func TestCouplingPartnersDirectional(t *testing.T) {
	o := newCouplingObserver()
	// hub.h is touched in every commit; leaf.c in 3 of them, always with hub.h.
	// peer.c changes with leaf.c twice.
	o.Observe(CommitInfo{}, commit("hub.h", "leaf.c", "peer.c"))
	o.Observe(CommitInfo{}, commit("hub.h", "leaf.c", "peer.c"))
	o.Observe(CommitInfo{}, commit("hub.h", "leaf.c"))
	o.Observe(CommitInfo{}, commit("hub.h"))
	o.Observe(CommitInfo{}, commit("hub.h"))
	o.Finalise(HistoryWindow{}, headWith("hub.h", "leaf.c", "peer.c"))

	partners := o.partnersFor("leaf.c")
	if len(partners) < 2 {
		t.Fatalf("expected leaf.c to couple with hub.h and peer.c, got %+v", partners)
	}

	// leaf.c changed in 3 commits; hub.h co-changed all 3 → Couple(hub|leaf)=100%.
	// But Reverse(leaf|hub) = 3/5 = 60% — the asymmetry that marks hub.h a hub.
	hub := findPartner(partners, "hub.h")
	if hub.Couple() != 100.0 {
		t.Errorf("Couple(hub.h | leaf.c) = %.1f, want 100.0", hub.Couple())
	}
	if got := hub.Reverse(); got < 59.9 || got > 60.1 {
		t.Errorf("Reverse(leaf.c | hub.h) = %.1f, want ~60.0", got)
	}

	// peer.c co-changed with leaf.c in 2 of leaf's 3 commits → Couple ~66.7%.
	peer := findPartner(partners, "peer.c")
	if got := peer.Couple(); got < 66.0 || got > 67.0 {
		t.Errorf("Couple(peer.c | leaf.c) = %.1f, want ~66.7", got)
	}

	// Degree is base-rate corrected, so the hub scores BELOW the peer even though
	// its Couple is a perfect 100%:
	//   hub.h  → 3/(3+5-3) = 60.0%   (present for all of leaf's commits, but it is
	//                                 present for everyone's commits)
	//   peer.c → 2/(3+2-2) = 66.7%   (never changes without leaf.c — a real partner)
	if got := hub.Degree(); got < 59.9 || got > 60.1 {
		t.Errorf("Degree(hub.h, leaf.c) = %.1f, want ~60.0", got)
	}
	if got := peer.Degree(); got < 66.0 || got > 67.0 {
		t.Errorf("Degree(peer.c, leaf.c) = %.1f, want ~66.7", got)
	}

	// Ranked by Degree: peer.c before hub.h. Ranking by Couple instead put the hub
	// first — the base-rate confound this report exists to avoid.
	if partners[0].Path != "peer.c" {
		t.Errorf("expected peer.c ranked first by Degree, got %s", partners[0].Path)
	}
}

func TestCouplingPartnersUnknownTarget(t *testing.T) {
	o := newCouplingObserver()
	o.Observe(CommitInfo{}, commit("a.go", "b.go"))
	o.Finalise(HistoryWindow{}, headWith("a.go", "b.go"))
	if got := o.partnersFor("nope.go"); got != nil {
		t.Errorf("unknown target should yield nil partners, got %+v", got)
	}
}

func findPartner(ps []CouplingPartner, path string) CouplingPartner {
	for _, p := range ps {
		if p.Path == path {
			return p
		}
	}
	return CouplingPartner{}
}

func TestCouplingDropsFilesAbsentFromHead(t *testing.T) {
	o := newCouplingObserver()
	o.Observe(CommitInfo{}, commit("a.go", "gone.go"))
	o.Observe(CommitInfo{}, commit("a.go", "gone.go"))
	// gone.go is not in HEAD (deleted before the window end).
	o.Finalise(HistoryWindow{}, headWith("a.go"))

	if len(o.pairs) != 0 {
		t.Errorf("pair referencing a deleted file should be dropped, got %+v", o.pairs)
	}
}

// headWithComplexity builds a HeadSnapshot whose files carry the given HEAD
// cyclomatic complexity, for exercising the complexity-weighted ranking.
func headWithComplexity(cx map[string]int64) HeadSnapshot {
	h := HeadSnapshot{Files: map[string]HeadFile{}}
	for p, c := range cx {
		h.Files[p] = HeadFile{Path: p, Complexity: c}
	}
	return h
}

// Weighted ranking must demote a high-churn pair that includes a
// zero-complexity (data/generated) file below a lower-churn pair of two complex
// files — the whole point of --coupling-weighted.
func TestCouplingWeightedDemotesDataFilePairs(t *testing.T) {
	prev := CouplingWeighted
	CouplingWeighted = true
	defer func() { CouplingWeighted = prev }()

	o := newCouplingObserver()
	// data.json + gen.go co-change often (5), but data.json has zero complexity.
	for i := 0; i < 5; i++ {
		o.Observe(CommitInfo{}, commit("data.json", "gen.go"))
	}
	// logic_a.go + logic_b.go co-change less (3), but both are complex.
	for i := 0; i < 3; i++ {
		o.Observe(CommitInfo{}, commit("logic_a.go", "logic_b.go"))
	}
	o.Finalise(HistoryWindow{}, headWithComplexity(map[string]int64{
		"data.json": 0, "gen.go": 120,
		"logic_a.go": 100, "logic_b.go": 100,
	}))

	if len(o.pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d: %+v", len(o.pairs), o.pairs)
	}
	top := o.pairs[0]
	if top.A != "logic_a.go" || top.B != "logic_b.go" {
		t.Errorf("weighted top pair = (%s, %s), want the two complex files first", top.A, top.B)
	}
	// The data-file pair's min-complexity is zero, so its weighted score is zero.
	dataPair, ok := findPair(o.pairs, "data.json", "gen.go")
	if !ok {
		t.Fatal("data.json/gen.go pair missing")
	}
	if dataPair.WeightedScore() != 0 {
		t.Errorf("data-file pair weighted score = %.1f, want 0", dataPair.WeightedScore())
	}
}

// With weighting off (the default), ranking is pure co-change volume — the
// zero-complexity pair with more shared commits leads.
func TestCouplingUnweightedRanksByVolume(t *testing.T) {
	o := newCouplingObserver()
	for i := 0; i < 5; i++ {
		o.Observe(CommitInfo{}, commit("data.json", "gen.go"))
	}
	for i := 0; i < 3; i++ {
		o.Observe(CommitInfo{}, commit("logic_a.go", "logic_b.go"))
	}
	o.Finalise(HistoryWindow{}, headWithComplexity(map[string]int64{
		"data.json": 0, "gen.go": 120,
		"logic_a.go": 100, "logic_b.go": 100,
	}))

	top := o.pairs[0]
	if top.A != "data.json" || top.B != "gen.go" {
		t.Errorf("unweighted top pair = (%s, %s), want the higher-volume data pair first", top.A, top.B)
	}
}

// resolveCouplingTarget's error must be context-neutral: it names the path, not
// any CLI flag, so an MCP client (which passed a `file` argument and has never
// seen the flag) gets a message that reads correctly. The CLI re-attaches the
// flag name at its own call site.
func TestResolveCouplingTargetMissIsFlagNeutral(t *testing.T) {
	repo := makeFixtureRepo(t, []map[string]string{
		{"processor/workers.go": "package processor\n// v0\n", "main.go": "package main\n// v0\n"},
	})

	_, err := resolveCouplingTarget(repo, "processor/wokers.go")
	if err == nil {
		t.Fatal("expected an error for a path not in HEAD, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not in HEAD") {
		t.Errorf("error = %q, want it to mention the target is not in HEAD", msg)
	}
	if strings.Contains(msg, "--") {
		t.Errorf("error = %q, want no CLI flag names in the neutral core error", msg)
	}
	// A basename typo (wokers vs workers) changes the basename, so the
	// same-basename suggestion heuristic does not fire — no suggestion here.
	if strings.Contains(msg, "did you mean") {
		t.Errorf("error = %q, did not expect a suggestion for a basename typo", msg)
	}
}

// Suggestions are built inside resolveCouplingTarget, so they reach every caller
// — CLI and MCP alike. A same-basename file in another directory triggers the
// "did you mean" hint; this asserts it is present in the returned error itself
// (not only in CLI-side rendering).
func TestResolveCouplingTargetSuggestsSameBasename(t *testing.T) {
	repo := makeFixtureRepo(t, []map[string]string{
		{"processor/workers.go": "package processor\n// v0\n", "main.go": "package main\n// v0\n"},
	})

	// Right basename, wrong directory — the heuristic matches on basename.
	_, err := resolveCouplingTarget(repo, "workers.go")
	if err == nil {
		t.Fatal("expected an error for a path not in HEAD, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "did you mean") {
		t.Fatalf("error = %q, want a did-you-mean suggestion", msg)
	}
	if !strings.Contains(msg, "processor/workers.go") {
		t.Errorf("error = %q, want it to suggest processor/workers.go", msg)
	}
	if strings.Contains(msg, "--") {
		t.Errorf("error = %q, want no CLI flag names in the neutral core error", msg)
	}
}
