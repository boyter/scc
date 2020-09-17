package processor

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	glang "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
	"gopkg.in/yaml.v2"
)

var tabularShortBreak = "───────────────────────────────────────────────────────────────────────────────\n"
var tabularShortBreakCi = "-------------------------------------------------------------------------------\n"
var tabularShortFormatHead = "%-20s %9s %9s %8s %9s %8s %10s\n"
var tabularShortFormatBody = "%-20s %9d %9d %8d %9d %8d %10d\n"
var tabularShortFormatFile = "%-30s %9d %8d %9d %8d %10d\n"
var shortFormatFileTrucate = 29
var shortNameTruncate = 20

var tabularShortFormatHeadNoComplexity = "%-22s %11s %11s %10s %11s %9s\n"
var tabularShortFormatBodyNoComplexity = "%-22s %11d %11d %10d %11d %9d\n"
var tabularShortFormatFileNoComplexity = "%-34s %11d %10d %11d %9d\n"
var longNameTruncate = 22

var tabularWideBreak = "─────────────────────────────────────────────────────────────────────────────────────────────────────────────\n"
var tabularWideBreakCi = "-------------------------------------------------------------------------------------------------------------\n"
var tabularWideFormatHead = "%-33s %9s %9s %8s %9s %8s %10s %16s\n"
var tabularWideFormatBody = "%-33s %9d %9d %8d %9d %8d %10d %16.2f\n"
var tabularWideFormatFile = "%-43s %9d %8d %9d %8d %10d %16.2f\n"
var wideFormatFileTrucate = 42

func sortSummaryFiles(summary *LanguageSummary) {
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "languages":
		sort.Slice(summary.Files, func(i, j int) bool {
			return strings.Compare(summary.Files[i].Location, summary.Files[j].Location) < 0
		})
	case SortBy == "line" || SortBy == "lines":
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Lines > summary.Files[j].Lines
		})
	case SortBy == "blank" || SortBy == "blanks":
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Blank > summary.Files[j].Blank
		})
	case SortBy == "code" || SortBy == "codes":
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Code > summary.Files[j].Code
		})
	case SortBy == "comment" || SortBy == "comments":
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Comment > summary.Files[j].Comment
		})
	case SortBy == "complexity" || SortBy == "complexitys":
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Complexity > summary.Files[j].Complexity
		})
	default:
		sort.Slice(summary.Files, func(i, j int) bool {
			return summary.Files[i].Lines > summary.Files[j].Lines
		})
	}
}

// LanguageSummary to generate output like cloc
type languageSummaryCloc struct {
	Name    string `yaml:"name"`
	Code    int64  `yaml:"code"`
	Comment int64  `yaml:"comment"`
	Blank   int64  `yaml:"blank"`
	Count   int64  `yaml:"nFiles"`
}

type summaryStruct struct {
	Code    int64 `yaml:"code"`
	Comment int64 `yaml:"comment"`
	Blank   int64 `yaml:"blank"`
	Count   int64 `yaml:"nFiles"`
}

type headerStruct struct {
	Url            string  `yaml:"url"`
	Version        string  `yaml:"version"`
	ElapsedSeconds float64 `yaml:"elapsed_seconds"`
	NFiles         int64   `yaml:"n_files"`
	NLines         int64   `yaml:"n_lines"`
	FilesPerSecond float64 `yaml:"files_per_second"`
	LinesPerSecond float64 `yaml:"lines_per_second"`
}

type languageReportStart struct {
	Header headerStruct
}

type languageReportEnd struct {
	Sum summaryStruct `yaml:"SUM"`
}

func getTabularShortBreak() string {
	if Ci {
		return tabularShortBreakCi
	}

	return tabularShortBreak
}

func getTabularWideBreak() string {
	if Ci {
		return tabularWideBreakCi
	}

	return tabularWideBreak
}

