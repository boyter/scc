package processor

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"sync"
)

const (
	S_BLANK              int64 = 1
	S_CODE               int64 = 2
	S_COMMENT            int64 = 3
	S_COMMENT_CODE       int64 = 4 // Indicates comment after code
	S_MULTICOMMENT       int64 = 5
	S_MULTICOMMENT_CODE  int64 = 6 // Indicates multi comment after code
	S_MULTICOMMENT_BLANK int64 = 7 // Indicates multi comment ended with blank afterwards
	S_STRING             int64 = 8
)

type LineType int32

const (
	LINE_BLANK LineType = iota
	LINE_CODE
	LINE_COMMENT
)

func checkForMatch(currentByte byte, index int, endPoint int, matches [][]byte, fileJob *FileJob) bool {
	potentialMatch := true
	for i := 0; i < len(matches); i++ {
		if currentByte == matches[i][0] {
			for j := 0; j < len(matches[i]); j++ {
				if index+j >= endPoint || matches[i][j] != fileJob.Content[index+j] {
					potentialMatch = false
					break
				}
			}

			if potentialMatch {
				return true
			}
		}
	}

	return false
}

func checkForMatchSingle(currentByte byte, index int, endPoint int, matches []byte, fileJob *FileJob) bool {
	potentialMatch := true

	if currentByte == matches[0] {
		for j := 0; j < len(matches); j++ {
			if index+j >= endPoint || matches[j] != fileJob.Content[index+j] {
				potentialMatch = false
				break
			}
		}

		if potentialMatch {
			return true
		}
	}

	return false
}

func checkForMatchMultiOpen(currentByte byte, index int, endPoint int, matches []OpenClose, fileJob *FileJob) (int, []byte) {

	potentialMatch := true
	for i := 0; i < len(matches); i++ {
		if currentByte == matches[i].Open[0] {
			potentialMatch = true

			for j := 1; j < len(matches[i].Open); j++ {
				if index+j > endPoint || matches[i].Open[j] != fileJob.Content[index+j] {
					potentialMatch = false
					break
				}
			}

			if potentialMatch {
				return len(matches[i].Open), matches[i].Close
			}
		}
	}

	return 0, nil
}

func checkForMatchMultiClose(currentByte byte, index int, endPoint int, matches []OpenClose, fileJob *FileJob) int {

	potentialMatch := true
	for i := 0; i < len(matches); i++ {
		if currentByte == matches[i].Close[0] {
			potentialMatch = true

			for j := 1; j < len(matches[i].Close); j++ {
				if index+j > endPoint || matches[i].Close[j] != fileJob.Content[index+j] {
					potentialMatch = false
					break
				}
			}

			if potentialMatch {
				return len(matches[i].Close)
			}
		}
	}

	return 0
}

// What I want to know is given a list of strings and a current position are any
// of them there starting from where we check, and if yes say so

func checkComplexity(currentByte byte, index int, endPoint int, matches [][]byte, fileJob *FileJob) int {
	// Special case if the thing we are matching is not the first thing in the file
	// then we need to check that there was a whitespace before it
	if index != 0 {
		// If the byte before our current postion is not a whitespace then return false
		if fileJob.Content[index-1] != ' ' && fileJob.Content[index-1] != '\t' && fileJob.Content[index-1] != '\n' && fileJob.Content[index-1] != '\r' {
			return 0
		}
	}

	// Because the number of complexity checks is usually quite high this check speeds
	// up the processing quite a lot and is worth implementing
	// NB this allocation is much cheaper than refering to things directly
	complexityBytes := LanguageFeatures[fileJob.Language].ComplexityBytes

	hasMatch := false
	for i := 0; i < len(complexityBytes); i++ {
		if complexityBytes[i] == currentByte {
			hasMatch = true
			break
		}
	}

	if !hasMatch {
		return 0
	}

	potentialMatch := true
	for i := 0; i < len(matches); i++ { // Loop each match
		if currentByte == matches[i][0] { // If the first byte of the match is not the current byte skip
			potentialMatch = true

			// Assume that we have a match and then see if we don't
			// Start from 1 as we already checked the first byte for a match
			for j := 1; j < len(matches[i]); j++ {
				// Bounds check first and if that is ok check if the bytes match
				if index+j > endPoint || matches[i][j] != fileJob.Content[index+j] {
					potentialMatch = false
					break
				}
			}

			// Return the length of match and use that to step past the bytes we just checked
			if potentialMatch {
				return len(matches[i])
			}
		}
	}

	return 0
}

