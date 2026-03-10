// SPDX-License-Identifier: MIT

package processor

import (
	"math"
	"testing"
)

func TestLocomoComplexityDensityZeroCode(t *testing.T) {
	got := LocomoComplexityDensity(10, 0)
	if got != 0 {
		t.Errorf("Expected 0 for zero code lines, got %f", got)
	}
}

func TestLocomoComplexityDensity(t *testing.T) {
	got := LocomoComplexityDensity(30, 100)
	if math.Abs(got-0.3) > 0.001 {
		t.Errorf("Expected 0.3, got %f", got)
	}
}

func TestLocomoComplexityFactor(t *testing.T) {
	// density 0.3, weight 5 → 1 + sqrt(0.3)*5 ≈ 1 + 0.5477*5 ≈ 3.738
	got := LocomoComplexityFactor(0.3, 5)
	if got < 3.7 || got > 3.8 {
		t.Errorf("Expected ~3.74, got %f", got)
	}
}

func TestLocomoComplexityFactorLowDensity(t *testing.T) {
	// density 0.05, weight 5 → 1 + sqrt(0.05)*5 ≈ 1 + 0.2236*5 ≈ 2.118
	got := LocomoComplexityFactor(0.05, 5)
	if got < 2.0 || got > 2.2 {
		t.Errorf("Expected ~2.12, got %f", got)
	}
}

func TestLocomoIterationFactor(t *testing.T) {
	// density 0.3, base 1.5, weight 2 → 1.5 + sqrt(0.3)*2 ≈ 1.5 + 1.095 ≈ 2.595
	got := LocomoIterationFactor(0.3, 1.5, 2)
	if got < 2.5 || got > 2.7 {
		t.Errorf("Expected ~2.60, got %f", got)
	}
}

func TestLocomoIterationFactorLowDensity(t *testing.T) {
	// density 0.05, base 1.5, weight 2 → 1.5 + sqrt(0.05)*2 ≈ 1.5 + 0.447 ≈ 1.947
	got := LocomoIterationFactor(0.05, 1.5, 2)
	if got < 1.9 || got > 2.0 {
		t.Errorf("Expected ~1.95, got %f", got)
	}
}

func TestLocomoEstimateBasic(t *testing.T) {
	// Reset to defaults
	LocomoPresetName = "medium"
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoCyclesSet = false

	result := LocomoEstimate(1000, 100)

	// density = 0.1, complexityFactor ≈ 2.58, iterationFactor ≈ 2.13
	// outputTokens = 1000 * 10 * 2.13 = 21322
	// inputTokens = 1000 * 20 * 2.58 * 2.13 = 109929
	// cost = (109929/1M * 3) + (21322/1M * 15) ≈ 0.33 + 0.32 ≈ 0.65

	if result.OutputTokens <= 0 {
		t.Error("Expected positive output tokens")
	}
	if result.InputTokens <= 0 {
		t.Error("Expected positive input tokens")
	}
	if result.Cost <= 0 {
		t.Error("Expected positive cost")
	}
	if result.GenerationSeconds <= 0 {
		t.Error("Expected positive generation time")
	}
	if result.ReviewHours <= 0 {
		t.Error("Expected positive review hours")
	}
	if result.AverageComplexityMult <= 1 {
		t.Error("Expected complexity multiplier > 1")
	}
	if result.Preset != "medium" {
		t.Errorf("Expected preset medium, got %s", result.Preset)
	}
}

func TestLocomoEstimateZeroCode(t *testing.T) {
	LocomoPresetName = "medium"
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01

	result := LocomoEstimate(0, 0)

	if result.OutputTokens != 0 {
		t.Errorf("Expected 0 output tokens for zero code, got %f", result.OutputTokens)
	}
	if result.Cost != 0 {
		t.Errorf("Expected 0 cost for zero code, got %f", result.Cost)
	}
}

func TestLocomoEstimateHighComplexity(t *testing.T) {
	LocomoPresetName = "medium"
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01

	low := LocomoEstimate(1000, 50)   // density 0.05
	high := LocomoEstimate(1000, 300) // density 0.3

	if high.Cost <= low.Cost {
		t.Error("Higher complexity should produce higher cost")
	}

	// The ratio should be in a reasonable range (not 12x like v1 linear would produce)
	ratio := high.Cost / low.Cost
	if ratio > 5 {
		t.Errorf("Cost ratio between high and low complexity seems too high: %f", ratio)
	}
}

