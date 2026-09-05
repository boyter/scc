// SPDX-License-Identifier: MIT

package processor

import "sync"

// summaryTotals is a run folded down to what the tabular summaries print: one
// entry per language and the run wide totals.
//
// It exists so the fold can happen in more than one place. The counting workers
// each keep one of these and merge it once when they finish, which is what lets
// the default summary skip the per file channel entirely — see
// summariseInWorkers. Every other output format still folds one job at a time
// off the channel, which is the same code with keepFiles set.
type summaryTotals struct {
	// keepFiles retains the FileJob of every file counted, which only the
	// --by-file output needs. Leaving it off is what stops a whole tree of
	// jobs staying alive until the run ends.
	keepFiles bool
	// keepLineLength collects the per line lengths that --max-mean reports.
	keepLineLength bool
	// weighted fills in FileJob.WeightedComplexity, which only the --by-file
	// rows of the wide output read.
	weighted bool

	languages map[string]*LanguageSummary

	sumFiles      int64
	sumLines      int64
	sumCode       int64
	sumComment    int64
	sumBlank      int64
	sumComplexity int64
	sumCognitive  int64
	sumBytes      int64
}

func newSummaryTotals(keepFiles, keepLineLength, weighted bool) *summaryTotals {
	return &summaryTotals{
		keepFiles:      keepFiles,
		keepLineLength: keepLineLength,
		weighted:       weighted,
		languages:      map[string]*LanguageSummary{},
	}
}

// add folds one counted file into the totals.
func (t *summaryTotals) add(res *FileJob) {
	t.sumFiles++
	t.sumLines += res.Lines
	t.sumCode += res.Code
	t.sumComment += res.Comment
	t.sumBlank += res.Blank
	t.sumComplexity += res.Complexity
	t.sumCognitive += res.Cognitive
	t.sumBytes += res.Bytes

	if t.weighted {
		var weightedComplexity float64
		if res.Code != 0 {
			weightedComplexity = (float64(res.Complexity) / float64(res.Code)) * 100
		}
		res.WeightedComplexity = weightedComplexity
	}

	summary, ok := t.languages[res.Language]
	if !ok {
		summary = &LanguageSummary{Name: res.Language}
		t.languages[res.Language] = summary
	}

	summary.Lines += res.Lines
	summary.Code += res.Code
	summary.Comment += res.Comment
	summary.Blank += res.Blank
	summary.Complexity += res.Complexity
	summary.Cognitive += res.Cognitive
	summary.Count++

	if t.keepFiles {
		summary.Files = append(summary.Files, res)
	}
	if t.keepLineLength {
		summary.LineLength = append(summary.LineLength, res.LineLength...)
	}
}

// merge folds another set of totals into this one. Addition is the only
// operation, so the result does not depend on the order the workers finish in.
func (t *summaryTotals) merge(other *summaryTotals) {
	t.sumFiles += other.sumFiles
	t.sumLines += other.sumLines
	t.sumCode += other.sumCode
	t.sumComment += other.sumComment
	t.sumBlank += other.sumBlank
	t.sumComplexity += other.sumComplexity
	t.sumCognitive += other.sumCognitive
	t.sumBytes += other.sumBytes

	for name, summary := range other.languages {
		existing, ok := t.languages[name]
		if !ok {
			t.languages[name] = summary
			continue
		}

		existing.Lines += summary.Lines
		existing.Code += summary.Code
		existing.Comment += summary.Comment
		existing.Blank += summary.Blank
		existing.Complexity += summary.Complexity
		existing.Cognitive += summary.Cognitive
		existing.Count += summary.Count
		existing.Files = append(existing.Files, summary.Files...)
		existing.LineLength = append(existing.LineLength, summary.LineLength...)
	}
}

// sorted returns the languages in the order the summary prints them.
func (t *summaryTotals) sorted() []LanguageSummary {
	language := make([]LanguageSummary, 0, len(t.languages))
	for _, summary := range t.languages {
		language = append(language, *summary)
	}

	return sortLanguageSummary(language)
}

// sharedSummaryTotals is a set of totals several goroutines fold into. Each
// counting worker keeps its own unshared summaryTotals for the whole run and
// merges it here once, so a tree of a hundred thousand files costs one lock
// acquisition per worker rather than a channel send per file.
type sharedSummaryTotals struct {
	mutex  sync.Mutex
	totals *summaryTotals
}

func newSharedSummaryTotals() *sharedSummaryTotals {
	return &sharedSummaryTotals{totals: newSummaryTotals(false, false, false)}
}

func (s *sharedSummaryTotals) mergeLocked(local *summaryTotals) {
	s.mutex.Lock()
	s.totals.merge(local)
	s.mutex.Unlock()
}

// workerSummary is where Process leaves the totals its counting workers folded,
// for the summary that follows to pick up. It is handed over rather than passed
// as an argument because the format functions are reached through a switch on
// several package level flags and are called directly by tests, none of which
// know about it. collectSummaryTotals takes it, so a set of totals is read
// exactly once by the run that produced it and can never be seen by a later
// one.
//
// Only the goroutine running Process touches this variable. The workers share
// the sharedSummaryTotals it points at, under that struct's own mutex.
var workerSummary *sharedSummaryTotals

func takeWorkerSummary() *sharedSummaryTotals {
	taken := workerSummary
	workerSummary = nil
	return taken
}

// collectSummaryTotals folds everything the summary needs to print. It always
// drains input, which both handles the formats that fold job by job and, where
// the workers folded the run themselves, waits on the channel close that says
// every worker has merged its share.
func collectSummaryTotals(input chan *FileJob, keepFiles, keepLineLength, weighted bool) *summaryTotals {
	totals := newSummaryTotals(keepFiles, keepLineLength, weighted)

	for res := range input {
		totals.add(res)
	}

	if worker := takeWorkerSummary(); worker != nil {
		totals.merge(worker.totals)
	}

	return totals
}
