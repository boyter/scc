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

// LocomoPreset defines the pricing and throughput for an LLM tier.
// Presets are tier-based (large/medium/small/local) rather than model-specific,
// so they don't go stale as specific models are retired or renamed.
type LocomoPreset struct {
	Name        string
	InputPrice  float64 // cost per 1M input tokens
	OutputPrice float64 // cost per 1M output tokens
	TPS         float64 // output tokens per second
}

var locomoPresets = map[string]LocomoPreset{
	// large: frontier models (Claude Opus, GPT-4.5, Gemini Ultra, etc.)
	"large": {Name: "large", InputPrice: 10.00, OutputPrice: 30.00, TPS: 30},
	// medium: balanced models (Claude Sonnet, GPT-4o, Gemini Pro, etc.) — default
	"medium": {Name: "medium", InputPrice: 3.00, OutputPrice: 15.00, TPS: 50},
	// small: fast/cheap models (Claude Haiku, GPT-4o-mini, Gemini Flash, etc.)
	"small": {Name: "small", InputPrice: 0.50, OutputPrice: 2.00, TPS: 100},
	// local: self-hosted models (Llama, Mistral, etc.) — no API cost
	"local": {Name: "local", InputPrice: 0.00, OutputPrice: 0.00, TPS: 15},
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

// GetLocomoPreset returns the preset for the given name, falling back to medium
func GetLocomoPreset(name string) LocomoPreset {
	p, ok := locomoPresets[strings.ToLower(name)]
	if ok {
		return p
	}
	return locomoPresets["medium"]
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