func TestLocomoEstimateLocalLlama(t *testing.T) {
	LocomoPresetName = "local"
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01

	result := LocomoEstimate(1000, 100)

	if result.Cost != 0 {
		t.Errorf("Expected 0 cost for local, got %f", result.Cost)
	}
	if result.GenerationSeconds <= 0 {
		t.Error("Expected positive generation time even for local model")
	}
}

func TestGetLocomoPresetUnknown(t *testing.T) {
	p := GetLocomoPreset("nonexistent")
	if p.Name != "medium" {
		t.Errorf("Expected fallback to medium, got %s", p.Name)
	}
}

func TestParseLocomoConfig(t *testing.T) {
	var a, b, c, d, e float64
	parseLocomoConfig("8,15,3,2.0,1.5", &a, &b, &c, &d, &e)

	if a != 8 || b != 15 || c != 3 || d != 2.0 || e != 1.5 {
		t.Errorf("Config parsing failed: got %f,%f,%f,%f,%f", a, b, c, d, e)
	}
}

func TestParseLocomoConfigInvalid(t *testing.T) {
	a, b, c, d, e := 10.0, 20.0, 5.0, 1.5, 2.0
	parseLocomoConfig("bad,config", &a, &b, &c, &d, &e)

	// Should remain unchanged
	if a != 10 || b != 20 || c != 5 || d != 1.5 || e != 2 {
		t.Error("Invalid config should not change defaults")
	}
}

func TestLocomoEstimateIterationFactorPopulated(t *testing.T) {
	LocomoPresetName = "medium"
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoCyclesSet = false
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01

	result := LocomoEstimate(1000, 100)

	if result.IterationFactor <= 0 {
		t.Error("Expected positive IterationFactor")
	}
	// density = 0.1, iFactor = 1.5 + sqrt(0.1)*2 ≈ 2.13
	if result.IterationFactor < 2.0 || result.IterationFactor > 2.3 {
		t.Errorf("Expected IterationFactor ~2.13, got %f", result.IterationFactor)
	}
}

func TestLocomoCyclesOverride(t *testing.T) {
	LocomoPresetName = "medium"
	LocomoConfig = ""
	LocomoInputPriceSet = false
	LocomoOutputPriceSet = false
	LocomoTPSSet = false
	LocomoTokensPerLine = 10
	LocomoBaseInputPerLine = 20
	LocomoComplexityWeight = 5
	LocomoIterations = 1.5
	LocomoIterationWeight = 2
	LocomoReviewMinutesPerLine = 0.01

	// First get the default result
	LocomoCyclesSet = false
	defaultResult := LocomoEstimate(1000, 100)

	// Now override with 100 cycles
	LocomoCyclesSet = true
	LocomoCyclesOverride = 100
	overrideResult := LocomoEstimate(1000, 100)

	// Reset
	LocomoCyclesSet = false
	LocomoCyclesOverride = 0

	if overrideResult.IterationFactor != 100 {
		t.Errorf("Expected IterationFactor 100, got %f", overrideResult.IterationFactor)
	}
	if overrideResult.Cost <= defaultResult.Cost {
		t.Error("Expected override cost to be higher than default")
	}
	// Cost should scale roughly proportionally with cycles
	ratio := overrideResult.Cost / defaultResult.Cost
	if ratio < 30 || ratio > 70 {
		t.Errorf("Cost ratio seems off: %f (expected ~47x for 100 vs ~2.13 cycles)", ratio)
	}
}

func TestParseLocomoConfigPartialInvalid(t *testing.T) {
	a, b, c, d, e := 10.0, 20.0, 5.0, 1.5, 2.0
	parseLocomoConfig("8,15,bad,2.0,1.5", &a, &b, &c, &d, &e)

	// Should remain unchanged since one part failed
	if a != 10 || b != 20 || c != 5 || d != 1.5 || e != 2 {
		t.Error("Partially invalid config should not change defaults")
	}
}
