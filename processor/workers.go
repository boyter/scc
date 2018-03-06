package processor

import (
	// "bytes"
	"io/ioutil"
	"sync"
)

// If the file contains anything even just a newline its lines > 1
// If the file size is 0 its lines = 0
// Newlines belong to the line they started on so a file of \n means only 1 line
func countStats(fileJob *FileJob) {

	// If the file has a length of 0 it is is empty then we say it has no lines
	fileJob.Bytes = int64(len(fileJob.Content))
	if fileJob.Bytes == 0 {
		fileJob.Lines = 0
		return
	}

	fileJob.Lines = 1
	endPoint := fileJob.Bytes - 1

	// This means we look at every byte so there may be a better way to do this

	// State

	for i, b := range fileJob.Content {
		if b == '\n' && int64(i) != endPoint {
			fileJob.Lines++
		}
	}

	// If the file is not empty then it has at least 1 line
	// fileJob.Lines = int64(bytes.Count(fileJob.Content, []byte("\n")))   // Fastest way to count newlines but buggy
	// fileJob.Blank = int64(bytes.Count(fileJob.Content, []byte("\n\n"))) // Cheap way to calculate blanks but probably wrong

	// Cater for file thats not empty but no newlines
	if fileJob.Lines == 0 && fileJob.Bytes != 0 {
		fileJob.Lines = 1
	}

	// is it? what about the langage "whitespace" where whitespace is significant....

	// Find first instance of a \n
	// Check the slice before for interesting
	// Determine if newline
	// keep running counter
	// check if spaces etc....
}

// Reads file into memory
func fileReaderWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			content, err := ioutil.ReadFile(res.Location)

			if err == nil {
				res.Content = content
				*output <- res
			}

			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}

// Does the actual processing of stats and is the hot path
func fileProcessorWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			countStats(res)
			*output <- res
			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}
