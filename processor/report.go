// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/boyter/gocodewalker"
	"github.com/go-git/go-git/v5"
)

// DefaultReportName is the file name used when --report is invoked without
// a path (pflag's NoOptDefVal). main.go wires this in as the bare-flag
// default; runReport compares ReportOut to it to decide whether the user
// supplied an explicit path or relied on the default.
const DefaultReportName = "scc-report.html"

// ReportOut is the output path supplied via --report. Empty means report
// mode is off; any other value (including DefaultReportName when the user
// passed a bare `--report`) flips Process() into the HTML-report branch.
var ReportOut = ""

// ReportSkip is the raw comma-separated value supplied via --report-skip.
// Process() parses it into ReportSkipNames before the report runs.
var ReportSkip = ""

// ReportSkipNames is the parsed, lower-cased set of section names supplied
// via --report-skip. Wired from main.go (spec 05). CollectReportData reads
// this through ReportSkipped to decide which *Result pointers to nil out
// before returning.
var ReportSkipNames = map[string]bool{}

// ReportTitle is the override for the repo name used in the report banner
// (spec 05). Empty means "auto-detect".
var ReportTitle = ""

// reportSkipRecognised is the set of section names --report-skip accepts.
// Kept here (next to ReportSkipped) so future template authors can find the
// authoritative list in one place. Names must match what the report template
// and CollectReportData branch on. Spec 05 fixes this set.
var reportSkipRecognised = map[string]bool{
	"cocomo":     true,
	"locomo":     true,
	"hotspots":   true,
	"coupling":   true,
	"authors":    true,
	"timeline":   true,
	"files":      true,
	"uloc":       true,
	"linelength": true,
	"card":       true,
}

// ReportSkipped reports whether the given section name was listed in
// --report-skip. Section names are case-insensitive — callers can pass
// either case.
func ReportSkipped(section string) bool {
	if len(ReportSkipNames) == 0 {
		return false
	}
	return ReportSkipNames[strings.ToLower(section)]
}

// Totals captures the headline numbers shown in the report's Overview strip.
// Mirrors the sums computed by the tabular formatter (sumFiles / sumLines /
// …) but pulled into a struct so the template can read them by name.
type Totals struct {
	Files      int64
	Lines      int64
	Code       int64
	Comment    int64
	Blank      int64
	Complexity int64
	Bytes      int64
}

// ULOCResult is the unique-lines-of-code rollup. Maps are converted to a
// stable slice here so the template can range deterministically.
type ULOCResult struct {
	Global      int
	PerLanguage []ULOCLanguage
	TotalLines  int64
	Dryness     float64
}

// ULOCLanguage is one row of the per-language ULOC slice. Sorted by ULOC
// descending, then name ascending.
type ULOCLanguage struct {
	Language string
	ULOC     int
}

// LineLengthBucket is one bar in the line-length histogram. Edges are
// inclusive-left, exclusive-right except for the open-ended tail bucket.
type LineLengthBucket struct {
	Start int // inclusive
	End   int // exclusive; 0 means "no upper bound" (the tail bucket)
	Count int64
	Label string // e.g. "0–20", "120+"
}

// LineLengthOutlier is one entry in the longest-lines callout list.
type LineLengthOutlier struct {
	File       string
	Language   string
	LineLength int
}

// LineLengthResult is the line-length histogram and summary statistics.
type LineLengthResult struct {
	Buckets    []LineLengthBucket
	Mean       float64
	Max        int
	Outliers   []LineLengthOutlier
	TotalLines int64
}

// HotspotsResult mirrors the data the tabular hotspot formatter consumes.
// Records is already sorted by Score desc.
type HotspotsResult struct {
	Window    HistoryWindow
	Records   []HotspotRow
	TotalRaw  int
	Available bool
}

// HotspotRow is one row of the hotspots table. Pulled out so report consumers
// don't depend on the private hotspotsRecord type.
type HotspotRow struct {
	File         string
	Language     string
	Complexity   int64
	Commits      int
	LinesChanged int64
	Authors      int
	CodeChurn    int64
	CommentChurn int64
	Score        float64
}

// CouplingResult mirrors the all-pairs change-coupling data. Pairs is already
// sorted strongest-first (raw co-change volume — the report never uses the
// complexity-weighted ranking).
type CouplingResult struct {
	Window     HistoryWindow
	Pairs      []CouplingPairRow
	TotalPairs int
	Available  bool
}

