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

// noGlobalConfig forces SCC_CONFIG_PATH empty (treated as unset) so a value in
// the developer's environment cannot pollute config tests. Pass it as the env
// argument to runSCCDir for any test that does not set its own global config.
var noGlobalConfig = []string{SccConfigEnv + "="}

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
		{
			name:            "end-of-flags marker dropped in config",
			content:         "--\n--by-file\n",
			allowPositional: false,
			want:            []string{"--by-file"},
		},
		{
			name:            "end-of-flags marker dropped mid-line in config",
			content:         "--exclude-dir vendor --\n",
			allowPositional: false,
			want:            []string{"--exclude-dir", "vendor"},
		},
		{
			name:            "end-of-flags marker kept in @file",
			content:         "--\nsrc/\n",
			allowPositional: true,
			want:            []string{"--", "src/"},
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
		{name: "long find-root-config", args: []string{"scc", "--find-root-config"}, wantFindRoot: true},
		{name: "no -r shorthand", args: []string{"scc", "-r"}},
		{name: "-r no longer clusters into find-root", args: []string{"scc", "-rv"}},
		{name: "non-r cluster", args: []string{"scc", "-vd"}},
		{name: "config space form", args: []string{"scc", "--config", "team.scc"}, wantPath: "team.scc"},
		{name: "config equals form", args: []string{"scc", "--config=team.scc"}, wantPath: "team.scc"},
		{name: "config as final token (typo)", args: []string{"scc", "--config"}},
		{name: "all three", args: []string{"scc", "--no-config", "--find-root-config", "--config=x"}, wantNoConfig: true, wantFindRoot: true, wantPath: "x"},
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
// register is present in the resulting flag set. The write-only mode must
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
	out, err := runSCCDir(t, dir, noGlobalConfig)
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
	out, err := runSCCDir(t, sub, noGlobalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("config should NOT be picked up from a subdirectory (no walk-up), output:\n%s", out)
	}
}

func TestConfigPrecedenceCLIWins(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n")
	out, err := runSCCDir(t, dir, noGlobalConfig, "--format", "json")
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
		_, err := runSCCDir(t, dir, noGlobalConfig)
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
	_, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "-o", target)
	if err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("genuine CLI -o should write the file even with config present")
	}
}

