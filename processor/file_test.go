package processor

import (
	"math/rand"
	"path/filepath"
	"testing"
	"strings"
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

func TestWalkDirectoryParallel(t *testing.T) {
	isLazy = false
	ProcessConstants()

	WhiteListExtensions = []string{"go"}
	Exclude = []string{"vendor"}
	PathBlacklist = []string{"vendor"}
	Verbose = true
	Trace = true
	Debug = true
	GcFileCount = 10

	inputChan := make(chan *FileJob, 10000)
	walkDirectoryParallel("../", inputChan)
	close(inputChan)

	count := 0
	for range inputChan {
		count++
	}

	if count == 0 {
		t.Errorf("Expected at least one file got %d", count)
	}
}

func TestWalkDirectoryParallelWorksWithSingleInputFile(t *testing.T) {
	isLazy = false
	ProcessConstants()

	WhiteListExtensions = []string{"go"}
	Exclude = []string{"vendor"}
	PathBlacklist = []string{"vendor"}
	Verbose = true
	Trace = true
	Debug = true
	GcFileCount = 10

	inputChan := make(chan *FileJob, 10000)
	walkDirectoryParallel("file_test.go", inputChan)
	close(inputChan)

	count := 0
	for range inputChan {
		count++
	}

	if count != 1 {
		t.Errorf("Expected exactly one file got %d", count)
	}
}

func TestWalkDirectoryParallelIgnoresRootTrailingSlash(t *testing.T) {
	isLazy = false
	ProcessConstants()

	WhiteListExtensions = []string{"go"}
	Exclude = []string{"vendor"}
	PathBlacklist = []string{"vendor"}
	Verbose = true
	Trace = true
	Debug = true
	GcFileCount = 10

	inputChan := make(chan *FileJob, 10000)
	walkDirectoryParallel("file_test.go/", inputChan)
	close(inputChan)

	count := 0
	for range inputChan {
		count++
	}

	if count != 1 {
		t.Errorf("Expected exactly one file got %d", count)
	}
}

// Issue #82 - project .git directory not being filtered when using absolute
// path argument
func TestWalkDirectoryParallelIgnoresAbsoluteGitPath(t *testing.T) {
	isLazy = false
	ProcessConstants()

	// master is a file extension for ASP.NET, and also a filename (almost)
	// certain to appear in the .git directory.
	// This test also relies on the behaviour of treating `master` as a file
	// with the `master` file extension.
	WhiteListExtensions = []string{"master", "go"}
	Exclude = []string{"vendor"}
	PathBlacklist = []string{".git", "vendor"}
	Verbose = true
	Trace = true
	Debug = true
	GcFileCount = 10

	inputChan := make(chan *FileJob, 10000)
	absBaseDir, _ := filepath.Abs("../")
	absGitDir := filepath.Join(absBaseDir, ".git")

	walkDirectoryParallel(absBaseDir, inputChan)
	close(inputChan)

	sawGit := false
	for fileJob := range inputChan {
		if strings.HasPrefix(fileJob.Location, absGitDir) {
			sawGit = true
			break
		}
	}

	if sawGit {
		t.Errorf("Expected .git folder to be ignored")
	}
}

func TestWalkDirectory(t *testing.T) {
	Debug = true
	Exclude = []string{"test"}
	ProcessConstants()
	files := walkDirectory(".", []string{}, ExtensionToLanguage)

	if len(files) == 0 {
		t.Error("Expected at least one file")
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
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
