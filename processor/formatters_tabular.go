// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"github.com/clipperhouse/uax29/v2/graphemes"
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

// The wide layout is fixed-width and totals 109 columns (120 with the extra
// Cognitive column) so every row aligns under the ─── break lines. The Language
// column is deliberately narrow (24) so the numeric columns can hold large,
// comma-grouped counts (tens of millions) without overflowing on big trees such
// as the Linux kernel. Any width change here must keep the totals at 109/120 and
// be mirrored across the file/MaxMean/ULOC/Percentage/DRYness rows below.
var tabularWideFormatHead = "%-24s %8s %12s %10s %10s %12s %10s %16s\n"
var tabularWideFormatBody = "%-24s %8d %12d %10d %10d %12d %10d %16.2f\n"
var tabularWideFormatFile = "%s %12d %10d %10d %12d %10d %16.2f\n"

// Cognitive variants add a right-aligned "Cognitive" column after Complexity.
// Selected only when processor.Cognitive is set so the default layout is untouched.
var tabularWideFormatHeadCognitive = "%-24s %8s %12s %10s %10s %12s %10s %10s %16s\n"
var tabularWideFormatBodyCognitive = "%-24s %8d %12d %10d %10d %12d %10d %10d %16.2f\n"
var tabularWideFormatFileCognitive = "%s %12d %10d %10d %12d %10d %10d %16.2f\n"
var tabularWideFormatFileMaxMean = "MaxLine / MeanLine %14d %12d\n"
var wideFormatFileTruncate = 32
var tabularWideUlocLanguageFormatBody = "(ULOC) %39d\n"
var tabularWideUlocGlobalFormatBody = "Unique Lines of Code (ULOC) %18d\n"
var tabularWideFormatBodyPercent = "Percentage %21.1f%% %11.1f%% %9.1f%% %9.1f%% %11.1f%% %9.1f%%\n"
var tabularWideDrynessFormatBody = "DRYness %% %36.2f\n"

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

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	totals := collectSummaryTotals(input, Files, MaxMean, Files)
	sumFiles := totals.sumFiles
	sumLines := totals.sumLines
	sumCode := totals.sumCode
	sumComment := totals.sumComment
	sumBlank := totals.sumBlank
	sumComplexity := totals.sumComplexity
	sumCognitive := totals.sumCognitive
	sumBytes := totals.sumBytes

	language := totals.sorted()

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
			_, _ = p.Fprintf(str, tabularWideFormatBodyCognitive, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summary.Cognitive, summaryWeightedComplexity)
		} else {
			_, _ = p.Fprintf(str, tabularWideFormatBody, trimmedName, summary.Count, summary.Lines, summary.Blank, summary.Comment, summary.Code, summary.Complexity, summaryWeightedComplexity)
		}

		if Percent {
			_, _ = p.Fprintf(str,
				tabularWideFormatBodyPercent,
				pct(summary.Count, sumFiles),
				pct(summary.Lines, sumLines),
				pct(summary.Blank, sumBlank),
				pct(summary.Comment, sumComment),
				pct(summary.Code, sumCode),
				pct(summary.Complexity, sumComplexity),
			)

			if !UlocMode {
				if !Files && summary.Name != language[len(language)-1].Name {
					str.WriteString(tabularWideBreakCi)
				}
			}
		}

		if MaxMean {
			_, _ = p.Fprintf(str, tabularWideFormatFileMaxMean, maxIn(summary.LineLength), meanIn(summary.LineLength))
		}

		if UlocMode {
			_, _ = p.Fprintf(str, tabularWideUlocLanguageFormatBody, len(ulocLanguageCount[summary.Name]))
			if !Files && summary.Name != language[len(language)-1].Name {
				str.WriteString(tabularWideBreakCi)
			}
		}

		if Files {
			sortSummaryFiles(&summary)
			str.WriteString(getTabularWideBreak())

			for _, res := range summary.Files {
				tmp := unicodeAwareTrim(res.Location, wideFormatFileTruncate)
				tmp = unicodeAwareRightPad(tmp, 33)

				if Cognitive {
					_, _ = p.Fprintf(str, tabularWideFormatFileCognitive, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.Cognitive, res.WeightedComplexity)
				} else {
					_, _ = p.Fprintf(str, tabularWideFormatFile, tmp, res.Lines, res.Blank, res.Comment, res.Code, res.Complexity, res.WeightedComplexity)
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
		_, _ = p.Fprintf(str, tabularWideFormatBodyCognitive, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, sumCognitive, totalWeightedComplexity)
	} else {
		_, _ = p.Fprintf(str, tabularWideFormatBody, "Total", sumFiles, sumLines, sumBlank, sumComment, sumCode, sumComplexity, totalWeightedComplexity)
	}
	str.WriteString(getTabularWideBreak())

	if UlocMode {
		_, _ = p.Fprintf(str, tabularWideUlocGlobalFormatBody, len(ulocGlobalCount))
		if Dryness {
			// Guard the divide: a tree of only empty files has zero lines, and an
			// unguarded float division prints DRYness as ∞. Matches the guard
			// snapshotULOC already applies to the same calculation.
			dryness := 0.0
			if sumLines > 0 {
				dryness = float64(len(ulocGlobalCount)) / float64(sumLines)
			}
			_, _ = p.Fprintf(str, tabularWideDrynessFormatBody, dryness)
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
//
// What is wanted is the longest tail of the string that fits the width, with a
// ~ in front of it to say something was cut. Removing one rune at a time and
// asking the width again after each is the obvious way to find it and costs a
// fresh string and a fresh walk of that string for every rune removed, which on
// a --by-file run over a tree of long paths was two thirds of the whole run.
// Walking back from the end and adding up the widths finds the same place in
// one pass, StringWidth being the sum of RuneWidth over the string.
func unicodeAwareTrim(tmp string, size int) string {
	// A string of size bytes or fewer cannot hold more than size runes, and a
	// rune is never narrower than nothing, so there is nothing to cut. This is
	// the answer for nearly every name.
	if len(tmp) <= size {
		return tmp
	}

	r := []rune(tmp)
	if len(r) <= size {
		return tmp
	}

	// Width is asked per grapheme cluster, not per rune, because that is what
	// StringWidth answers and the two disagree: a family emoji is four runes
	// joined by zero width joiners and is two cells wide, where the runes add
	// up to eight. Summing runes cut a name that already fitted, and cut it
	// through the middle of a cluster, leaving a dangling joiner.
	//
	// Two walks rather than one so nothing is allocated: the first asks how
	// wide the whole name is, the second drops clusters off the front until
	// what is left fits.
	total := 0
	forward := graphemes.FromString(tmp)
	for forward.Next() {
		total += graphemeWidth(forward.Value())
	}

	dropped := 0
	offset := 0
	trim := graphemes.FromString(tmp)
	for total-dropped > size && trim.Next() {
		cluster := trim.Value()
		dropped += graphemeWidth(cluster)
		offset += len(cluster)
	}

	return "~" + strings.TrimSpace(tmp[offset:])
}

// graphemeWidth is the width of one grapheme cluster, which runewidth takes to
// be the width of the first rune in it that has one.
func graphemeWidth(cluster string) int {
	for _, r := range cluster {
		if w := runewidth.RuneWidth(r); w > 0 {
			return w
		}
	}

	return 0
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

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	totals := collectSummaryTotals(input, Files, MaxMean, false)
	sumFiles := totals.sumFiles
	sumLines := totals.sumLines
	sumCode := totals.sumCode
	sumComment := totals.sumComment
	sumBlank := totals.sumBlank
	sumComplexity := totals.sumComplexity
	sumCognitive := totals.sumCognitive
	sumBytes := totals.sumBytes

	language := totals.sorted()

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
					pct(summary.Count, sumFiles),
					pct(summary.Lines, sumLines),
					pct(summary.Blank, sumBlank),
					pct(summary.Comment, sumComment),
					pct(summary.Code, sumCode),
					pct(activeComplexity(summary.Complexity, summary.Cognitive), activeComplexity(sumComplexity, sumCognitive)),
				)
			} else {
				_, _ = p.Fprintf(str,
					tabularShortPercentLanguageFormatBodyNoComplexity,
					pct(summary.Count, sumFiles),
					pct(summary.Lines, sumLines),
					pct(summary.Blank, sumBlank),
					pct(summary.Comment, sumComment),
					pct(summary.Code, sumCode),
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
			// Guard the divide: a tree of only empty files has zero lines, and an
			// unguarded float division prints DRYness as ∞. Matches the guard
			// snapshotULOC already applies to the same calculation.
			dryness := 0.0
			if sumLines > 0 {
				dryness = float64(len(ulocGlobalCount)) / float64(sumLines)
			}
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

// pct returns value as a percentage of total, guarding the total==0 case so
// the tabular --percent output never prints NaN% for a category that has no
// lines (e.g. a project whose files have zero blank/comment/complexity lines).
// Mirrors the total!=0 guard already used by addLanguagePercentages for JSON.
func pct(value, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(value) / float64(total) * 100
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