// CouplingPairRow is one file pair. Public so report consumers don't depend on
// the private CouplingCount type.
type CouplingPairRow struct {
	FileA  string
	FileB  string
	Shared int
	Degree float64
}

// AuthorsResult mirrors the data the authors tabular formatter consumes. The
// Sentinel pseudo-row (`(before window)`) is included in Rows; consumers
// filter or call it out separately.
type AuthorsResult struct {
	Window       HistoryWindow
	Rows         []AuthorRow
	BusFactor    int
	BusAuthors   []string
	BusCovered   float64
	InWindowCode int64
}

// AuthorRow is one row of the authors rollup table. Mirrors authorRow but
// public for template consumers.
type AuthorRow struct {
	Name            string
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

// LangTimelineResult mirrors the language-timeline observer output.
type LangTimelineResult struct {
	Window  HistoryWindow
	Bucket  Bucketing
	Rows    []LangTimelineRow
	Buckets int
}

// LangTimelineRow is one row of the language timeline table.
type LangTimelineRow struct {
	Language      string
	StartingLines int64
	CodeNow       int64
	Change        int64
	SharePercent  float64
	Deltas        []int64
	Trajectory    []int64
}

// AuthorTimelineResult mirrors the author-timeline observer output.
type AuthorTimelineResult struct {
	Window  HistoryWindow
	Bucket  Bucketing
	Rows    []AuthorTimelineRow
	Buckets int
}

// AuthorTimelineRow is one row of the author timeline table.
type AuthorTimelineRow struct {
	Name         string
	Email        string
	TotalCommits int
	CodeDelta    int64
	Series       []AuthorTimelineBucket
}

// AuthorTimelineBucket is one bucket of an author's timeline series.
type AuthorTimelineBucket struct {
	Commits   int
	CodeDelta int64
}

// ReportData is the in-memory aggregate produced by CollectReportData. The
// HTML template consumes one of these values per report run.
type ReportData struct {
	// Metadata
	RepoName     string
	GeneratedAt  time.Time
	SccVersion   string
	Duration     time.Duration
	GitAvailable bool

	// Default rollup (always present)
	Summary []LanguageSummary
	Totals  Totals

	// Optional analyses — nil/empty if skipped or unavailable.
	ULOC             *ULOCResult
	LineLength       *LineLengthResult
	Hotspots         *HotspotsResult
	Coupling         *CouplingResult
	Authors          *AuthorsResult
	LanguageTimeline *LangTimelineResult
	AuthorTimeline   *AuthorTimelineResult
	Files            []*FileJob

	// Cost
	Cocomo *CocomoResult
	Locomo *LocomoResult

	// Rendered share-card SVG (data: URL safe). Populated by RenderReport
	// before the main template runs so it can be embedded as og:image.
	CardSVG template.HTML
}

// reportFlagState snapshots the package-level flag vars CollectReportData
// flips on entry so they can be restored on exit.
//
// scc's analysis modes (ULOC, line-length, per-file table) are gated by
// process-wide globals. The report mode flips them on inside a single
// invocation; we snapshot and restore via defer so panics, errors, or
// in-process re-entrancy don't leak state into a later scc call.
type reportFlagState struct {
	UlocMode         bool
	MaxMean          bool
	Files            bool
	CouplingWeighted bool
}

func saveReportFlags() reportFlagState {
	return reportFlagState{
		UlocMode:         UlocMode,
		MaxMean:          MaxMean,
		Files:            Files,
		CouplingWeighted: CouplingWeighted,
	}
}

func (s reportFlagState) restore() {
	UlocMode = s.UlocMode
	MaxMean = s.MaxMean
	Files = s.Files
	CouplingWeighted = s.CouplingWeighted
}

// CollectReportData orchestrates the full scc analysis surface for one
// report. It walks the tree once for default counts, runs the git-history
// observers (when git is available), computes cost estimates, and returns a
// ReportData ready for HTML templating.
//
// IMPORTANT: this function mutates the package-level analysis flags
// (UlocMode, MaxMean, Files) while it runs. The previous values are
// snapshotted and restored via defer, but callers should not assume the
// flags retain their on-entry values during the call.
func CollectReportData(path string) (ReportData, error) {
	start := time.Now()

	saved := saveReportFlags()
	defer saved.restore()

	if !ReportSkipped("uloc") {
		UlocMode = true
	}
	if !ReportSkipped("linelength") {
		MaxMean = true
	}
	if !ReportSkipped("files") {
		Files = true
	}

	// Reset the package-level ULOC accumulators so repeated in-process
	// invocations don't see stale data from an earlier walk.
	ulocMutex.Lock()
	ulocGlobalCount = map[string]struct{}{}
	ulocLanguageCount = map[string]map[string]struct{}{}
	ulocMutex.Unlock()

	gitAvailable := detectGit(path)

	data := ReportData{
		GeneratedAt:  time.Now().UTC(),
		SccVersion:   Version,
		GitAvailable: gitAvailable,
		RepoName:     detectRepoName(path),
	}

	files, summary, totals, err := walkAndAggregate(path)
	if err != nil {
		return ReportData{}, err
	}
	data.Files = files
	data.Summary = summary
	data.Totals = totals

	if !ReportSkipped("uloc") {
		data.ULOC = snapshotULOC(totals.Lines)
	}

	if !ReportSkipped("linelength") {
		data.LineLength = bucketLineLengths(files)
	}

	// The report always shows raw co-change coupling, never the opt-in
	// complexity-weighted ranking. saved.restore() puts the flag back on exit.
	CouplingWeighted = false

	if gitAvailable {
		if !ReportSkipped("hotspots") {
			obs := newHotspotsObserver()
			if window, err := runHistory(path, obs); err == nil {
				data.Hotspots = hotspotsResultFromObserver(obs, window)
			} else {
				printWarnF("report: hotspots observer failed: %s", err)
			}
		}
		if !ReportSkipped("coupling") {
			obs := newCouplingObserver()
			if window, err := runHistory(path, obs); err == nil {
				data.Coupling = couplingResultFromObserver(obs, window)
			} else {
				printWarnF("report: coupling observer failed: %s", err)
			}
		}
		if !ReportSkipped("authors") {
			obs := newHistoryAuthorsObserver()
			if window, err := runHistory(path, obs); err == nil {
				data.Authors = authorsResultFromObserver(obs, window)
			} else {
				printWarnF("report: authors observer failed: %s", err)
			}
		}
		if !ReportSkipped("timeline") {
			lObs := newHistoryLanguagesObserver(HistoryBuckets)
			if window, err := runHistory(path, lObs); err == nil {
				data.LanguageTimeline = languageTimelineResultFromObserver(lObs, window)
			} else {
				printWarnF("report: language timeline observer failed: %s", err)
			}
			aObs := newHistoryAuthorTimelineObserver(HistoryBuckets)
			if window, err := runHistory(path, aObs); err == nil {
				data.AuthorTimeline = authorTimelineResultFromObserver(aObs, window)
			} else {
				printWarnF("report: author timeline observer failed: %s", err)
			}
		}
	}

	if !Cocomo && !ReportSkipped("cocomo") {
		c := computeCocomo(totals.Code)
		data.Cocomo = &c
	}
	if !ReportSkipped("locomo") {
		l := computeLocomo(totals.Code, totals.Complexity)
		data.Locomo = &l
	}

	data.Duration = time.Since(start)
	return data, nil
}

// detectGit returns true if the path (or any parent) contains a git working
// directory. Uses go-git's PlainOpenWithOptions with DetectDotGit so callers
// can pass a subdirectory of a repo. Cached behaviour is not needed here —
// this is called once at the start of CollectReportData.
func detectGit(path string) bool {
	_, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	return err == nil
}

// detectRepoName implements the resolution chain from spec 05:
//  1. ReportTitle (set from --report-title) if non-empty.
//  2. Last path segment of `git config --get remote.origin.url` (strip `.git`).
//  3. Basename of the analysed path.
//  4. "scc report" fallback.
func detectRepoName(path string) string {
	if ReportTitle != "" {
		return ReportTitle
	}
	if name := remoteOriginName(path); name != "" {
		return name
	}
	abs, err := filepath.Abs(path)
	if err == nil && abs != "" {
		base := filepath.Base(abs)
		if base != "" && base != "." && base != string(filepath.Separator) {
			return base
		}
	}
	return "scc report"
}

// remoteOriginName runs `git config --get remote.origin.url` inside path and
// returns the last segment of the URL with a trailing `.git` stripped. Empty
// when git is unavailable, the command fails, or the remote isn't set.
func remoteOriginName(path string) string {
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	if url == "" {
		return ""
	}
	// Strip transport prefix (https://, git@host:) and trailing slash.
	url = strings.TrimSuffix(url, "/")
	// Take last path-or-colon segment.
	for _, sep := range []string{"/", ":"} {
		if idx := strings.LastIndex(url, sep); idx >= 0 {
			url = url[idx+1:]
		}
	}
	url = strings.TrimSuffix(url, ".git")
	return url
}

// walkAndAggregate runs scc's standard file walker against path, drains the
// resulting FileJob channel once, and tees the results into the language
// rollup and a flat per-file slice. Reuses aggregateLanguageSummary by
// feeding it the same FileJobs through a buffered channel.
func walkAndAggregate(path string) ([]*FileJob, []LanguageSummary, Totals, error) {
	if path == "" {
		path = "."
	}

	fpath := filepath.Clean(path)
	info, err := os.Stat(fpath)
	if err != nil {
		return nil, nil, Totals{}, fmt.Errorf("file or directory could not be read: %s", fpath)
	}

	dirPaths := []string{}
	filePaths := []string{}
	if info.IsDir() {
		dirPaths = append(dirPaths, fpath)
	} else {
		filePaths = append(filePaths, fpath)
	}

	ctx := processorContext{remap: newRemapConfig(RemapAll, RemapUnknown)}

	potentialFilesQueue := make(chan *gocodewalker.File, FileListQueueSize)
	fileListQueue := make(chan *FileJob, FileListQueueSize)
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)

	if len(dirPaths) > 0 {
		fileWalker := gocodewalker.NewParallelFileWalker(dirPaths, potentialFilesQueue)
		fileWalker.SetErrorHandler(func(e error) bool {
			printError(e.Error())
			return true
		})
		fileWalker.IgnoreGitIgnore = GitIgnore
		fileWalker.IgnoreIgnoreFile = Ignore
		fileWalker.IgnoreGitModules = GitModuleIgnore
		fileWalker.IncludeHidden = true
		fileWalker.ExcludeDirectory = PathDenyList
		fileWalker.SetConcurrency(DirectoryWalkerJobWorkers)
		if !SccIgnore {
			fileWalker.CustomIgnore = []string{".sccignore"}
		}
		fileWalker.CustomIgnoreFiles = IgnoreFiles

		var excludePathRegexes []*regexp.Regexp
		for _, exclude := range Exclude {
			re, err := regexp.Compile(exclude)
			if err == nil {
				fileWalker.ExcludeFilenameRegex = append(fileWalker.ExcludeFilenameRegex, re)
				fileWalker.ExcludeDirectoryRegex = append(fileWalker.ExcludeDirectoryRegex, re)
				excludePathRegexes = append(excludePathRegexes, re)
			} else {
				printError(err.Error())
			}
		}

		go func() {
			if err := fileWalker.Start(); err != nil {
				printError(err.Error())
			}
		}()

		go func() {
			for fi := range potentialFilesQueue {
				shouldExclude := false
				for _, re := range excludePathRegexes {
					if re.MatchString(fi.Location) {
						shouldExclude = true
						break
					}
				}
				if shouldExclude {
					continue
				}
				fileInfo, err := os.Lstat(fi.Location)
				if err != nil {
					continue
				}
				if !fileInfo.IsDir() {
					if job := newFileJob(fi.Location, fi.Filename, fileInfo); job != nil {
						fileListQueue <- job
					}
				}
			}
			close(fileListQueue)
		}()
	} else {
		go func() {
			for _, f := range filePaths {
				fileInfo, err := os.Lstat(f)
				if err != nil {
					continue
				}
				if job := newFileJob(f, f, fileInfo); job != nil {
					fileListQueue <- job
				}
			}
			close(fileListQueue)
		}()
	}

	go ctx.fileProcessorWorker(fileListQueue, fileSummaryJobQueue)

	// Tee: as each FileJob arrives, append to the flat slice and forward to
	// a buffered channel that aggregateLanguageSummary drains. We forward
	// synchronously so totals/files always see the same set.
	aggregateInput := make(chan *FileJob, FileSummaryJobQueueSize)
	var (
		files  []*FileJob
		totals Totals
		mu     sync.Mutex
	)

	go func() {
		for job := range fileSummaryJobQueue {
			mu.Lock()
			files = append(files, job)
			totals.Files++
			totals.Lines += job.Lines
			totals.Code += job.Code
			totals.Comment += job.Comment
			totals.Blank += job.Blank
			totals.Complexity += job.Complexity
			totals.Bytes += job.Bytes
			mu.Unlock()
			aggregateInput <- job
		}
		close(aggregateInput)
	}()

	summary := aggregateLanguageSummary(aggregateInput)
	summary = sortLanguageSummary(summary)

	// Ensure deterministic ordering of the flat Files slice — the worker
	// pool can interleave file emissions.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Location < files[j].Location
	})

	return files, summary, totals, nil
}

