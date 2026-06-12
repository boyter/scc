// SPDX-License-Identifier: MIT

package processor

import (
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

func TestParseCountAsPattern(t *testing.T) {
	testCases := []struct {
		in      string
		wantErr bool
		want    CountRule
	}{
		{
			in:   "glob:*_spec.rb:Ruby Spec:Ruby",
			want: CountRule{Engine: MatchGlob, Pattern: "*_spec.rb", Name: "Ruby Spec", BaseLanguage: "Ruby"},
		},
		{
			// no engine prefix defaults to glob
			in:   "*_spec.rb:Ruby Spec:Ruby",
			want: CountRule{Engine: MatchGlob, Pattern: "*_spec.rb", Name: "Ruby Spec", BaseLanguage: "Ruby"},
		},
		{
			in:   `re:\.test\.js$:JavaScript Tests:JavaScript`,
			want: CountRule{Engine: MatchRegex, Pattern: `\.test\.js$`, Name: "JavaScript Tests", BaseLanguage: "JavaScript"},
		},
		{
			// colons inside the pattern (non-capturing group) must be preserved
			in:   `re:(?:blah|foo)\.test\.js$:JS Tests:JavaScript`,
			want: CountRule{Engine: MatchRegex, Pattern: `(?:blah|foo)\.test\.js$`, Name: "JS Tests", BaseLanguage: "JavaScript"},
		},
		{in: "glob:*.test.js", wantErr: true},          // missing name and baselang
		{in: "glob:*.test.js:JS Tests", wantErr: true}, // missing baselang
		{in: "glob:*.test.js::Ruby", wantErr: true},    // empty name
		{in: "glob::JS Tests:Ruby", wantErr: true},     // empty pattern
	}

	for _, tc := range testCases {
		got, err := parseCountAsPattern(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseCountAsPattern(%q) expected error, got %+v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseCountAsPattern(%q) unexpected error: %s", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseCountAsPattern(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestGlobToRegex(t *testing.T) {
	testCases := []struct {
		glob string
		want string
	}{
		{"*_spec.rb", `^.*_spec\.rb$`},
		{"*.test.js", `^.*\.test\.js$`},
		{"foo?.go", `^foo.\.go$`},
	}

	for _, tc := range testCases {
		if got := globToRegex(tc.glob); got != tc.want {
			t.Errorf("globToRegex(%q) = %q, want %q", tc.glob, got, tc.want)
		}
	}
}

func TestSetupCountRules(t *testing.T) {
	defer func() {
		CountAsPattern = nil
		CountRules = nil
		compiledCountRules = nil
		delete(languageDatabase, "Ruby Spec")
	}()

	CountAsPattern = []string{"glob:*_spec.rb:Ruby Spec:Ruby"}
	CountRules = nil
	compiledCountRules = nil
	ProcessConstants()

	// The minted category must exist in the language database cloned from Ruby
	if _, ok := languageDatabase["Ruby Spec"]; !ok {
		t.Fatal("expected Ruby Spec to be registered in the language database")
	}

	// It must not pollute normal extension detection
	if langs, ok := ExtensionToLanguage["rb"]; ok {
		for _, l := range langs {
			if l == "Ruby Spec" {
				t.Error("Ruby Spec should not be registered against the rb extension")
			}
		}
	}

	// A compiled rule must be present so newFileJob can match it
	found := false
	for _, r := range compiledCountRules {
		if r.name == "Ruby Spec" {
			found = true
			if !r.re.MatchString("some/path/foo_spec.rb") {
				t.Error("compiled rule should match foo_spec.rb path")
			}
			if r.re.MatchString("some/path/foo.rb") {
				t.Error("compiled rule should not match foo.rb path")
			}
		}
	}
	if !found {
		t.Fatal("expected a compiled count rule for Ruby Spec")
	}
}

func TestSetupCountRulesUnknownBaseSkipped(t *testing.T) {
	defer func() {
		CountAsPattern = nil
		CountRules = nil
		compiledCountRules = nil
	}()

	CountAsPattern = []string{"glob:*_spec.rb:Ruby Spec:Nonexistent"}
	CountRules = nil
	compiledCountRules = nil
	ProcessConstants()

	if _, ok := languageDatabase["Ruby Spec"]; ok {
		t.Error("Ruby Spec should not be registered when base language is unknown")
	}
	if len(compiledCountRules) != 0 {
		t.Error("no compiled rule should be created for an unknown base language")
	}
}