func isWhitespace(currentByte byte) bool {
	if currentByte != ' ' && currentByte != '\t' && currentByte != '\n' && currentByte != '\r' {
		return false
	}

	return true
}

// CountStats will process the fileJob
// If the file contains anything even just a newline its line count should be >= 1.
// If the file has a size of 0 its line count should be 0.
// Newlines belong to the line they started on so a file of \n means only 1 line
// This is the 'hot' path for the application and needs to be as fast as possible
func CountStats(fileJob *FileJob) {

	// If the file has a length of 0 it is is empty then we say it has no lines
	fileJob.Bytes = int64(len(fileJob.Content))
	if fileJob.Bytes == 0 {
		fileJob.Lines = 0
		return
	}

	complexityChecks := LanguageFeatures[fileJob.Language].ComplexityChecks
	singleLineCommentChecks := LanguageFeatures[fileJob.Language].SingleLineComment
	multiLineCommentChecks := LanguageFeatures[fileJob.Language].MultiLineComment
	stringChecks := LanguageFeatures[fileJob.Language].StringChecks

	endPoint := int(fileJob.Bytes - 1)
	currentState := S_BLANK
	endString := []byte{}

	// If we have checked bytes ahead of where we are we can jump ahead and save time
	// this value stores that jump
	offsetJump := 0

	// For determining duplicates we need the below. The reason for creating
	// the byte array here is to avoid GC pressure. MD5 is in the standard library
	// and is fast enough to not warrent murmur3 hashing. No need to be
	// crypto secure here either so no need to eat the performance cost of a better
	// hash method
	digest := md5.New()
	digestible := []byte{' '}

	for index := 0; index < len(fileJob.Content); index++ {
		offsetJump = 0

		if Duplicates {
			// Technically this is wrong because we skip bytes so this is not a true
			// hash of the file contents, but for duplicate files it shouldn't matter
			// as both will skip the same way
			digestible[0] = fileJob.Content[index]
			digest.Write(digestible)
		}

		// Check if this file is binary by checking for nul byte and if so bail out
		// this is how GNU Grep, git and ripgrep check for binary files
		if !DisableCheckBinary && fileJob.Content[index] == 0 {
			fileJob.Binary = true
			return
		}

		// Based on our current state determine if the state should change by checking
		// what the character is. The below is very CPU bound so need to be careful if
		// changing anything in here and profile/measure afterwards!
	state:
		switch {
		case isWhitespace(fileJob.Content[index]):
			break state
		case currentState == S_BLANK || currentState == S_MULTICOMMENT_BLANK:
			// From blank we can move into comment, move into a multiline comment
			// or move into code but we can only do one.
			if checkForMatch(fileJob.Content[index], index, endPoint, singleLineCommentChecks, fileJob) {
				currentState = S_COMMENT
				break state
			}

			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, multiLineCommentChecks, fileJob)
			if offsetJump != 0 {
				currentState = S_MULTICOMMENT
				break state
			}

			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, stringChecks, fileJob)
			if offsetJump != 0 {
				currentState = S_STRING
				break state
			}

			if !isWhitespace(fileJob.Content[index]) {
				currentState = S_CODE

				if !Complexity {
					offsetJump = checkComplexity(fileJob.Content[index], index, endPoint, complexityChecks, fileJob)
					if offsetJump != 0 {
						fileJob.Complexity++
					}
				}
				break state
			}
		case currentState == S_CODE:
			// From code we can move into a multiline comment or string
			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, multiLineCommentChecks, fileJob)
			if offsetJump != 0 {
				currentState = S_MULTICOMMENT_CODE
				break state
			}

			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, stringChecks, fileJob)
			if offsetJump != 0 {
				currentState = S_STRING
				break state
			} else {
				if !Complexity {
					offsetJump = checkComplexity(fileJob.Content[index], index, endPoint, complexityChecks, fileJob)
					if offsetJump != 0 {
						fileJob.Complexity++
					}
				}
				break state
			}
		case currentState == S_STRING:
			// Its not possible to enter this state without checking at least 1 byte so it is safe to check -1 here
			// without checking if it is out of bounds first
			if fileJob.Content[index-1] != '\\' && checkForMatchSingle(fileJob.Content[index], index, endPoint, endString, fileJob) {
				currentState = S_CODE
			}
			break state
		case currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_CODE:
			offsetJump = checkForMatchMultiClose(fileJob.Content[index], index, endPoint, multiLineCommentChecks, fileJob)

			if offsetJump != 0 {

				//If we started as multiline code switch back to code so we count correctly
				if currentState == S_MULTICOMMENT_CODE {
					currentState = S_CODE
				} else {
					// If we are the end of the file OR next byte is whitespace move to comment blank

					if index+offsetJump >= endPoint || isWhitespace(fileJob.Content[index+offsetJump]) {
						currentState = S_MULTICOMMENT_BLANK
					} else {
						currentState = S_MULTICOMMENT_CODE
					}
				}

				index += offsetJump
			}
		}

		// This means the end of processing the line so calculate the stats according to what state
		// we are currently in
		if index >= endPoint || fileJob.Content[index] == '\n' {
			fileJob.Lines++

			if Trace {
				printTrace(fmt.Sprintf("%s line %d ended with state: %d", fileJob.Location, fileJob.Lines, currentState))
			}

			switch {
			case currentState == S_BLANK:
				{
					fileJob.Blank++
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_BLANK) {
							return
						}
					}
				}
			case currentState == S_CODE || currentState == S_STRING || currentState == S_COMMENT_CODE || currentState == S_MULTICOMMENT_CODE:
				{
					fileJob.Code++
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_CODE) {
							return
						}
					}
				}
			case currentState == S_COMMENT || currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_BLANK:
				{
					fileJob.Comment++
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_COMMENT) {
							return
						}
					}
				}
			}

			// If we are in a multiline comment that started after some code then we need
			// to move to a multiline comment if a multiline comment then stay there
			// otherwise we reset back into a blank state
			if currentState != S_MULTICOMMENT && currentState != S_MULTICOMMENT_CODE {
				currentState = S_BLANK
			} else {
				currentState = S_MULTICOMMENT
			}
		}
	}

	if Duplicates {
		hashed := make([]byte, 0)
		fileJob.Hash = digest.Sum(hashed)
	}

	// Save memory by unsetting the content as we no longer require it
	fileJob.Content = []byte{}
}

