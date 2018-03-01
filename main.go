package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type FileJob struct {
	Location string
}

// A buffered channel that we can send work requests on.
var FileJobQueue = make(chan FileJob, 100)

/// Get all the files that exist in the directory
func walkDirectory(directory string) []FileJob {
	all, _ := ioutil.ReadDir(directory)

	directories := []string{}
	files := []FileJob{}

	// Work out which directories and files we want to investigate
	for _, f := range all {
		if f.IsDir() {
			directories = append(directories, f.Name())
		} else {
			files = append(files, FileJob{Location: filepath.Join(directory, f.Name())})

			FileJobQueue <- FileJob{Location: filepath.Join(directory, f.Name())}
		}
	}

	for _, newdirectory := range directories {
		files = append(files, walkDirectory(filepath.Join(directory, newdirectory))...)
	}

	return files
}

func main() {
	fmt.Println("Main")
	fmt.Println(walkDirectory("../"))
}
