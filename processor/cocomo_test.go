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

	// Should be around 2602469. The yearly wage is divided by 12 in floating
	// point (56000/12 = 4666.666...); the prior code truncated it to 4666 via
	// integer division, which underestimated the cost by ~$373 here.
	if got < 2602400 || got > 2602500 {
		t.Errorf("Got %f", got)
	}
}

// TestEstimateCostTable pins the monthly-wage conversion. The yearly wage must
// be divided by 12 as a floating-point value so a wage that is not a clean
// multiple of 12 keeps its fractional month. The expected values below are
// derived independently of EstimateCost (plain arithmetic on literals); the
// "wage not divisible by 12" case would fail under the old integer-division
// code, which truncated 4666.666... down to 4666.
func TestEstimateCostTable(t *testing.T) {
	tests := []struct {
		name          string
		effortApplied float64
		averageWage   int64
		overhead      float64
		want          float64
	}{
		{"zero effort", 0, 56286, 2.4, 0},
		{"zero overhead", 1, 56286, 0, 0},
		{"wage divisible by 12", 1, 60000, 1, 60000.0 / 12},
		{"wage not divisible by 12", 1, 56000, 1, 56000.0 / 12},
		{"default wage one person-month", 1, 56286, 2.4, 56286.0 / 12 * 2.4},
		{"default wage twelve person-months", 12, 56286, 2.4, 56286.0 * 2.4},
		{"two person-months fractional wage", 2, 56000, 1.5, 2 * 56000.0 / 12 * 1.5},
	}

	const relTol = 1e-9
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.effortApplied, tt.averageWage, tt.overhead)
			if math.Abs(got-tt.want) > relTol*(1+math.Abs(tt.want)) {
				t.Errorf("EstimateCost(%v, %d, %v) = %v, want %v",
					tt.effortApplied, tt.averageWage, tt.overhead, got, tt.want)
			}
		})
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
