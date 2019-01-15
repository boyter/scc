package processor

import (
	"testing"
)

func TestProcessConstants(t *testing.T) {
	Trace = true
	ProcessConstants()

	if len(ExtensionToLanguage) == 0 {
		t.Error("Should not be 0")
	}

	if len(LanguageFeatures) == 0 {
		t.Error("Should not be 0")
	}
}

func TestConfigureGc(t *testing.T) {
	ConfigureGc()
}

func TestConfigureLazy(t *testing.T) {
	ConfigureLazy(true)
	if isLazy != true {
		t.Error("isLazy should be true")
	}

	ConfigureLazy(false)
	if isLazy != false {
		t.Error("isLazy should be false")
	}
}

func TestLoadLanguageFeature(t *testing.T) {
	isLazy = true
	LoadLanguageFeature("Go")
	_, ok := LanguageFeatures["Go"]

	if !ok {
		t.Error("Language should have been loaded")
	}
}

func TestLoadLanguageFeatureNew(t *testing.T) {
	isLazy = true
	LanguageFeatures = map[string]LanguageFeature{}
	LoadLanguageFeature("Go")
	LoadLanguageFeature("Go")

	_, ok := LanguageFeatures["Go"]

	if !ok {
		t.Error("Language should have been loaded")
	}

	isLazy = false
	ProcessConstants()
}

func TestProcessFlags(t *testing.T) {
	Debug = true
	More = true
	Complexity = true

	processFlags()

	if Complexity {
		t.Error("Complexity should be false")
	}
}

func TestPrintLanguages(t *testing.T) {
	printLanguages()
}

func TestProcess(t *testing.T) {
	Process()
}
