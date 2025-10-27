package processor

import (
	"cmp"
	"slices"
	"strings"

	"github.com/agnivade/levenshtein"
	"github.com/spf13/pflag"
)

const SimilarStringThreshold float64 = 0.8

// StringSimilarRatio calculates the similarity ratio between s1 and s2.
//
// The function is based on the Levenshtein distance. The ratio is calculated using the formula:
//
// 1 - (Levenshtein Distance / length of the longer string).
//
// It returns a float64 between 0.0 (completely dissimilar) and 1.0 (identical).
// Based on experience, a ratio >= [SimilarStringThreshold] can generally be
// considered to indicate that two strings are highly similar.
//
// Note: The comparison is case-sensitive. For example, "hello" and "HELLO"
// will be treated as completely different strings (their similarity ratio is 0).
func StringSimilarRatio(s1, s2 string) float64 {
	distance := levenshtein.ComputeDistance(s1, s2)
	return 1 - float64(distance)/float64(max(len(s1), len(s2)))
}

type similarFlag struct {
	name  string
	ratio float64
}

func GetMostSimilarFlags(flagSet *pflag.FlagSet, flag string) []string {
	similarFlags := make([]similarFlag, 0)
	flagSet.VisitAll(func(f *pflag.Flag) {
		if len(f.Name) <= 1 {
			return
		}
		ratio := StringSimilarRatio(flag, f.Name)
		if ratio >= SimilarStringThreshold {
			similarFlags = append(similarFlags, similarFlag{
				name:  f.Name,
				ratio: ratio,
			})
		}
	})
	if len(similarFlags) == 0 {
		return []string{}
	}

	slices.SortFunc(similarFlags, func(a, b similarFlag) int {
		result := cmp.Compare(b.ratio, a.ratio)
		if result != 0 {
			return result
		}
		return strings.Compare(a.name, b.name)
	})
	result := make([]string, 0, len(similarFlags))
	for _, sc := range similarFlags {
		result = append(result, sc.name)
	}
	return result
}
