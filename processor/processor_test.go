package processor

import (
	"github.com/MichaelTJones/walk"
	"github.com/iafan/cwalk"
	"github.com/karrick/godirwalk"
	"os"
	"path"
	"path/filepath"
	"strings"
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
	expected := "yml"

	if got != expected {
		t.Errorf("Expected %s got %s", expected, got)
	}
}

func BenchmarkNativeWalk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		filepath.Walk("./", func(root string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if !info.IsDir() {
				strings.ToLower(path.Ext(info.Name()))
			}

			return nil
		})
	}
}

func BenchmarkCWalk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cwalk.Walk("./", func(root string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if !info.IsDir() {
				strings.ToLower(path.Ext(info.Name()))
			}

			return nil
		})
	}
}

func BenchmarkWalk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		walk.Walk("./", func(root string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !info.Mode().IsRegular() {
				return nil
			}

			if !info.IsDir() {
				strings.ToLower(path.Ext(info.Name()))
			}

			return nil
		})
	}
}

func BenchmarkGoDirWalk(b *testing.B) {
	for i := 0; i < b.N; i++ {
		godirwalk.Walk("./", &godirwalk.Options{
			Callback: func(osPathname string, de *godirwalk.Dirent) error {
				strings.ToLower(osPathname)
				return nil
			},
			ErrorCallback: func(osPathname string, err error) godirwalk.ErrorAction {
				return godirwalk.SkipNode
			},
		})
	}
}

func BenchmarkGetExtension(b *testing.B) {
	name := "something.c"
	for i := 0; i < b.N; i++ {
		getExtension(name)
	}
}
