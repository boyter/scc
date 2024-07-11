// SPDX-License-Identifier: MIT OR Unlicense

package processor

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"

	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
	"gopkg.in/yaml.v2"
)

var tabularShortBreak = "───────────────────────────────────────────────────────────────────────────────\n"
var tabularShortBreakCi = "-------------------------------------------------------------------------------\n"

var tabularShortFormatHead = "%-20s %9s %9s %8s %9s %8s %10s\n"
var tabularShortFormatBody = "%-20s %9d %9d %8d %9d %8d %10d\n"
var tabularShortFormatFile = "%s %9d %8d %9d %8d %10d\n"
var tabularShortFormatFileMaxMean = "MaxLine / MeanLine %11d %9d\n"
var shortFormatFileTruncate = 29
var shortNameTruncate = 20
var tabularShortUlocLanguageFormatBody = "(ULOC) %33d\n"
var tabularShortPercentLanguageFormatBody = "Percentage %18.1f%% %8.1f%% %7.1f%% %8.1f%% %7.1f%% %9.1f%%\n"
var tabularShortUlocGlobalFormatBody = "Unique Lines of Code (ULOC) %12d\n"

var tabularShortFormatHeadNoComplexity = "%-22s %11s %11s %10s %11s %9s\n"
var tabularShortFormatBodyNoComplexity = "%-22s %11d %11d %10d %11d %9d\n"
var tabularShortFormatFileNoComplexity = "%s %11d %10d %11d %9d\n"
var tabularShortFormatFileMaxMeanNoComplexity = "MaxLine / MeanLine %15d %11d\n"
var longNameTruncate = 22
var tabularShortUlocLanguageFormatBodyNoComplexity = "(ULOC) %39d\n"
var tabularShortPercentLanguageFormatBodyNoComplexity = "Percentage %22.1f%% %10.1f%% %9.1f%% %10.1f%% %8.1f%%\n"

var tabularWideBreak = "─────────────────────────────────────────────────────────────────────────────────────────────────────────────\n"
var tabularWideBreakCi = "-------------------------------------------------------------------------------------------------------------\n"
var tabularWideFormatHead = "%-33s %9s %9s %8s %9s %8s %10s %16s\n"
var tabularWideFormatBody = "%-33s %9d %9d %8d %9d %8d %10d %16.2f\n"
var tabularWideFormatFile = "%s %9d %8d %9d %8d %10d %16.2f\n"
var tabularWideFormatFileMaxMean = "MaxLine / MeanLine %24d %9d\n"
var wideFormatFileTruncate = 42
var tabularWideUlocLanguageFormatBody = "(ULOC) %46d\n"
var tabularWideUlocGlobalFormatBody = "Unique Lines of Code (ULOC) %25d\n"
var tabularWideFormatBodyPercent = "Percentage %31.1f%% %8.1f%% %7.1f%% %8.1f%% %7.1f%% %9.1f%%\n"

var openMetricsMetadata = `# TYPE scc_files count
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines count
# HELP scc_lines Number of lines.
# TYPE scc_code count
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments count
# HELP scc_comments Number of comments.
# TYPE scc_blanks count
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity count
# HELP scc_complexity Code complexity.
# TYPE scc_bytes count
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
`
var openMetricsSummaryRecordFormat = "scc_%s{language=\"%s\"} %d\n"
var openMetricsFileRecordFormat = "scc_%s{language=\"%s\",file=\"%s\"} %d\n"

func sortSummaryFiles(summary *LanguageSummary) {
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "languages" || SortBy == "lang":
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
	case SortBy == "complexity" || SortBy == "complexitys" || SortBy == "comp":
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
		return tabularWideBreakCi
	}

	return tabularWideBreak
}

