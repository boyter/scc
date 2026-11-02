// SPDX-License-Identifier: MIT

package processor

import (
	"math"
)

// Basic COCOMO Params from Boehm
//
// Organic – A software project is said to be an organic type if the team size required is adequately small, the
// problem is well understood and has been solved in the past and also the team members have a nominal experience
// regarding the problem.
// Semi-detached – A software project is said to be a Semi-detached type if the vital characteristics such as team-size,
// experience, knowledge of the various programming environment lie in between that of organic and Embedded.
// The projects classified as Semi-Detached are comparatively less familiar and difficult to develop compared to
// the organic ones and require more experience and better guidance and creativity. Eg: Compilers or
// different Embedded Systems can be considered of Semi-Detached type.
// Embedded – A software project with requiring the highest level of complexity, creativity, and experience
// requirement fall under this category. Such software requires a larger team size than the other two models
// and also the developers need to be sufficiently experienced and creative to develop such complex models.
var projectType = map[string][]float64{
	"organic":       {2.4, 1.05, 2.5, 0.38},
	"semi-detached": {3.0, 1.12, 2.5, 0.35},
	"embedded":      {3.6, 1.20, 2.5, 0.32},
}

// EstimateCost calculates the cost in dollars applied using generic COCOMO weighted values based
// on the average yearly wage
func EstimateCost(effortApplied float64, averageWage int64, overhead float64) float64 {
	return effortApplied * float64(averageWage) / 12 * overhead
}

// cocomoCoefficients returns the four Basic COCOMO coefficients [a, b, c, d]
// for the configured project type. It guards the projectType lookup so an
// unknown or malformed project type can never yield a nil/short slice and
// panic the indexers below — falling back to the organic model instead.
func cocomoCoefficients() []float64 {
	if v, ok := projectType[CocomoProjectType]; ok && len(v) >= 4 {
		return v
	}
	return projectType["organic"]
}

// EstimateEffort calculate the effort applied using generic COCOMO weighted values
func EstimateEffort(sloc int64, eaf float64) float64 {
	c := cocomoCoefficients()
	var effortApplied = c[0] * math.Pow(float64(sloc)/1000, c[1]) * eaf
	return effortApplied
}

// EstimateScheduleMonths estimates the effort in months based on the result from EstimateEffort
func EstimateScheduleMonths(effortApplied float64) float64 {
	c := cocomoCoefficients()
	return c[2] * math.Pow(effortApplied, c[3])
}
