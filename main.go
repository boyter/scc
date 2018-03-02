package main

import (
	"fmt"
	"github.com/ryanuber/columnize"
	"io/ioutil"
	"os"
	"path/filepath"
	// "runtime"
	"sync"
)

type FileJob struct {
	Filename string
	Location string
	Content  []byte
	Count    int64
}

// A buffered channel that we can send work requests on.
var FileReadJobQueue = make(chan FileJob, 100)
var FileProcessJobQueue = make(chan FileJob, 100)
var FileSummaryJobQueue = make(chan FileJob, 100)

/// Get all the files that exist in the directory
func walkDirectory(root string) {
	var wg sync.WaitGroup

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		wg.Add(1)
		go func() {
			FileReadJobQueue <- FileJob{Location: path, Filename: info.Name()}
			wg.Done()
		}()

		return nil
	})

	go func() {
		wg.Wait()
		close(FileReadJobQueue)
	}()
}

func fileReaderWorker() {
	var wg sync.WaitGroup
	for res := range FileReadJobQueue {
		wg.Add(1)
		go func() {
			content, _ := ioutil.ReadFile(res.Location)
			FileProcessJobQueue <- FileJob{Filename: res.Filename, Location: res.Location, Content: content}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(FileProcessJobQueue)
	}()
}

func fileProcessorWorker() {
	var wg sync.WaitGroup
	for res := range FileProcessJobQueue {
		// Do some pointless work
		wg.Add(1)
		go func() {
			count := 0
			count2 := 0
			for _, i := range res.Content {
				if i == 0 {
					count++
				}

				if i == 1 {
					count2++
				}
			}
			FileSummaryJobQueue <- FileJob{Filename: res.Filename, Location: res.Location, Count: int64(count)}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(FileSummaryJobQueue)
	}()
}

func main() {
	go fileReaderWorker()
	go fileProcessorWorker()

	walkDirectory("../../")

	total := int64(0)
	count := 0
	for res := range FileSummaryJobQueue {
		fmt.Println(res.Filename, res.Location, len(res.Content), res.Count)
		total += res.Count
		count++
	}

	fmt.Println(total)
	fmt.Println("COUNT:", count)
	// Once done lets print it all out
	output := []string{
		"Directory | File | License | Confidence | Size",
	}

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}
