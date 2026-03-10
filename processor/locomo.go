// SPDX-License-Identifier: MIT

package processor

import (
	"math"
	"strconv"
	"strings"
)

// LOCOMO — LLM Output COst MOdel
// Estimates the cost to regenerate a codebase using an LLM.
// Analogous to COCOMO for human development — a rough ballpark estimator.

// LocomoPreset defines the pricing and throughput for a specific LLM model
type LocomoPreset struct {
	Name        string
	InputPrice  float64 // cost per 1M input tokens
	OutputPrice float64 // cost per 1M output tokens
	TPS         float64 // output tokens per second
}

var locomoPresets = map[string]LocomoPreset{
	"claude-sonnet": {Name: "claude-sonnet", InputPrice: 3.00, OutputPrice: 15.00, TPS: 50},
	"claude-haiku":  {Name: "claude-haiku", InputPrice: 0.80, OutputPrice: 4.00, TPS: 100},
	"gpt-4o":        {Name: "gpt-4o", InputPrice: 2.50, OutputPrice: 10.00, TPS: 50},
	"gpt-4o-mini":   {Name: "gpt-4o-mini", InputPrice: 0.15, OutputPrice: 0.60, TPS: 100},
	"local-llama":   {Name: "local-llama", InputPrice: 0.00, OutputPrice: 0.00, TPS: 15},
}

// LocomoResult holds the computed estimates from the LOCOMO model
type LocomoResult struct {
	InputTokens              float64
	OutputTokens             float64
	Cost                     float64
	GenerationSeconds        float64
	ReviewHours              float64
	AverageComplexityMult    float64
	Preset                   string
}

// GetLocomoPreset returns the preset for the given name, falling back to claude-sonnet
func GetLocomoPreset(name string) LocomoPreset {
	p, ok := locomoPresets[strings.ToLower(name)]
	if ok {
		return p
	}
	return locomoPresets["claude-sonnet"]
}

// LocomoComplexityDensity calculates complexity/code with a guard for division by zero
func LocomoComplexityDensity(complexity, code int64) float64 {
	if code == 0 {
		return 0
	}
	return float64(complexity) / float64(code)
}

// LocomoComplexityFactor calculates the input token scaling factor based on complexity density
// Uses sqrt scaling to prevent runaway compounding
func LocomoComplexityFactor(complexityDensity, complexityWeight float64) float64 {
	return 1 + math.Sqrt(complexityDensity)*complexityWeight
}

// LocomoIterationFactor calculates the iteration/retry multiplier based on complexity density
// Uses sqrt scaling to prevent runaway compounding
func LocomoIterationFactor(complexityDensity, baseIterations, iterationWeight float64) float64 {
	return baseIterations + math.Sqrt(complexityDensity)*iterationWeight
}

// LocomoEstimate computes the full LOCOMO estimate for a project
func LocomoEstimate(sumCode, sumComplexity int64) LocomoResult {
	// Resolve preset, then apply any explicit overrides
	preset := GetLocomoPreset(LocomoPresetName)

	inputPrice := preset.InputPrice
	outputPrice := preset.OutputPrice
	tps := preset.TPS

	if LocomoInputPriceSet {
		inputPrice = LocomoInputPrice
	}
	if LocomoOutputPriceSet {
		outputPrice = LocomoOutputPrice
	}
	if LocomoTPSSet {
		tps = LocomoTPS
	}

	// Parse config overrides if provided
	tokensPerLine := LocomoTokensPerLine
	baseInputPerLine := LocomoBaseInputPerLine
	complexityWeight := LocomoComplexityWeight
	baseIterations := LocomoIterations
	iterationWeight := LocomoIterationWeight

	if LocomoConfig != "" {
		parseLocomoConfig(LocomoConfig, &tokensPerLine, &baseInputPerLine, &complexityWeight, &baseIterations, &iterationWeight)
	}

	density := LocomoComplexityDensity(sumComplexity, sumCode)
	cFactor := LocomoComplexityFactor(density, complexityWeight)
	iFactor := LocomoIterationFactor(density, baseIterations, iterationWeight)

	outputTokens := float64(sumCode) * tokensPerLine * iFactor
	inputTokens := float64(sumCode) * baseInputPerLine * cFactor * iFactor

	cost := (inputTokens/1_000_000)*inputPrice + (outputTokens/1_000_000)*outputPrice

	generationSeconds := 0.0
	if tps > 0 {
		generationSeconds = outputTokens / tps
	}

	reviewHours := float64(sumCode) * LocomoReviewMinutesPerLine / 60

	return LocomoResult{
		InputTokens:           inputTokens,
		OutputTokens:          outputTokens,
		Cost:                  cost,
		GenerationSeconds:     generationSeconds,
		ReviewHours:           reviewHours,
		AverageComplexityMult: cFactor * iFactor,
		Preset:                preset.Name,
	}
}

// parseLocomoConfig parses a comma-separated config string "tokensPerLine,baseInputPerLine,complexityWeight,iterations,iterationWeight"
func parseLocomoConfig(config string, tokensPerLine, baseInputPerLine, complexityWeight, iterations, iterationWeight *float64) {
	parts := strings.Split(config, ",")
	if len(parts) != 5 {
		return
	}

	vals := make([]float64, 5)
	for i, part := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			return // if any part fails, use defaults
		}
		vals[i] = f
	}

	*tokensPerLine = vals[0]
	*baseInputPerLine = vals[1]
	*complexityWeight = vals[2]
	*iterations = vals[3]
	*iterationWeight = vals[4]
}
