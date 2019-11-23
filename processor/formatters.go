package processor

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	glang "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
	"gopkg.in/yaml.v2"
	"math"
	"os"
	"sort"
	"strings"
	"time"
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
var shortFormatFileTrucateNoComplexity = 33
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
			return summary.Files[i].Lines > summary.Files[j].Lines
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
	Url              string  `yaml:"url"`
	Version          string  `yaml:"version"`
	Elapsed_seconds  float64 `yaml:"elapsed_seconds"`
	N_files          int64   `yaml:"n_files"`
	N_lines          int64   `yaml:"n_lines"`
	Files_per_second float64 `yaml:"files_per_second"`
	Lines_per_second float64 `yaml:"lines_per_second"`
}

type LanguageReportStart struct {
	Header headerStruct
}

type LanguageReportEnd struct {
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
		Url:              "https://github.com/boyter/scc/",
		Version:          Version,
		N_files:          sumFiles,
		N_lines:          sumLines,
		Elapsed_seconds:  es,
		Files_per_second: float64(float64(sumFiles) / es),
		Lines_per_second: float64(float64(sumLines) / es),
	}
	summary := summaryStruct{
		Blank:   sumBlank,
		Comment: sumComment,
		Code:    sumCode,
		Count:   sumFiles,
	}
	reportStart := LanguageReportStart{
		Header: header,
	}
	reportEnd := LanguageReportEnd{
		Sum: summary,
	}

	report_yaml, _ := yaml.Marshal(reportStart)
	sum_yaml, _ := yaml.Marshal(reportEnd)
	language_yaml, _ := yaml.Marshal(languages)
	yamlString := "# https://github.com/boyter/scc/\n" + string(report_yaml) + string(language_yaml) + string(sum_yaml)

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	return yamlString
}

func toJSON(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	languages := map[string]LanguageSummary{}
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
		"Complexity"},
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
			fmt.Sprint(result.Complexity)})
	}

	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	w.WriteAll(records)
	w.Flush()

	return b.String()
}

func fileSummarize(input chan *FileJob) string {
	switch {
	case More || strings.ToLower(Format) == "wide":
		return fileSummarizeLong(input)
	case strings.ToLower(Format) == "json":
		return toJSON(input)
	case strings.ToLower(Format) == "cloc-yaml" || strings.ToLower(Format) == "cloc-yml":
		return toClocYAML(input)
	case strings.ToLower(Format) == "csv":
		return toCSV(input)
	}

	return fileSummarizeShort(input)
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
