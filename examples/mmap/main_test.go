package main

import (
	mmapgo "github.com/edsrzf/mmap-go"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkIoUtilOpenSingleFile(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		by, _ := ioutil.ReadFile("./linuxaverage")
		count = len(by)
	}
	b.Log(count)
}

func BenchmarkMmapUtilOpenSingleFile(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		file, _ := os.OpenFile("./linuxaverage", os.O_RDONLY, 0644)
		defer file.Close()

		mmap, _ := mmapgo.Map(file, mmapgo.RDONLY, 0)
		count = len(mmap)
		mmap.Unmap()
	}
	b.Log(count)
}

func BenchmarkIoUtilOpenDirectory(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {

		filepath.Walk("/root/linux", func(root string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			by, _ := ioutil.ReadFile(filepath.Join(root, info.Name()))

			count += len(by)
			return nil
		})
	}
	b.Log(count)
}

func BenchmarkMmapUtilOpenDirectory(b *testing.B) {
	count := 0
	for i := 0; i < b.N; i++ {
		filepath.Walk("/root/linux", func(root string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			file, _ := os.OpenFile(filepath.Join(root, info.Name()), os.O_RDONLY, 0644)
			defer file.Close()

			mmap, _ := mmapgo.Map(file, mmapgo.RDONLY, 0)
			count += len(mmap)
			mmap.Unmap()
			return nil
		})
	}
	b.Log(count)
}