// snapshotULOC converts the package-level ULOC maps into a sorted slice so
// the template can range deterministically. totalLines drives the DRYness
// number — unique lines / total lines, capped at 1.0.
func snapshotULOC(totalLines int64) *ULOCResult {
	ulocMutex.Lock()
	defer ulocMutex.Unlock()

	res := &ULOCResult{
		Global:     len(ulocGlobalCount),
		TotalLines: totalLines,
	}
	if totalLines > 0 {
		res.Dryness = float64(res.Global) / float64(totalLines)
	}

	res.PerLanguage = make([]ULOCLanguage, 0, len(ulocLanguageCount))
	for lang, set := range ulocLanguageCount {
		res.PerLanguage = append(res.PerLanguage, ULOCLanguage{Language: lang, ULOC: len(set)})
	}
	sort.Slice(res.PerLanguage, func(i, j int) bool {
		if res.PerLanguage[i].ULOC != res.PerLanguage[j].ULOC {
			return res.PerLanguage[i].ULOC > res.PerLanguage[j].ULOC
		}
		return res.PerLanguage[i].Language < res.PerLanguage[j].Language
	})

	return res
}

// lineLengthBucketEdges defines the histogram bins used in the report — six
// 20-wide bins plus an open-ended tail.
var lineLengthBucketEdges = []struct {
	start, end int
	label      string
}{
	{0, 20, "0–20"},
	{20, 40, "20–40"},
	{40, 60, "40–60"},
	{60, 80, "60–80"},
	{80, 100, "80–100"},
	{100, 120, "100–120"},
	{120, 0, "120+"},
}

