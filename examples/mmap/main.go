package main

import (
	"fmt"
	mmapgo "github.com/edsrzf/mmap-go"
	"os"
	"path/filepath"
)

var testPath = filepath.Join(os.TempDir(), "testdata")

func openFile(flags int) *os.File {
	file, err := os.OpenFile("/home/bboyter/Go/src/github.com/boyter/scc/examples/mmap/main.go", flags, 0644)

	if err != nil {
		panic(err.Error())
	}

	return file
}

func main() {
	file := openFile(os.O_RDONLY)
	defer file.Close()

	mmap, err := mmapgo.Map(file, mmapgo.RDONLY, 0)

	fmt.Println(len(mmap))

	count := 0
	for _, currentByte := range mmap {
		if currentByte == '\n' {
			count++
		}
	}

	fmt.Println(count)

	if err != nil {
		fmt.Println("error mapping:", err)
	}

	if err := mmap.Unmap(); err != nil {
		fmt.Println("error unmapping:", err)
	}
}
