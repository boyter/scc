// SPDX-License-Identifier: MIT

package processor

import (
	"math"
	"strings"
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

// TestCocomoProjectTypeSelection is a table-driven regression test for the
// panic triggered by `scc --cocomo-project-type Organic`. projectType is keyed
// by the canonical lowercase name; before the fix, the membership check in
// ProcessConstants lower-cased the value only for the lookup while the real
// COCOMO indexers used the raw value, so any case variant of a built-in type
// (Organic, SEMI-DETACHED, Embedded, ...) indexed a nil slice and panicked.
// Each row drives ProcessConstants (selection) then EstimateEffort/
// EstimateScheduleMonths (lookup) and must not panic.
func TestCocomoProjectTypeSelection(t *testing.T) {
	// ProcessConstants may register a custom projectType entry; both
	// functions also read the CocomoProjectType package global, so snapshot
	// and restore both to keep the test hermetic. (No test in this package
	// calls t.Parallel, so sequential mutation is safe.)
	savedType := cloneProjectTypeMap(projectType)
	savedCocomo := CocomoProjectType
	defer func() {
		projectType = savedType
		CocomoProjectType = savedCocomo
	}()

	// Baselines computed against the canonical lowercase keys (the trusted
	// reference). Case-variant rows must reproduce these exactly, which proves
	// the right coefficients are selected rather than silently falling back.
	CocomoProjectType = "organic"
	organicEffort := EstimateEffort(537, 1)
	organicSchedule := EstimateScheduleMonths(organicEffort)
	CocomoProjectType = "semi-detached"
	semiEffort := EstimateEffort(537, 1)
	semiSchedule := EstimateScheduleMonths(semiEffort)
	CocomoProjectType = "embedded"
	embeddedEffort := EstimateEffort(537, 1)
	embeddedSchedule := EstimateScheduleMonths(embeddedEffort)

	// Sanity: the three built-ins must produce distinct numbers, otherwise the
	// per-type numeric assertions below would be vacuous.
	if !(organicEffort != semiEffort && semiEffort != embeddedEffort && organicEffort != embeddedEffort) {
		t.Fatalf("baselines not distinct: organic=%v semi=%v embedded=%v", organicEffort, semiEffort, embeddedEffort)
	}

	// Independent oracle for the custom 5-tuple: compute with its own
	// (non-organic) coefficients so the row proves the custom tuple is selected,
	// not a fallback. eaf is 1 here, matching EstimateEffort(537, 1).
	customABCD := []float64{3.6, 1.20, 2.5, 0.38}
	customEffort := customABCD[0] * math.Pow(537.0/1000, customABCD[1])
	customSchedule := customABCD[2] * math.Pow(customEffort, customABCD[3])
	if customEffort == organicEffort {
		t.Fatalf("custom baseline coincides with organic — row would be vacuous")
	}

	tests := []struct {
		name       string
		input      string
		wantNorm   string  // expected CocomoProjectType after ProcessConstants
		wantEffort float64 // expected EstimateEffort(537, 1); 0 skips the check
		wantSched  float64 // expected EstimateScheduleMonths; 0 skips the check
	}{
		{
			name:       "organic lowercase canonical",
			input:      "organic",
			wantNorm:   "organic",
			wantEffort: organicEffort,
			wantSched:  organicSchedule,
		},
		{
			name:       "Organic capitalized panics before fix",
			input:      "Organic",
			wantNorm:   "organic",
			wantEffort: organicEffort,
			wantSched:  organicSchedule,
		},
		{
			name:       "ORGANIC all-caps",
			input:      "ORGANIC",
			wantNorm:   "organic",
			wantEffort: organicEffort,
			wantSched:  organicSchedule,
		},
		{
			name:       "Semi-Detached mixed case resolves to semi-detached",
			input:      "Semi-Detached",
			wantNorm:   "semi-detached",
			wantEffort: semiEffort,
			wantSched:  semiSchedule,
		},
		{
			name:       "Embedded capitalized resolves to embedded",
			input:      "Embedded",
			wantNorm:   "embedded",
			wantEffort: embeddedEffort,
			wantSched:  embeddedSchedule,
		},
		{
			name:       "unknown type falls back to organic",
			input:      "does-not-exist",
			wantNorm:   "organic",
			wantEffort: organicEffort,
			wantSched:  organicSchedule,
		},
		{
			name:       "custom 5-tuple with non-organic coefficients",
			input:      "custom,3.6,1.20,2.5,0.38",
			wantNorm:   "custom,3.6,1.20,2.5,0.38",
			wantEffort: customEffort,
			wantSched:  customSchedule,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CocomoProjectType = tt.input
			ProcessConstants()

			if CocomoProjectType != tt.wantNorm {
				t.Fatalf("CocomoProjectType = %q, want %q", CocomoProjectType, tt.wantNorm)
			}

			// Recover so any panic is contained to this subtest and reported as a
			// clean failure. For the case-variant rows the normalization assertion
			// above is what first catches the original regression; this net guards
			// the custom/unknown rows where a future guard regression could panic.
			var effort, sched float64
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("COCOMO lookup panicked for %q: %v", tt.input, r)
					}
				}()
				effort = EstimateEffort(537, 1)
				sched = EstimateScheduleMonths(effort)
			}()

			if tt.wantEffort != 0 && !floatApproxEqual(effort, tt.wantEffort) {
				t.Errorf("effort = %v, want %v", effort, tt.wantEffort)
			}
			if tt.wantSched != 0 && !floatApproxEqual(sched, tt.wantSched) {
				t.Errorf("schedule = %v, want %v", sched, tt.wantSched)
			}
		})
	}
}

