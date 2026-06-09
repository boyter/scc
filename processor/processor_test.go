// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func TestProcessConstantsPathExclude(t *testing.T) {
	PathDenyList = []string{"testing/"}
	ProcessConstants()

	if PathDenyList[0] != "testing" {
		t.Error("expected / to be trimmed")
	}

	PathDenyList = []string{}
}

func TestConfigureGc(t *testing.T) {
	ConfigureGc()
}

func TestConfigureLazy(t *testing.T) {
	ConfigureLazy(true)
	if !isLazy {
		t.Error("isLazy should be true")
	}

	ConfigureLazy(false)
	if isLazy {
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
	result := &strings.Builder{}
	PrintLanguages(result)
	if !strings.Contains(result.String(), "Go Template (tmpl,gohtml,gotxt)\n") {
		t.Fatal("printLanguages test failed")
	}
}

func TestProcess(t *testing.T) {
	Process()
}

func TestSetupCountAsLanguage(t *testing.T) {
	ProcessConstants()
	CountAs = "boyter:C Header"
	setupCountAs()
	v := ExtensionToLanguage["boyter"]

	if v[0] != "C Header" {
		t.Error("Expected boyter to map to C Header")
	}

	CountAs = ""
}

func TestSetupCountAsLanguageCase(t *testing.T) {
	ProcessConstants()
	CountAs = "BoYtER:C Header"
	setupCountAs()
	v := ExtensionToLanguage["boyter"]

	if v[0] != "C Header" {
		t.Error("Expected boyter to map to C Header")
	}

	CountAs = ""
}

func TestSetupCountAsExtension(t *testing.T) {
	ProcessConstants()
	CountAs = "boyter:j2"
	setupCountAs()
	v := ExtensionToLanguage["boyter"]

	if v[0] != "Jinja" {
		t.Error("Expected boyter to map to Jinja")
	}

	CountAs = ""
}

func TestSetupCountAsMultiple(t *testing.T) {
	ProcessConstants()
	CountAs = "boyter:j2,retyob:JAVA"
	setupCountAs()
	v := ExtensionToLanguage["boyter"]

	if v[0] != "Jinja" {
		t.Error("Expected boyter to map to Jinja")
	}

	v = ExtensionToLanguage["retyob"]

	if v[0] != "Java" {
		t.Error("Expected retyob to map to Java")
	}

	CountAs = ""
}
func TestLanguagesFileIsValidJSON(t *testing.T) {
	data, err := os.ReadFile("../languages.json")
	if err != nil {
		t.Fatal("failed to read languages.json", err)
	}

	var languages map[string]Language
	if err := json.Unmarshal(data, &languages); err != nil {
		t.Error("languages.json is not valid JSON", err)
	}
}

func TestLanguagesFileInvalidJSON(t *testing.T) {
	dir, _ := os.MkdirTemp("", "test-languages")
	langFile := filepath.Join(dir, "languages.json")
	_ = os.WriteFile(langFile, []byte(`not valid json`), 0644)

	LanguagesFile = langFile
	// should not panic
	ProcessConstants()
}
