// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"

	jsoniter "github.com/json-iterator/go"
)

const constantsFile = "./processor/constants.go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Quote copy from processor/structs.go for code generation
type Quote struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	IgnoreEscape bool   `json:"ignoreEscape"`
	DocString    bool   `json:"docString"`
}

// Language copy from processor/structs.go for code generation
type Language struct {
	LineComment      []string   `json:"line_comment"`
	ComplexityChecks []string   `json:"complexitychecks"`
	Extensions       []string   `json:"extensions"`
	ExtensionFile    bool       `json:"extensionFile"`
	MultiLine        [][]string `json:"multi_line"`
	Quotes           []Quote    `json:"quotes"`
	NestedMultiLine  bool       `json:"nestedmultiline"`
	Keywords         []string   `json:"keywords"`
	FileNames        []string   `json:"filenames"`
	SheBangs         []string   `json:"shebangs"`
}

func formatLanguage(l Language) Language {
	ret := Language{
		LineComment:      slices.Clone(l.LineComment),
		ComplexityChecks: slices.Clone(l.ComplexityChecks),
		Extensions:       slices.Clone(l.Extensions),
		ExtensionFile:    l.ExtensionFile,
		Quotes:           slices.Clone(l.Quotes),
		NestedMultiLine:  l.NestedMultiLine,
		Keywords:         slices.Clone(l.Keywords),
		FileNames:        slices.Clone(l.FileNames),
		SheBangs:         slices.Clone(l.SheBangs),
	}
	ret.MultiLine = make([][]string, len(l.MultiLine))
	for i := range l.MultiLine {
		ret.MultiLine[i] = slices.Clone(l.MultiLine[i])
	}

	for i := range ret.LineComment {
		ret.LineComment[i] = strconv.Quote(ret.LineComment[i])
	}
	for i := range ret.ComplexityChecks {
		ret.ComplexityChecks[i] = strconv.Quote(ret.ComplexityChecks[i])
	}
	for i := range ret.Extensions {
		ret.Extensions[i] = strconv.Quote(ret.Extensions[i])
	}
	for i := range ret.MultiLine {
		for j := range ret.MultiLine[i] {
			ret.MultiLine[i][j] = strconv.Quote(ret.MultiLine[i][j])
		}
	}
	for i := range ret.Quotes {
		ret.Quotes[i].Start = strconv.Quote(ret.Quotes[i].Start)
		ret.Quotes[i].End = strconv.Quote(ret.Quotes[i].End)
	}
	for i := range ret.Keywords {
		ret.Keywords[i] = strconv.Quote(ret.Keywords[i])
	}
	for i := range ret.FileNames {
		ret.FileNames[i] = strconv.Quote(ret.FileNames[i])
	}
	for i := range ret.SheBangs {
		ret.SheBangs[i] = strconv.Quote(ret.SheBangs[i])
	}
	return ret
}

//go:embed languages.tmpl
var langTemplate string

// Reads all .json files in the current folder
// and encodes them as strings literals in constants.go
func generateConstants() error {
	files, _ := os.ReadDir(".")
	buf := &bytes.Buffer{}

	langs := map[string]Language{}

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "languages") && strings.HasSuffix(f.Name(), ".json") {
			f, err := os.Open(f.Name())
			if err != nil {
				return fmt.Errorf("failed to open file '%s': %v", f.Name(), err)
			}
			defer f.Close()

			data := map[string]Language{}

			// validate the json by decoding into an empty struct
			if err := json.NewDecoder(f).Decode(&data); err != nil {
				return fmt.Errorf("failed to validate json in file '%s': %v", f.Name(), err)
			}

			for k, v := range data {
				langs[k] = formatLanguage(v)
			}
		}
	}

	t, err := template.New("codeGenerator").Parse(langTemplate)
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

	out, err := os.OpenFile(constantsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open constants file: %v", err)
	}
	defer out.Close()

	if _, err := out.Write(source); err != nil {
		return fmt.Errorf("failed to write constants file %v", err)
	}

	return nil
}

func main() {
	if err := generateConstants(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate constants: %v\n", err)
		os.Exit(1)
	}
}
