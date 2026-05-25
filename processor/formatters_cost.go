// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	glanguage "golang.org/x/text/language"
	gmessage "golang.org/x/text/message"
)

func calculateCocomoSLOCCount(sumCode int64, str *strings.Builder) {
	estimatedEffort := EstimateEffort(int64(sumCode), EAF)
	estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
	estimatedPeopleRequired := 0.0
	if estimatedScheduleMonths > 0 {
		estimatedPeopleRequired = estimatedEffort / estimatedScheduleMonths
	}
	estimatedCost := EstimateCost(estimatedEffort, AverageWage, Overhead)

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	_, _ = p.Fprintf(str, "Total Physical Source Lines of Code (SLOC)                     = %d\n", sumCode)
	_, _ = p.Fprintf(str, "Development Effort Estimate, Person-Years (Person-Months)      = %.2f (%.2f)\n", estimatedEffort/12, estimatedEffort)
	_, _ = p.Fprintf(str, " (Basic COCOMO model, Person-Months = %.2f*(KSLOC**%.2f)*%.2f)\n", projectType[CocomoProjectType][0], projectType[CocomoProjectType][1], EAF)
	_, _ = p.Fprintf(str, "Schedule Estimate, Years (Months)                              = %.2f (%.2f)\n", estimatedScheduleMonths/12, estimatedScheduleMonths)
	_, _ = p.Fprintf(str, " (Basic COCOMO model, Months = %.2f*(person-months**%.2f))\n", projectType[CocomoProjectType][2], projectType[CocomoProjectType][3])
	_, _ = p.Fprintf(str, "Estimated Average Number of Developers (Effort/Schedule)       = %.2f\n", estimatedPeopleRequired)
	_, _ = p.Fprintf(str, "Total Estimated Cost to Develop                                = %s%.0f\n", CurrencySymbol, estimatedCost)
	_, _ = p.Fprintf(str, " (average salary = %s%d/year, overhead = %.2f)\n", CurrencySymbol, AverageWage, Overhead)
}

func calculateCocomo(sumCode int64, str *strings.Builder) {
	estimatedCost, estimatedScheduleMonths, estimatedPeopleRequired := esstimateCostScheduleMonths(sumCode)

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	_, _ = p.Fprintf(str, "Estimated Cost to Develop (%s) %s%d\n", CocomoProjectType, CurrencySymbol, int64(estimatedCost))
	_, _ = p.Fprintf(str, "Estimated Schedule Effort (%s) %.2f months\n", CocomoProjectType, estimatedScheduleMonths)
	if math.IsNaN(estimatedPeopleRequired) {
		_, _ = p.Fprintf(str, "Estimated People Required 1 Grandparent\n")
	} else {
		_, _ = p.Fprintf(str, "Estimated People Required (%s) %.2f\n", CocomoProjectType, estimatedPeopleRequired)
	}
}

func esstimateCostScheduleMonths(sumCode int64) (float64, float64, float64) {
	estimatedEffort := EstimateEffort(int64(sumCode), EAF)
	estimatedCost := EstimateCost(estimatedEffort, AverageWage, Overhead)
	estimatedScheduleMonths := EstimateScheduleMonths(estimatedEffort)
	estimatedPeopleRequired := 0.0
	if estimatedScheduleMonths > 0 {
		estimatedPeopleRequired = estimatedEffort / estimatedScheduleMonths
	}
	return estimatedCost, estimatedScheduleMonths, estimatedPeopleRequired
}

func calculateLocomo(sumCode, sumComplexity int64, str *strings.Builder) {
	result := LocomoEstimate(sumCode, sumComplexity)

	p := gmessage.NewPrinter(glanguage.Make(os.Getenv("LANG")))

	_, _ = p.Fprintf(str, "LOCOMO LLM Cost Estimate (%s)\n", result.Preset)
	_, _ = p.Fprintf(str, "  Tokens Required (in/out) %.1fM / %.1fM\n", result.InputTokens/1_000_000, result.OutputTokens/1_000_000)
	_, _ = p.Fprintf(str, "  Cost to Generate %s%.0f\n", CurrencySymbol, result.Cost)
	_, _ = p.Fprintf(str, "  Estimated Cycles %.1f\n", result.IterationFactor)

	if result.GenerationSeconds > 86400 {
		_, _ = p.Fprintf(str, "  Generation Time (serial) %.1f days\n", result.GenerationSeconds/86400)
	} else if result.GenerationSeconds > 3600 {
		_, _ = p.Fprintf(str, "  Generation Time (serial) %.1f hours\n", result.GenerationSeconds/3600)
	} else {
		_, _ = p.Fprintf(str, "  Generation Time (serial) %.1f minutes\n", result.GenerationSeconds/60)
	}

	_, _ = p.Fprintf(str, "  Human Review Time %.1f hours\n", result.ReviewHours)
	str.WriteString("  Disclaimer: rough ballpark for regenerating code using a LLM.\n")
	str.WriteString("  Does not account for context reuse, test generation, or heavy debugging.\n")
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
		_, _ = fmt.Fprintf(str, "Processed %d bytes, %s megabytes (%s)\n", sumBytes, `¯\_(ツ)_/¯`, strings.ToUpper(SizeUnit))
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
		_, _ = fmt.Fprintf(str, "Processed %d bytes, %.3f megabytes (%s)\n", sumBytes, size, strings.ToUpper(SizeUnit))
	}
}

func isLeapYear(year int) bool {
	leapFlag := false
	if year%4 == 0 {
		if year%100 == 0 {
			leapFlag = year%400 == 0
		} else {
			leapFlag = true
		}
	}
	return leapFlag
}