func toClocYAML(input chan *FileJob) string {
	startTime := makeTimestampMilli()

	langs := map[string]languageSummaryCloc{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity int64 = 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity

		_, ok := langs[res.Language]

		if !ok {
			langs[res.Language] = languageSummaryCloc{
				Name:    res.Language,
				Code:    res.Code,
				Comment: res.Comment,
				Blank:   res.Blank,
				Count:   1,
			}
		} else {
			tmp := langs[res.Language]

			langs[res.Language] = languageSummaryCloc{
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
	languageYaml, _ := yaml.Marshal(langs)
	yamlString := "# https://github.com/boyter/scc/\n" + string(reportYaml) + string(languageYaml) + string(sumYaml)

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return yamlString
}

func toJSON(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	jsonString, _ := json.Marshal(language)

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return string(jsonString)
}

type Json2 struct {
	LanguageSummary         []LanguageSummary `json:"languageSummary"`
	EstimatedCost           float64           `json:"estimatedCost"`
	EstimatedScheduleMonths float64           `json:"estimatedScheduleMonths"`
	EstimatedPeople         float64           `json:"estimatedPeople"`
}

func toJSON2(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	var sumCode int64
	for _, l := range language {
		sumCode += l.Code
	}

	cost, schedule, people := esstimateCostScheduleMonths(sumCode)

	jsonString, _ := json.Marshal(Json2{
		LanguageSummary:         language,
		EstimatedCost:           cost,
		EstimatedScheduleMonths: schedule,
		EstimatedPeople:         people,
	})

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return string(jsonString)
}

func toCSV(input chan *FileJob) string {
	if Files {
		return toCSVFiles(input)
	}

	return toCSVSummary(input)
}

func toCSVSummary(input chan *FileJob) string {
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	records := make([][]string, 0, len(language))

	for _, result := range language {
		records = append(records, []string{
			result.Name,
			fmt.Sprint(result.Lines),
			fmt.Sprint(result.Code),
			fmt.Sprint(result.Comment),
			fmt.Sprint(result.Blank),
			fmt.Sprint(result.Complexity),
			fmt.Sprint(result.Bytes),
			fmt.Sprint(result.Count),
			fmt.Sprint(len(ulocLanguageCount[result.Name])),
		})
	}

	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "languages":
		sort.Slice(records, func(i, j int) bool {
			return strings.Compare(records[i][0], records[j][0]) < 0
		})
	case SortBy == "line" || SortBy == "lines":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][1], 10, 64)
			i2, _ := strconv.ParseInt(records[j][1], 10, 64)
			return i1 > i2
		})
	case SortBy == "blank" || SortBy == "blanks":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][4], 10, 64)
			i2, _ := strconv.ParseInt(records[j][4], 10, 64)
			return i1 > i2
		})
	case SortBy == "code" || SortBy == "codes":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][2], 10, 64)
			i2, _ := strconv.ParseInt(records[j][2], 10, 64)
			return i1 > i2
		})
	case SortBy == "comment" || SortBy == "comments":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][3], 10, 64)
			i2, _ := strconv.ParseInt(records[j][3], 10, 64)
			return i1 > i2
		})
	case SortBy == "complexity" || SortBy == "complexitys":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][5], 10, 64)
			i2, _ := strconv.ParseInt(records[j][5], 10, 64)
			return i1 > i2
		})
	case SortBy == "byte" || SortBy == "bytes":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][6], 10, 64)
			i2, _ := strconv.ParseInt(records[j][6], 10, 64)
			return i1 > i2
		})
	default:
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][1], 10, 64)
			i2, _ := strconv.ParseInt(records[j][1], 10, 64)
			return i1 > i2
		})
	}

	recordsEnd := [][]string{{
		"Language",
		"Lines",
		"Code",
		"Comments",
		"Blanks",
		"Complexity",
		"Bytes",
		"Files",
		"ULOC",
	}}

	recordsEnd = append(recordsEnd, records...)

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.WriteAll(recordsEnd)
	w.Flush()

	return b.String()
}

