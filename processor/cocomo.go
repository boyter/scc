package processor

import (
	"math"
)

// Calculate the cost in dollars applied using generic COCOMO2 weighted values based
// on the average yearly wage
func EstimateCost(effortApplied float64, averageWage int64) float64 {
	return effortApplied * float64(averageWage/12) * float64(1.8)
}

// Calculate the effort applied using generic COCOMO2 weighted values
func EstimateEffort(sloc int64) float64 {
	var eaf float64 = 1

	// Numbers based on organic project, small team, good experience working with requirements
	var effortApplied float64 = float64(3.2) * math.Pow(float64(sloc)/1000, 1.05) * eaf
	return effortApplied
}

func EstimateScheduleMonths(effortApplied float64) float64 {
	// Numbers based on organic project small team, good experience working with requirements
	return float64(2.5) * math.Pow(effortApplied, 0.38)
}
