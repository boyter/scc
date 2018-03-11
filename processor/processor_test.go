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
