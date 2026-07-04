// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/mattn/go-runewidth"

	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
)

var tabularShortFormatHead = "%-15s %9s %11s %9s %9s %10s %10s\n"
var tabularShortFormatBody = "%-15s %9d %11d %9d %9d %10d %10d\n"
var tabularShortFormatFile = "%s %9d %9d %9d %10d %10d\n"
var tabularShortFormatFileMaxMean = "MaxLine / MeanLine %6d %11d\n"
var shortFormatFileTruncate = 26
var shortNameTruncate = 15
var tabularShortUlocLanguageFormatBody = "(ULOC) %30d\n"
var tabularShortPercentLanguageFormatBody = "Percentage %13.1f%% %10.1f%% %8.1f%% %8.1f%% %9.1f%% %9.1f%%\n"
var tabularShortUlocGlobalFormatBody = "Unique Lines of Code (ULOC) %9d\n"
var tabularShortDrynessFormatBody = "DRYness %% %27.2f\n"

var tabularShortFormatHeadNoComplexity = "%-21s %11s %11s %10s %11s %10s\n"
var tabularShortFormatBodyNoComplexity = "%-21s %11d %11d %10d %11d %10d\n"
var tabularShortFormatFileNoComplexity = "%s %10d %10d %11d %10d\n"
var tabularShortFormatFileMaxMeanNoComplexity = "MaxLine / MeanLine %14d %11d\n"
var longNameTruncate = 22
var tabularShortUlocLanguageFormatBodyNoComplexity = "(ULOC) %38d\n"
var tabularShortPercentLanguageFormatBodyNoComplexity = "Percentage %21.1f%% %10.1f%% %9.1f%% %10.1f%% %9.1f%%\n"

var tabularWideFormatHead = "%-33s %9s %9s %8s %9s %8s %10s %16s\n"
var tabularWideFormatBody = "%-33s %9d %9d %8d %9d %8d %10d %16.2f\n"
var tabularWideFormatFile = "%s %9d %8d %9d %8d %10d %16.2f\n"

// Cognitive variants add a right-aligned "Cognitive" column after Complexity.
// Selected only when processor.Cognitive is set so the default layout is untouched.
var tabularWideFormatHeadCognitive = "%-33s %9s %9s %8s %9s %8s %10s %10s %16s\n"
var tabularWideFormatBodyCognitive = "%-33s %9d %9d %8d %9d %8d %10d %10d %16.2f\n"
var tabularWideFormatFileCognitive = "%s %9d %8d %9d %8d %10d %10d %16.2f\n"
var tabularWideFormatFileMaxMean = "MaxLine / MeanLine %24d %9d\n"
var wideFormatFileTruncate = 42
var tabularWideUlocLanguageFormatBody = "(ULOC) %46d\n"
var tabularWideUlocGlobalFormatBody = "Unique Lines of Code (ULOC) %25d\n"
var tabularWideFormatBodyPercent = "Percentage %31.1f%% %8.1f%% %7.1f%% %8.1f%% %7.1f%% %9.1f%%\n"
var tabularWideDrynessFormatBody = "DRYness %% %43.2f\n"

