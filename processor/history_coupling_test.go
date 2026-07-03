// SPDX-License-Identifier: MIT

package processor

import "testing"

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

	// Ranked by Couple: hub.h (100) before peer.c (66.7).
	if partners[0].Path != "hub.h" {
		t.Errorf("expected hub.h ranked first by Couple, got %s", partners[0].Path)
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