func toClocYAML(input chan *FileJob) string {
	startTime := makeTimestampMilli()

	languages := map[string]languageSummaryCloc{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity int64 = 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity

		_, ok := languages[res.Language]

		if !ok {
			languages[res.Language] = languageSummaryCloc{
				Name:    res.Language,
				Code:    res.Code,
				Comment: res.Comment,
				Blank:   res.Blank,
				Count:   1,
			}
		} else {
			tmp := languages[res.Language]

			languages[res.Language] = languageSummaryCloc{
				Name:    res.Language,
				Code:    tmp.Code + res.Code,
				Comment: tmp.Comment + res.Comment,
				Blank:   tmp.Blank + res.Blank,
				Count:   tmp.Count + 1,
			}
		}
	}

	es := float64(makeTimestampMilli()-startTimeMilli) * float64(0.001)

	header := headerStruct{
		Url:            "https://github.com/boyter/scc/",
		Version:        Version,
		NFiles:         sumFiles,
		NLines:         sumLines,
		ElapsedSeconds: es,
		FilesPerSecond: float64(float64(sumFiles) / es),
		LinesPerSecond: float64(float64(sumLines) / es),
	}
	summary := summaryStruct{
		Blank:   sumBlank,
		Comment: sumComment,
		Code:    sumCode,
		Count:   sumFiles,
	}
	reportStart := languageReportStart{
		Header: header,
	}
	reportEnd := languageReportEnd{
		Sum: summary,
	}

	reportYaml, _ := yaml.Marshal(reportStart)
	sumYaml, _ := yaml.Marshal(reportEnd)
	languageYaml, _ := yaml.Marshal(languages)
	yamlString := "# https://github.com/boyter/scc/\n" + string(reportYaml) + string(languageYaml) + string(sumYaml)

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return yamlString
}

func toJSON(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	languages := map[string]LanguageSummary{}

	for res := range input {
		_, ok := languages[res.Language]

		if !ok {
			files := []*FileJob{}
			if Files {
				files = append(files, res)
			}

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
				Files:      files,
				Bytes:      res.Bytes,
			}
		} else {
			tmp := languages[res.Language]
			files := tmp.Files
			if Files {
				files = append(files, res)
			}

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
				Files:      files,
				Bytes:      res.Bytes + tmp.Bytes,
			}
		}
	}

	language := []LanguageSummary{}
	for _, summary := range languages {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	jsonString, _ := json.Marshal(language)

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return string(jsonString)
}

func toCSV(input chan *FileJob) string {
	records := [][]string{{
		"Language",
		"Location",
		"Filename",
		"Lines",
		"Code",
		"Comments",
		"Blanks",
		"Complexity",
		"Bytes"},
	}

	for result := range input {
		records = append(records, []string{
			result.Language,
			result.Location,
			result.Filename,
			fmt.Sprint(result.Lines),
			fmt.Sprint(result.Code),
			fmt.Sprint(result.Comment),
			fmt.Sprint(result.Blank),
			fmt.Sprint(result.Complexity),
			fmt.Sprint(result.Bytes)})
	}

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	w.WriteAll(records)
	w.Flush()

	return b.String()
}

func toHtml(input chan *FileJob) string {
	return `<html lang="en"><head><meta charset="utf-8" /><title>scc html output</title><style>table { border-collapse: collapse; }td, th { border: 1px solid #999; padding: 0.5rem; text-align: left;}</style></head><body>` +
		toHtmlTable(input) +
		`</body></html>`
}

func toHtmlTable(input chan *FileJob) string {
	languages := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumBytes += res.Bytes

		_, ok := languages[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
				Files:      files,
				Bytes:      res.Bytes,
			}
		} else {
			tmp := languages[res.Language]
			files := append(tmp.Files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
				Files:      files,
				Bytes:      tmp.Bytes + res.Bytes,
			}
		}
	}

	language := []LanguageSummary{}
	for _, summary := range languages {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	var str strings.Builder

	str.WriteString(`<table id="scc-table">
	<thead><tr>
		<th>Language</th>
		<th>Files</th>
		<th>Lines</th>
		<th>Blank</th>
		<th>Comment</th>
		<th>Code</th>
		<th>Complexity</th>
		<th>Bytes</th>
	</tr></thead>
	<tbody>`)

	for _, r := range language {
		str.WriteString(fmt.Sprintf(`<tr>
		<th>%s</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
	</tr>`, r.Name, len(r.Files), r.Lines, r.Blank, r.Comment, r.Code, r.Complexity, r.Bytes))

		if Files {
			sortSummaryFiles(&r)

			for _, res := range r.Files {
				str.WriteString(fmt.Sprintf(`<tr>
		<td>%s</td>
		<td></td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
	    <td>%d</td>
	</tr>`, res.Location, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.Bytes))
			}
		}

	}

	str.WriteString(fmt.Sprintf(`</tbody>
	<tfoot><tr>
		<th>Total</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
    	<th>%d</th>
	</tr></tfoot>
	</table>`, sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumBytes))

	return str.String()
}