func fileSummarizeLong(input chan *FileJob) string {
	str := &strings.Builder{}

	str.WriteString(getTabularWideBreak())
	if Cognitive {
		_, _ = fmt.Fprintf(str, tabularWideFormatHeadCognitive, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity", "Cognitive", "Complexity/Lines")
	} else {
		_, _ = fmt.Fprintf(str, tabularWideFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity", "Complexity/Lines")
	}

	if !Files {
		str.WriteString(getTabularWideBreak())
	}

	langs := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumCognitive, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0, 0

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumCognitive += res.Cognitive
		sumBytes += res.Bytes

		var weightedComplexity float64
		if res.Code != 0 {
			weightedComplexity = (float64(res.Complexity) / float64(res.Code)) * 100
		}
		res.WeightedComplexity = weightedComplexity

		_, ok := langs[res.Language]

		if !ok {
			files := []*FileJob{}
			files = append(files, res)

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
				LineLength: res.LineLength,
			}
		} else {
			tmp := langs[res.Language]
			files := append(tmp.Files, res)
			lineLength := append(tmp.LineLength, res.LineLength...)

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
				LineLength: lineLength,
			}
		}
	}

	language := make([]LanguageSummary, 0, len(langs))
	for _, summary := range langs {
		language = append(language, summary)
	}

	language = sortLanguageSummary(language)

	startTime := makeTimestampMilli()
	for _, summary := range language {
		if Files {
			str.WriteString(getTabularWideBreak())
		}

		trimmedName := summary.Name
		if len(summary.Name) > longNameTruncate {
			trimmedName = summary.Name[:longNameTruncate-1] + "…"
		}

		var summaryWeightedComplexity float64
		if summary.Code != 0 {
			summaryWeightedComplexity = (float64(summary.Complexity) / float64(summary.Code)) * 100
		}

		if Cognitive {
			_, _ = fmt.Fprintf(str, tabularWideFormatBodyCognitive, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summary.Cognitive, summaryWeightedComplexity)
		} else {
			_, _ = fmt.Fprintf(str, tabularWideFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summaryWeightedComplexity)
		}

		if Percent {
			_, _ = fmt.Fprintf(str,
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
			_, _ = fmt.Fprintf(str, tabularWideFormatFileMaxMean, maxIn(summary.LineLength), meanIn(summary.LineLength))
		}

		if UlocMode {
			_, _ = fmt.Fprintf(str, tabularWideUlocLanguageFormatBody, len(ulocLanguageCount[summary.Name]))
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

				if Cognitive {
					_, _ = fmt.Fprintf(str, tabularWideFormatFileCognitive, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.Cognitive, res.WeightedComplexity)
				} else {
					_, _ = fmt.Fprintf(str, tabularWideFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.WeightedComplexity)
				}
			}
		}
	}

	printDebugF("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime)

	var totalWeightedComplexity float64
	if sumCode != 0 {
		totalWeightedComplexity = (float64(sumComplexity) / float64(sumCode)) * 100
	}

	str.WriteString(getTabularWideBreak())
	if Cognitive {
		_, _ = fmt.Fprintf(str, tabularWideFormatBodyCognitive, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumCognitive, totalWeightedComplexity)
	} else {
		_, _ = fmt.Fprintf(str, tabularWideFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, totalWeightedComplexity)
	}
	str.WriteString(getTabularWideBreak())

	if UlocMode {
		_, _ = fmt.Fprintf(str, tabularWideUlocGlobalFormatBody, len(ulocGlobalCount))
		if Dryness {
			dryness := float64(len(ulocGlobalCount)) / float64(sumLines)
			_, _ = fmt.Fprintf(str, tabularWideDrynessFormatBody, dryness)
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
	if Locomo {
		calculateLocomo(sumCode, sumComplexity, str)
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
			r = r[1:]
			tmp = string(r)
		}

		tmp = "~" + strings.TrimSpace(tmp)
	}

	return tmp
}

// Using %-30s in string format does not appear to be unicode aware with characters such as
// 文中 meaning the size is off... which is annoying, so we implement this ourselves to get it
// right
func unicodeAwareRightPad(tmp string, size int) string {
	return runewidth.FillRight(tmp, size)
}

func fileSummarizeShort(input chan *FileJob) string {
	str := &strings.Builder{}

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		_, _ = fmt.Fprintf(str, tabularShortFormatHead, "Language", "Files", "Lines", "Blanks", "Comments", "Code", "Complexity")
	} else {
		_, _ = fmt.Fprintf(str, tabularShortFormatHeadNoComplexity, "Language", "Files", "Lines", "Blanks", "Comments", "Code")
	}

	if !Files {
		str.WriteString(getTabularShortBreak())
	}

	lang := map[string]LanguageSummary{}
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumCognitive, sumBytes int64 = 0, 0, 0, 0, 0, 0, 0, 0

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	for res := range input {
		sumFiles++
		sumLines += res.Lines
		sumCode += res.Code
		sumComment += res.Comment
		sumBlank += res.Blank
		sumComplexity += res.Complexity
		sumCognitive += res.Cognitive
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
				Cognitive:  res.Cognitive,
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
				Cognitive:  tmp.Cognitive + res.Cognitive,
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
			_, _ = p.Fprintf(str, tabularShortFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, activeComplexity(summary.Complexity, summary.Cognitive))
		} else {
			_, _ = p.Fprintf(str, tabularShortFormatBodyNoComplexity, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code)
		}

		if Percent {
			if !Complexity {
				_, _ = p.Fprintf(str,
					tabularShortPercentLanguageFormatBody,
					float64(len(summary.Files))/float64(sumFiles)*100,
					float64(summary.Lines)/float64(sumLines)*100,
					float64(summary.Blank)/float64(sumBlank)*100,
					float64(summary.Comment)/float64(sumComment)*100,
					float64(summary.Code)/float64(sumCode)*100,
					float64(activeComplexity(summary.Complexity, summary.Cognitive))/float64(activeComplexity(sumComplexity, sumCognitive))*100,
				)
			} else {
				_, _ = p.Fprintf(str,
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
				_, _ = p.Fprintf(str, tabularShortFormatFileMaxMean, maxIn(summary.LineLength), meanIn(summary.LineLength))
			} else {
				_, _ = p.Fprintf(str, tabularShortFormatFileMaxMeanNoComplexity, maxIn(summary.LineLength), meanIn(summary.LineLength))
			}

			addBreak = true
		}

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularShortBreak())

			for _, res := range summary.Files {
				tmp := unicodeAwareTrim(res.Location, shortFormatFileTruncate)

				if !Complexity {
					tmp = unicodeAwareRightPad(tmp, 27)
					_, _ = p.Fprintf(str, tabularShortFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, activeComplexity(res.Complexity, res.Cognitive))
				} else {
					tmp = unicodeAwareRightPad(tmp, 34)
					_, _ = p.Fprintf(str, tabularShortFormatFileNoComplexity, tmp, res.Lines, res.Blank, res.Comment, res.Code)
				}
			}
		}

		if UlocMode {
			if !Complexity {
				_, _ = p.Fprintf(str, tabularShortUlocLanguageFormatBody, len(ulocLanguageCount[summary.Name]))
			} else {
				_, _ = p.Fprintf(str, tabularShortUlocLanguageFormatBodyNoComplexity, len(ulocLanguageCount[summary.Name]))
			}

			addBreak = true
		}

		if addBreak {
			if !Files && summary.Name != language[len(language)-1].Name {
				str.WriteString(tabularShortBreakCi)
			}
		}
	}

	printDebugF("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime)

	str.WriteString(getTabularShortBreak())
	if !Complexity {
		_, _ = p.Fprintf(str, tabularShortFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, activeComplexity(sumComplexity, sumCognitive))
	} else {
		_, _ = p.Fprintf(str, tabularShortFormatBodyNoComplexity, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode)
	}
	str.WriteString(getTabularShortBreak())

	if UlocMode {
		_, _ = p.Fprintf(str, tabularShortUlocGlobalFormatBody, len(ulocGlobalCount))
		if Dryness {
			dryness := float64(len(ulocGlobalCount)) / float64(sumLines)
			_, _ = p.Fprintf(str, tabularShortDrynessFormatBody, dryness)
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
	if Locomo {
		calculateLocomo(sumCode, sumComplexity, str)
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

	return slices.Max(i)
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
