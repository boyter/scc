// SPDX-License-Identifier: MIT

package processor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	fdiff "github.com/go-git/go-git/v5/plumbing/format/diff"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/utils/merkletrie"
)

// HistoryDepth is the maximum number of commits the history engine walks. 0
// means "entire history". Wired to --depth in main.go.
var HistoryDepth = 1000

// LineRange is a half-open line span [Start, Start+Count) in 1-based line
// numbers. A FileChange carries one entry per contiguous run of added (or
// removed) lines emitted by go-git's diff.
type LineRange struct {
	Start int
	Count int
}

// CommitInfo is the per-commit metadata handed to observers.
type CommitInfo struct {
	Hash   plumbing.Hash
	Author string
	Email  string
	When   time.Time
}

// FileChange is one changed file inside a commit. AddedRanges/RemovedRanges
// describe the diff against the first parent; LineTypes and Complexity are
// scc's classifier output for the new blob (one LineType per line, one entry
// in Complexity per line that fired a complexity tick).
type FileChange struct {
	Path             string
	FromPath         string // != Path on a detected rename; "" on a pure add
	Language         string
	AddedRanges      []LineRange
	RemovedRanges    []LineRange
	LineTypes        []LineType
	RemovedLineTypes []LineType // old-blob line types, for code-filtered removals
	Complexity       []int
	NewBlob          []byte
}

// CommitObserver is implemented by each report's accumulator. The engine
// invokes Observe once per commit oldest-first, then Finalise once with the
// window metadata and a snapshot of the HEAD tree (latest language /
// complexity per surviving file).
type CommitObserver interface {
	Observe(c CommitInfo, changes []FileChange)
	Finalise(window HistoryWindow, head HeadSnapshot)
}

// HistoryWindow describes the commit window the engine walked.
type HistoryWindow struct {
	Depth   int
	Commits int
	From    time.Time
	To      time.Time
	Head    plumbing.Hash
}

// HeadFile is one file in the HEAD tree, classified by scc's engine.
type HeadFile struct {
	Path       string
	Language   string
	Complexity int64
	Cognitive  int64 // nesting-weighted complexity; zero unless the Cognitive global is on
}

// HeadSnapshot is the set of files in HEAD, keyed by path.
type HeadSnapshot struct {
	Files map[string]HeadFile
}

// BaselineFile is one file from the window's start-commit tree, classified by
// scc's engine. Carries per-line type and complexity placement so observers
// can attribute lines that survive untouched from before the window.
type BaselineFile struct {
	Path       string
	Language   string
	LineTypes  []LineType
	Complexity []int // 1-based line numbers that fired a complexity tick
}

// BaselineSnapshot is the optional pre-walk state handed to observers that
// implement BaselineObserver. Files holds the classified contents of the
// window's start-commit tree (empty when the window covers all history);
// Mailmap is the parsed .mailmap from the HEAD tree, if present.
type BaselineSnapshot struct {
	Files   map[string]BaselineFile
	Mailmap *mailmap
}

// BaselineObserver is an optional extension to CommitObserver. When an
// observer implements it, the engine builds the baseline snapshot before the
// walk and calls Seed once. Observers that don't need the baseline (e.g.
// Hotspots) skip the expense by not implementing the interface.
type BaselineObserver interface {
	Seed(BaselineSnapshot)
}

// MailmapObserver is an optional extension to CommitObserver. The engine
// always parses the repo's .mailmap from HEAD — one small blob — and hands
// it to observers that implement this, before the walk. Unlike
// BaselineObserver it does NOT trigger the expensive start-tree
// classification, so observers that only need author folding (e.g. Hotspots,
// the author timeline) can implement it cheaply.
type MailmapObserver interface {
	SetMailmap(*mailmap)
}

// errStopIter is a local sentinel used to terminate iter.ForEach once we've
// collected --depth commits. iter.ForEach surfaces whatever the callback
// returns, so we can compare it back at the call site directly.
var errStopIter = errors.New("history: stop iteration")

// Bucketing divides [From, To] into N equal time slices. Used by the timeline
// reports (plans 04 and 05) to map per-commit timestamps to a fixed-resolution
// per-bucket series independent of terminal width.
type Bucketing struct {
	From  time.Time
	To    time.Time
	N     int
	Width time.Duration
}

