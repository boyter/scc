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
	Path          string
	Language      string
	AddedRanges   []LineRange
	RemovedRanges []LineRange
	LineTypes     []LineType
	Complexity    []int
	NewBlob       []byte
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
	if walkErr != nil && !errors.Is(walkErr, errStopIter) {
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

	if bo, ok := observer.(BaselineObserver); ok {
		baseline := buildBaselineForObserver(collected, ignore)
		bo.Seed(baseline)
	}

	ctx := context.Background()
	for i := len(collected) - 1; i >= 0; i-- {
		commit := collected[i]
		changes, err := commitChanges(ctx, commit, ignore)
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

	snapshot, err := buildHeadSnapshot(collected[0], ignore)
	if err != nil {
		printWarnF("history: head snapshot: %s", err)
		snapshot = emptySnapshot()
	}

	observer.Finalise(window, snapshot)
	return window, nil
}

// buildBaselineForObserver loads the mailmap from HEAD and classifies the
// tree at the window's start commit. The start commit is the first-parent of
// the oldest commit in the window; if that commit has no parents (the window
// covers all history) the baseline files map is empty.
func buildBaselineForObserver(collected []*object.Commit, ignore *historyIgnore) BaselineSnapshot {
	baseline := BaselineSnapshot{Files: map[string]BaselineFile{}}
	if len(collected) == 0 {
		return baseline
	}

	headCommit := collected[0]
	if headTree, err := headCommit.Tree(); err == nil {
		baseline.Mailmap = loadMailmapFromTree(headTree)
	}

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
		job, lineTypes, ok := classifyHistoryBlob(f.Name, blob)
		if !ok {
			return nil
		}
		baseline.Files[f.Name] = BaselineFile{
			Path:       f.Name,
			Language:   job.Language,
			LineTypes:  lineTypes,
			Complexity: complexityLineNumbers(job),
		}
		return nil
	})

	return baseline
}

// buildHeadSnapshot walks the HEAD commit's tree and runs scc's classifier
// on each file. Used by hotspots (and future reports) to know each surviving
// file's current language and complexity.
func buildHeadSnapshot(headCommit *object.Commit, ignore *historyIgnore) (HeadSnapshot, error) {
	tree, err := headCommit.Tree()
	if err != nil {
		return emptySnapshot(), err
	}

	snap := HeadSnapshot{Files: map[string]HeadFile{}}
	err = tree.Files().ForEach(func(f *object.File) error {
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

		job, _, ok := classifyHistoryBlob(f.Name, blob)
		if !ok {
			return nil
		}

		snap.Files[f.Name] = HeadFile{
			Path:       f.Name,
			Language:   job.Language,
			Complexity: job.Complexity,
		}
		return nil
	})
	return snap, err
}

// commitChanges computes the first-parent diff for commit and projects every
// change into a FileChange. Skips paths that the engine can't count
// (binary blobs, no language detected, submodules, symlinks, ignored paths).
// Deletes are dropped because hotspots-style reports can't render files that
// no longer exist.
func commitChanges(ctx context.Context, commit *object.Commit, ignore *historyIgnore) ([]FileChange, error) {
	toTree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var fromTree *object.Tree
	if commit.NumParents() > 0 {
		parent, err := commit.Parent(0)
		if err != nil {
			return nil, err
		}
		fromTree, err = parent.Tree()
		if err != nil {
			return nil, err
		}
	}

	changes, err := object.DiffTreeWithOptions(ctx, fromTree, toTree, object.DefaultDiffTreeOptions)
	if err != nil {
		return nil, err
	}

	out := make([]FileChange, 0, len(changes))
	for _, change := range changes {
		fc, ok := buildFileChange(change, ignore)
		if !ok {
			continue
		}
		out = append(out, fc)
	}
	return out, nil
}

// buildFileChange converts a single object.Change into a FileChange.
func buildFileChange(change *object.Change, ignore *historyIgnore) (FileChange, bool) {
	action, err := change.Action()
	if err != nil {
		return FileChange{}, false
	}
	if action == merkletrie.Delete {
		return FileChange{}, false
	}

	path := change.To.Name
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

	job, lineTypes, ok := classifyHistoryBlob(path, blob)
	if !ok {
		return FileChange{}, false
	}

	return FileChange{
		Path:          path,
		Language:      job.Language,
		AddedRanges:   added,
		RemovedRanges: removed,
		LineTypes:     lineTypes,
		Complexity:    complexityLineNumbers(job),
		NewBlob:       blob,
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
		Location:          path,
		Filename:          basename(path),
		Extension:         extension,
		PossibleLanguages: languages,
		Bytes:             int64(len(blob)),
		Content:           blob,
	}

	job.Language = DetermineLanguage(job.Filename, job.Language, job.PossibleLanguages, job.Content)
	if job.Language == SheBang {
		cutoff := len(blob)
		if cutoff > 200 {
			cutoff = 200
		}
		lang, err := DetectSheBang(string(blob[:cutoff]))
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
