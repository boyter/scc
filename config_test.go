// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"
)

// runSCCDir runs the test binary as scc in the given working directory, with
// optional extra environment entries (KEY=VALUE).
func runSCCDir(t *testing.T, dir string, env []string, args ...string) (string, error) {
	t.Helper()
	bin, err := filepath.Abs(sccBinPath)
	if err != nil {
		t.Fatal(err)
	}
	full := slices.Insert(slices.Clone(args), 0, sccTestFlag)
	cmd := exec.Command(bin, full...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = append(os.Environ(), env...)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestParseConfigArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		content         string
		allowPositional bool
		want            []string
	}{
		{
			name:    "single flag per line",
			content: "--no-cocomo\n--by-file\n",
			want:    []string{"--no-cocomo", "--by-file"},
		},
		{
			name:    "multiple tokens per line",
			content: "--exclude-dir vendor\n",
			want:    []string{"--exclude-dir", "vendor"},
		},
		{
			name:    "single quotes group a value",
			content: "--count-as 'jsp:html'\n",
			want:    []string{"--count-as", "jsp:html"},
		},
		{
			name:    "double quotes allow embedded single quote",
			content: "--count-as \"a'b\"\n",
			want:    []string{"--count-as", "a'b"},
		},
		{
			name:    "whole-line comment skipped",
			content: "# a comment\n--by-file\n",
			want:    []string{"--by-file"},
		},
		{
			name:    "inline trailing comment stripped",
			content: "--format wide   # comment\n",
			want:    []string{"--format", "wide"},
		},
		{
			name:    "blank lines produce no tokens",
			content: "\n\n--by-file\n\n   \n",
			want:    []string{"--by-file"},
		},
		{
			name:    "backslash is literal (windows path)",
			content: "--exclude-dir C:\\build\\out\n",
			want:    []string{"--exclude-dir", "C:\\build\\out"},
		},
		{
			name:    "crlf line endings",
			content: "--no-cocomo\r\n--by-file\r\n",
			want:    []string{"--no-cocomo", "--by-file"},
		},
		{
			name:            "positional line dropped when not allowed",
			content:         "src/\n--by-file\n",
			allowPositional: false,
			want:            []string{"--by-file"},
		},
		{
			name:            "positional line kept when allowed",
			content:         "src/\n--by-file\n",
			allowPositional: true,
			want:            []string{"src/", "--by-file"},
		},
		{
			name:    "empty quoted value still a token",
			content: "--report-title \"\"\n",
			want:    []string{"--report-title", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseConfigArgs(tc.content, tc.allowPositional)
			if !slices.Equal(got, tc.want) {
				t.Errorf("parseConfigArgs(%q, %v) = %q, want %q", tc.content, tc.allowPositional, got, tc.want)
			}
		})
	}
}

func TestPreScanConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		args         []string
		wantNoConfig bool
		wantFindRoot bool
		wantPath     string
	}{
		{name: "nothing", args: []string{"scc", "."}},
		{name: "no-config", args: []string{"scc", "--no-config"}, wantNoConfig: true},
		{name: "long find-root", args: []string{"scc", "--find-root"}, wantFindRoot: true},
		{name: "short find-root", args: []string{"scc", "-r"}, wantFindRoot: true},
		{name: "clustered -rv", args: []string{"scc", "-rv"}, wantFindRoot: true},
		{name: "clustered -vr", args: []string{"scc", "-vr"}, wantFindRoot: true},
		{name: "clustered -rzd", args: []string{"scc", "-rzd"}, wantFindRoot: true},
		{name: "non-r cluster", args: []string{"scc", "-vd"}},
		{name: "config space form", args: []string{"scc", "--config", "team.scc"}, wantPath: "team.scc"},
		{name: "config equals form", args: []string{"scc", "--config=team.scc"}, wantPath: "team.scc"},
		{name: "config as final token (typo)", args: []string{"scc", "--config"}},
		{name: "all three", args: []string{"scc", "--no-config", "-r", "--config=x"}, wantNoConfig: true, wantFindRoot: true, wantPath: "x"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			noConfig, findRoot, path := preScanConfig(tc.args)
			if noConfig != tc.wantNoConfig || findRoot != tc.wantFindRoot || path != tc.wantPath {
				t.Errorf("preScanConfig(%q) = (%v,%v,%q), want (%v,%v,%q)",
					tc.args, noConfig, findRoot, path, tc.wantNoConfig, tc.wantFindRoot, tc.wantPath)
			}
		})
	}
}

func TestMergeSliceDefault(t *testing.T) {
	t.Parallel()
	defaults := []string{".git", ".hg", ".svn"}

	if got := mergeSliceDefault(nil, defaults); !slices.Equal(got, defaults) {
		t.Errorf("empty set should fall back to defaults, got %q", got)
	}
	if got := mergeSliceDefault([]string{"vendor"}, defaults); !slices.Equal(got, []string{".git", ".hg", ".svn", "vendor"}) {
		t.Errorf("defaults should be preserved, got %q", got)
	}
	if got := mergeSliceDefault([]string{"vendor", "dist"}, defaults); !slices.Equal(got, []string{".git", ".hg", ".svn", "vendor", "dist"}) {
		t.Errorf("union should be order-stable, got %q", got)
	}
	if got := mergeSliceDefault([]string{".git", "vendor"}, defaults); !slices.Equal(got, []string{".git", ".hg", ".svn", "vendor"}) {
		t.Errorf("duplicate should be removed, got %q", got)
	}
}

