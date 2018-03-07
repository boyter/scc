package processor

import (
	"io/ioutil"
	"sync"
)

const (
	S_BLANK        int64 = 1
	S_CODE         int64 = 2
	S_COMMENT      int64 = 3
	S_MULTICOMMENT int64 = 4
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

	endPoint := int(fileJob.Bytes - 1)
	currentState := S_BLANK

	for i, b := range fileJob.Content {

		if b != ' ' && b != '\t' && b != '\n' && b != '\r' { // TODO Check if another if to avoid setting S_CODE is faster
			currentState = S_CODE
		}

		if b == '\n' || i == endPoint { // This means the end of processing the line so calculate the stats
			switch {
			case currentState == S_BLANK:
				fileJob.Blank++
			case currentState == S_CODE:
				fileJob.Code++
			}

			fileJob.Lines++
			currentState = S_BLANK
		}
	}
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
