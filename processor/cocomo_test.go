// SPDX-License-Identifier: MIT

package processor

import (
	"math"
	"testing"
)

func TestEstimateCost(t *testing.T) {
	eff := EstimateEffort(26, 1)
	got := EstimateCost(eff, 56000, 2.4)

	// Should be around 582
	if got < 580 || got > 585 {
		t.Errorf("Got %f", got)
	}
}

func TestEstimateCostManyLines(t *testing.T) {
	eff := EstimateEffort(77873, 1)
	got := EstimateCost(eff, 56000, 2.4)

	// Should be around 2602469
	if got < 2602460 || got > 2602480 {
		t.Errorf("Got %f", got)
	}
}

func TestEstimateScheduleMonths(t *testing.T) {
	eff := EstimateEffort(537, 1)
	got := EstimateScheduleMonths(eff)

	// Should be around 2.7
	if got < 2.6 || got > 2.8 {
		t.Errorf("Got %f", got)
	}
}

// TestEstimateCostMonthlyWagePrecision guards the monthly-wage division: the annual
// wage must be converted to float BEFORE dividing by 12, otherwise an annual wage
// that is not evenly divisible by 12 (such as the shipped default --avg-wage 56286,
// whose monthly wage is 4690.5) is truncated to a whole number and every cost line
// is silently under-counted.
func TestEstimateCostMonthlyWagePrecision(t *testing.T) {
	eff := EstimateEffort(100000, 1) // organic, default EAF
	got := EstimateCost(eff, 56286, 2.4)
	// 56286/12 = 4690.5, NOT 4690.
	want := eff * (56286.0 / 12.0) * 2.4
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("EstimateCost truncated the monthly wage: got %v want %v (diff %v)", got, want, got-want)
	}
}
