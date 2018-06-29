package processor

import (
	"testing"
)

func TestProcessConstants(t *testing.T) {
	ProcessConstants()

	if len(ExtensionToLanguage) == 0 {
		t.Error("Should not be 0")
	}

	if len(LanguageFeatures) == 0 {
		t.Error("Should not be 0")
	}
}
