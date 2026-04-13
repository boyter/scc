// SPDX-License-Identifier: MIT

package processor

import (
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetExtension(t *testing.T) {
	got := getExtension("something.c")
	expected := "c"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionNoExtension(t *testing.T) {
	got := getExtension("something")
	expected := "something"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionMultipleDots(t *testing.T) {
	got := getExtension(".travis.yml")
	expected := "travis.yml"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionMultipleExtensions(t *testing.T) {
	got := getExtension("something.go.yml")
	expected := "go.yml"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionStartsWith(t *testing.T) {
	got := getExtension(".gitignore")
	expected := ".gitignore"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionTypeScriptDefinition(t *testing.T) {
	got := getExtension("test.d.ts")
	expected := "d.ts"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestGetExtensionSecondPass(t *testing.T) {
	got := getExtension("test.d.ts")
	got = getExtension(got)
	expected := "ts"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func TestNewFileJobFullname(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()
	AllowListExtensions = []string{}

	fi, _ := os.Stat("../examples/issue114/makefile")
	job := newFileJob("../examples/issue114/", "makefile", fi)

	if job.PossibleLanguages[0] != "Makefile" {
		t.Error("Expected makefile got", job.PossibleLanguages[0])
	}
}

func TestNewFileJob(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()

	fi, _ := os.Stat("../examples/issue114/java")
	job := newFileJob("../examples/issue114/", "java", fi)

	if job.PossibleLanguages[0] != "#!" {
		t.Error("Expected special value #! got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobGitIgnore(t *testing.T) {
	AllowListExtensions = []string{}
	ProcessConstants()
	cleanVisitedPaths()
	CountIgnore = true

	fi, _ := os.Stat("../examples/issue114/.gitignore")
	job := newFileJob("../examples/issue114/", ".gitignore", fi)

	if job.PossibleLanguages[0] != "gitignore" {
		t.Error("Expected gitignore got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobIgnore(t *testing.T) {
	AllowListExtensions = []string{}
	ProcessConstants()
	cleanVisitedPaths()

	fi, _ := os.Stat("../examples/issue114/.ignore")
	job := newFileJob("../examples/issue114/", ".ignore", fi)

	if job.PossibleLanguages[0] != "ignore" {
		t.Error("Expected ignore got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobLicense(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()

	fi, _ := os.Stat("../examples/issue114/license")
	job := newFileJob("../examples/issue114/", "license", fi)

	if job.PossibleLanguages[0] != "License" {
		t.Error("Expected License got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobYAML(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()

	fi, _ := os.Stat("../examples/issue114/.travis.yml")
	job := newFileJob("../examples/issue114/", ".travis.yml", fi)

	found := false
	for _, j := range job.PossibleLanguages {
		if j == "YAML" {
			found = true
		}
	}

	if !found {
		t.Error("Expected YAML in but didn't find", job.PossibleLanguages)
	}
}

func TestNewFileJobYAMLCloudformation(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()

	fi, _ := os.Stat("../examples/issue114/.travis.yml")
	job := newFileJob("../examples/issue114/", ".travis.yml", fi)

	found := false
	for _, j := range job.PossibleLanguages {
		if j == "CloudFormation (YAML)" {
			found = true
		}
	}

	if !found {
		t.Error("Expected CloudFormation in but didn't find", job.PossibleLanguages)
	}
}

func TestNewFileJobSize(t *testing.T) {
	ProcessConstants()
	cleanVisitedPaths()
	NoLarge = true
	LargeByteCount = 1

	fi, _ := os.Stat("file_test.go")
	job := newFileJob("file_test.go", "file_test.go", fi)

	if job != nil {
		t.Error("Expected nil got", job)
	}

	NoLarge = false
	LargeByteCount = 1000000
}

func TestNewFileJobBrokenSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping symlink test on Windows due to privilege requirements")
	}

	ProcessConstants()
	cleanVisitedPaths()
	IncludeSymLinks = true
	defer func() {
		IncludeSymLinks = false
	}()

	// Create a temp directory to work in
	file := filepath.Join(t.TempDir(), "source.go")
	err := os.WriteFile(file, []byte("package main\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	symPath := filepath.Join(t.TempDir(), "broken.go")
	err = os.Symlink(file, symPath)
	if err != nil {
		t.Fatal(err)
	}
	// Delete the file breaks the symlink
	err = os.Remove(file)
	if err != nil {
		t.Fatal(err)
	}

	fi, _ := os.Lstat(symPath)
	job := newFileJob(symPath, "broken.go", fi)

	if job != nil {
		t.Error("Expected nil for broken symlink got", job)
	}
}

func BenchmarkGetExtensionDifferent(b *testing.B) {
	for i := 0; i < b.N; i++ {

		b.StopTimer()
		name := randStringBytes(3) + "." + randStringBytes(2)
		b.StartTimer()

		getExtension(name)
	}
}

func BenchmarkGetExtensionSame(b *testing.B) {
	name := randStringBytes(7) + "." + randStringBytes(3)

	for i := 0; i < b.N; i++ {
		getExtension(name)
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.IntN(len(letterBytes))]
	}
	return string(b)
}

func TestNewFileJobCircularSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping symlink test on Windows due to privilege requirements")
	}
	ProcessConstants()
	cleanVisitedPaths()
	IncludeSymLinks = true
	defer func() { IncludeSymLinks = false }()

	// Create a temp directory to work in
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	link1 := filepath.Join(dir, "link1.go")
	link2 := filepath.Join(dir, "link2.go")
	// Create a loop: link1 -> link2 and link2 -> link1
	if err := os.Symlink(link2, link1); err != nil {
		t.Fatal("Failed to create first link:", err)
	}
	if err := os.Symlink(link1, link2); err != nil {
		t.Fatal("Failed to create circular link:", err)
	}

	fi, err := os.Lstat(link1)
	if err != nil {
		t.Fatal(err)
	}
	// It should return the 'too many links' error.
	job := newFileJob(link1, "link1.go", fi)

	if job != nil {
		t.Error("Expected nil for circular symlink, but got a FileJob")
	}
}

func TestNewFileJobDuplicateCounting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping symlink test on Windows due to privilege requirements")
	}
	ProcessConstants()
	cleanVisitedPaths()
	IncludeSymLinks = true
	defer func() { IncludeSymLinks = false }()

	// on some systems like macOS, t.TempDir is also a symlink
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	// Create a test file
	testFile := filepath.Join(dir, "file.go")

	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to the same file
	linkFile := filepath.Join(dir, "link.go")
	if err := os.Symlink(testFile, linkFile); err != nil {
		t.Fatalf("Failed to create link file: %s", err)
	}
	// Process the test file
	fi1, _ := os.Lstat(testFile)
	job1 := newFileJob(testFile, "file.go", fi1)

	// Process the symlink (same target)
	fi2, _ := os.Lstat(linkFile)
	job2 := newFileJob(linkFile, "link.go", fi2)

	// First count should go through
	if job1 == nil {
		t.Fatal("Expected first file job to be created")
	}

	// Second count should be skipped
	if job2 != nil {
		t.Error("Expected nil for duplicate file through symlink, but got a FileJob")
	}
}