// NewBucketing constructs a Bucketing covering [from, to] divided into n
// equal-width slices. n must be > 0; n <= 0 is normalised to 1 so callers can
// pass user input unchecked. A degenerate window (from == to or to before
// from) yields Width=0; all commits land in bucket 0 / N-1.
func NewBucketing(from, to time.Time, n int) Bucketing {
	if n <= 0 {
		n = 1
	}
	b := Bucketing{From: from, To: to, N: n}
	if to.After(from) {
		b.Width = to.Sub(from) / time.Duration(n)
	}
	return b
}

// Index returns the 0..N-1 bucket slot for commit time t. Times before From
// clamp to 0 (defensive — should not happen given the walk window). Times at
// or after To clamp to N-1.
func (b Bucketing) Index(t time.Time) int {
	if b.N <= 0 {
		return 0
	}
	if b.Width <= 0 {
		return 0
	}
	if !t.After(b.From) {
		return 0
	}
	if !t.Before(b.To) {
		return b.N - 1
	}
	idx := int(t.Sub(b.From) / b.Width)
	if idx < 0 {
		return 0
	}
	if idx >= b.N {
		return b.N - 1
	}
	return idx
}

// Start returns the wall-clock start time of bucket i. Indexes outside
// [0, N) are clamped.
func (b Bucketing) Start(i int) time.Time {
	if b.N <= 0 {
		return b.From
	}
	if i <= 0 {
		return b.From
	}
	if i >= b.N {
		i = b.N - 1
	}
	return b.From.Add(time.Duration(i) * b.Width)
}

// emptySnapshot is what observers see when HEAD is missing or empty.
func emptySnapshot() HeadSnapshot {
	return HeadSnapshot{Files: map[string]HeadFile{}}
}

// runHistory opens the repo at repoPath, walks up to HistoryDepth commits
// (newest first → oldest first), and feeds every commit's first-parent diff
// to the observer.
func runHistory(repoPath string, observer CommitObserver) (HistoryWindow, error) {
	// Turn GC back on because we have no idea how much we are about to process
	EnableGc()

	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return HistoryWindow{}, fmt.Errorf("open git repository: %w", err)
	}

	head, err := repo.Head()
	if err != nil {
		if errors.Is(err, plumbing.ErrReferenceNotFound) {
			observer.Finalise(HistoryWindow{}, emptySnapshot())
			return HistoryWindow{}, nil
		}
		return HistoryWindow{}, fmt.Errorf("read HEAD: %w", err)
	}

	iter, err := repo.Log(&git.LogOptions{
		From:  head.Hash(),
		Order: git.LogOrderCommitterTime,
	})
	if err != nil {
		return HistoryWindow{}, fmt.Errorf("walk log: %w", err)
	}

	collected := make([]*object.Commit, 0)
	walkErr := iter.ForEach(func(c *object.Commit) error {
		collected = append(collected, c)
		if HistoryDepth > 0 && len(collected) >= HistoryDepth {
			return errStopIter
		}
		return nil
	})
	// A shallow clone (e.g. CI's default `git checkout --depth 1`) stores a
	// parent hash for its oldest commit but not the parent object itself.
	// go-git's commit walker resolves each commit's parents as it advances, so
	// it surfaces that absent object as ErrObjectNotFound — exactly the same
	// "no more history to walk" situation as the root commit reaching zero
	// parents, just reached via a missing-object error instead of a count.
	// Treat it as end-of-history and keep what we walked, rather than aborting.
	if walkErr != nil && !errors.Is(walkErr, errStopIter) && !errors.Is(walkErr, plumbing.ErrObjectNotFound) {
		return HistoryWindow{}, fmt.Errorf("collect commits: %w", walkErr)
	}

	if len(collected) == 0 {
		observer.Finalise(HistoryWindow{Head: head.Hash()}, emptySnapshot())
		return HistoryWindow{Head: head.Hash()}, nil
	}

	window := HistoryWindow{
		Depth:   HistoryDepth,
		Commits: len(collected),
		From:    collected[len(collected)-1].Author.When,
		To:      collected[0].Author.When,
		Head:    head.Hash(),
	}

	ignore, err := buildHistoryIgnore(repo, head.Hash())
	if err != nil {
		printWarnF("history: ignore matcher: %s", err)
	}

	cache := newBlobClassifyCache()

	if mo, ok := observer.(MailmapObserver); ok {
		mo.SetMailmap(loadMailmapForHead(collected[0]))
	}

	if bo, ok := observer.(BaselineObserver); ok {
		baseline := buildBaselineForObserver(collected, ignore, cache)
		bo.Seed(baseline)
	}

	ctx := context.Background()
	for i := len(collected) - 1; i >= 0; i-- {
		commit := collected[i]
		changes, err := commitChanges(ctx, commit, ignore, cache)
		if err != nil {
			printWarnF("history: diff %s: %s", commit.Hash, err)
			continue
		}
		observer.Observe(CommitInfo{
			Hash:   commit.Hash,
			Author: commit.Author.Name,
			Email:  commit.Author.Email,
			When:   commit.Author.When,
		}, changes)
	}

	snapshot, err := buildHeadSnapshot(collected[0], ignore, cache)
	if err != nil {
		printWarnF("history: head snapshot: %s", err)
		snapshot = emptySnapshot()
	}

	observer.Finalise(window, snapshot)
	return window, nil
}