// Reads entire file into memory and then pushes it onto the next queue
func fileReaderWorker(input *chan *FileJob, output *chan *FileJob) {
	var startTime int64 = 0
	var wg sync.WaitGroup

	for res := range *input {
		if startTime == 0 {
			startTime = makeTimestampMilli()
		}

		wg.Add(1)
		go func(res *FileJob) {
			fileStartTime := makeTimestampNano()
			content, err := ioutil.ReadFile(res.Location)

			if Trace {
				printTrace(fmt.Sprintf("nanoseconds read into memory: %s: %d", res.Location, makeTimestampNano()-fileStartTime))
			}

			if err == nil {
				res.Content = content
				*output <- res
			} else {
				if Verbose {
					printWarn(fmt.Sprintf("error reading: %s %s", res.Location, err))
				}
			}

			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)

		if Debug {
			printDebug(fmt.Sprintf("milliseconds reading files into memory: %d", makeTimestampMilli()-startTime))
		}
	}()
}

var duplicates = CheckDuplicates{
	hashes: make(map[int64][][]byte),
}

// Does the actual processing of stats and as such contains the hot path CPU call
func fileProcessorWorker(input *chan *FileJob, output *chan *FileJob) {
	var startTime int64 = 0
	var wg sync.WaitGroup
	for res := range *input {
		if startTime == 0 {
			startTime = makeTimestampMilli()
		}

		wg.Add(1)
		go func(res *FileJob) {
			fileStartTime := makeTimestampNano()
			CountStats(res)

			if Duplicates {
				if duplicates.Check(res.Bytes, res.Hash) {
					if Verbose {
						printWarn(fmt.Sprintf("skipping duplicate file: %s", res.Location))
					}
					wg.Done()
					return
				} else {
					duplicates.Add(res.Bytes, res.Hash)
				}
			}

			if Trace {
				printTrace(fmt.Sprintf("nanoseconds process: %s: %d", res.Location, makeTimestampNano()-fileStartTime))
			}

			if !res.Binary {
				*output <- res
			} else {
				if Verbose {
					printWarn(fmt.Sprintf("skipping file identified as binary: %s", res.Location))
				}
			}

			wg.Done()
		}(res)
	}

	go func() {
		wg.Wait()
		close(*output)
	}()

	if Debug {
		printDebug(fmt.Sprintf("milliseconds proessing files: %d", makeTimestampMilli()-startTime))
	}
}
