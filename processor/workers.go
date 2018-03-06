package processor

import (
	"bytes"
	"io/ioutil"
	"sync"
)

func countStats(fileJob *FileJob) {

}

func fileReaderWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			content, _ := ioutil.ReadFile(res.Location)
			res.Content = content
			*output <- res
			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}

func fileProcessorWorker(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		// Do some pointless work
		wg.Add(1)
		go func(res *FileJob) {
			res.Lines = int64(bytes.Count(res.Content, []byte("\n")))   // Fastest way to count newlines
			res.Blank = int64(bytes.Count(res.Content, []byte("\n\n"))) // Cheap way to calculate blanks but probably wrong
			// is it? what about the langage "whitespace" where whitespace is significant....

			// Find first instance of a \n
			// Check the slice before for interesting
			// Determine if newline
			// keep running counter
			// check if spaces etc....

			res.Bytes = int64(len(res.Content))
			*output <- res
			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
}