func toCSVFiles(input chan *FileJob) string {
	records := [][]string{}

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
			fmt.Sprint(result.Bytes),
			fmt.Sprint(result.Uloc),
		})
	}

	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
	switch {
	case SortBy == "name" || SortBy == "names":
		sort.Slice(records, func(i, j int) bool {
			return strings.Compare(records[i][2], records[j][2]) < 0
		})
	case SortBy == "language" || SortBy == "languages":
		sort.Slice(records, func(i, j int) bool {
			return strings.Compare(records[i][0], records[j][0]) < 0
		})
	case SortBy == "line" || SortBy == "lines":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][3], 10, 64)
			i2, _ := strconv.ParseInt(records[j][3], 10, 64)
			return i1 > i2
		})
	case SortBy == "blank" || SortBy == "blanks":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][6], 10, 64)
			i2, _ := strconv.ParseInt(records[j][6], 10, 64)
			return i1 > i2
		})
	case SortBy == "code" || SortBy == "codes":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][4], 10, 64)
			i2, _ := strconv.ParseInt(records[j][4], 10, 64)
			return i1 > i2
		})
	case SortBy == "comment" || SortBy == "comments":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][5], 10, 64)
			i2, _ := strconv.ParseInt(records[j][5], 10, 64)
			return i1 > i2
		})
	case SortBy == "complexity" || SortBy == "complexitys":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][7], 10, 64)
			i2, _ := strconv.ParseInt(records[j][7], 10, 64)
			return i1 > i2
		})
	case SortBy == "byte" || SortBy == "bytes":
		sort.Slice(records, func(i, j int) bool {
			i1, _ := strconv.ParseInt(records[i][8], 10, 64)
			i2, _ := strconv.ParseInt(records[j][8], 10, 64)
			return i1 > i2
		})
	default:
		sort.Slice(records, func(i, j int) bool {
			return records[i][2] > records[j][2]
		})
	}

	recordsEnd := [][]string{{
		"Language",
		"Provider",
		"Filename",
		"Lines",
		"Code",
		"Comments",
		"Blanks",
		"Complexity",
		"Bytes",
		"ULOC",
	}}

	recordsEnd = append(recordsEnd, records...)

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	_ = w.WriteAll(recordsEnd)
	w.Flush()

	return b.String()
}

func toOpenMetrics(input chan *FileJob) string {
	if Files {
		return toOpenMetricsFiles(input)
	}

	return toOpenMetricsSummary(input)
}

func toOpenMetricsSummary(input chan *FileJob) string {
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	sb := &strings.Builder{}
	sb.WriteString(openMetricsMetadata)
	for _, result := range language {
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "files", result.Name, result.Count)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "lines", result.Name, result.Lines)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "code", result.Name, result.Code)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "comments", result.Name, result.Comment)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "blanks", result.Name, result.Blank)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "complexity", result.Name, result.Complexity)
		fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "bytes", result.Name, result.Bytes)
	}
	return sb.String()
}

func toOpenMetricsFiles(input chan *FileJob) string {
	sb := &strings.Builder{}
	sb.WriteString(openMetricsMetadata)
	for file := range input {
		var filename = strings.ReplaceAll(file.Location, "\\", "\\\\")
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "lines", file.Language, filename, file.Lines)
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "code", file.Language, filename, file.Code)
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "comments", file.Language, filename, file.Comment)
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "blanks", file.Language, filename, file.Blank)
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "complexity", file.Language, filename, file.Complexity)
		fmt.Fprintf(sb, openMetricsFileRecordFormat, "bytes", file.Language, filename, file.Bytes)
	}
	sb.WriteString("# EOF")
	return sb.String()
}

