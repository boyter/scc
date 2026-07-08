// SPDX-License-Identifier: MIT

package processor

import (
	jsoniter "github.com/json-iterator/go"
)

func addLanguagePercentages(language []LanguageSummary) {
	var sumFiles, sumLines, sumCode, sumComment, sumBlank, sumComplexity, sumBytes int64
	for _, l := range language {
		sumFiles += l.Count
		sumLines += l.Lines
		sumCode += l.Code
		sumComment += l.Comment
		sumBlank += l.Blank
		sumComplexity += l.Complexity
		sumBytes += l.Bytes
	}

	percent := func(value, total int64) *float64 {
		var p float64
		if total != 0 {
			p = float64(value) / float64(total) * 100
		}
		return &p
	}

	for i := range language {
		language[i].FilePercent = percent(language[i].Count, sumFiles)
		language[i].LinePercent = percent(language[i].Lines, sumLines)
		language[i].CodePercent = percent(language[i].Code, sumCode)
		language[i].CommentPercent = percent(language[i].Comment, sumComment)
		language[i].BlankPercent = percent(language[i].Blank, sumBlank)
		language[i].ComplexityPercent = percent(language[i].Complexity, sumComplexity)
		language[i].BytePercent = percent(language[i].Bytes, sumBytes)
	}
}

func toJSON(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	if Percent {
		addLanguagePercentages(language)
	}

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	jsonString, _ := json.Marshal(language)

	printDebugF("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime)

	return string(jsonString)
}

type Json2 struct {
	LanguageSummary         []LanguageSummary `json:"languageSummary"`
	EstimatedCost           float64           `json:"estimatedCost"`
	EstimatedScheduleMonths float64           `json:"estimatedScheduleMonths"`
	EstimatedPeople         float64           `json:"estimatedPeople"`

	// LOCOMO fields (only populated when --locomo or --cost-comparison is enabled)
	EstimatedLLMCost                  *float64 `json:"estimatedLLMCost,omitempty"`
	EstimatedLLMInputTokens           *float64 `json:"estimatedLLMInputTokens,omitempty"`
	EstimatedLLMOutputTokens          *float64 `json:"estimatedLLMOutputTokens,omitempty"`
	EstimatedLLMGenerationSeconds     *float64 `json:"estimatedLLMGenerationSeconds,omitempty"`
	EstimatedLLMReviewHours           *float64 `json:"estimatedLLMReviewHours,omitempty"`
	EstimatedLLMPreset                *string  `json:"estimatedLLMPreset,omitempty"`
	EstimatedLLMAverageComplexityMult *float64 `json:"estimatedLLMAverageComplexityMultiplier,omitempty"`
	EstimatedLLMCycles                *float64 `json:"estimatedLLMCycles,omitempty"`
}

func toJSON2(input chan *FileJob) string {
	startTime := makeTimestampMilli()
	language := aggregateLanguageSummary(input)
	language = sortLanguageSummary(language)

	if Percent {
		addLanguagePercentages(language)
	}

	var sumCode, sumComplexity int64
	for _, l := range language {
		sumCode += l.Code
		sumComplexity += l.Complexity
	}

	cost, schedule, people := esstimateCostScheduleMonths(sumCode)

	j2 := Json2{
		LanguageSummary:         language,
		EstimatedCost:           cost,
		EstimatedScheduleMonths: schedule,
		EstimatedPeople:         people,
	}

	if Locomo {
		result := LocomoEstimate(sumCode, sumComplexity)
		j2.EstimatedLLMCost = &result.Cost
		j2.EstimatedLLMInputTokens = &result.InputTokens
		j2.EstimatedLLMOutputTokens = &result.OutputTokens
		j2.EstimatedLLMGenerationSeconds = &result.GenerationSeconds
		j2.EstimatedLLMReviewHours = &result.ReviewHours
		j2.EstimatedLLMPreset = &result.Preset
		j2.EstimatedLLMAverageComplexityMult = &result.AverageComplexityMult
		j2.EstimatedLLMCycles = &result.IterationFactor
	}

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	jsonString, _ := json.Marshal(j2)

	printDebugF("milliseconds to build formatted string: %d", makeTimestampMilli()-startTime)

	return string(jsonString)
}
