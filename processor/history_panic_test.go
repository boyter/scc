// SPDX-License-Identifier: MIT

package processor

import (
	"testing"
)

// TestSafeClassifyRecoversFromPanic confirms a panic inside the classifier is
// caught and surfaced as ok=false rather than crashing the report. Uses the
// classifyFn indirection so the test doesn't need a real-world bad blob.
func TestSafeClassifyRecoversFromPanic(t *testing.T) {
	ProcessConstants()
	save := classifyFn
	t.Cleanup(func() { classifyFn = save })

	classifyFn = func(path string, blob []byte) (*FileJob, []LineType, bool) {
		panic("synthetic classifier panic")
	}

	job, lineTypes, ok := safeClassify("evil.go", []byte("package a\n"))
	if ok {
		t.Fatalf("expected ok=false on classifier panic, got ok=true")
	}
	if job != nil {
		t.Errorf("expected nil job on panic, got %v", job)
	}
	if lineTypes != nil {
		t.Errorf("expected nil lineTypes on panic, got %v", lineTypes)
	}
}

// TestSafeClassifyPassesThroughOnSuccess sanity-checks that the wrapper is a
// no-op for clean inputs — same classification, no behaviour change.
func TestSafeClassifyPassesThroughOnSuccess(t *testing.T) {
	ProcessConstants()
	job, lineTypes, ok := safeClassify("a.go", []byte("package a\nfunc A() {}\n"))
	if !ok {
		t.Fatalf("expected ok=true for a valid Go file")
	}
	if job == nil || job.Language != "Go" {
		t.Fatalf("expected Go classification, got %v", job)
	}
	if len(lineTypes) == 0 {
		t.Errorf("expected non-empty lineTypes")
	}
}

// TestCommitChangesRecoversFromPanic confirms a panicking classifier for one
// path doesn't abort the walk: every commit still produces an Observe call,
// the panicking file is skipped, and other files in the same commit survive.
// Also covers HEAD snapshot: a HEAD-present panicking file is skipped, not
// fatal.
func TestCommitChangesRecoversFromPanic(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 10
	t.Cleanup(func() { HistoryDepth = saveDepth })

	save := classifyFn
	t.Cleanup(func() { classifyFn = save })
	classifyFn = func(path string, blob []byte) (*FileJob, []LineType, bool) {
		if path == "bad.go" {
			panic("synthetic per-path panic")
		}
		return classifyHistoryBlob(path, blob)
	}

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\nfunc A() {}\n", "bad.go": "package bad\nfunc B() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc A2() {}\n", "bad.go": "package bad\nfunc B() {}\nfunc B2() {}\n"},
		{"a.go": "package a\nfunc A() {}\nfunc A2() {}\nfunc A3() {}\n", "bad.go": "package bad\nfunc B() {}\nfunc B2() {}\nfunc B3() {}\n"},
	})

	cap := &captureObserver{}
	if _, err := runHistory(dir, cap); err != nil {
		t.Fatalf("runHistory: %v", err)
	}

	if len(cap.commits) != 3 {
		t.Fatalf("expected 3 commits observed despite panic, got %d", len(cap.commits))
	}

	for i, changes := range cap.changes {
		var sawGood bool
		for _, fc := range changes {
			if fc.Path == "bad.go" {
				t.Errorf("commit %d: bad.go should have been skipped, but appeared in changes", i)
			}
			if fc.Path == "a.go" {
				sawGood = true
			}
		}
		if !sawGood {
			t.Errorf("commit %d: a.go missing from changes — recover should not affect other files", i)
		}
	}

	if _, ok := cap.snapshot.Files["bad.go"]; ok {
		t.Errorf("HEAD snapshot should not contain bad.go (classifier panics), but it does")
	}
	if _, ok := cap.snapshot.Files["a.go"]; !ok {
		t.Errorf("HEAD snapshot should contain a.go")
	}
}

// TestSafePatchRecoversFromPanicRegression re-asserts the original
// change.Patch() recovery still triggers — we don't want the broadened
// recovery work to silently drop the existing safety net. Calling
// change.Patch() on a nil *object.Change dereferences a nil pointer inside
// the go-git implementation, which is the simplest available panic trigger
// that exercises safePatch's recover.
func TestSafePatchRecoversFromPanicRegression(t *testing.T) {
	patch, ok := safePatch(nil)
	if ok {
		t.Fatalf("expected ok=false on nil change, got ok=true")
	}
	if patch != nil {
		t.Fatalf("expected nil patch on panic, got %v", patch)
	}
}
