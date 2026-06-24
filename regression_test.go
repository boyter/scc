// SPDX-License-Identifier: MIT

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The tests in this file guard against regressions to pre-existing behaviour
// introduced by the config-dotfile work (spec/01-config-dotfile). They focus on
// functionality that existed before the feature and could have been silently
// changed by the registerFlags refactor, the shared @file tokenizer and the
// config discovery/merge pipeline - cases the feature's own tests do not cover.
//
// They run with noGlobalConfig (defined in config_test.go) so a SCC_CONFIG_PATH
// in the developer's environment cannot pollute the results.

// TestRegressionExcludeDirPreservesDefaults exercises the phase-02 slice-default
// change on the ordinary no-config CLI path (the ~99% case). pflag replaces a
// slice's default on the first Set, so before this work `--exclude-dir vendor`
// dropped the built-in .git/.hg/.svn and scc descended into them. The post-parse
// union must keep the defaults as a non-removable safety net. .svn is the canary:
// nothing else skips it, so if preservation breaks it reappears in the output.
func TestRegressionExcludeDirPreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	layout := map[string]string{".svn": "x.go", "vendor": "y.go", "keep": "z.go"}
	for sub, file := range layout {
		if err := os.Mkdir(filepath.Join(dir, sub), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, file), []byte("package x\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "--by-file", "--exclude-dir", "vendor")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "z.go") {
		t.Errorf("keep/z.go should be counted, output:\n%s", out)
	}
	if strings.Contains(out, "y.go") {
		t.Errorf("vendor/y.go should be excluded by --exclude-dir, output:\n%s", out)
	}
	if strings.Contains(out, "x.go") {
		t.Errorf(".svn/x.go reappeared: a CLI --exclude-dir must not replace the built-in defaults (union, not replace), output:\n%s", out)
	}
}

// TestRegressionExcludeFilePreservesDefaults is the --exclude-file counterpart of
// the above: adding one ignored filename on the CLI must not drop the built-in
// lockfile defaults. package-lock.json is the canary - before the fix pflag's
// replace-on-first-Set would have let it through.
func TestRegressionExcludeFilePreservesDefaults(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.go":           "package main\n",
		"package-lock.json": "{\"a\":1}\n",
		"myignore.txt":      "hello\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	out, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "--by-file", "--exclude-file", "myignore.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("main.go should be counted, output:\n%s", out)
	}
	if strings.Contains(out, "myignore.txt") {
		t.Errorf("myignore.txt should be excluded by --exclude-file, output:\n%s", out)
	}
	if strings.Contains(out, "package-lock.json") {
		t.Errorf("package-lock.json reappeared: a CLI --exclude-file must not replace the built-in lockfile defaults, output:\n%s", out)
	}
}

// TestRegressionConfigWithPositionalDir guards the merged argument ordering: a
// positional path must still be scanned when a project .scc is present. Config
// tokens are prepended and the genuine CLI (including the path) comes last, so
// cobra must still receive the path as a positional. The .scc sets --by-file, so
// the per-file row for the scanned path proves both that config applied and that
// the positional argument survived the prepend.
func TestRegressionConfigWithPositionalDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "code"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "code", "a.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".scc"), []byte("--by-file\n"), 0644); err != nil {
		t.Fatal(err)
	}

	out, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "code")
	if err != nil {
		t.Fatal(err)
	}
	// a.go only appears in --by-file output, so its presence proves both the
	// config (--by-file) applied and the positional 'code' dir was scanned.
	if !strings.Contains(out, "a.go") {
		t.Errorf("positional 'code' dir should be scanned with a project .scc present, output:\n%s", out)
	}
}

// TestRegressionAtFileCommentsAndMultiToken confirms the shared tokenizer's new
// capabilities reach the existing @file syntax: multiple tokens per line, plus
// whole-line and inline '#' comments. Under the old whole-line splitter "-f csv"
// was a single unknown token and "main.go # x" an unreadable path, so this both
// proves the improvement and pins the new @file contract.
func TestRegressionAtFileCommentsAndMultiToken(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	atFile := filepath.Join(dir, "flags.txt")
	content := "# count the project\n\n-f csv\n--by-file\nmain.go # the entrypoint\n"
	if err := os.WriteFile(atFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// @file must be the sole argument; the sole-argument trigger is preserved.
	out, err := runSCCDir(t, dir, noGlobalConfig, "@"+atFile)
	if err != nil {
		t.Fatalf("@file errored: %v\n%s", err, out)
	}
	if strings.Contains(out, "could not be read") {
		t.Errorf("inline '#' comment was not stripped from the path line, output:\n%s", out)
	}
	if !strings.Contains(out, "main.go") {
		t.Errorf("@file multi-token (-f csv --by-file) + main.go path not honoured, output:\n%s", out)
	}
}

// TestRegressionHelpShowsSliceDefaults guards the user-visible default display.
// The three slice flags are registered with an empty runtime default (to defuse
// pflag's replace-on-first-Set); their "(default ...)" text is restored via
// DefValue. Skipping that restoration would silently drop the signal of what scc
// excludes out of the box, so assert --help still advertises the defaults.
func TestRegressionHelpShowsSliceDefaults(t *testing.T) {
	out, err := runSCC("--help")
	if err != nil {
		t.Fatalf("--help should exit 0: %v\n%s", err, out)
	}
	wants := []string{
		"[.git,.hg,.svn]",
		"[package-lock.json,Cargo.lock,yarn.lock,pubspec.lock,Podfile.lock,pnpm-lock.yaml]",
		"[do not edit,<auto-generated />]",
	}
	for _, w := range wants {
		if !strings.Contains(out, w) {
			t.Errorf("--help should display slice default %q, output:\n%s", w, out)
		}
	}
}

// TestRegressionCommentOnlyConfigAllowsWrite documents an edge of the security
// model: a .scc that contributes zero tokens (only comments/blanks) must not
// engage the write-blocking config path. With nothing injected there is nothing
// to defend against, so the no-config fast path runs and a CLI -o still writes.
func TestRegressionCommentOnlyConfigAllowsWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".scc"), []byte("# just a comment\n\n"), 0644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "out.csv")
	if _, err := runSCCDir(t, dir, noGlobalConfig, "-f", "csv", "-o", target); err != nil {
		t.Fatal(err)
	}
	if info, statErr := os.Stat(target); statErr != nil || info.Size() == 0 {
		t.Errorf("a comment-only .scc should not block a genuine CLI -o write")
	}
}
