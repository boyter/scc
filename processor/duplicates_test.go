// SPDX-License-Identifier: MIT

package processor

import "testing"

// TestDuplicatesClearedBetweenRuns guards the bug where the package-level
// duplicate-detection hashes were never reset between runs. scc was historically
// a one-shot CLI so the state died with the process, but the long-lived MCP
// server calls ProcessResult once per tool call: every file in the second call
// matched a hash recorded by the first, so the result came back empty.
func TestDuplicatesClearedBetweenRuns(t *testing.T) {
	savedDuplicates := Duplicates
	savedPaths := DirFilePaths
	savedAllow, savedExclude := AllowListExtensions, ExcludeListExtensions
	defer func() {
		Duplicates = savedDuplicates
		DirFilePaths = savedPaths
		AllowListExtensions, ExcludeListExtensions = savedAllow, savedExclude
		cleanDuplicates()
	}()

	Duplicates = true
	AllowListExtensions = []string{}
	ExcludeListExtensions = []string{}

	run := func() []LanguageSummary {
		t.Helper()
		DirFilePaths = []string{"../examples/duplicates"}
		summary, err := ProcessResult()
		if err != nil {
			t.Fatalf("ProcessResult: %s", err)
		}
		return summary
	}

	first := run()
	if len(first) == 0 {
		t.Fatal("first run counted nothing, test cannot detect the leak")
	}

	second := run()
	if len(second) != len(first) {
		t.Fatalf("duplicate state leaked between runs: first run counted %d languages, second counted %d", len(first), len(second))
	}
	for i := range first {
		if first[i].Count != second[i].Count {
			t.Errorf("language %s: first run counted %d files, second counted %d", first[i].Name, first[i].Count, second[i].Count)
		}
	}
}
