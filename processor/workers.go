package processor

import (
	"io/ioutil"
	"sync"
)

const (
	S_BLANK        int64 = 1
	S_CODE         int64 = 2
	S_COMMENT      int64 = 3
	S_COMMENT_CODE int64 = 4 // Indicates comment AFTER code
	S_MULTICOMMENT int64 = 5
)

func checkForSingleLineComment(currentByte byte, index int, endPoint int, singleLineComments [][]byte, fileJob *FileJob) bool {
	for _, edge := range singleLineComments {
		if currentByte == edge[0] {

			// Start at 1 to avoid doing the check we just did again
			// Check BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
			for j := 1; j < len(edge); j++ {
				if index+j >= endPoint || edge[j] != fileJob.Content[index+j] {
					return false
				}
			}

			return true
		}
	}

	return false
}

// If the file contains anything even just a newline its lines > 1
// If the file size is 0 its lines = 0
// Newlines belong to the line they started on so a file of \n means only 1 line
// This is the 'hot' path for the application and needs to be as fast as possible
func countStats(fileJob *FileJob) {
	// If the file has a length of 0 it is is empty then we say it has no lines
	fileJob.Bytes = int64(len(fileJob.Content))
	if fileJob.Bytes == 0 {
		fileJob.Lines = 0
		return
	}

	// WIP should be in the list of languages
	complexityChecks := [][]byte{
		[]byte("for "),
		[]byte("for("),
		[]byte("if "),
		[]byte("if("),
		[]byte("switch "),
		[]byte("while "),
		[]byte("else "),
		[]byte("|| "),
		[]byte("&& "),
		[]byte("!= "),
		[]byte("== "),
	}

	// WIP should be in the list of lanugages
	singleLineCommentChecks := [][]byte{
		[]byte("#"),
		[]byte("//"),
	}

	// WIP should be in the list of languages
	multiLineCommentChecks := []MultiLineComment{
		MultiLineComment{
			Open:  []byte("/*"),
			Close: []byte("*/"),
		},
	}

	/* test */
	endPoint := int(fileJob.Bytes - 1)
	currentState := S_BLANK

	for index, currentByte := range fileJob.Content {

		// WIP If the line is still blank we can move into single line comment otherwise its still a code line just with a comment at the end
		if currentState == S_BLANK {
			if checkForSingleLineComment(currentByte, index, endPoint, singleLineCommentChecks, fileJob) {
				currentState = S_COMMENT
			}
		}

		// If we are in code its possible to move into single line comment BUT we don't count it later
		// TODO merge with the above
		if currentState == S_CODE {
			if checkForSingleLineComment(currentByte, index, endPoint, singleLineCommentChecks, fileJob) {
				currentState = S_COMMENT_CODE
			}
		}

		// If we arent in a comment its possible to enter multiline comment
		if currentState != S_COMMENT && currentState != S_COMMENT_CODE {
			for _, edge := range multiLineCommentChecks {
				if currentByte == edge.Open[0] {
					potentialMatch := true

					// Start at 1 to avoid doing the check we just did again
					// Check BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
					for j := 1; j < len(edge.Open); j++ {
						if index+j >= endPoint || edge.Open[j] != fileJob.Content[index+j] {
							potentialMatch = false
							break
						}
					}

					if potentialMatch {
						currentState = S_MULTICOMMENT
					}
				}
			}
		}

		// Check currentState first to save on the extra checks for a small speed boost, then check in order of most common characters
		if currentState == S_BLANK && currentByte != ' ' && currentByte != '\t' && currentByte != '\n' && currentByte != '\r' {
			currentState = S_CODE
		}

		// Complexity calculation
		// In reality this is going to need to pull from the list of languages to see how to do this
		if currentState == S_BLANK || currentState == S_CODE {
			for _, edge := range complexityChecks {
				if currentByte == edge[0] {
					potentialMatch := true

					// Start at 1 to avoid doing the check we just did again
					// Check BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
					for j := 1; j < len(edge); j++ {
						if index+j > endPoint || edge[j] != fileJob.Content[index+j] {
							potentialMatch = false
							break
						}
					}

					// Check if the previous byte is space tab or newline otherwise its not a match
					if index != 0 {
						if fileJob.Content[index-1] != ' ' && fileJob.Content[index-1] != '\t' && fileJob.Content[index-1] != '\n' && fileJob.Content[index-1] != '\r' {
							potentialMatch = false
						}
					}

					if potentialMatch {
						fileJob.Complexity++
					}
				}
			}
		}

		// This means the end of processing the line so calculate the stats
		if currentByte == '\n' || index == endPoint {
			switch {
			case currentState == S_BLANK:
				fileJob.Blank++
			case currentState == S_CODE || currentState == S_COMMENT_CODE:
				fileJob.Code++
			case currentState == S_COMMENT:
				fileJob.Comment++
			case currentState == S_MULTICOMMENT:
				fileJob.Comment++
			}

			fileJob.Lines++

			if currentState != S_MULTICOMMENT {
				currentState = S_BLANK
			}
		}
	}
}

// Reads from the file first queue and pushes to the next
// use this to chain from buffers where we don't want processing to
// stop into more CPU bound task we want to run on the number of CPU's
func fileBufferReader(input *chan *FileJob, output *chan *FileJob) {
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			*output <- res
			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
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