// TestRegisterFlagsExhaustive asserts every flag registerFlags is meant to
// register is present in the resulting flag set. The write-only mode (§5.2) must
// give every non-write flag a sink; a missing flag there is silent state
// corruption, so this check is the insurance.
func TestRegisterFlagsExhaustive(t *testing.T) {
	t.Parallel()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var out, report, multi string
	registerFlags(fs, &flagBindings{output: &out, report: &report, formatMulti: &multi})

	expected := []string{
		"character", "percent", "uloc", "dryness", "binary", "by-file", "ci",
		"no-ignore", "no-scc-ignore", "no-gitignore", "no-gitmodule", "count-ignore",
		"debug", "exclude-dir", "file-gc-count", "file-list-queue-size",
		"file-process-job-workers", "file-summary-job-queue-size",
		"directory-walker-job-workers", "format", "report", "report-skip",
		"report-title", "include-ext", "exclude-ext", "exclude-file", "languages",
		"avg-wage", "overhead", "eaf", "sloccount-format", "no-cocomo",
		"cocomo-project-type", "no-size", "no-hborder", "size-unit", "no-complexity",
		"no-duplicates", "min-gen", "min", "gen", "generated-markers", "no-min-gen",
		"no-min", "no-gen", "min-gen-line-length", "not-match", "output", "sort",
		"trace", "verbose", "wide", "no-large", "include-symlinks", "large-line-count",
		"large-byte-count", "count-as", "count-as-pattern", "format-multi",
		"sql-project", "remap-unknown", "remap-all", "currency-symbol", "locomo",
		"cost-comparison", "locomo-preset", "locomo-review", "locomo-config",
		"locomo-input-price", "locomo-output-price", "locomo-tps", "locomo-cycles",
		"hotspots", "by-author", "depth", "timeline", "buckets", "no-fold-authors",
		"mcp",
	}
	for _, name := range expected {
		if fs.Lookup(name) == nil {
			t.Errorf("registerFlags did not register --%s", name)
		}
	}
}

func TestRegisterFlagsSliceDefValue(t *testing.T) {
	t.Parallel()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	var out, report, multi string
	registerFlags(fs, &flagBindings{output: &out, report: &report, formatMulti: &multi})

	cases := map[string]string{
		"exclude-dir":       "[.git,.hg,.svn]",
		"exclude-file":      "[package-lock.json,Cargo.lock,yarn.lock,pubspec.lock,Podfile.lock,pnpm-lock.yaml]",
		"generated-markers": "[do not edit,<auto-generated />]",
	}
	for name, want := range cases {
		f := fs.Lookup(name)
		if f == nil {
			t.Fatalf("missing flag --%s", name)
		}
		if f.DefValue != want {
			t.Errorf("--%s DefValue = %q, want %q", name, f.DefValue, want)
		}
	}
}

// writeSccConfig creates a temp dir with a .scc file and returns the dir.
func writeSccConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".scc"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// a source file so output is meaningful
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestConfigProjectLoaded(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n--no-cocomo\n")
	out, err := runSCCDir(t, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("project .scc --format csv not applied, output:\n%s", out)
	}
	if strings.Contains(out, "Estimated Cost to Develop") {
		t.Errorf("project .scc --no-cocomo not applied, output:\n%s", out)
	}
}

func TestConfigNoWalkUp(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n")
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, sub, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("config should NOT be picked up from a subdirectory (no walk-up), output:\n%s", out)
	}
}

