// SPDX-License-Identifier: MIT

package main

import (
	"bytes"
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
	buf := &bytes.Buffer{}

	// Open constants
	_, _ = buf.WriteString("package processor\n\nconst (\n")

	for _, f := range files {
		if strings.HasPrefix(f.Name(), "languages") && strings.HasSuffix(f.Name(), ".json") {
			f, err := os.Open(f.Name())
			if err != nil {
				return fmt.Errorf("failed to open file '%s': %v", f.Name(), err)
			}
			defer f.Close()

			// validate the json by decoding into an empty struct
			if err := json.NewDecoder(f).Decode(&struct{}{}); err != nil {
				return fmt.Errorf("failed to validate json in file '%s': %v", f.Name(), err)
			}

			// Reset position
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				return fmt.Errorf("failed to reset file position '%s': %v", f.Name(), err)
			}

			// The constant variable name
			_ = buf.WriteByte('\t')
			_, _ = buf.WriteString(strings.TrimSuffix(f.Name(), ".json") + " = `")

			enc := base64.NewEncoder(base64.StdEncoding, buf)
			gz, _ := gzip.NewWriterLevel(enc, gzip.BestSpeed)
			if _, err := io.Copy(gz, f); err != nil {
				return fmt.Errorf("failed to encode file '%s': %v", f.Name(), err)
			}
			gz.Close()
			enc.Close()

			_, _ = buf.WriteString("`\n")
		}
	}

	// Close out constants
	_, _ = buf.WriteString(")\n")

	out, err := os.OpenFile(constantsFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open constants file: %v", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, buf); err != nil {
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