// lineLengthOutlierCount is the maximum number of longest-line outliers
// surfaced in the report. The tabular formatter only shows top-N; the
// HTML report has more vertical room so we collect a slightly larger set.
const lineLengthOutlierCount = 10

// bucketLineLengths walks every file's per-line lengths into the histogram
// buckets and tracks mean / max / longest-N outliers. Returns nil if no file
// had per-line length data (e.g. MaxMean was off everywhere).
func bucketLineLengths(files []*FileJob) *LineLengthResult {
	res := &LineLengthResult{}
	res.Buckets = make([]LineLengthBucket, len(lineLengthBucketEdges))
	for i, e := range lineLengthBucketEdges {
		res.Buckets[i] = LineLengthBucket{Start: e.start, End: e.end, Label: e.label}
	}

	type outlier struct {
		file, lang string
		length     int
	}
	var (
		total     int64
		count     int64
		maxLength int
		outliers  []outlier
	)

	for _, fj := range files {
		fileMax := 0
		for _, ll := range fj.LineLength {
			count++
			total += int64(ll)
			if ll > maxLength {
				maxLength = ll
			}
			if ll > fileMax {
				fileMax = ll
			}
			for i, edge := range lineLengthBucketEdges {
				if edge.end == 0 {
					if ll >= edge.start {
						res.Buckets[i].Count++
						break
					}
				} else if ll >= edge.start && ll < edge.end {
					res.Buckets[i].Count++
					break
				}
			}
		}
		if fileMax > 0 {
			outliers = append(outliers, outlier{
				file:   fj.Location,
				lang:   fj.Language,
				length: fileMax,
			})
		}
	}

	if count == 0 {
		return nil
	}

	res.TotalLines = count
	res.Mean = float64(total) / float64(count)
	res.Max = maxLength

	sort.Slice(outliers, func(i, j int) bool {
		if outliers[i].length != outliers[j].length {
			return outliers[i].length > outliers[j].length
		}
		return outliers[i].file < outliers[j].file
	})
	if len(outliers) > lineLengthOutlierCount {
		outliers = outliers[:lineLengthOutlierCount]
	}
	res.Outliers = make([]LineLengthOutlier, 0, len(outliers))
	for _, o := range outliers {
		res.Outliers = append(res.Outliers, LineLengthOutlier{
			File:       o.file,
			Language:   o.lang,
			LineLength: o.length,
		})
	}
	return res
}

