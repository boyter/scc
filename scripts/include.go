// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/boyter/scc/v3/processor"
	jsoniter "github.com/json-iterator/go"
)

const (
	constantsFile     = "./processor/constants.go"
	languagesListFile = "./LANGUAGES.md"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//go:embed languages.tmpl
var langTemplate string

// loadLanguages reads and validates all languages*.json files in the current
// folder, returning the merged language database. Both generated artifacts are
// produced from this single freshly parsed map so that one generation pass is
// fully consistent. Previously generateLanguagesList read the languageDatabase
// compiled into this binary, which is the *old* constants.go (compiled before
// this run rewrote it), so LANGUAGES.md always lagged the JSON by one pass.
func loadLanguages() (map[string]processor.Language, error) {
	files, _ := os.ReadDir(".")
	langs := map[string]processor.Language{}

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "languages") && strings.HasSuffix(f.Name(), ".json") {
			file, err := os.Open(f.Name())
			if err != nil {
				return nil, fmt.Errorf("failed to open file '%s': %v", f.Name(), err)
			}

			data := map[string]processor.Language{}

			// validate the json by decoding into an empty struct
			if err := json.NewDecoder(file).Decode(&data); err != nil {
				_ = file.Close()
				return nil, fmt.Errorf("failed to validate json in file '%s': %v", f.Name(), err)
			}
			_ = file.Close()

			// validate that every regex heuristic compiles ahead of time so a
			// broken pattern fails the build rather than at runtime
			for name, lang := range data {
				for _, h := range lang.Heuristics {
					if _, err := regexp.Compile(h.Pattern); err != nil {
						return nil, fmt.Errorf("invalid heuristic regex %q for language '%s' in file '%s': %v", h.Pattern, name, f.Name(), err)
					}
				}
			}

			maps.Insert(langs, maps.All(data))
		}
	}

	return langs, nil
}

// encodes the language database as string literals in constants.go
func generateConstants(langs map[string]processor.Language) error {
	buf := &bytes.Buffer{}

	t, err := template.New("codeGenerator").Funcs(template.FuncMap{
		"quote": strconv.Quote,
	}).Parse(langTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template file: %v", err)
	}

	if err := t.Execute(buf, langs); err != nil {
		return fmt.Errorf("failed to execute template file: %v", err)
	}

	source, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("failed to format code: %v", err)
	}

	out, err := os.OpenFile(constantsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open constants file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(out)

	if _, err := out.Write(source); err != nil {
		return fmt.Errorf("failed to write constants file %v", err)
	}

	return nil
}

// generateLanguagesList writes LANGUAGES.md from the freshly parsed language
// map. It mirrors the formatting of processor.PrintLanguages but works on the
// passed in data rather than the compiled-in languageDatabase so a single
// generation pass stays consistent with constants.go.
func generateLanguagesList(langs map[string]processor.Language) error {
	out, err := os.OpenFile(languagesListFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open languages list file: %v", err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(out)

	names := make([]string, 0, len(langs))
	for name := range langs {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	_, _ = out.WriteString("```\n")
	for _, name := range names {
		exts := append(append([]string{}, langs[name].Extensions...), langs[name].FileNames...)
		_, _ = fmt.Fprintf(out, "%s (%s)\n", name, strings.Join(exts, ","))
	}
	_, _ = out.WriteString("```\n")

	return nil
}

func main() {
	langs, err := loadLanguages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load languages: %v\n", err)
		os.Exit(1)
	}
	if err := generateConstants(langs); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate constants: %v\n", err)
		os.Exit(1)
	}
	if err := generateLanguagesList(langs); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate languages list: %v\n", err)
		os.Exit(1)
	}
}
