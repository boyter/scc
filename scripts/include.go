// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"maps"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/boyter/scc/v3/processor"
	jsoniter "github.com/json-iterator/go"
)

const constantsFile = "./processor/constants.go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//go:embed languages.tmpl
var langTemplate string

// Reads all .json files in the current folder
// and encodes them as strings literals in constants.go
func generateConstants() error {
	files, _ := os.ReadDir(".")
	buf := &bytes.Buffer{}

	langs := map[string]processor.Language{}

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "languages") && strings.HasSuffix(f.Name(), ".json") {
			f, err := os.Open(f.Name())
			if err != nil {
				return fmt.Errorf("failed to open file '%s': %v", f.Name(), err)
			}
			defer func(file *os.File) {
				_ = file.Close()
			}(f)

			data := map[string]processor.Language{}

			// validate the json by decoding into an empty struct
			if err := json.NewDecoder(f).Decode(&data); err != nil {
				return fmt.Errorf("failed to validate json in file '%s': %v", f.Name(), err)
			}

			maps.Insert(langs, maps.All(data))
		}
	}

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

	out, err := os.OpenFile(constantsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
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

func main() {
	if err := generateConstants(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate constants: %v\n", err)
		os.Exit(1)
	}
}