func hotspotsResultFromObserver(o *hotspotsObserver, window HistoryWindow) *HotspotsResult {
	res := &HotspotsResult{
		Window:    window,
		TotalRaw:  o.totalRaw,
		Available: true,
	}
	res.Records = make([]HotspotRow, 0, len(o.records))
	for _, r := range o.records {
		res.Records = append(res.Records, HotspotRow{
			File:         r.File,
			Language:     r.Language,
			Complexity:   r.Complexity,
			Commits:      r.Commits,
			LinesChanged: r.LinesChanged,
			Authors:      len(r.Authors),
			CodeChurn:    r.CodeChurn,
			CommentChurn: r.CommentChurn,
			Score:        r.Score,
		})
	}
	return res
}

func couplingResultFromObserver(o *couplingObserver, window HistoryWindow) *CouplingResult {
	res := &CouplingResult{
		Window:     window,
		TotalPairs: o.totalPairs,
		Available:  true,
	}
	res.Pairs = make([]CouplingPairRow, 0, len(o.pairs))
	for _, p := range o.pairs {
		res.Pairs = append(res.Pairs, CouplingPairRow{
			FileA:  p.A,
			FileB:  p.B,
			Shared: p.Shared,
			Degree: p.Degree(),
		})
	}
	return res
}

