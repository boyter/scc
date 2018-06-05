package main

// Goal is to determine the fastest way to iterate directories in go
// Recursion or while with stack?
// Can we go faster than godirwalk

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func walk(directory string) int {
	count := 0
	all, _ := ioutil.ReadDir(directory)

	for _, f := range all {
		if f.IsDir() {
			count += walk(filepath.Join(directory, f.Name()))
		} else {
			count++
		}
	}

	return count
}

func main() {
	count := walk("./")
	fmt.Println(count)
}