// loadMailmapForHead parses .mailmap from the HEAD commit's tree. Returns
// nil when there is no .mailmap or the HEAD tree cannot be read. Cheap
// compared to building the full baseline, so observers that only need
// author folding (Hotspots, author timeline) can satisfy MailmapObserver
// without paying for the start-tree classification.
func loadMailmapForHead(headCommit *object.Commit) *mailmap {
	if headCommit == nil {
		return nil
	}
	tree, err := headCommit.Tree()
	if err != nil {
		return nil
	}
	return loadMailmapFromTree(tree)
}

// buildBaselineForObserver loads the mailmap from HEAD and classifies the
// tree at the window's start commit. The start commit is the first-parent of
// the oldest commit in the window; if that commit has no parents (the window
// covers all history) the baseline files map is empty.
func buildBaselineForObserver(collected []*object.Commit, ignore *historyIgnore, cache *blobClassifyCache) (baseline BaselineSnapshot) {
	baseline = BaselineSnapshot{Files: map[string]BaselineFile{}}
	// Backstop for panics outside the per-file recover below — go-git's
	// tree.Files() iterator can itself panic on a corrupt object, and the
	// per-file handler is not in scope for that. Return whatever was
	// accumulated so far rather than crashing the report.
	defer func() {
		if r := recover(); r != nil {
			printWarnF("history: baseline walk panicked, using partial result: %v", r)
		}
	}()
	if len(collected) == 0 {
		return baseline
	}

	baseline.Mailmap = loadMailmapForHead(collected[0])

	oldest := collected[len(collected)-1]
	if oldest.NumParents() == 0 {
		return baseline
	}
	parent, err := oldest.Parent(0)
	if err != nil {
		printWarnF("history: baseline parent: %s", err)
		return baseline
	}
	tree, err := parent.Tree()
	if err != nil {
		printWarnF("history: baseline tree: %s", err)
		return baseline
	}

	_ = tree.Files().ForEach(func(f *object.File) error {
		defer func() {
			if r := recover(); r != nil {
				name := "<unknown>"
				if f != nil {
					name = f.Name
				}
				printWarnF("history: skipping %s in baseline — panicked: %v", name, r)
			}
		}()
		if f.Mode == filemode.Dir || f.Mode == filemode.Submodule || f.Mode == filemode.Symlink {
			return nil
		}
		if ignore != nil && ignore.Match(f.Name, false) {
			return nil
		}
		reader, err := f.Reader()
		if err != nil {
			return nil
		}
		defer reader.Close()
		blob, err := io.ReadAll(reader)
		if err != nil {
			return nil
		}
		res := cache.classify(f.Hash, f.Name, blob)
		if !res.ok {
			return nil
		}
		baseline.Files[f.Name] = BaselineFile{
			Path:       f.Name,
			Language:   res.language,
			LineTypes:  res.lineTypes,
			Complexity: res.complexLine,
		}
		return nil
	})

	return baseline
}