func authorsResultFromObserver(o *historyAuthorsObserver, window HistoryWindow) *AuthorsResult {
	res := &AuthorsResult{
		Window:       window,
		BusFactor:    o.busFactor,
		BusAuthors:   append([]string(nil), o.busAuthors...),
		BusCovered:   o.busCovered,
		InWindowCode: o.inWindowCode,
	}
	res.Rows = make([]AuthorRow, 0, len(o.rows))
	for _, r := range o.rows {
		res.Rows = append(res.Rows, AuthorRow{
			Name:            r.Name,
			Email:           r.Email,
			Code:            r.Code,
			Comment:         r.Comment,
			Complexity:      r.Complexity,
			Files:           r.Files,
			OwnsPercent:     r.OwnsPercent,
			InWindowPercent: r.InWindowPercent,
			LastCommit:      r.LastCommit,
			Sentinel:        r.Sentinel,
		})
	}
	return res
}

func languageTimelineResultFromObserver(o *historyLanguagesObserver, window HistoryWindow) *LangTimelineResult {
	res := &LangTimelineResult{
		Window:  window,
		Bucket:  o.bucket,
		Buckets: o.bucket.N,
	}
	res.Rows = make([]LangTimelineRow, 0, len(o.rows))
	for _, r := range o.rows {
		row := LangTimelineRow{
			Language:      r.Language,
			StartingLines: r.StartingLines,
			CodeNow:       r.CodeNow,
			Change:        r.Change,
			SharePercent:  r.SharePercent,
			Deltas:        append([]int64(nil), r.Deltas...),
			Trajectory:    append([]int64(nil), r.Trajectory...),
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}

func authorTimelineResultFromObserver(o *historyAuthorTimelineObserver, window HistoryWindow) *AuthorTimelineResult {
	res := &AuthorTimelineResult{
		Window:  window,
		Bucket:  o.bucket,
		Buckets: o.bucket.N,
	}
	res.Rows = make([]AuthorTimelineRow, 0, len(o.rows))
	for _, r := range o.rows {
		row := AuthorTimelineRow{
			Name:         r.Name,
			Email:        r.Email,
			TotalCommits: r.TotalCommits,
			CodeDelta:    r.CodeDelta,
			Series:       make([]AuthorTimelineBucket, len(r.Series)),
		}
		for i, b := range r.Series {
			row.Series[i] = AuthorTimelineBucket{
				Commits:   b.Commits,
				CodeDelta: b.CodeDelta,
			}
		}
		res.Rows = append(res.Rows, row)
	}
	return res
}
