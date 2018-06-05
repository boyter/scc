// SPDX-License-Identifier: MIT

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const constantsFile = "./processor/constants.go"

func fatalf(f string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, f+"\n", v...)
	os.Exit(1)
}

// Reads all .json files in the current folder
// and encodes them as strings literals in constants.go
func generateConstants() error {
	files, _ := ioutil.ReadDir(".")
	out, err := ioutil.TempFile(".", "temp_constants")
	if err != nil {
		return fmt.Errorf("failed to open temp file: %v", err)
	}
	defer os.Remove(out.Name())

	// Open constants
	out.Write([]byte("package processor \n\nconst (\n"))

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
			out.Write([]byte(strings.TrimSuffix(f.Name(), ".json") + " = `"))

			enc := base64.NewEncoder(base64.StdEncoding, out)
			if _, err := io.Copy(enc, f); err != nil {
				return fmt.Errorf("failed to encode file '%s': %v", f.Name(), err)
			}
			enc.Close()

			out.Write([]byte("`\n"))
		}
	}

	// Close out constants
	out.Write([]byte(")\n"))
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
