// SPDX-License-Identifier: MIT

package processor

import (
	"cmp"
	"fmt"
	"os"
	"slices"
	"strings"
)

var tabularShortBreak = "───────────────────────────────────────────────────────────────────────────────\n"
var tabularShortBreakCi = "-------------------------------------------------------------------------------\n"

var tabularWideBreak = "─────────────────────────────────────────────────────────────────────────────────────────────────────────────\n"
var tabularWideBreakCi = "-------------------------------------------------------------------------------------------------------------\n"

// Wider break variants matching the extra Cognitive column (11 chars wider).
var tabularWideBreakCognitive = strings.Repeat("─", 120) + "\n"
var tabularWideBreakCiCognitive = strings.Repeat("-", 120) + "\n"

// activeComplexity returns the complexity magnitude to display and sort by: the
// nesting-weighted Cognitive value when --cognitive is active, otherwise the
// cyclomatic Complexity. The default tabular "Complexity" column and the
// "complexity" sort key both follow the active metric, so --cognitive overrides
// them in place without changing the header. Machine formats (JSON/CSV/SQL/MCP)
// keep both fields distinct and do not use this.
func activeComplexity(complexity, cognitive int64) int64 {
	if Cognitive {
		return cognitive
	}
	return complexity
}

func sortSummaryFiles(summary *LanguageSummary) {
	switch SortBy {
	case "name", "names", "language", "languages", "lang", "langs":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return strings.Compare(a.Location, b.Location)
		})
	case "line", "lines":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Lines, a.Lines)
		})
	case "blank", "blanks":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Blank, a.Blank)
		})
	case "code", "codes":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Code, a.Code)
		})
	case "comment", "comments":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Comment, a.Comment)
		})
	case "complexity", "complexitys", "comp":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(activeComplexity(b.Complexity, b.Cognitive), activeComplexity(a.Complexity, a.Cognitive))
		})
	case "cognitive", "cognitives", "cog":
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Cognitive, a.Cognitive)
		})
	default:
		slices.SortFunc(summary.Files, func(a, b *FileJob) int {
			return cmp.Compare(b.Lines, a.Lines)
		})
	}
}

func getTabularShortBreak() string {
	if HBorder {
		return ""
	}

	if Ci {
		return tabularShortBreakCi
	}

	return tabularShortBreak
}

func getTabularWideBreak() string {
	if HBorder {
		return ""
	}

	if Ci {
		if Cognitive {
			return tabularWideBreakCiCognitive
		}
		return tabularWideBreakCi
	}

	if Cognitive {
		return tabularWideBreakCognitive
	}

	return tabularWideBreak
}

func fileSummarize(input chan *FileJob) string {
	if FormatMulti != "" {
		return fileSummarizeMulti(input)
	}

	switch {
	case More || strings.EqualFold(Format, "wide"):
		return fileSummarizeLong(input)
	case strings.EqualFold(Format, "json"):
		return toJSON(input)
	case strings.EqualFold(Format, "json2"):
		return toJSON2(input)
	case strings.EqualFold(Format, "cloc-yaml") || strings.EqualFold(Format, "cloc-yml"):
		return toClocYAML(input)
	case strings.EqualFold(Format, "csv"):
		return toCSV(input)
	case strings.EqualFold(Format, "csv-stream"):
		return toCSVStream(input)
	case strings.EqualFold(Format, "html"):
		return toHtml(input)
	case strings.EqualFold(Format, "html-table"):
		return toHtmlTable(input)
	case strings.EqualFold(Format, "sql"):
		return toSql(input)
	case strings.EqualFold(Format, "sql-insert"):
		return toSqlInsert(input)
	case strings.EqualFold(Format, "openmetrics"):
		return toOpenMetrics(input)
	}

	return fileSummarizeShort(input)
}

// Deals with the case of CI/CD where you might want to run with multiple outputs
// both to files and to stdout. Not the most efficient way to do it in terms of memory
// but seeing as the files are just summaries by this point it shouldn't be too bad
func fileSummarizeMulti(input chan *FileJob) string {
	// collect all the results
	var results []*FileJob
	for res := range input {
		results = append(results, res)
	}

	var str strings.Builder

	// for each output pump the results into
	for s := range strings.SplitSeq(FormatMulti, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {
			i := make(chan *FileJob, len(results))

			for _, r := range results {
				i <- r
			}
			close(i)

			var val string

			switch strings.ToLower(t[0]) {
			case "tabular":
				val = fileSummarizeShort(i)
			case "wide":
				val = fileSummarizeLong(i)
			case "json":
				val = toJSON(i)
			case "json2":
				val = toJSON2(i)
			case "cloc-yaml":
				val = toClocYAML(i)
			case "cloc-yml":
				val = toClocYAML(i)
			case "csv":
				val = toCSV(i)
			case "csv-stream":
				// special case where we want to ignore writing to stdout to disk as it's already done
				_ = toCSVStream(i)
				continue
			case "html":
				val = toHtml(i)
			case "html-table":
				val = toHtmlTable(i)
			case "sql":
				val = toSql(i)
			case "sql-insert":
				val = toSqlInsert(i)
			case "openmetrics":
				val = toOpenMetrics(i)
			}

			if t[1] == "stdout" {
				str.WriteString(val)
				str.WriteString("\n")
			} else {
				err := os.WriteFile(t[1], []byte(val), 0600)
				if err != nil {
					fmt.Printf("%s unable to be written to for format %s: %s", t[1], t[0], err)
				}
			}
		}
	}

	return str.String()
}

func aggregateLanguageSummary(input chan *FileJob) []LanguageSummary {
	langs := map[string]LanguageSummary{}

	for res := range input {
		_, ok := langs[res.Language]

		if !ok {
			files := []*FileJob{}
			if Files {
				files = append(files, res)
			}

			langs[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Cognitive:  res.Cognitive,
				Count:      1,
				Files:      files,
				Bytes:      res.Bytes,
				ULOC:       0,
			}
		} else {
			tmp := langs[res.Language]
			files := tmp.Files
			if Files {
				files = append(files, res)
			}

			langs[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Cognitive:  tmp.Cognitive + res.Cognitive,
				Count:      tmp.Count + 1,
				Files:      files,
				Bytes:      res.Bytes + tmp.Bytes,
				ULOC:       0,
			}
		}
	}

	language := make([]LanguageSummary, 0, len(langs))
	for _, summary := range langs {
		summary.ULOC = len(ulocLanguageCount[summary.Name]) // for #498
		language = append(language, summary)
	}

	return language
}

func sortLanguageSummary(language []LanguageSummary) []LanguageSummary {
	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
	// NB in any non name cases if the values are the same we sort by name to ensure
	// deterministic output
	switch SortBy {
	case "name", "names", "language", "languages", "lang", "langs":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			return strings.Compare(a.Name, b.Name)
		})
	case "line", "lines":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Lines, a.Lines); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "blank", "blanks":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Blank, a.Blank); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "code", "codes":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Code, a.Code); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "comment", "comments":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Comment, a.Comment); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "complexity", "complexitys":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(activeComplexity(b.Complexity, b.Cognitive), activeComplexity(a.Complexity, a.Cognitive)); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "cognitive", "cognitives", "cog":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Cognitive, a.Cognitive); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "byte", "bytes":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Bytes, a.Bytes); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	case "file", "files":
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Count, a.Count); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	default: // Files IE default falls into this category
		slices.SortFunc(language, func(a, b LanguageSummary) int {
			if order := cmp.Compare(b.Count, a.Count); order != 0 {
				return order
			}
			return strings.Compare(a.Name, b.Name)
		})
	}

	return language
}
