// SPDX-License-Identifier: MIT

package processor

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// validateHistoryFlags checks the global flag state for the history reports
// (--hotspots, --by-author, --timeline). Hard errors are returned and should
// abort the run; recoverable conditions are written to warnDst as a single
// line each and execution continues.
func validateHistoryFlags(warnDst io.Writer) error {
	if !Hotspots && !ByAuthor && !Timeline {
		return nil
	}

	if HistoryDepth < 0 {
		return errors.New("--depth must be >= 0 (0 means entire history)")
	}

	if Timeline && HistoryBuckets < 1 {
		return errors.New("--buckets must be >= 1")
	}

	if len(DirFilePaths) > 1 {
		_, _ = fmt.Fprintf(
			warnDst,
			"history reports run against a single repository; ignoring extra paths: %s\n",
			strings.Join(DirFilePaths[1:], ", "),
		)
	}

	ignored := collectIgnoredHistoryFlags()
	if len(ignored) > 0 {
		_, _ = fmt.Fprintf(
			warnDst,
			"history reports ignore these flags: %s (they apply to the working-tree counter only)\n",
			strings.Join(ignored, ", "),
		)
	}

	return nil
}

// collectIgnoredHistoryFlags returns the CLI flag names that were set but
// have no effect under a history report. Order matches the user-facing
// flag list in --help to keep the warning readable.
func collectIgnoredHistoryFlags() []string {
	var ignored []string

	add := func(active bool, name string) {
		if active {
			ignored = append(ignored, name)
		}
	}

	add(Files, "--by-file")
	add(UlocMode, "--uloc")
	add(Dryness, "--dryness")
	add(MaxMean, "--character")
	add(Duplicates, "--no-duplicates")
	add(MinifiedGenerated, "--min-gen")
	add(Minified, "--min")
	add(Generated, "--gen")
	add(IgnoreMinifiedGenerate, "--no-min-gen")
	add(IgnoreMinified, "--no-min")
	add(IgnoreGenerated, "--no-gen")
	add(Cocomo, "--no-cocomo")
	add(Locomo, "--locomo")
	add(CostComparison, "--cost-comparison")
	add(SLOCCountFormat, "--sloccount-format")
	add(NoLarge, "--no-large")

	sort := strings.ToLower(strings.TrimSpace(SortBy))
	if sort != "" && sort != "files" {
		ignored = append(ignored, "--sort")
	}

	return ignored
}
