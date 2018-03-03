package main

import (
	"fmt"
	"github.com/ryanuber/columnize"
	"io/ioutil"
	"os"
	"path/filepath"
	// "runtime"
	"strings"
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
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		if !info.IsDir() {
			FileReadJobQueue <- FileJob{Location: path, Filename: info.Name()}
		}

		return nil
	})

	close(FileReadJobQueue)
}

// func walkDirectory(directory string) {
// 	var wg sync.WaitGroup
// 	all, _ := ioutil.ReadDir(directory)

// 	directories := []string{}

// 	// Work out which directories and files we want to investigate
// 	for _, f := range all {
// 		if f.IsDir() {
// 			directories = append(directories, f.Name())
// 		} else {
// 			wg.Add(1)
// 			go func() {
// 				FileReadJobQueue <- FileJob{Location: filepath.Join(directory, f.Name()), Filename: f.Name()}
// 				wg.Done()
// 			}()
// 		}
// 	}

// 	for _, newdirectory := range directories {
// 		walkDirectory(filepath.Join(directory, newdirectory))
// 	}

// 	wg.Wait()
// }

func fileReaderWorker() {
	var wg sync.WaitGroup
	for res := range FileReadJobQueue {
		fileReadJob := res // Avoid race condition
		wg.Add(1)
		go func() {
			content, _ := ioutil.ReadFile(fileReadJob.Location)
			FileProcessJobQueue <- FileJob{Filename: fileReadJob.Filename, Location: fileReadJob.Location, Content: content}
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
		fileReadJob := res // Avoid race condition
		// Do some pointless work
		wg.Add(1)
		go func() {
			count := 0
			count2 := 0
			for _, i := range fileReadJob.Content {
				if i == 0 {
					count++
				} else {
					count2++
				}
			}
			FileSummaryJobQueue <- FileJob{Filename: fileReadJob.Filename, Location: fileReadJob.Location, Count: int64(count)}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(FileSummaryJobQueue)
	}()
}

func fileSummeriser() {
	total := int64(0)
	count := 0

	languages := map[string]int64{}

	for res := range FileSummaryJobQueue {

		// strings.Split(res.Filename, "sep") res.Filename

		if strings.HasSuffix(res.Filename, ".go") {
			_, ok := languages["Go"]

			if ok {
				languages["Go"] = languages["Go"] + 1
			} else {
				languages["Go"] = 1
			}

		}

		if strings.HasSuffix(res.Filename, ".yml") {
			_, ok := languages["Yaml"]

			if ok {
				languages["Yaml"] = languages["Yaml"] + 1
			} else {
				languages["Yaml"] = 1
			}

		}

		// fmt.Println(res.Filename, res.Location, len(res.Content), res.Count)
		total += res.Count
		count++
	}

	for name, count := range languages {
		fmt.Println(name, count)
	}
}

func main() {
	go fileReaderWorker()
	go fileProcessorWorker()

	walkDirectory("./")
	fileSummeriser()

	// for res := range FileSummaryJobQueue {
	// 	fmt.Println(res.Location)
	// }

	// Once done lets print it all out
	output := []string{
		"Directory | File | License | Confidence | Size",
	}

	result := columnize.SimpleFormat(output)
	fmt.Println(result)
}
