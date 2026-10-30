// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/boyter/scc/v4/processor"
)

func TestExplicitFileLanguageDetection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		path, content, language string
	}{
		{"a/.hooks/pre-commit", "#!/usr/bin/env bash\necho hello\n", "BASH"},
		{"release.v1/tool", "#!/usr/bin/env python3\nprint(1)\n", "Python"},
		{"build/Makefile", "all:\n\techo hello\n", "Makefile"},
		{"build/CMakeLists.txt", "project(hello)\n", "CMake"},
		{"release.v1/main.go", "package main\n", "Go"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			for _, mode := range []string{"directory", "relative", "absolute"} {
				t.Run(mode, func(t *testing.T) {
					arg := path
					if mode == "directory" {
						arg = filepath.Dir(path)
					} else if mode == "relative" {
						arg = filepath.FromSlash(tc.path)
					}
					out, err := runSCCDir(t, dir, noGlobalConfig, "--no-config", "--by-file", "--format", "json", arg)
					if err != nil {
						t.Fatalf("scc: %v\n%s", err, out)
					}
					var languages []processor.LanguageSummary
					if err := json.Unmarshal([]byte(out), &languages); err != nil {
						t.Fatalf("decode output: %v\n%s", err, out)
					}
					if len(languages) != 1 || languages[0].Name != tc.language || len(languages[0].Files) != 1 {
						t.Fatalf("want one %s file, got %s", tc.language, out)
					}
					file := languages[0].Files[0]
					if file.Filename != filepath.Base(path) || file.Language != tc.language || file.Lines != int64(strings.Count(tc.content, "\n")) {
						t.Fatalf("incorrect file result: %+v", file)
					}
					wantLocation := path
					if mode == "relative" {
						wantLocation = arg
					}
					if filepath.Clean(file.Location) != filepath.Clean(wantLocation) {
						t.Fatalf("location = %q, want %q", file.Location, wantLocation)
					}
				})
			}
		})
	}
}
