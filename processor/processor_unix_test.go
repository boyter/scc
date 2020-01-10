// +build linux darwin

package processor

import (
	"testing"
)

func TestScaleWorkersToLimit(t *testing.T) {
	scaleWorkersToLimit(10, 10)
}

func TestConfigureLimitsUnix(t *testing.T) {
	ConfigureLimitsUnix()
}
