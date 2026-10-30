// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/boyter/scc/v4/processor"
)

func TestEnvSplitStringShebangCLI(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"normal":   "#!/usr/bin/env python3\nprint(1)\n",
		"split":    "#!/usr/bin/env -S python3 -u\nprint(2)\n",
		"attached": "#!/usr/bin/env -Spython3 -u\nprint(3)\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := runSCCDir(t, dir, noGlobalConfig, "--no-config", "--by-file", "--format", "json", ".")
	if err != nil {
		t.Fatalf("scc: %v\n%s", err, out)
	}
	var got []processor.LanguageSummary
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "Python" || got[0].Count != 3 {
		t.Fatalf("want three Python files, got %s", out)
	}
}
