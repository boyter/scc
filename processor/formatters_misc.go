// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v2"
)

var openMetricsMetadata = `# TYPE scc_files gauge
# HELP scc_files Number of sourcecode files.
# TYPE scc_lines gauge
# HELP scc_lines Number of lines.
# TYPE scc_code gauge
# HELP scc_code Number of lines of actual code.
# TYPE scc_comments gauge
# HELP scc_comments Number of comments.
# TYPE scc_blanks gauge
# HELP scc_blanks Number of blank lines.
# TYPE scc_complexity gauge
# HELP scc_complexity Code complexity.
# TYPE scc_bytes gauge
# UNIT scc_bytes bytes
# HELP scc_bytes Size in bytes.
`
var openMetricsSummaryRecordFormat = "scc_%s{language=\"%s\"} %d\n"
var openMetricsFileRecordFormat = "scc_%s{language=\"%s\",file=\"%s\"} %d\n"

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

	printDebugF("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime)

	return yamlString
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
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "files", result.Name, result.Count)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "lines", result.Name, result.Lines)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "code", result.Name, result.Code)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "comments", result.Name, result.Comment)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "blanks", result.Name, result.Blank)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "complexity", result.Name, result.Complexity)
		_, _ = fmt.Fprintf(sb, openMetricsSummaryRecordFormat, "bytes", result.Name, result.Bytes)
	}
	return sb.String()
}

func toOpenMetricsFiles(input chan *FileJob) string {
	sb := &strings.Builder{}
	sb.WriteString(openMetricsMetadata)
	for file := range input {
		var filename = strings.ReplaceAll(file.Location, "\\", "\\\\")
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "lines", file.Language, filename, file.Lines)
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "code", file.Language, filename, file.Code)
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "comments", file.Language, filename, file.Comment)
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "blanks", file.Language, filename, file.Blank)
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "complexity", file.Language, filename, file.Complexity)
		_, _ = fmt.Fprintf(sb, openMetricsFileRecordFormat, "bytes", file.Language, filename, file.Bytes)
	}
	sb.WriteString("# EOF\n")
	return sb.String()
}

func toHtml(input chan *FileJob) string {
	return `<html lang="en"><head><meta charset="utf-8" /><title>scc html output</title><style>table { border-collapse: collapse; }td, th { border: 1px solid #999; padding: 0.5rem; text-align: left;}</style></head><body>` +
		toHtmlTable(input) +
		"</body></html>\n"
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
		_, _ = fmt.Fprintf(str, `<tr>
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
				_, _ = fmt.Fprintf(str, `<tr>
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

	_, _ = fmt.Fprintf(str, `</tbody>
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

	hasCostOutput := false
	if !Cocomo {
		var sb strings.Builder
		calculateCocomo(sumCode, &sb)
		_, _ = fmt.Fprintf(str, `
	<tr>
		<th colspan="9">%s</th>
	</tr>`, strings.ReplaceAll(sb.String(), "\n", "<br>"))
		hasCostOutput = true
	}
	if Locomo {
		var sb strings.Builder
		calculateLocomo(sumCode, sumComplexity, &sb)
		_, _ = fmt.Fprintf(str, `
	<tr>
		<th colspan="9">%s</th>
	</tr>`, strings.ReplaceAll(sb.String(), "\n", "<br>"))
		hasCostOutput = true
	}
	if hasCostOutput {
		str.WriteString(`</tfoot>
	</table>`)
	} else {
		str.WriteString(`</tfoot></table>`)
	}

	return str.String()
}
