// SPDX-License-Identifier: MIT OR Unlicense

package processor

import (
	"math/rand"
	"os"
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
	AllowListExtensions = []string{}

	fi, _ := os.Stat("../examples/issue114/makefile")
	job := newFileJob("../examples/issue114/", "makefile", fi)

	if job.PossibleLanguages[0] != "Makefile" {
		t.Error("Expected makefile got", job.PossibleLanguages[0])
	}
}

func TestNewFileJob(t *testing.T) {
	ProcessConstants()

	fi, _ := os.Stat("../examples/issue114/java")
	job := newFileJob("../examples/issue114/", "java", fi)

	if job.PossibleLanguages[0] != "#!" {
		t.Error("Expected special value #! got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobGitIgnore(t *testing.T) {
	AllowListExtensions = []string{}
	ProcessConstants()
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

	fi, _ := os.Stat("../examples/issue114/.ignore")
	job := newFileJob("../examples/issue114/", ".ignore", fi)

	if job.PossibleLanguages[0] != "ignore" {
		t.Error("Expected ignore got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobLicense(t *testing.T) {
	ProcessConstants()

	fi, _ := os.Stat("../examples/issue114/license")
	job := newFileJob("../examples/issue114/", "license", fi)

	if job.PossibleLanguages[0] != "License" {
		t.Error("Expected License got", job.PossibleLanguages[0])
	}
}

func TestNewFileJobYAML(t *testing.T) {
	ProcessConstants()

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