func toSqlInsert(input chan *FileJob) string {
	var str strings.Builder
	projectName := SQLProject
	if projectName == "" {
		projectName = strings.Join(DirFilePaths, ",")
	}

	str.WriteString("\nbegin transaction;")
	count := 0
	for res := range input {
		count++

		dir, _ := filepath.Split(res.Location)

		str.WriteString(fmt.Sprintf("\ninsert into t values('%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d);",
			projectName, res.Language, res.Location, dir, res.Filename, res.Bytes, res.Blank, res.Comment, res.Code, res.Complexity))

		// every 1000 files commit and start a new transaction to avoid overloading
		if count == 1000 {
			str.WriteString("\ncommit;")
			str.WriteString("\nbegin transaction;")
			count = 0
		}
	}
	if count != 1000 {
		str.WriteString("\ncommit;")
	}

	currentTime := time.Now()
	es := float64(makeTimestampMilli()-startTimeMilli) * 0.001
	str.WriteString("\nbegin transaction;")
	str.WriteString(fmt.Sprintf("\ninsert into metadata values('%s', '%s', %f);", currentTime.Format("2006-01-02 15:04:05"), projectName, es))
	str.WriteString("\ncommit;")

	return str.String()
}

func toSql(input chan *FileJob) string {
	var str strings.Builder

	str.WriteString(`create table metadata (   -- github.com/boyter/scc v ` + Version + `
             timestamp text,
             Project   text,
             elapsed_s real);
create table t        (
             Project       text   ,
             Language      text   ,
             File          text   ,
             File_dirname  text   ,
             File_basename text   ,
             nByte         integer,
             nBlank        integer,
             nComment      integer,
             nCode         integer,
             nComplexity   integer   );`)

	str.WriteString(toSqlInsert(input))
	return str.String()
}

