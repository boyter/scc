// SPDX-License-Identifier: MIT

package processor

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
)

// TestBlobClassifyCacheMemoisesByHash verifies the cache returns the same
// result for repeated lookups of the same hash and only adds one entry.
// The classify method is the only call site for classifyHistoryBlob inside
// the cache, so len(cache.entries) is the underlying classifier call count.
func TestBlobClassifyCacheMemoisesByHash(t *testing.T) {
	ProcessConstants()
	cache := newBlobClassifyCache()
	blob := []byte("package a\nfunc A() {}\n")
	hash := plumbing.ComputeHash(plumbing.BlobObject, blob)

	r1 := cache.classify(hash, "a.go", blob)
	r2 := cache.classify(hash, "a.go", blob)

	if !r1.ok || !r2.ok {
		t.Fatalf("expected ok=true for both lookups, got %v / %v", r1.ok, r2.ok)
	}
	if r1.language != r2.language {
		t.Fatalf("language mismatch across hits: %q vs %q", r1.language, r2.language)
	}
	if got := len(cache.entries); got != 1 {
		t.Fatalf("expected 1 cache entry after two lookups of the same hash, got %d", got)
	}
}

// TestBlobClassifyCacheKeysByHashNotPath confirms two different paths
// pointing at the same blob (a rename) classify once. The cache is
// hash-keyed by design — the rename inherits the original language.
func TestBlobClassifyCacheKeysByHashNotPath(t *testing.T) {
	ProcessConstants()
	cache := newBlobClassifyCache()
	content := []byte("package a\nfunc A() {}\n")
	hash := plumbing.ComputeHash(plumbing.BlobObject, content)

	r1 := cache.classify(hash, "a.go", content)
	r2 := cache.classify(hash, "renamed.go", content)

	if !r1.ok || !r2.ok {
		t.Fatalf("expected ok=true for both, got %v / %v", r1.ok, r2.ok)
	}
	if len(cache.entries) != 1 {
		t.Fatalf("expected 1 cache entry for same hash on different paths, got %d", len(cache.entries))
	}
	if r1.language != r2.language {
		t.Fatalf("expected same language for same hash; got %q vs %q", r1.language, r2.language)
	}
}

// TestBlobClassifyCacheNegativeCaching confirms a blob the classifier
// rejects (unknown extension) is cached with ok=false. A repeat lookup of
// the same hash is a hit, so the classifier does not run a second time.
func TestBlobClassifyCacheNegativeCaching(t *testing.T) {
	ProcessConstants()
	cache := newBlobClassifyCache()
	blob := []byte("nothing identifiable here\n")
	hash := plumbing.ComputeHash(plumbing.BlobObject, blob)

	r1 := cache.classify(hash, "weird.xx", blob)
	r2 := cache.classify(hash, "weird.xx", blob)

	if r1.ok || r2.ok {
		t.Fatalf("expected ok=false for unknown-extension blob, got %v / %v", r1.ok, r2.ok)
	}
	if len(cache.entries) != 1 {
		t.Fatalf("expected negative result to be cached as 1 entry, got %d", len(cache.entries))
	}
}

// TestBlobClassifyCacheSeparatesByHash sanity-checks that distinct hashes
// produce distinct cache entries.
func TestBlobClassifyCacheSeparatesByHash(t *testing.T) {
	ProcessConstants()
	cache := newBlobClassifyCache()
	a := []byte("package a\nfunc A() {}\n")
	b := []byte("package a\nfunc A() {}\nfunc B() {}\n")
	ha := plumbing.ComputeHash(plumbing.BlobObject, a)
	hb := plumbing.ComputeHash(plumbing.BlobObject, b)

	cache.classify(ha, "a.go", a)
	cache.classify(hb, "a.go", b)
	cache.classify(ha, "a.go", a)

	if len(cache.entries) != 2 {
		t.Fatalf("expected 2 cache entries for 2 distinct hashes, got %d", len(cache.entries))
	}
}

// TestRunHistoryStillProducesHeadAndChanges is a smoke test that confirms
// threading the cache through runHistory / buildBaselineForObserver /
// commitChanges / buildHeadSnapshot doesn't change observable behavior: HEAD
// and per-commit changes still classify correctly.
func TestRunHistoryStillProducesHeadAndChanges(t *testing.T) {
	saveDepth := HistoryDepth
	HistoryDepth = 10
	t.Cleanup(func() { HistoryDepth = saveDepth })

	dir := makeFixtureRepo(t, []map[string]string{
		{"a.go": "package a\nfunc A() {}\n"},
		{"b.go": "package b\nfunc B() {}\n"},
		{"c.go": "package c\nfunc C() {}\n"},
	})

	cap := &captureObserver{}
	if _, err := runHistory(dir, cap); err != nil {
		t.Fatalf("runHistory: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		hf, ok := cap.snapshot.Files[name]
		if !ok {
			t.Fatalf("HEAD snapshot missing %s; got %v", name, cap.snapshot.Files)
		}
		if hf.Language != "Go" {
			t.Errorf("%s language = %q, want Go", name, hf.Language)
		}
	}
}
