package processor

import (
	"slices"
	"testing"

	"github.com/spf13/pflag"
)

func TestStringSimilarRatio(t *testing.T) {
	testCases := []struct {
		s1, s2    string
		isSimilar bool
	}{
		{
			s1:        "",
			s2:        "",
			isSimilar: false,
		},
		{
			s1:        "",
			s2:        "hello",
			isSimilar: false,
		},
		{
			s1:        "hello",
			s2:        "",
			isSimilar: false,
		},
		{
			s1:        "hello",
			s2:        "hello",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "Hello",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "helle",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "hallo",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "helo",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "hell",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "heelo",
			isSimilar: true,
		},
		{
			s1:        "uloc",
			s2:        "ulc",
			isSimilar: true,
		},
		{
			s1:        "uloc",
			s2:        "ulcc",
			isSimilar: true,
		},
		{
			s1:        "uloc",
			s2:        "ulocc",
			isSimilar: true,
		},
		{
			s1:        "--ci",
			s2:        "--cii",
			isSimilar: true,
		},
		{
			s1:        "hello",
			s2:        "hello-world",
			isSimilar: false,
		},
		{
			s1:        "hello",
			s2:        "how",
			isSimilar: false,
		},
		{
			s1:        "hello",
			s2:        "HELLO",
			isSimilar: false,
		},
		{
			s1:        "python",
			s2:        "golang",
			isSimilar: false,
		},
	}
	for _, tc := range testCases {
		result := StringSimilarRatio(tc.s1, tc.s2) >= SimilarStringThreshold
		if result != tc.isSimilar {
			t.Errorf("StringSimilarRatio(%q, %q) failed, got %v, want %v", tc.s1, tc.s2, result, tc.isSimilar)
		}
	}
}

func TestGetMostSimilarFlags(t *testing.T) {
	flags := pflag.NewFlagSet("testing", pflag.ExitOnError)
	_ = flags.Bool("no-ignore", false, "test")
	_ = flags.Bool("no-gitignore", false, "test")
	_ = flags.Int("no-gitmodule", 0, "test")
	_ = flags.String("format", "", "test")
	_ = flags.String("uloc", "", "test")
	_ = flags.String("gen", "", "test")
	_ = flags.String("ci", "", "test")

	testCases := []struct {
		name    string
		expects []string
	}{
		{
			name:    "",
			expects: []string{},
		},
		{
			name:    "unknown",
			expects: []string{},
		},
		{
			name:    "no-gignore",
			expects: []string{"no-ignore", "no-gitignore"},
		},
		{
			name:    "no-gitmodyle",
			expects: []string{"no-gitmodule"},
		},
		{
			name:    "formet",
			expects: []string{"format"},
		},
		{
			name:    "ulc",
			expects: []string{"uloc"},
		},
		{
			name:    "gan",
			expects: []string{"gen"},
		},
		{
			name:    "cii",
			expects: []string{"ci"},
		},
		// these two are not included in the set, but still need to be matched
		{
			name:    "vrsion",
			expects: []string{"version"},
		},
		{
			name:    "hellp",
			expects: []string{"help"},
		},
	}
	for _, tc := range testCases {
		result := GetMostSimilarFlags(flags, tc.name)
		if !slices.Equal(result, tc.expects) {
			t.Errorf("got: %v, want: %v", result, tc.expects)
		}
	}
}
