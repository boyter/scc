package main

import (
	"fmt"
	"github.com/ryanuber/columnize"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type FileJob struct {
	Filename string
	Location string
	Content  []byte
	Count    int64
}

// A buffered channel that we can send work requests on.
var FileReadJobQueue = make(chan FileJob, runtime.NumCPU()*10)
var FileProcessJobQueue = make(chan FileJob, runtime.NumCPU()*10)
var FileSummaryJobQueue = make(chan FileJob, runtime.NumCPU()*10)

/// Get all the files that exist in the directory
/// We do it this way rather than walk because we want
/// to group files together potentially
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
		go func() { // HL
			FileReadJobQueue <- FileJob{Location: path}
			wg.Done()
		}()

		return nil
	})
	// Walk has returned, so all calls to wg.Add are done.  Start a
	// goroutine to close c once all the sends are done.
	go func() { // HL
		wg.Wait()
		close(FileReadJobQueue) // HL
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

	wg.Wait()
	close(FileProcessJobQueue)
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

	wg.Wait()
	close(FileSummaryJobQueue)
}

func main() {
	go fileReaderWorker()
	go fileProcessorWorker()

	walkDirectory("../")
	fmt.Println("Finished running...")

	for res := range FileSummaryJobQueue {
		fmt.Println(res.Filename, res.Location, len(res.Content), res.Count)
	}

	// Once done lets print it all out
	output := []string{
		"Directory | File | License | Confidence | Size",
	}

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}
