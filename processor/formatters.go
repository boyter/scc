package processor

import (
	"fmt"
	glang "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
	"sort"
	"strings"
	"time"
)

var tabularShortBreak = "-------------------------------------------------------------------------------\n"
var tabularShortFormatHead = "%-20s %9s %9s %8s %9s %8s %10s\n"
var tabularShortFormatBody = "%-20s %9d %9d %8d %9d %8d %10d\n"
var tabularShortFormatFile = "%-30s %9d %8d %9d %8d %10d\n"
var shortFormatFileTrucate = 29

var tabularWideBreak = "-------------------------------------------------------------------------------------------------------------\n"
var tabularWideFormatHead = "%-33s %9s %9s %8s %9s %8s %10s %16s\n"
var tabularWideFormatBody = "%-33s %9d %9d %8d %9d %8d %10d %16f\n"
var tabularWideFormatFile = "%-43s %9d %8d %9d %8d %10d %16f\n"
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

func fileSummerize(input *chan *FileJob) string {
	switch {
	case More:
		return fileSummerizeLong(input)
	case !More:
		return fileSummerizeShort(input)
	}

	return ""
}

func fileSummerizeLong(input *chan *FileJob) string {
	var str strings.Builder

	str.WriteString(tabularWideBreak)
	str.WriteString(fmt.Sprintf(tabularWideFormatHead, "Language", "Files", "Lines", "Code", "Comments", "Blanks", "Complexity", "Complexity/Lines"))

	if !Files {
		str.WriteString(tabularWideBreak)
	}

	languages := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity int64 = 0, 0, 0, 0, 0, 0
	var sumWeightedComplexity float64 = 0

	for res := range *input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity

		var weightedComplexity float64 = 0
		if res.Lines != 0 {
			weightedComplexity = float64(res.Complexity) / float64(res.Lines)
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
			str.WriteString(tabularWideBreak)
		}

		str.WriteString(fmt.Sprintf(tabularWideFormatBody, summary.Name, summary.Count, summary.Lines, summary.Code, summary.Comment, summary.Blank, summary.Complexity, sumWeightedComplexity))

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(tabularWideBreak)

			for _, res := range summary.Files {
				tmp := res.Location

				if len(tmp) >= wideFormatFileTrucate {
					totrim := len(tmp) - wideFormatFileTrucate
					tmp = "~" + tmp[totrim:]
				}

				str.WriteString(fmt.Sprintf(tabularWideFormatFile, tmp, res.Lines, res.Code, res.Comment, res.Blank, res.Complexity, res.WeightedComplexity))
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(tabularWideBreak)
	str.WriteString(fmt.Sprintf(tabularWideFormatBody, "Total", sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumWeightedComplexity))
	str.WriteString(tabularWideBreak)

	return str.String()
}

func fileSummerizeShort(input *chan *FileJob) string {
	var str strings.Builder

	str.WriteString(tabularShortBreak)
	str.WriteString(fmt.Sprintf(tabularShortFormatHead, "Language", "Files", "Lines", "Code", "Comments", "Blanks", "Complexity"))

	if !Files {
		str.WriteString(tabularShortBreak)
	}

	languages := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity int64 = 0, 0, 0, 0, 0, 0

	for res := range *input {
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
			str.WriteString(tabularShortBreak)
		}

		str.WriteString(fmt.Sprintf(tabularShortFormatBody, summary.Name, summary.Count, summary.Lines, summary.Code, summary.Comment, summary.Blank, summary.Complexity))

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(tabularShortBreak)

			for _, res := range summary.Files {
				tmp := res.Location

				if len(tmp) >= shortFormatFileTrucate {
					totrim := len(tmp) - shortFormatFileTrucate
					tmp = "~" + tmp[totrim:]
				}

				str.WriteString(fmt.Sprintf(tabularShortFormatFile, tmp, res.Lines, res.Code, res.Comment, res.Blank, res.Complexity))
			}
		}
	}

	if Debug {
		printDebug(fmt.Sprintf("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime))
	}

	str.WriteString(tabularShortBreak)
	str.WriteString(fmt.Sprintf(tabularShortFormatBody, "Total", sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity))
	str.WriteString(tabularShortBreak)

	p := gmessage.NewPrinter(glang.English)
	str.WriteString(p.Sprintf("Estimated Cost to Develop $%d\n", int64(EstimateCost(EstimateEffort(int64(sumCode)), 56286))))
	str.WriteString(tabularShortBreak)

	return str.String()
}

// Get the time as standard UTC/Zulu format
func getFormattedTime() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// Prints a message to stdout if flag to enable warning output is set
func printWarn(msg string) {
	if Verbose {
		fmt.Println(fmt.Sprintf(" WARN %s: %s", getFormattedTime(), msg))
	}
}

// Prints a message to stdout if flag to enable debug output is set
func printDebug(msg string) {
	if Debug {
		fmt.Println(fmt.Sprintf("DEBUG %s: %s", getFormattedTime(), msg))
	}
}

// Prints a message to stdout if flag to enable trace output is set
func printTrace(msg string) {
	if Trace {
		fmt.Println(fmt.Sprintf("TRACE %s: %s", getFormattedTime(), msg))
	}
}
