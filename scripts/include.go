// SPDX-License-Identifier: MIT

package main

import (
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const constantsFile = "./processor/constants.go"

// Reads all .json files in the current folder
// and encodes them as strings literals in constants.go
func generateConstants() error {
	files, _ := os.ReadDir(".")
	out, err := os.CreateTemp(".", "temp_constants")
	if err != nil {
		return fmt.Errorf("failed to open temp file: %v", err)
	}
	defer os.Remove(out.Name())

	// Open constants
	out.WriteString("package processor \n\nconst (\n")

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "languages") && strings.HasSuffix(f.Name(), ".json") {
			f, err := os.Open(f.Name())
			if err != nil {
				return fmt.Errorf("failed to open file '%s': %v", f.Name(), err)
			}

			// validate the json by decoding into an empty struct
			if err := json.NewDecoder(f).Decode(&struct{}{}); err != nil {
				return fmt.Errorf("failed to validate json in file '%s': %v", f.Name(), err)
			}

			// Reset position
			f.Seek(0, io.SeekStart)

			// The constant variable name
			out.WriteString(strings.TrimSuffix(f.Name(), ".json") + " = `")

			enc := base64.NewEncoder(base64.StdEncoding, out)
			gz, _ := gzip.NewWriterLevel(enc, gzip.BestSpeed)
			if _, err := io.Copy(gz, f); err != nil {
				return fmt.Errorf("failed to encode file '%s': %v", f.Name(), err)
			}
			gz.Close()
			enc.Close()

			out.WriteString("`\n")
		}
	}

	// Close out constants
	out.WriteString(")\n")
	out.Close()

	if err := os.Rename(out.Name(), constantsFile); err != nil {
		return fmt.Errorf("%v", err)
	}

	return nil
}

func main() {
	if err := generateConstants(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate constants: %v\n", err)
		os.Exit(1)
	}
}