func fileSummarize(input chan *FileJob) string {
	if FormatMulti != "" {
		return fileSummarizeMulti(input)
	}

	switch {
	case More || strings.ToLower(Format) == "wide":
		return fileSummarizeLong(input)
	case strings.ToLower(Format) == "json":
		return toJSON(input)
	case strings.ToLower(Format) == "cloc-yaml" || strings.ToLower(Format) == "cloc-yml":
		return toClocYAML(input)
	case strings.ToLower(Format) == "csv":
		return toCSV(input)
	case strings.ToLower(Format) == "html":
		return toHtml(input)
	case strings.ToLower(Format) == "html-table":
		return toHtmlTable(input)
	case strings.ToLower(Format) == "sql":
		return toSql(input)
	case strings.ToLower(Format) == "sql-insert":
		return toSqlInsert(input)
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
	for _, s := range strings.Split(FormatMulti, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {
			i := make(chan *FileJob, len(results))

			for _, r := range results {
				i <- r
			}
			close(i)

			var val = ""

			switch strings.ToLower(t[0]) {
			case "tabular":
				val = fileSummarizeShort(i)
			case "wide":
				val = fileSummarizeLong(i)
			case "json":
				val = toJSON(i)
			case "cloc-yaml":
				val = toClocYAML(i)
			case "cloc-yml":
				val = toClocYAML(i)
			case "csv":
				val = toCSV(i)
			case "html":
				val = toHtml(i)
			case "html-table":
				val = toHtmlTable(i)
			case "sql":
				val = toSql(i)
			case "sql-insert":
				val = toSqlInsert(i)
			}

			if t[1] == "stdout" {
				str.WriteString(val)
				str.WriteString("\n")
			} else {
				_ = ioutil.WriteFile(t[1], []byte(val), 0600)
			}

		}
	}

	return str.String()
}

func fileSummarizeLong(input chan *FileJob) string {
	var str strings.Builder

	str.WriteString(getTabularWideBreak())
	str.WriteString(fmt.Sprintf(tabularWideFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity", "Complexity/Lines"))

	if !Files {
		str.WriteString(getTabularWideBreak())
	}

	languages := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity int64 = 0, 0, 0, 0, 0, 0
	var sumWeightedComplexity float64

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity

		var weightedComplexity float64
		if res.Code != 0 {
			weightedComplexity = (float64(res.Complexity) / float64(res.Code)) * 100
		}
		res.WeightedComplexity = weightedComplexity
		sumWeightedComplexity += weightedComplexity

		_, ok := languages[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

			languages[res.Language] = LanguageSummary{
				Name:               res.Language,
				Lines:              res.Lines,
				Code:               res.Code,
				Comment:            res.Comment,
				Blank:              res.Blank,
				Complexity:         res.Complexity,
				Count:              1,
				WeightedComplexity: weightedComplexity,
				Files:              files,
			}
		} else {
			tmp := languages[res.Language]
			files := append(tmp.Files, res)

			languages[res.Language] = LanguageSummary{
				Name:               res.Language,
				Lines:              tmp.Lines + res.Lines,
				Code:               tmp.Code + res.Code,
				Comment:            tmp.Comment + res.Comment,
				Blank:              tmp.Blank + res.Blank,
				Complexity:         tmp.Complexity + res.Complexity,
				Count:              tmp.Count + 1,
				WeightedComplexity: tmp.WeightedComplexity + weightedComplexity,
				Files:              files,
			}
		}
	}

	language := []LanguageSummary{}
	for _, summary := range languages {
		language = append(language, summary)
	}

	// Cater for the common case of adding plural even for those options that don't make sense
	// as its quite common for those who English is not a first language to make a simple mistake
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "languages":
		sort.Slice(language, func(i, j int) bool {
			return strings.Compare(language[i].Name, language[j].Name) < 0
		})
	case SortBy == "line" || SortBy == "lines":
		sort.Slice(language, func(i, j int) bool {
			return language[i].Lines > language[j].Lines
		})
	case SortBy == "blank" || SortBy == "blanks":
		sort.Slice(language, func(i, j int) bool {
			return language[i].Blank > language[j].Blank
		})
	case SortBy == "code" || SortBy == "codes":
		sort.Slice(language, func(i, j int) bool {
			return language[i].Code > language[j].Code
		})
	case SortBy == "comment" || SortBy == "comments":
		sort.Slice(language, func(i, j int) bool {
			return language[i].Comment > language[j].Comment
		})
	case SortBy == "complexity" || SortBy == "complexitys":
		sort.Slice(language, func(i, j int) bool {
			return language[i].Complexity > language[j].Complexity
		})
	default:
		sort.Slice(language, func(i, j int) bool {
			return language[i].Count > language[j].Count
		})
	}

	startTime := makeTimestampMilli()
	for _, summary := range language {
		if Files {
			str.WriteString(getTabularWideBreak())
		}

		trimmedName := summary.Name
		if len(summary.Name) > longNameTruncate {
			trimmedName = summary.Name[:longNameTruncate-1] + "…"
		}

		str.WriteString(fmt.Sprintf(tabularWideFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summary.WeightedComplexity))

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularWideBreak())

			for _, res := range summary.Files {
				tmp := res.Location

				if len(tmp) >= wideFormatFileTrucate {
					totrim := len(tmp) - wideFormatFileTrucate
					tmp = "~" + tmp[totrim:]
				}

				str.WriteString(fmt.Sprintf(tabularWideFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.WeightedComplexity))
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(getTabularWideBreak())
	str.WriteString(fmt.Sprintf(tabularWideFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumWeightedComplexity))
	str.WriteString(getTabularWideBreak())

	if !Cocomo {
		estimatedEffort := EstimateEffort(int64(sumCode))
		estimatedCost := EstimateCost(estimatedEffort, AverageWage)
		estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
		estimatedPeopleRequired := estimatedEffort / estimatedScheduleMonths

		p := gmessage.NewPrinter(glang.English)

		str.WriteString(p.Sprintf("Estimated Cost to Develop $%d\n", int64(estimatedCost)))
		str.WriteString(fmt.Sprintf("Estimated Schedule Effort %f months\n", estimatedScheduleMonths))
		if math.IsNaN(estimatedPeopleRequired) {
			str.WriteString(fmt.Sprintf("Estimated People Required 1 Grandparent\n"))
		} else {
			str.WriteString(fmt.Sprintf("Estimated People Required %f\n", estimatedPeopleRequired))
		}
		str.WriteString(getTabularWideBreak())
	}

	return str.String()
}

func fileSummarizeShort(input chan *FileJob) string {
	var str strings.Builder

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		str.WriteString(fmt.Sprintf(tabularShortFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity"))
	} else {
		str.WriteString(fmt.Sprintf(tabularShortFormatHeadNoComplexity, "Language", "Files", "Lines", "Blanks", "Comments", "Code"))
	}

	if !Files {
		str.WriteString(getTabularShortBreak())
	}

	languages := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumBytes += res.Bytes

		_, ok := languages[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
				Files:      files,
			}
		} else {
			tmp := languages[res.Language]
			files := append(tmp.Files, res)

			languages[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
				Files:      files,
			}
		}
	}

	language := []LanguageSummary{}
	for _, summary := range languages {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	startTime := makeTimestampMilli()
	for _, summary := range language {
		if Files {
			str.WriteString(getTabularShortBreak())
		}

		trimmedName := summary.Name
		trimmedName = trimNameShort(summary, trimmedName)

		if !Complexity {
			str.WriteString(fmt.Sprintf(tabularShortFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity))
		} else {
			str.WriteString(fmt.Sprintf(tabularShortFormatBodyNoComplexity, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code))
		}

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularShortBreak())

			for _, res := range summary.Files {
				tmp := res.Location

				if len(tmp) >= shortFormatFileTrucate {
					totrim := len(tmp) - shortFormatFileTrucate
					tmp = "~" + tmp[totrim:]
				}

				if !Complexity {
					str.WriteString(fmt.Sprintf(tabularShortFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity))
				} else {
					str.WriteString(fmt.Sprintf(tabularShortFormatFileNoComplexity, tmp, res.Lines, res.Blank, res.Comment, res.Code))
				}
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		str.WriteString(fmt.Sprintf(tabularShortFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity))
	} else {
		str.WriteString(fmt.Sprintf(tabularShortFormatBodyNoComplexity, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode))
	}
	str.WriteString(getTabularShortBreak())

	calculateCocomo(sumCode, &str)
	calculateSize(sumBytes, &str)
	return str.String()
}

func trimNameShort(summary LanguageSummary, trimmedName string) string {
	if len(summary.Name) > shortNameTruncate {
		trimmedName = summary.Name[:shortNameTruncate-1] + "…"
	}
	return trimmedName
}

func calculateCocomo(sumCode int64, str *strings.Builder) {
	if !Cocomo {
		estimatedEffort := EstimateEffort(int64(sumCode))
		estimatedCost := EstimateCost(estimatedEffort, AverageWage)
		estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
		estimatedPeopleRequired := estimatedEffort / estimatedScheduleMonths

		p := gmessage.NewPrinter(glang.English)

		str.WriteString(p.Sprintf("Estimated Cost to Develop $%d\n", int64(estimatedCost)))
		str.WriteString(fmt.Sprintf("Estimated Schedule Effort %f months\n", estimatedScheduleMonths))
		if math.IsNaN(estimatedPeopleRequired) {
			str.WriteString(fmt.Sprintf("Estimated People Required 1 Grandparent\n"))
		} else {
			str.WriteString(fmt.Sprintf("Estimated People Required %f\n", estimatedPeopleRequired))
		}
		str.WriteString(getTabularShortBreak())
	}
}

func calculateSize(sumBytes int64, str *strings.Builder) {
	if !Size {
		var size float64

		switch strings.ToLower(SizeUnit) {
		case "binary":
			size = float64(sumBytes) / 1_048_576
		case "mixed":
			size = float64(sumBytes) / 1_024_000
		case "xkcd-kb":
			str.WriteString("1000 bytes during leap years, 1024 otherwise\n")
			tim := time.Now()
			if isLeapYear(tim.Year()) {
				size = float64(sumBytes) / 1_000_000
			}
		case "xkcd-kelly":
			str.WriteString("compromise between 1000 and 1024 bytes\n")
			size = float64(sumBytes) / (1012 * 1012)
		case "xkcd-imaginary":
			str.WriteString("used in quantum computing\n")
			str.WriteString(fmt.Sprintf("Processed %d bytes, %s megabytes (%s)\n", sumBytes, `¯\_(ツ)_/¯`, strings.ToUpper(SizeUnit)))
		case "xkcd-intel":
			str.WriteString("calculated on pentium F.P.U.\n")
			size = float64(sumBytes) / (1023.937528 * 1023.937528)
		case "xkcd-drive":
			str.WriteString("shrinks by 4 bytes every year for marketing reasons\n")
			tim := time.Now()

			s := 908 - ((tim.Year() - 2013) * 4) // comic starts with 908 in 2013 hence hardcoded values
			s = min(s, 908)                      // just in case the clock is stupidly set

			size = float64(sumBytes) / float64(s*s)
		case "xkcd-bakers":
			str.WriteString("9 bits to the byte since you're such a good customer\n")
			size = float64(sumBytes) / (1152 * 1152)
		default:
			// SI value of 1000 bytes
			size = float64(sumBytes) / 1_000_000
			SizeUnit = "SI"
		}

		if strings.ToLower(SizeUnit) != "xkcd-imaginary" {
			str.WriteString(fmt.Sprintf("Processed %d bytes, %.3f megabytes (%s)\n", sumBytes, size, strings.ToUpper(SizeUnit)))
		}

		str.WriteString(getTabularShortBreak())
	}
}

func isLeapYear(year int) bool {
	leapFlag := false
	if year%4 == 0 {
		if year%100 == 0 {
			if year%400 == 0 {
				leapFlag = true
			} else {
				leapFlag = false
			}
		} else {
			leapFlag = true
		}
	} else {
		leapFlag = false
	}
	return leapFlag
}

func sortLanguageSummary(language []LanguageSummary) []LanguageSummary {
	// Cater for the common case of adding plural even for those options that don't make sense
	// as its quite common for those who English is not a first language to make a simple mistake
	// NB in any non name cases if the values are the same we sort by name to ensure
	// deterministic output
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "languages":
		sort.Slice(language, func(i, j int) bool {
			return strings.Compare(language[i].Name, language[j].Name) < 0
		})
	case SortBy == "line" || SortBy == "lines":
		sort.Slice(language, func(i, j int) bool {
			if language[i].Lines == language[j].Lines {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Lines > language[j].Lines
		})
	case SortBy == "blank" || SortBy == "blanks":
		sort.Slice(language, func(i, j int) bool {
			if language[i].Blank == language[j].Blank {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Blank > language[j].Blank
		})
	case SortBy == "code" || SortBy == "codes":
		sort.Slice(language, func(i, j int) bool {
			if language[i].Code == language[j].Code {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Code > language[j].Code
		})
	case SortBy == "comment" || SortBy == "comments":
		sort.Slice(language, func(i, j int) bool {
			if language[i].Comment == language[j].Comment {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Comment > language[j].Comment
		})
	case SortBy == "complexity" || SortBy == "complexitys":
		sort.Slice(language, func(i, j int) bool {
			if language[i].Complexity == language[j].Complexity {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Complexity > language[j].Complexity
		})
	default: // Files IE default falls into this category
		sort.Slice(language, func(i, j int) bool {
			if language[i].Count == language[j].Count {
				return strings.Compare(language[i].Name, language[j].Name) < 0
			}

			return language[i].Count > language[j].Count
		})
	}

	return language
}

// Get the time as standard UTC/Zulu format
func getFormattedTime() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Prints a message to stdout if flag to enable warning output is set
func printWarn(msg string) {
	if Verbose {
		fmt.Printf(" WARN %s: %s\n", getFormattedTime(), msg)
	}
}

// Prints a message to stdout if flag to enable warning output is set
func printWarnf(msg string, args ...interface{}) {
	if Verbose {
		printWarn(fmt.Sprintf(msg, args...))
	}
}

// Prints a message to stdout if flag to enable debug output is set
func printDebug(msg string) {
	if Debug {
		fmt.Printf("DEBUG %s: %s\n", getFormattedTime(), msg)
	}
}

// Prints a message to stdout if flag to enable debug output is set
func printDebugf(msg string, args ...interface{}) {
	if Debug {
		printDebug(fmt.Sprintf(msg, args...))
	}
}

// Prints a message to stdout if flag to enable trace output is set
func printTrace(msg string) {
	if Trace {
		fmt.Printf("TRACE %s: %s\n", getFormattedTime(), msg)
	}
}

// Prints a message to stdout if flag to enable trace output is set
func printTracef(msg string, args ...interface{}) {
	if Trace {
		printTrace(fmt.Sprintf(msg, args...))
	}
}

// Used when explicitly for os.exit output when crashing out
func printError(msg string) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR %s: %s\n", getFormattedTime(), msg)
}