func TestConfigPrecedenceCLIWins(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n")
	out, err := runSCCDir(t, dir, nil, "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "[{") {
		t.Errorf("CLI --format json should override config csv, output:\n%s", out)
	}
}

func TestConfigNeverWritesFile(t *testing.T) {
	cases := []string{
		"-o evil.csv\n",
		"--output evil.csv\n",
		"-do evil.csv\n", // clustered: -d + -o (headline case)
		"-do/evil.csv\n", // clustered, no space
		"--report=evil.csv\n",
		"--format-multi tabular:stdout,csv:evil.csv\n",
	}
	for _, content := range cases {
		dir := writeSccConfig(t, content)
		_, err := runSCCDir(t, dir, nil)
		if err != nil {
			t.Fatalf("scc errored for config %q: %v", content, err)
		}
		if _, statErr := os.Stat(filepath.Join(dir, "evil.csv")); statErr == nil {
			t.Errorf("config %q caused a file write (evil.csv exists)", content)
		}
	}
}

func TestConfigPresentCLICanWrite(t *testing.T) {
	dir := writeSccConfig(t, "--no-cocomo\n")
	target := filepath.Join(dir, "good.csv")
	_, err := runSCCDir(t, dir, nil, "-f", "csv", "-o", target)
	if err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("genuine CLI -o should write the file even with config present")
	}
}

func TestConfigControlFlagWithWriteFlag(t *testing.T) {
	// scc --config x.scc -o out.csv: the CLI-only parse must accept --config so
	// -o still writes (§5.2).
	dir := t.TempDir()
	cfg := filepath.Join(dir, "team.scc")
	if err := os.WriteFile(cfg, []byte("--no-cocomo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "out.csv")
	_, err := runSCCDir(t, dir, nil, "--config", cfg, "-f", "csv", "-o", target)
	if err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("scc --config x.scc -o out.csv should still write out.csv")
	}
}

func TestConfigEnvVarGlobal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(dir, "global.scc")
	if err := os.WriteFile(global, []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, []string{SccConfigEnv + "=" + global})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("SCC_CONFIG_PATH global not applied, output:\n%s", out)
	}

	// set-but-empty is treated as unset (tabular default)
	out, err = runSCCDir(t, dir, []string{SccConfigEnv + "="})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("empty SCC_CONFIG_PATH should be treated as unset, output:\n%s", out)
	}
}

func TestConfigNoConfigComposition(t *testing.T) {
	// --config x.scc --no-config loads exactly x.scc and skips the project .scc.
	dir := writeSccConfig(t, "--by-file\n") // project .scc would set --by-file
	global := filepath.Join(dir, "iso.scc")
	if err := os.WriteFile(global, []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, nil, "--config", global, "--no-config")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--config should still load with --no-config, output:\n%s", out)
	}
	// --by-file from the project .scc must NOT apply under --no-config: the
	// per-file CSV header carries a Location column, the summary one does not.
	if strings.Contains(out, "Location") {
		t.Errorf("--no-config should skip the project .scc, output:\n%s", out)
	}
}

func TestConfigInsideFileIsInert(t *testing.T) {
	// A --config written inside a project .scc must not chain-load another file.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	chained := filepath.Join(dir, "chained.scc")
	if err := os.WriteFile(chained, []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".scc"), []byte("--config "+chained+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--config inside a .scc must be inert (no chain-load), output:\n%s", out)
	}
}

func TestConfigUnreadableExplicitErrors(t *testing.T) {
	dir := t.TempDir()
	out, err := runSCCDir(t, dir, nil, "--config", filepath.Join(dir, "does-not-exist.scc"))
	if err == nil {
		t.Fatalf("scc should exit non-zero for an unreadable --config, output:\n%s", out)
	}
	if !strings.Contains(out, "could not read config") {
		t.Errorf("expected a clear error for unreadable --config, output:\n%s", out)
	}
}

func TestConfigCoupledMinFlags(t *testing.T) {
	// config --min + CLI --no-min must resolve exactly as without the feature:
	// the CLI-only write parse must not re-fire the coupled min/gen closures.
	dir := writeSccConfig(t, "--min\n")
	// --no-min on the CLI wins (last); the run should still succeed and not crash.
	if _, err := runSCCDir(t, dir, nil, "--no-min"); err != nil {
		t.Fatalf("coupled min flags run failed: %v", err)
	}
}

func TestConfigUnknownFlagAttribution(t *testing.T) {
	dir := writeSccConfig(t, "--not-a-real-flag\n")
	out, err := runSCCDir(t, dir, nil)
	if err == nil {
		t.Fatalf("unknown flag in config should exit non-zero, output:\n%s", out)
	}
	if !strings.Contains(out, "project") || !strings.Contains(out, "unknown flag --not-a-real-flag") {
		t.Errorf("unknown config flag should be attributed to the project file, output:\n%s", out)
	}
}

func TestConfigCompletionUnaffected(t *testing.T) {
	// scc completion --shell bash in a dir containing ./.scc must still emit
	// completions (config prepend must not shift args[1]).
	dir := writeSccConfig(t, "--format csv\n")
	out, err := runSCCDir(t, dir, nil, "completion", "--shell", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bash completion") && !strings.Contains(out, "complete -") {
		t.Errorf("completion should still work with ./.scc present, output:\n%s", out[:min(len(out), 200)])
	}
}

func TestConfigInsideAtFileHonored(t *testing.T) {
	// --config inside an @file IS honored (asymmetry vs config files, §4): @file
	// replaces os.Args before the pre-scan, so it is part of the genuine CLI.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(dir, "from-atfile.scc")
	if err := os.WriteFile(global, []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	atFile := filepath.Join(dir, "flags.txt")
	if err := os.WriteFile(atFile, []byte("--config "+global+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, nil, "@"+atFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--config inside an @file should be honored, output:\n%s", out)
	}
}

func TestNoConfigFastPathWritesFile(t *testing.T) {
	// No config present: -o on the CLI writes exactly as today (fast path).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "out.csv")
	if _, err := runSCCDir(t, dir, nil, "-f", "csv", "-o", target); err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("no-config fast path: CLI -o should write the file")
	}
}