// TestCocomoCoefficientsGuard proves the defense-in-depth guard in
// cocomoCoefficients: even if a caller sets an unknown project type and never
// runs ProcessConstants (so no normalization/fallback runs), the COCOMO
// functions fall back to organic instead of panicking on a nil slice.
func TestCocomoCoefficientsGuard(t *testing.T) {
	saved := CocomoProjectType
	defer func() { CocomoProjectType = saved }()

	organicEffort := EstimateEffort(537, 1)
	organicSchedule := EstimateScheduleMonths(organicEffort)

	CocomoProjectType = "totally-bogus-type"

	var effort, sched float64
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("guarded lookup panicked for unknown type: %v", r)
			}
		}()
		effort = EstimateEffort(537, 1)
		sched = EstimateScheduleMonths(effort)
	}()

	if !floatApproxEqual(effort, organicEffort) {
		t.Errorf("unknown type effort = %v, want organic %v", effort, organicEffort)
	}
	if !floatApproxEqual(sched, organicSchedule) {
		t.Errorf("unknown type schedule = %v, want organic %v", sched, organicSchedule)
	}
}

// TestCalculateCocomoSLOCCountProjectType covers the --sloccount-format COCOMO
// path (calculateCocomoSLOCCount in formatters_cost.go) — the second panic site
// the fix touched, which had its own direct projectType[CocomoProjectType][0..3]
// indexings in the printed "Basic COCOMO model" lines. A case-variant project
// type and an unknown type must both render without panicking, emitting the
// organic coefficients via the cocomoCoefficients guard.
func TestCalculateCocomoSLOCCountProjectType(t *testing.T) {
	savedType := cloneProjectTypeMap(projectType)
	savedCocomo := CocomoProjectType
	defer func() {
		projectType = savedType
		CocomoProjectType = savedCocomo
	}()

	tests := []struct {
		name  string
		input string
	}{
		{name: "Organic case variant", input: "Organic"},
		{name: "unknown type falls back to organic", input: "does-not-exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CocomoProjectType = tt.input
			ProcessConstants()

			var str strings.Builder
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("calculateCocomoSLOCCount panicked for %q: %v", tt.input, r)
					}
				}()
				calculateCocomoSLOCCount(537, &str)
			}()

			out := str.String()
			if !strings.Contains(out, "Basic COCOMO model") {
				t.Fatalf("missing COCOMO model line in output:\n%s", out)
			}
			// Both rows resolve to organic, so the printed effort coefficients
			// must be organic's a,b = {2.4, 1.05}, produced via the guard rather
			// than a raw projectType[CocomoProjectType] lookup that would panic.
			if !strings.Contains(out, "Person-Months = 2.40*(KSLOC**1.05)") {
				t.Errorf("expected organic coefficients in output, got:\n%s", out)
			}
		})
	}
}

func cloneProjectTypeMap(m map[string][]float64) map[string][]float64 {
	out := make(map[string][]float64, len(m))
	for k, v := range m {
		// Deep-copy the slice so the snapshot is independent of any future
		// in-place mutation of the live map's values.
		cp := make([]float64, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

func floatApproxEqual(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9
}
