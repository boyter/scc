package processor

import (
	"fmt"
	"io/ioutil"
	"sync"
)

const (
	S_BLANK             int64 = 1
	S_CODE              int64 = 2
	S_COMMENT           int64 = 3
	S_COMMENT_CODE      int64 = 4 // Indicates comment after code
	S_MULTICOMMENT      int64 = 5
	S_MULTICOMMENT_CODE int64 = 6 // Indicates multi comment after code
)

func checkForMatch(currentByte byte, index int, endPoint int, matches [][]byte, fileJob *FileJob) bool {
	for _, edge := range matches {
		if currentByte == edge[0] {

			// Start at 1 to avoid doing the check we just did again
			// see BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
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

func checkForMatchMultiOpen(currentByte byte, index int, endPoint int, matches []MultiLineComment, fileJob *FileJob) bool {
	for _, edge := range matches {
		if currentByte == edge.Open[0] {

			// Start at 1 to avoid doing the check we just did again
			// see BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
			for j := 1; j < len(edge.Open); j++ {
				if index+j >= endPoint || edge.Open[j] != fileJob.Content[index+j] {
					return false
				}
			}

			return true
		}
	}

	return false
}

func checkForMatchMultiClose(currentByte byte, index int, endPoint int, matches []MultiLineComment, fileJob *FileJob) bool {
	for _, edge := range matches {
		if currentByte == edge.Close[0] {

			// Start at 1 to avoid doing the check we just did again
			// see BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
			for j := 1; j < len(edge.Close); j++ {
				if index+j >= endPoint || edge.Close[j] != fileJob.Content[index+j] {
					return false
				}
			}

			return true
		}
	}

	return false
}

// If the file contains anything even just a newline its line count should be >= 1.
// If the file has a size of 0 its line count should be 0.
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

	endPoint := int(fileJob.Bytes - 1)
	currentState := S_BLANK
	// It is possible to have a comment like /*/**/*/ which requires a primitive stack
	// implementation to ensure that it is closed off
	currentMultiLine := 0

	for index, currentByte := range fileJob.Content {

		// If the line is still blank we can move into single line comment otherwise its still a code line just with a comment at the end
		if currentState == S_BLANK && checkForMatch(currentByte, index, endPoint, singleLineCommentChecks, fileJob) {
			currentState = S_COMMENT
		}

		// If we are in code its possible to move into single line comment
		if currentState == S_CODE && checkForMatch(currentByte, index, endPoint, singleLineCommentChecks, fileJob) {
			currentState = S_COMMENT_CODE
		}

		// If we arent in a comment its possible to enter multiline comment
		if (currentState == S_BLANK || currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_CODE) && checkForMatchMultiOpen(currentByte, index, endPoint, multiLineCommentChecks, fileJob) {
			currentState = S_MULTICOMMENT
			currentMultiLine++
		}

		// If we are in code its possible to move unto a multie line comment
		if currentState == S_CODE && checkForMatchMultiOpen(currentByte, index, endPoint, multiLineCommentChecks, fileJob) {
			currentState = S_MULTICOMMENT_CODE
			currentMultiLine++
		}

		// If we are in multiline comment its possible to move back to code
		if (currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_CODE) && checkForMatchMultiClose(currentByte, index, endPoint, multiLineCommentChecks, fileJob) {
			currentMultiLine--
			if currentMultiLine == 0 {
				currentState = S_MULTICOMMENT_CODE
			}
		}

		// Check currentState first to save on the extra checks for a small speed boost, then check in order of most common characters
		if currentState == S_BLANK && currentByte != ' ' && currentByte != '\t' && currentByte != '\n' && currentByte != '\r' {
			currentState = S_CODE
		}

		// Complexity calculations for this file
		if currentState == S_BLANK || currentState == S_CODE {
			for _, edge := range complexityChecks {
				if currentByte == edge[0] {
					potentialMatch := true

					// Start at 1 to avoid doing the check we just did again
					// see BenchmarkCheckByteEquality if you doubt this is the fastest way to do it
					for j := 1; j < len(edge); j++ {
						if index+j > endPoint || edge[j] != fileJob.Content[index+j] {
							potentialMatch = false
							break
						}
					}

					// Check if the previous byte is space tab or newline otherwise it is not a match
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

		// This means the end of processing the line so calculate the stats according to what state
		// we are currently in
		if currentByte == '\n' || index == endPoint {
			fileJob.Lines++
			printTrace(fmt.Sprintf("%s line %d ended with state: %d", fileJob.Location, fileJob.Lines, currentState))

			switch {
			case currentState == S_BLANK:
				fileJob.Blank++
			case currentState == S_CODE || currentState == S_COMMENT_CODE || currentState == S_MULTICOMMENT_CODE:
				fileJob.Code++
			case currentState == S_COMMENT:
				fileJob.Comment++
			case currentState == S_MULTICOMMENT:
				fileJob.Comment++
			}

			if currentState != S_MULTICOMMENT && currentState != S_MULTICOMMENT_CODE {
				currentState = S_BLANK
			}

			// If we are in a multiline comment that started after some code then we need
			// to move to a normal multiline comment
			if currentState == S_MULTICOMMENT_CODE {
				currentState = S_MULTICOMMENT
			}
		}
	}

	// Save memory by unsetting the content as we no longer require it
	fileJob.Content = []byte{}
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

// Reads entire file into memory and then pushes it onto the next queue
func fileReaderWorker(input *chan *FileJob, output *chan *FileJob) {
	startTime := makeTimestampMilli()
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			fileStartTime := makeTimestampNano()
			content, err := ioutil.ReadFile(res.Location)
			printTrace(fmt.Sprintf("nanoseconds read into memory: %s: %d", res.Location, makeTimestampNano()-fileStartTime))

			if err == nil {
				res.Content = content
				*output <- res
			} else {
				printWarn(fmt.Sprintf("error reading: %s %s", res.Location, err))
			}

			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
	printDebug(fmt.Sprintf("milliseconds reading files into memory: %d", makeTimestampMilli()-startTime))
}

// Does the actual processing of stats and as such contains the hot path CPU call
func fileProcessorWorker(input *chan *FileJob, output *chan *FileJob) {
	startTime := makeTimestampMilli()
	var wg sync.WaitGroup
	for res := range *input {
		wg.Add(1)
		go func(res *FileJob) {
			fileStartTime := makeTimestampNano()
			countStats(res)
			printTrace(fmt.Sprintf("nanoseconds process: %s: %d", res.Location, makeTimestampNano()-fileStartTime))

			*output <- res
			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()
	printDebug(fmt.Sprintf("milliseconds proessing files: %d", makeTimestampMilli()-startTime))
}