// buildHeadSnapshot walks the HEAD commit's tree and runs scc's classifier
// on each file. Used by hotspots (and future reports) to know each surviving
// file's current language and complexity.
func buildHeadSnapshot(headCommit *object.Commit, ignore *historyIgnore, cache *blobClassifyCache) (snap HeadSnapshot, err error) {
	snap = emptySnapshot()
	// Backstop for panics outside the per-file recover below — go-git's
	// tree.Files() iterator can itself panic on a corrupt object. Return the
	// partial snapshot with no error so the caller keeps what we collected.
	defer func() {
		if r := recover(); r != nil {
			printWarnF("history: HEAD snapshot walk panicked, using partial result: %v", r)
			err = nil
		}
	}()

	tree, err := headCommit.Tree()
	if err != nil {
		return emptySnapshot(), err
	}

	snap = HeadSnapshot{Files: map[string]HeadFile{}}
	err = tree.Files().ForEach(func(f *object.File) error {
		defer func() {
			if r := recover(); r != nil {
				name := "<unknown>"
				if f != nil {
					name = f.Name
				}
				printWarnF("history: skipping %s in HEAD snapshot — panicked: %v", name, r)
			}
		}()
		if f.Mode == filemode.Dir || f.Mode == filemode.Submodule || f.Mode == filemode.Symlink {
			return nil
		}
		if ignore != nil && ignore.Match(f.Name, false) {
			return nil
		}
		reader, err := f.Reader()
		if err != nil {
			return nil
		}
		defer reader.Close()
		blob, err := io.ReadAll(reader)
		if err != nil {
			return nil
		}

		res := cache.classify(f.Hash, f.Name, blob)
		if !res.ok {
			return nil
		}

		snap.Files[f.Name] = HeadFile{
			Path:       f.Name,
			Language:   res.language,
			Complexity: res.complexity,
			Cognitive:  res.cognitive,
		}
		return nil
	})
	return snap, err
}

// historyDiffOptions forces rename detection on. The rename-aware reports
// (author rollup, hotspots) depend on renames arriving as a single change
// rather than a delete + add pair, so pin it explicitly — a future go-git
// bump can't silently disable it.
var historyDiffOptions = &object.DiffTreeOptions{DetectRenames: true}

// commitChanges computes the first-parent diff for commit and projects every
// change into a FileChange. Skips paths that the engine can't count
// (binary blobs, no language detected, submodules, symlinks, ignored paths).
// Deletes are dropped because hotspots-style reports can't render files that
// no longer exist.
//
// The outer recover catches anything the per-call wrappers don't (corrupt
// packfiles via go-git object resolution, future regressions in the diff
// pipeline). One bad commit becomes a warning, not a crash.
func commitChanges(ctx context.Context, commit *object.Commit, ignore *historyIgnore, cache *blobClassifyCache) (out []FileChange, err error) {
	defer func() {
		if r := recover(); r != nil {
			printWarnF("history: skipping commit %s — diff pipeline panicked: %v", commit.Hash, r)
			out = nil
			err = nil
		}
	}()

	toTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var fromTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		// A shallow-clone boundary commit carries a parent hash whose object is
		// absent. Treat the missing parent as no parent (leave fromTree nil, so
		// the commit diffs against the empty tree) instead of failing — the
		// same end-of-history handling as commits with zero parents.
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		fromTree, err = parent.Tree()
		if errors.Is(err, plumbing.ErrObjectNotFound) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}

	changes, err := object.DiffTreeWithOptions(ctx, fromTree, toTree, historyDiffOptions)
	if err != nil {
		return nil, err
	}

	out = make([]FileChange, 0, len(changes))
	for _, change := range changes {
		fc, ok := buildFileChange(change, ignore, cache)
		if !ok {
			continue
		}
		out = append(out, fc)
	}
	return out, nil
}