func TestConfigControlFlagWithWriteFlag(t *testing.T) {
	// scc --config x.scc -o out.csv: the CLI-only parse must accept --config so
	// -o still writes.
	dir := t.TempDir()
	cfg := filepath.Join(dir, "team.scc")
	if err := os.WriteFile(cfg, []byte("--no-cocomo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "out.csv")
	_, err := runSCCDir(t, dir, noGlobalConfig, "--config", cfg, "-f", "csv", "-o", target)
	if err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("scc --config x.scc -o out.csv should still write out.csv")
	}
}

// TestConfigDashDashDoesNotDisableCLIFlags guards against a '--' end-of-flags
// marker in a config file. Config tokens are prepended, so an unstripped '--'
// would terminate flag parsing and turn the genuine CLI's flags into positional
// paths (here --by-file would be read as a file path).
func TestConfigDashDashDoesNotDisableCLIFlags(t *testing.T) {
	dir := writeSccConfig(t, "--\n")
	out, err := runSCCDir(t, dir, noGlobalConfig, "--by-file")
	if err != nil {
		t.Fatalf("scc errored: %v\n%s", err, out)
	}
	if strings.Contains(out, "could not be read") {
		t.Errorf("--by-file was treated as a path; '--' from config leaked through:\n%s", out)
	}
}

// TestConfigExplicitEmptyWriteFlagNoFalseWarn checks that an explicit empty
// write flag on the CLI (--output=) is recognised as "set on the CLI" and does
// not trigger the "ignoring --output from config" warning, while a config-only
// write still does warn.
func TestConfigExplicitEmptyWriteFlagNoFalseWarn(t *testing.T) {
	dir := writeSccConfig(t, "--output evil.csv\n")
	const warn = "ignoring --output from config"

	// CLI explicitly sets --output= (empty -> stdout): the flag WAS set on the
	// command line, so no warning should fire.
	out, err := runSCCDir(t, dir, noGlobalConfig, "--output=")
	if err != nil {
		t.Fatalf("scc errored: %v\n%s", err, out)
	}
	if strings.Contains(out, warn) {
		t.Errorf("explicit --output= on CLI should not warn about config:\n%s", out)
	}

	// Control: config-only write (no CLI override) should still warn.
	out, err = runSCCDir(t, dir, noGlobalConfig)
	if err != nil {
		t.Fatalf("scc errored: %v\n%s", err, out)
	}
	if !strings.Contains(out, warn) {
		t.Errorf("config-only --output should still warn, got:\n%s", out)
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
	out, err := runSCCDir(t, dir, noGlobalConfig, "--config", global, "--no-config")
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
	out, err := runSCCDir(t, dir, noGlobalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--config inside a .scc must be inert (no chain-load), output:\n%s", out)
	}
}

func TestConfigUnreadableExplicitErrors(t *testing.T) {
	dir := t.TempDir()
	out, err := runSCCDir(t, dir, noGlobalConfig, "--config", filepath.Join(dir, "does-not-exist.scc"))
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
	if _, err := runSCCDir(t, dir, noGlobalConfig, "--no-min"); err != nil {
		t.Fatalf("coupled min flags run failed: %v", err)
	}
}

func TestConfigUnknownFlagAttribution(t *testing.T) {
	dir := writeSccConfig(t, "--not-a-real-flag\n")
	out, err := runSCCDir(t, dir, noGlobalConfig)
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
	out, err := runSCCDir(t, dir, noGlobalConfig, "completion", "--shell", "bash")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "bash completion") && !strings.Contains(out, "complete -") {
		t.Errorf("completion should still work with ./.scc present, output:\n%s", out[:min(len(out), 200)])
	}
}

func TestConfigInsideAtFileHonored(t *testing.T) {
	// --config inside an @file IS honored (asymmetry vs config files): @file
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
	out, err := runSCCDir(t, dir, noGlobalConfig, "@"+atFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--config inside an @file should be honored, output:\n%s", out)
	}
}

// TestConfigCompleteDynamicUnaffected is the regression guard for cobra's
// hidden __complete command (the one a shell calls on every TAB). Config tokens
// are prepended ahead of the genuine argv, so without the completion bypass the
// __complete subcommand is shifted out of args[0] and dynamic completion breaks
// in any directory containing a ./.scc. (TestConfigCompletionUnaffected covers
// only the separate `completion --shell` script-generation path.)
func TestConfigCompleteDynamicUnaffected(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n")
	out, err := runSCCDir(t, dir, noGlobalConfig, "__complete", "--for")
	if err != nil {
		t.Fatalf("__complete errored with ./.scc present: %v\n%s", err, out)
	}
	if !strings.Contains(out, "--format") {
		t.Errorf("dynamic completion broken with ./.scc present, output:\n%s", out)
	}
	// A leaked config token would surface as an unknown-flag / read error.
	if strings.Contains(out, "unknown flag") || strings.Contains(out, "could not be read") {
		t.Errorf("config tokens leaked into the completion invocation, output:\n%s", out)
	}
}

// TestConfigSliceUnion locks the §7 union semantics end-to-end: a slice flag set
// in the config file and again on the CLI must combine with each other AND with
// the built-in defaults (defaults ∪ config ∪ CLI), rather than replacing them.
func TestConfigSliceUnion(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"vendor", "dist", "keep", ".git"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"vendor/v.go", "dist/d.go", "keep/k.go", ".git/g.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package x\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// config excludes vendor; CLI excludes dist; .git is a built-in default.
	if err := os.WriteFile(filepath.Join(dir, ".scc"), []byte("--exclude-dir vendor\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := runSCCDir(t, dir, noGlobalConfig, "--exclude-dir", "dist", "-f", "csv", "--by-file")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "k.go") {
		t.Errorf("keep/k.go should be counted, output:\n%s", out)
	}
	for _, gone := range []string{"v.go", "d.go", "g.go"} {
		if strings.Contains(out, gone) {
			t.Errorf("%s should be excluded (defaults ∪ config ∪ CLI), output:\n%s", gone, out)
		}
	}
}

// TestAtFileMultiTokenAndComments locks the phase-01 @file improvements: a line
// may carry multiple whitespace-separated tokens, '#' comments are stripped and
// blank lines are dropped (the old splitter emitted one token per line and a
// stray empty arg per blank line).
func TestAtFileMultiTokenAndComments(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	atFile := filepath.Join(dir, "flags.txt")
	// multi-token flag line, a whole-line comment, an inline comment and a blank.
	content := "# count this project\n\n--format csv   # as CSV\n--no-cocomo\n"
	if err := os.WriteFile(atFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, noGlobalConfig, "@"+atFile)
	if err != nil {
		t.Fatalf("@file errored: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("@file multi-token --format csv not applied, output:\n%s", out)
	}
	if strings.Contains(out, "Estimated Cost to Develop") {
		t.Errorf("@file --no-cocomo not applied, output:\n%s", out)
	}
}

// TestAtFileQuotedSpacePath documents the @file consequence of the shared
// quote-aware tokenizer: a path containing spaces must be quoted (the old
// splitter kept a whole line as one token, so unquoted spaces used to work).
// This pins the supported workaround.
func TestAtFileQuotedSpacePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "my file.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	atFile := filepath.Join(dir, "flags.txt")
	if err := os.WriteFile(atFile, []byte("'my file.go'\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := runSCCDir(t, dir, noGlobalConfig, "@"+atFile)
	if err != nil {
		t.Fatalf("@file errored: %v\n%s", err, out)
	}
	if strings.Contains(out, "could not be read") {
		t.Errorf("quoted space path in @file should resolve as one token, output:\n%s", out)
	}
	if !strings.Contains(out, "Go") {
		t.Errorf("quoted 'my file.go' should be counted, output:\n%s", out)
	}
}

// TestConfigFindRootWalkUp locks the headline --find-root-config feature
// end-to-end (§3.2): run from a subdirectory of a repo, --find-root-config must
// walk up to the repository root (detected via a .git dir) and load the root
// .scc, where the default ./.scc discovery finds nothing. TestConfigNoWalkUp
// already proves the default does NOT walk; this proves --find-root-config does.
func TestConfigFindRootWalkUp(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".scc"), []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "x.go"), []byte("package x\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Control: without --find-root-config, no walk-up, the root .scc is not seen (tabular).
	out, err := runSCCDir(t, sub, noGlobalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("default discovery must not walk up to the repo-root .scc, output:\n%s", out)
	}

	// --find-root-config walks up to the repo root and loads its .scc (csv).
	out, err = runSCCDir(t, sub, noGlobalConfig, "--find-root-config")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--find-root-config should walk up to the repo-root .scc, output:\n%s", out)
	}
}

// TestConfigFindRootOutsideRepoFallback locks the §3.2 graceful-degradation
// guarantee: when the CWD is not inside any repo, FindRepositoryRoot returns the
// supplied directory unchanged, so --find-root-config resolves to ./.scc -
// identical to default discovery, never an error. --find-root-config must
// therefore always be safe to leave on.
func TestConfigFindRootOutsideRepoFallback(t *testing.T) {
	dir := writeSccConfig(t, "--format csv\n") // ./.scc, no .git anywhere above
	out, err := runSCCDir(t, dir, noGlobalConfig, "--find-root-config")
	if err != nil {
		t.Fatalf("--find-root-config outside a repo should not error, got: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--find-root-config outside a repo should fall back to ./.scc, output:\n%s", out)
	}
}

// TestConfigEnvVarUnreadableErrors covers the SCC_CONFIG_PATH arm of the §8
// "explicit source the user asked for" rule: a set-but-unreadable global must
// exit non-zero with a clear error. TestConfigUnreadableExplicitErrors only
// covers the --config arm; the env-var resolution is a separate branch.
func TestConfigEnvVarUnreadableErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "does-not-exist.scc")
	out, err := runSCCDir(t, dir, []string{SccConfigEnv + "=" + missing})
	if err == nil {
		t.Fatalf("an unreadable SCC_CONFIG_PATH should exit non-zero, output:\n%s", out)
	}
	if !strings.Contains(out, "could not read config") {
		t.Errorf("expected a clear error for an unreadable SCC_CONFIG_PATH, output:\n%s", out)
	}
}

