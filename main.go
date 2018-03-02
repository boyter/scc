package main

import (
	"fmt"
	"github.com/ryanuber/columnize"
	"io/ioutil"
	"path/filepath"
	"runtime"
)

type FileJob struct {
	Location string
	Content  []byte
}

// A buffered channel that we can send work requests on.
var FileReadJobQueue = make(chan FileJob, runtime.NumCPU()*10)
var FileProcessJobQueue = make(chan FileJob, runtime.NumCPU()*10)

func readFile(filepath string) []byte {
	// TODO only read as deep into the file as we need
	bytes, err := ioutil.ReadFile(filepath)

	if err != nil {
		fmt.Print(err)
	}

	return bytes
}

/// Get all the files that exist in the directory
func walkDirectory(directory string) {
	all, _ := ioutil.ReadDir(directory)
	directories := []string{}

	// Work out which directories and files we want to investigate
	for _, f := range all {
		if f.IsDir() {
			directories = append(directories, f.Name())
		} else {
			FileReadJobQueue <- FileJob{Location: filepath.Join(directory, f.Name())}
		}
	}

	for _, newdirectory := range directories {
		walkDirectory(filepath.Join(directory, newdirectory))
	}
}

func fileProcessorWorker() {
	for {
		res := <-FileProcessJobQueue

		count := 0
		for _, i := range res.Content {
			if i == 0 {
				count++
			}
		}

		fmt.Println(res.Location, len(res.Content), count)
	}
}

func fileReaderWorker() {
	for {
		res := <-FileReadJobQueue
		content := readFile(res.Location)
		FileProcessJobQueue <- FileJob{Location: res.Location, Content: content}
	}
}

func main() {
	for i := 0; i < runtime.NumCPU()+2; i++ {
		go fileProcessorWorker()
		go fileReaderWorker()
	}

	fmt.Println("Main")
	walkDirectory("../")

	// Once done lets print it all out
	output := []string{
		"Directory | File | License | Confidence | Size",
	}

	result := columnize.SimpleFormat(output)

	fmt.Println(result)
}
