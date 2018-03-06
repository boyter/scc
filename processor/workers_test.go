package processor

import (
	"testing"
)

func TestCountStats(t *testing.T) {
	fileJob := FileJob{}
	countStats(&fileJob)

	// if result != "da39a3ee5e6b4b0d3255bfef95601890afd80709" {
	// 	t.Errorf("Expect generatePackageVerificationCode to have da39a3ee5e6b4b0d3255bfef95601890afd80709 but got %q", result)
	// }
}
