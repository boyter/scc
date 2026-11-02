// SPDX-License-Identifier: MIT

package processor

import "testing"

// TestRemapLoadsLanguageFeatures guards the bug where --remap-all / --remap-unknown
// relabelled a file without ever loading the target language's features. In lazy
// mode (every CLI run) CountStats then found no features for the new language and
// counted the file as plain text: no comments, no complexity, and silently wrong
// numbers that depended on whether some other file had already loaded that language.
func TestRemapLoadsLanguageFeatures(t *testing.T) {
	savedLazy := isLazy
	savedAll, savedUnknown := RemapAll, RemapUnknown
	savedComplexity, savedDuplicates := Complexity, Duplicates
	defer func() {
		isLazy = savedLazy
		RemapAll, RemapUnknown = savedAll, savedUnknown
		Complexity, Duplicates = savedComplexity, savedDuplicates
	}()

	Complexity = false
	Duplicates = false
	ConfigureLazy(true)
	ProcessConstants()

	// Drop any already-built Go features so this exercises the remap path doing
	// the load rather than piggybacking on an earlier test having loaded them.
	LanguageFeaturesMutex.Lock()
	delete(LanguageFeatures, "Go")
	LanguageFeaturesMutex.Unlock()

	content := []byte("// a comment\npackage main\n\nfunc main() {\n\tif true {\n\t}\n}\n")

	for _, tc := range []struct {
		name         string
		all, unknown string
		possible     []string
	}{
		{name: "remap-all", all: "package main:Go", possible: []string{"Plain Text"}},
		{name: "remap-unknown", unknown: "package main:Go", possible: []string{SheBang}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := processorContext{remap: newRemapConfig(tc.all, tc.unknown)}

			job := &FileJob{
				Location:          "example.txt",
				Filename:          "example.txt",
				Extension:         "txt",
				PossibleLanguages: tc.possible,
				Content:           content,
				Bytes:             int64(len(content)),
			}

			if !ctx.processFile(job) {
				t.Fatal("processFile rejected the remapped file")
			}
			if job.Language != "Go" {
				t.Fatalf("expected language Go, got %q", job.Language)
			}
			if job.Comment == 0 {
				t.Error("remapped file counted 0 comments, so the Go features were never loaded")
			}
			if job.Complexity == 0 {
				t.Error("remapped file counted 0 complexity, so the Go features were never loaded")
			}
		})
	}
}

// TestLoadLanguageFeatureUnknownName pins the map lookup in LoadLanguageFeature.
// It used to range over languageDatabase and break on a match, which left an
// unknown name holding whatever language the iteration landed on last — so a
// bogus --remap-all target would silently borrow a random language's comment and
// complexity rules. A miss must register nothing at all.
func TestLoadLanguageFeatureUnknownName(t *testing.T) {
	savedLazy := isLazy
	defer func() { isLazy = savedLazy }()

	ConfigureLazy(true)
	ProcessConstants()

	const bogus = "NotARealLanguageName"
	LoadLanguageFeature(bogus)

	LanguageFeaturesMutex.Lock()
	_, ok := LanguageFeatures[bogus]
	LanguageFeaturesMutex.Unlock()

	if ok {
		t.Errorf("LoadLanguageFeature(%q) registered features for a language that does not exist", bogus)
	}
}