// TestConfigNoConfigDisablesEnvGlobal proves --no-config skips a *set*
// SCC_CONFIG_PATH global (§4). TestConfigNoConfigComposition only exercises
// --config (which is honored even under --no-config); the env-var global is the
// arm that --no-config actually disables.
func TestConfigNoConfigDisablesEnvGlobal(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	global := filepath.Join(dir, "global.scc")
	if err := os.WriteFile(global, []byte("--format csv\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Sanity: the env global applies when --no-config is absent.
	out, err := runSCCDir(t, dir, []string{SccConfigEnv + "=" + global})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Language,Lines,Code") {
		t.Fatalf("env global should apply without --no-config, output:\n%s", out)
	}

	// --no-config must disable the env global (back to tabular default).
	out, err = runSCCDir(t, dir, []string{SccConfigEnv + "=" + global}, "--no-config")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "Language,Lines,Code") {
		t.Errorf("--no-config should disable the SCC_CONFIG_PATH global, output:\n%s", out)
	}
}

// TestConfigPresentCLIFormatMultiWrites is the --format-multi counterpart of
// TestConfigPresentCLICanWrite: with config present (so the two-mode write split
// engages), a genuine-CLI --format-multi must still write its file. The §5
// capability model blocks config from writing, not the CLI; resolveWriteFlags
// must therefore source --format-multi from the genuine CLI too, not just -o.
func TestConfigPresentCLIFormatMultiWrites(t *testing.T) {
	dir := writeSccConfig(t, "--no-cocomo\n")
	target := filepath.Join(dir, "multi.csv")
	_, err := runSCCDir(t, dir, noGlobalConfig, "--format-multi", "csv:"+target)
	if err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("genuine CLI --format-multi should write the file even with config present")
	}
}

func TestNoConfigFastPathWritesFile(t *testing.T) {
	// No config present: -o on the CLI writes exactly as today (fast path).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "out.csv")
	if _, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "-o", target); err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("no-config fast path: CLI -o should write the file")
	}
}