// buildFileChange converts a single object.Change into a FileChange.
func buildFileChange(change *object.Change, ignore *historyIgnore, cache *blobClassifyCache) (FileChange, bool) {
	action, err := change.Action()
	if err != nil {
		return FileChange{}, false
	}
	if action == merkletrie.Delete {
		return FileChange{}, false
	}

	path := change.To.Name
	fromPath := change.From.Name
	toEntry := change.To.TreeEntry
	if toEntry.Mode == filemode.Dir || toEntry.Mode == filemode.Submodule || toEntry.Mode == filemode.Symlink {
		return FileChange{}, false
	}

	if ignore != nil && ignore.Match(path, false) {
		return FileChange{}, false
	}

	languages, _ := DetectLanguage(path)
	if len(languages) == 0 {
		return FileChange{}, false
	}

	patch, ok := safePatch(change)
	if !ok {
		return FileChange{}, false
	}

	var added, removed []LineRange
	for _, fp := range patch.FilePatches() {
		if fp.IsBinary() {
			return FileChange{}, false
		}
		toLine, fromLine := 1, 1
		for _, chunk := range fp.Chunks() {
			lines := lineCount(chunk.Content())
			switch chunk.Type() {
			case fdiff.Equal:
				toLine += lines
				fromLine += lines
			case fdiff.Add:
				if lines > 0 {
					added = append(added, LineRange{Start: toLine, Count: lines})
				}
				toLine += lines
			case fdiff.Delete:
				if lines > 0 {
					removed = append(removed, LineRange{Start: fromLine, Count: lines})
				}
				fromLine += lines
			}
		}
	}

	blob, err := readBlob(change.To.Tree, &toEntry)
	if err != nil {
		return FileChange{}, false
	}

	res := cache.classify(toEntry.Hash, path, blob)
	if !res.ok {
		return FileChange{}, false
	}

	// Classify the parent blob so removed lines can be filtered to code —
	// the timeline reports need a symmetric code-only delta. The blob cache
	// makes this near-free: the old blob is normally an earlier commit's
	// new blob. Skip entirely when there are no removals (pure adds, or
	// when the diff produced no removed ranges).
	var removedLineTypes []LineType
	if len(removed) > 0 && change.From.Name != "" {
		fromEntry := change.From.TreeEntry
		if fromEntry.Mode != filemode.Dir &&
			fromEntry.Mode != filemode.Submodule &&
			fromEntry.Mode != filemode.Symlink {
			if oldBlob, rerr := readBlob(change.From.Tree, &fromEntry); rerr == nil {
				if oldRes := cache.classify(fromEntry.Hash, change.From.Name, oldBlob); oldRes.ok {
					removedLineTypes = oldRes.lineTypes
				}
			}
		}
	}

	return FileChange{
		Path:             path,
		FromPath:         fromPath,
		Language:         res.language,
		AddedRanges:      added,
		RemovedRanges:    removed,
		LineTypes:        res.lineTypes,
		RemovedLineTypes: removedLineTypes,
		Complexity:       res.complexLine,
		NewBlob:          blob,
	}, true
}

// safePatch wraps change.Patch() with panic recovery. The underlying
// sergi/go-diff line-to-rune encoding panics when a file has more distinct
// lines than fit in the Unicode code-point space (generated SQL, huge
// minified bundles, vendored data files). Treat any panic or error as
// "skip this file" so one bad file does not abort the whole report.
func safePatch(change *object.Change) (patch *object.Patch, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			path := ""
			if change != nil {
				path = change.To.Name
				if path == "" {
					path = change.From.Name
				}
			}
			printWarnF("history: skipping %s — diff library panicked: %v", path, r)
			patch = nil
			ok = false
		}
	}()
	p, err := change.Patch()
	if err != nil {
		return nil, false
	}
	return p, true
}

// classifyFn is the indirect reference to classifyHistoryBlob used by
// safeClassify. Tests substitute a panicking stub to exercise the recover
// path; production behaviour is unchanged.
var classifyFn = classifyHistoryBlob

// safeClassify wraps the classifier with panic recovery. The history walk
// feeds the classifier many more blob shapes than the working-tree counter
// ever sees (legacy encodings, partial UTF-8, oversized blobs, vendored
// data). A panic in any one path-blob pair must not abort the report —
// skip the file with a warning instead.
func safeClassify(path string, blob []byte) (job *FileJob, lineTypes []LineType, ok bool) {
	defer func() {
		if r := recover(); r != nil {
			printWarnF("history: skipping %s — classifier panicked: %v", path, r)
			job = nil
			lineTypes = nil
			ok = false
		}
	}()
	return classifyFn(path, blob)
}

