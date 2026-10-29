// SPDX-License-Identifier: MIT

package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCountStatsTopazLispex(t *testing.T) {
	tests := []struct {
		language, filename string
		code, comment      int64
	}{
		{"Topaz", "topaz.tpz", 12, 6},
		{"Lispex", "lispex.lspx", 10, 8},
	}
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata", "studiohaze", tt.filename))
			if err != nil {
				t.Fatal(err)
			}
			lines, code, comment, blank := countOne(t, tt.language, string(content))
			if lines != 21 || code != tt.code || comment != tt.comment || blank != 3 {
				t.Fatalf("got lines/code/comments/blanks %d/%d/%d/%d, want 21/%d/%d/3",
					lines, code, comment, blank, tt.code, tt.comment)
			}
			for _, filename := range []string{tt.filename, strings.ToUpper(tt.filename)} {
				languages, _ := DetectLanguage(filename)
				if len(languages) != 1 || languages[0] != tt.language {
					t.Errorf("DetectLanguage(%q) = %v, want only %s", filename, languages, tt.language)
				}
			}
		})
	}
}