// For very large repositories CSV stream can be used which prints results out as they come in
// with the express idea of lowering memory usage, see https://github.com/boyter/scc/issues/210 for
// the background on why this might be needed
func toCSVStream(input chan *FileJob) string {
	fmt.Println("Language,Provider,Filename,Lines,Code,Comments,Blanks,Complexity,Bytes,Uloc")

	var quoteRegex = regexp.MustCompile("\"")

	for result := range input {
		// Escape quotes in location and filename then surround with quotes.
		var location = "\"" + quoteRegex.ReplaceAllString(result.Location, "\"\"") + "\""
		var filename = "\"" + quoteRegex.ReplaceAllString(result.Filename, "\"\"") + "\""

		fmt.Printf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			result.Language,
			location,
			filename,
			fmt.Sprint(result.Lines),
			fmt.Sprint(result.Code),
			fmt.Sprint(result.Comment),
			fmt.Sprint(result.Blank),
			fmt.Sprint(result.Complexity),
			fmt.Sprint(result.Bytes),
			fmt.Sprint(result.Uloc),
		)
	}

	return ""
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

	language := make([]LanguageSummary, 0, len(languages))
	for _, summary := range languages {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	str := &strings.Builder{}

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
		<th>Uloc</th>
	</tr></thead>
	<tbody>`)

	for _, r := range language {
		fmt.Fprintf(str, `<tr>
		<th>%s</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
	</tr>`, r.Name, len(r.Files), r.Lines, r.Blank, r.Comment, r.Code, r.Complexity, r.Bytes, len(ulocLanguageCount[r.Name]))

		if Files {
			sortSummaryFiles(&r)

			for _, res := range r.Files {
				fmt.Fprintf(str, `<tr>
		<td>%s</td>
		<td></td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
		<td>%d</td>
	    <td>%d</td>
		<td>%d</td>
	</tr>`, res.Location, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.Bytes, res.Uloc)
			}
		}

	}

	fmt.Fprintf(str, `</tbody>
	<tfoot><tr>
		<th>Total</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
		<th>%d</th>
    	<th>%d</th>
		<th>%d</th>
	</tr>`, sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumBytes, len(ulocGlobalCount))

	if !Cocomo {
		var sb strings.Builder
		calculateCocomo(sumCode, &sb)
		fmt.Fprintf(str, `
	<tr>
		<th colspan="9">%s</th>
	</tr></tfoot>
	</table>`, strings.ReplaceAll(sb.String(), "\n", "<br>"))
	} else {
		str.WriteString(`</tfoot></table>`)
	}

	return str.String()
}

func toSqlInsert(input chan *FileJob) string {
	str := &strings.Builder{}
	projectName := SQLProject
	if projectName == "" {
		projectName = strings.Join(DirFilePaths, ",")
	}

	var sumCode int64
	str.WriteString("\nbegin transaction;")
	count := 0
	for res := range input {
		count++
		sumCode += res.Code

		dir, _ := filepath.Split(res.Location)

		fmt.Fprintf(str, "\ninsert into t values('%s', '%s', '%s', '%s', '%s', %d, %d, %d, %d, %d, %d);",
			escapeSQLString(projectName),
			escapeSQLString(res.Language),
			escapeSQLString(res.Location),
			escapeSQLString(dir),
			escapeSQLString(res.Filename), res.Bytes, res.Blank, res.Comment, res.Code, res.Complexity, res.Uloc)

		// every 1000 files commit and start a new transaction to avoid overloading
		if count == 1000 {
			str.WriteString("\ncommit;")
			str.WriteString("\nbegin transaction;")
			count = 0
		}
	}
	str.WriteString("\ncommit;")

	cost, schedule, people := esstimateCostScheduleMonths(sumCode)
	currentTime := time.Now()
	es := float64(makeTimestampMilli()-startTimeMilli) * 0.001
	str.WriteString("\nbegin transaction;")
	fmt.Fprintf(str, "\ninsert into metadata values('%s', '%s', %f, %f, %f, %f);",
		currentTime.Format("2006-01-02 15:04:05"),
		projectName,
		es,
		cost,
		schedule,
		people,
	)
	str.WriteString("\ncommit;")

	return str.String()
}

// attempt to manually escape everything that could be a problem
func escapeSQLString(input string) string {
	var buffer bytes.Buffer
	for _, char := range input {
		switch char {
		case '\x00':
			// Remove null characters
			continue
		case '\'':
			// Escape single quote with another single quote
			buffer.WriteRune('\'')
			buffer.WriteRune('\'')
		default:
			buffer.WriteRune(char)
		}
	}
	return buffer.String()
}

func toSql(input chan *FileJob) string {
	var str strings.Builder

	str.WriteString(`create table metadata (   -- github.com/boyter/scc v ` + Version + `
             timestamp text,
             Project   text,
             elapsed_s real,
             estimated_cost real,
             estimated_schedule_months real,
             estimated_people real);
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
             nComplexity   integer,
             nUloc         integer    
);`)

	str.WriteString(toSqlInsert(input))
	return str.String()
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
			case "json2":
				val = toJSON2(i)
			case "cloc-yaml":
				val = toClocYAML(i)
			case "cloc-yml":
				val = toClocYAML(i)
			case "csv":
				val = toCSV(i)
			case "csv-stream":
				val = toCSVStream(i)
				// special case where we want to ignore writing to stdout to disk as it's already done
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

func fileSummarizeLong(input chan *FileJob) string {
	str := &strings.Builder{}

	str.WriteString(getTabularWideBreak())
	fmt.Fprintf(str, tabularWideFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity", "Complexity/Lines")

	if !Files {
		str.WriteString(getTabularWideBreak())
	}

	langs := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0
	var sumWeightedComplexity float64

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumBytes += res.Bytes

		var weightedComplexity float64
		if res.Code != 0 {
			weightedComplexity = (float64(res.Complexity) / float64(res.Code)) * 100
		}
		res.WeightedComplexity = weightedComplexity
		sumWeightedComplexity += weightedComplexity

		_, ok := langs[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

			langs[res.Language] = LanguageSummary{
				Name:               res.Language,
				Lines:              res.Lines,
				Code:               res.Code,
				Comment:            res.Comment,
				Blank:              res.Blank,
				Complexity:         res.Complexity,
				Count:              1,
				WeightedComplexity: weightedComplexity,
				Files:              files,
				LineLength:         res.LineLength,
			}
		} else {
			tmp := langs[res.Language]
			files := append(tmp.Files, res)
			lineLength := append(tmp.LineLength, res.LineLength...)

			langs[res.Language] = LanguageSummary{
				Name:               res.Language,
				Lines:              tmp.Lines + res.Lines,
				Code:               tmp.Code + res.Code,
				Comment:            tmp.Comment + res.Comment,
				Blank:              tmp.Blank + res.Blank,
				Complexity:         tmp.Complexity + res.Complexity,
				Count:              tmp.Count + 1,
				WeightedComplexity: tmp.WeightedComplexity + weightedComplexity,
				Files:              files,
				LineLength:         lineLength,
			}
		}
	}

	language := make([]LanguageSummary, 0, len(langs))
	for _, summary := range langs {
		language = append(language, summary)
	}

	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
	switch {
	case SortBy == "name" || SortBy == "names" || SortBy == "language" || SortBy == "langs":
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

		fmt.Fprintf(str, tabularWideFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summary.WeightedComplexity)

		if Percent {
			fmt.Fprintf(str,
				tabularWideFormatBodyPercent,
				float64(len(summary.Files))/float64(sumFiles)*100,
				float64(summary.Lines)/float64(sumLines)*100,
				float64(summary.Blank)/float64(sumBlank)*100,
				float64(summary.Comment)/float64(sumComment)*100,
				float64(summary.Code)/float64(sumCode)*100,
				float64(summary.Complexity)/float64(sumComplexity)*100,
			)

			if !UlocMode {
				if !Files && summary.Name != language[len(language)-1].Name {
					str.WriteString(tabularWideBreakCi)
				}
			}
		}

		if MaxMean {
			fmt.Fprintf(str, tabularWideFormatFileMaxMean, maxIn(summary.LineLength), meanIn(summary.LineLength))
		}

		if UlocMode {
			fmt.Fprintf(str, tabularWideUlocLanguageFormatBody, len(ulocLanguageCount[summary.Name]))
			if !Files && summary.Name != language[len(language)-1].Name {
				str.WriteString(tabularWideBreakCi)
			}
		}

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularWideBreak())

			for _, res := range summary.Files {
				tmp := unicodeAwareTrim(res.Location, wideFormatFileTruncate)
				tmp = unicodeAwareRightPad(tmp, 43)

				fmt.Fprintf(str, tabularWideFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.WeightedComplexity)
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(getTabularWideBreak())
	fmt.Fprintf(str, tabularWideFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumWeightedComplexity)
	str.WriteString(getTabularWideBreak())

	if UlocMode {
		fmt.Fprintf(str, tabularWideUlocGlobalFormatBody, len(ulocGlobalCount))
		if Dryness {
			dryness := float64(len(ulocGlobalCount)) / float64(sumLines)
			fmt.Fprintf(str, "DRYness %% %43.2f\n", dryness)
		}
		str.WriteString(getTabularWideBreak())
	}

	if !Cocomo {
		if SLOCCountFormat {
			calculateCocomoSLOCCount(sumCode, str)
		} else {
			calculateCocomo(sumCode, str)
		}
	}
	if !Size {
		calculateSize(sumBytes, str)
		str.WriteString(getTabularWideBreak())
	}
	return str.String()
}

// We need to trim the file display for tabular output formats which this does in a unicode aware way
// to avoid cutting bytes... note that it needs to be expanded to deal with longer display characters at some
// point in the future
func unicodeAwareTrim(tmp string, size int) string {
	// iterate all the runes so we can cut off correctly and get the correct length
	r := []rune(tmp)

	if len(r) > size {
		for runewidth.StringWidth(tmp) > size {
			// remove character one at a time till we get the length we want
			tmp = string([]rune(tmp)[1:])
		}

		tmp = "~" + strings.TrimSpace(tmp)
	}

	return tmp
}

// Using %-30s in string format does not appear to be unicode aware with characters such as
// 文中 meaning the size is off... which is annoying, so we implement this ourselves to get it
// right
func unicodeAwareRightPad(tmp string, size int) string {
	for runewidth.StringWidth(tmp) < size {
		tmp += " "
	}

	return tmp
}

func fileSummarizeShort(input chan *FileJob) string {
	str := &strings.Builder{}

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		fmt.Fprintf(str, tabularShortFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity")
	} else {
		fmt.Fprintf(str, tabularShortFormatHeadNoComplexity, "Language", "Files", "Lines", "Blanks", "Comments", "Code")
	}

	if !Files {
		str.WriteString(getTabularShortBreak())
	}

	lang := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumBytes += res.Bytes

		_, ok := lang[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

			lang[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      res.Lines,
				Code:       res.Code,
				Comment:    res.Comment,
				Blank:      res.Blank,
				Complexity: res.Complexity,
				Count:      1,
				Files:      files,
				LineLength: res.LineLength,
			}
		} else {
			tmp := lang[res.Language]
			files := append(tmp.Files, res)
			lineLength := append(tmp.LineLength, res.LineLength...)

			lang[res.Language] = LanguageSummary{
				Name:       res.Language,
				Lines:      tmp.Lines + res.Lines,
				Code:       tmp.Code + res.Code,
				Comment:    tmp.Comment + res.Comment,
				Blank:      tmp.Blank + res.Blank,
				Complexity: tmp.Complexity + res.Complexity,
				Count:      tmp.Count + 1,
				Files:      files,
				LineLength: lineLength,
			}
		}
	}

	language := make([]LanguageSummary, 0, len(lang))
	for _, summary := range lang {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	startTime := makeTimestampMilli()
	for _, summary := range language {
		addBreak := false
		if Files {
			str.WriteString(getTabularShortBreak())
		}

		trimmedName := summary.Name
		trimmedName = trimNameShort(summary, trimmedName)

		if !Complexity {
			fmt.Fprintf(str, tabularShortFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity)
		} else {
			fmt.Fprintf(str, tabularShortFormatBodyNoComplexity, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code)
		}

		if Percent {
			if !Complexity {
				fmt.Fprintf(str,
					tabularShortPercentLanguageFormatBody,
					float64(len(summary.Files))/float64(sumFiles)*100,
					float64(summary.Lines)/float64(sumLines)*100,
					float64(summary.Blank)/float64(sumBlank)*100,
					float64(summary.Comment)/float64(sumComment)*100,
					float64(summary.Code)/float64(sumCode)*100,
					float64(summary.Complexity)/float64(sumComplexity)*100,
				)
			} else {
				fmt.Fprintf(str,
					tabularShortPercentLanguageFormatBodyNoComplexity,
					float64(len(summary.Files))/float64(sumFiles)*100,
					float64(summary.Lines)/float64(sumLines)*100,
					float64(summary.Blank)/float64(sumBlank)*100,
					float64(summary.Comment)/float64(sumComment)*100,
					float64(summary.Code)/float64(sumCode)*100,
				)
			}

			addBreak = true
		}

		if MaxMean {
			if !Complexity {
				fmt.Fprintf(str, tabularShortFormatFileMaxMean, maxIn(summary.LineLength), meanIn(summary.LineLength))
			} else {
				fmt.Fprintf(str, tabularShortFormatFileMaxMeanNoComplexity, maxIn(summary.LineLength), meanIn(summary.LineLength))
			}

			addBreak = true
		}

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularShortBreak())

			for _, res := range summary.Files {
				tmp := unicodeAwareTrim(res.Location, shortFormatFileTruncate)

				if !Complexity {
					tmp = unicodeAwareRightPad(tmp, 30)
					fmt.Fprintf(str, tabularShortFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity)
				} else {
					tmp = unicodeAwareRightPad(tmp, 34)
					fmt.Fprintf(str, tabularShortFormatFileNoComplexity, tmp, res.Lines, res.Blank, res.Comment, res.Code)
				}

				// if MaxMean {
				// 	fmt.Fprintf(str, tabularShortFormatFileMaxMean, maxIn(res.LineLength), meanIn(res.LineLength))
				// }
			}
		}

		if UlocMode {
			if !Complexity {
				fmt.Fprintf(str, tabularShortUlocLanguageFormatBody, len(ulocLanguageCount[summary.Name]))
			} else {
				fmt.Fprintf(str, tabularShortUlocLanguageFormatBodyNoComplexity, len(ulocLanguageCount[summary.Name]))
			}

			addBreak = true
		}

		if addBreak {
			if !Files && summary.Name != language[len(language)-1].Name {
				str.WriteString(tabularShortBreakCi)
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		fmt.Fprintf(str, tabularShortFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity)
	} else {
		fmt.Fprintf(str, tabularShortFormatBodyNoComplexity, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode)
	}
	str.WriteString(getTabularShortBreak())

	if UlocMode {
		fmt.Fprintf(str, tabularShortUlocGlobalFormatBody, len(ulocGlobalCount))
		if Dryness {
			dryness := float64(len(ulocGlobalCount)) / float64(sumLines)
			fmt.Fprintf(str, "DRYness %% %30.2f\n", dryness)
		}
		str.WriteString(getTabularShortBreak())
	}

	if !Cocomo {
		if SLOCCountFormat {
			calculateCocomoSLOCCount(sumCode, str)
		} else {
			calculateCocomo(sumCode, str)
		}
		str.WriteString(getTabularShortBreak())
	}
	if !Size {
		calculateSize(sumBytes, str)
		str.WriteString(getTabularShortBreak())
	}
	return str.String()
}

func maxIn(i []int) int {
	if len(i) == 0 {
		return 0
	}

	m := i[0]
	for _, x := range i {
		if x > m {
			m = x
		}
	}

	return m
}

func meanIn(i []int) int {
	if len(i) == 0 {
		return 0
	}

	sum := 0
	for _, x := range i {
		sum += x
	}

	return sum / len(i)
}

func trimNameShort(summary LanguageSummary, trimmedName string) string {
	if len(summary.Name) > shortNameTruncate {
		trimmedName = summary.Name[:shortNameTruncate-1] + "…"
	}
	return trimmedName
}

func calculateCocomoSLOCCount(sumCode int64, str *strings.Builder) {
	estimatedEffort := EstimateEffort(int64(sumCode), EAF)
	estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
	estimatedPeopleRequired := estimatedEffort / estimatedScheduleMonths
	estimatedCost := EstimateCost(estimatedEffort, AverageWage, Overhead)

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	p.Fprintf(str, "Total Physical Source Lines of Code (SLOC)                     = %d\n", sumCode)
	p.Fprintf(str, "Development Effort Estimate, Person-Years (Person-Months)      = %.2f (%.2f)\n", estimatedEffort/12, estimatedEffort)
	p.Fprintf(str, " (Basic COCOMO model, Person-Months = %.2f*(KSLOC**%.2f)*%.2f)\n", projectType[CocomoProjectType][0], projectType[CocomoProjectType][1], EAF)
	p.Fprintf(str, "Schedule Estimate, Years (Months)                              = %.2f (%.2f)\n", estimatedScheduleMonths/12, estimatedScheduleMonths)
	p.Fprintf(str, " (Basic COCOMO model, Months = %.2f*(person-months**%.2f))\n", projectType[CocomoProjectType][2], projectType[CocomoProjectType][3])
	p.Fprintf(str, "Estimated Average Number of Developers (Effort/Schedule)       = %.2f\n", estimatedPeopleRequired)
	p.Fprintf(str, "Total Estimated Cost to Develop                                = %s%.0f\n", CurrencySymbol, estimatedCost)
	p.Fprintf(str, " (average salary = %s%d/year, overhead = %.2f)\n", CurrencySymbol, AverageWage, Overhead)
}

func calculateCocomo(sumCode int64, str *strings.Builder) {
	estimatedCost, estimatedScheduleMonths, estimatedPeopleRequired := esstimateCostScheduleMonths(sumCode)

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	p.Fprintf(str, "Estimated Cost to Develop (%s) %s%d\n", CocomoProjectType, CurrencySymbol, int64(estimatedCost))
	p.Fprintf(str, "Estimated Schedule Effort (%s) %.2f months\n", CocomoProjectType, estimatedScheduleMonths)
	if math.IsNaN(estimatedPeopleRequired) {
		p.Fprintf(str, "Estimated People Required 1 Grandparent\n")
	} else {
		p.Fprintf(str, "Estimated People Required (%s) %.2f\n", CocomoProjectType, estimatedPeopleRequired)
	}
}

func esstimateCostScheduleMonths(sumCode int64) (float64, float64, float64) {
	estimatedEffort := EstimateEffort(int64(sumCode), EAF)
	estimatedCost := EstimateCost(estimatedEffort, AverageWage, Overhead)
	estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
	estimatedPeopleRequired := estimatedEffort / estimatedScheduleMonths
	return estimatedCost, estimatedScheduleMonths, estimatedPeopleRequired
}

func calculateSize(sumBytes int64, str *strings.Builder) {

	var size float64

	switch strings.ToLower(SizeUnit) {
	case "binary":
		size = float64(sumBytes) / 1_048_576
	case "mixed":
		size = float64(sumBytes) / 1_024_000
	case "xkcd-kb":
		str.WriteString("1000 bytes during leap years, 1024 otherwise\n")
		if isLeapYear(time.Now().Year()) {
			size = float64(sumBytes) / 1_000_000
		}
	case "xkcd-kelly":
		str.WriteString("compromise between 1000 and 1024 bytes\n")
		size = float64(sumBytes) / (1012 * 1012)
	case "xkcd-imaginary":
		str.WriteString("used in quantum computing\n")
		fmt.Fprintf(str, "Processed %d bytes, %s megabytes (%s)\n", sumBytes, `¯\_(ツ)_/¯`, strings.ToUpper(SizeUnit))
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

	if !strings.EqualFold(SizeUnit, "xkcd-imaginary") {
		fmt.Fprintf(str, "Processed %d bytes, %.3f megabytes (%s)\n", sumBytes, size, strings.ToUpper(SizeUnit))
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

func aggregateLanguageSummary(input chan *FileJob) []LanguageSummary {
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

	language := make([]LanguageSummary, 0, len(languages))
	for _, summary := range languages {
		language = append(language, summary)
	}

	return language
}

func sortLanguageSummary(language []LanguageSummary) []LanguageSummary {
	// Cater for the common case of adding plural even for those options that don't make sense
	// as it's quite common for those who English is not a first language to make a simple mistake
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