// readBlob fetches the raw bytes for a tree entry.
func readBlob(tree *object.Tree, entry *object.TreeEntry) ([]byte, error) {
	file, err := tree.TreeEntryFile(entry)
	if err != nil {
		return nil, err
	}
	reader, err := file.Reader()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// blobClassifyResult is the cached output of classifyHistoryBlob for a single
// blob hash. ok=false means the classifier rejected the blob (binary, no
// language); the vectors are nil in that case.
type blobClassifyResult struct {
	language      string
	complexity    int64
	cognitive     int64 // nesting-weighted complexity; zero unless the Cognitive global is on
	lineTypes     []LineType
	complexLine   []int
	cognitiveLine []int // 1-based lines that accrued cognitive weight; nil unless Cognitive is on
	ok            bool
}

// blobClassifyCache memoises classifyHistoryBlob output keyed by blob hash so
// the same blob seen in baseline, commit changes, and HEAD is classified once
// per runHistory. The walk is sequential, so no mutex is required.
type blobClassifyCache struct {
	entries map[plumbing.Hash]blobClassifyResult
}

func newBlobClassifyCache() *blobClassifyCache {
	return &blobClassifyCache{entries: make(map[plumbing.Hash]blobClassifyResult)}
}

// classify returns the classifier output for blob, computing and caching it
// on first sight. Slices in the returned result are shared between callers —
// they must be treated as read-only. Negative results (ok=false) are cached
// too so binary/unknown blobs aren't re-attempted.
func (c *blobClassifyCache) classify(hash plumbing.Hash, path string, blob []byte) blobClassifyResult {
	if c != nil {
		if hit, found := c.entries[hash]; found {
			return hit
		}
	}
	job, lineTypes, ok := safeClassify(path, blob)
	res := blobClassifyResult{ok: ok}
	if ok {
		res.language = job.Language
		res.complexity = job.Complexity
		res.cognitive = job.Cognitive
		res.lineTypes = lineTypes
		res.complexLine = complexityLineNumbers(job)
		res.cognitiveLine = cognitiveLineNumbers(job)
	}
	if c != nil {
		c.entries[hash] = res
	}
	return res
}

// classifyHistoryBlob runs scc's existing classifier on a git blob's bytes
// and returns the resulting FileJob (Language / Complexity / Code / Comment
// / Blank populated) plus the per-line type vector. ok=false means the file
// is binary or the language could not be resolved.
func classifyHistoryBlob(path string, blob []byte) (*FileJob, []LineType, bool) {
	languages, extension := DetectLanguage(path)
	if len(languages) == 0 {
		return nil, nil, false
	}
	for _, l := range languages {
		LoadLanguageFeature(l)
	}

	job := &FileJob{
		Location:             path,
		Filename:             basename(path),
		Extension:            extension,
		PossibleLanguages:    languages,
		Bytes:                int64(len(blob)),
		Content:              blob,
		TrackComplexityLines: true,
	}

	job.Language = DetermineLanguage(job.Filename, job.Language, job.PossibleLanguages, job.Content)
	if job.Language == SheBang {
		cutoff := min(len(blob), 200)
		lang, err := DetectSheBang(blob[:cutoff])
		if err != nil {
			return nil, nil, false
		}
		job.Language = lang
		LoadLanguageFeature(lang)
	}

	classifier := &historyLineCallback{}
	job.Callback = classifier

	CountStats(job)

	if job.Binary {
		return nil, nil, false
	}

	return job, classifier.lineTypes, true
}

// complexityLineNumbers returns the 1-based line numbers in job that fired a
// complexity tick. Convenience wrapper for observers that want per-line
// complexity placement (the per-line attribution reports in plans 03–04).
func complexityLineNumbers(job *FileJob) []int {
	out := make([]int, 0)
	for i, count := range job.ComplexityLine {
		if count > 0 {
			out = append(out, i+1)
		}
	}
	return out
}

// cognitiveLineNumbers returns the 1-based line numbers in job that accrued
// cognitive weight. Mirrors complexityLineNumbers for the cognitive per-line
// array; returns an empty slice when cognitive tracking is off (CognitiveLine
// nil), so callers get the same shape either way.
func cognitiveLineNumbers(job *FileJob) []int {
	out := make([]int, 0)
	for i, weight := range job.CognitiveLine {
		if weight > 0 {
			out = append(out, i+1)
		}
	}
	return out
}

type historyLineCallback struct {
	lineTypes []LineType
}

func (h *historyLineCallback) ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool {
	h.lineTypes = append(h.lineTypes, lineType)
	return true
}

func basename(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func lineCount(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
