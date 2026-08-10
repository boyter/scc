// SPDX-License-Identifier: MIT

package processor

import (
	"strings"
	"testing"
)

// A project whose files carry zero blank/comment/complexity lines used to make
// the tabular --percent output print NaN% for those columns, because the
// percentage divisions had no zero-total guard (unlike addLanguagePercentages
// in the JSON path). The pct helper now guards the total==0 case so each empty
// category reports 0.0% instead.

func TestPercentNoNaNOnZeroCategory(t *testing.T) {
	job := func() chan *FileJob {
		c := make(chan *FileJob, 1)
		c <- &FileJob{
			Language:  "JSON",
			Filename:  "a.json",
			Extension: "json",
			Location:  "./",
			Lines:     1,
			Code:      1,
			// Blank, Comment and Complexity left at zero on purpose.
		}
		close(c)
		return c
	}

	saved := Percent
	Percent = true
	defer func() { Percent = saved }()

	for name, fn := range map[string]func(chan *FileJob) string{
		"short": fileSummarizeShort,
		"wide":  fileSummarizeLong,
	} {
		t.Run(name, func(t *testing.T) {
			res := fn(job())
			if strings.Contains(res, "NaN") {
				t.Errorf("expected no NaN in --percent output, got:\n%s", res)
			}
			if !strings.Contains(res, "0.0%") {
				t.Errorf("expected a 0.0%% entry for the empty categories, got:\n%s", res)
			}
		})
	}
}
