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

func checkForMatch(currentByte byte, index int, endPoint int, mask byte, matches [][]byte, fileJob *FileJob) bool {
	if currentByte & mask != currentByte {
		return false
	}

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

func checkForMatchSingle(currentByte byte, index int, endPoint int, mask byte, matches []byte, fileJob *FileJob) bool {
	if currentByte & mask != currentByte {
		return false
	}

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

func checkForMatchMultiOpen(currentByte byte, index int, endPoint int, mask byte, matches []OpenClose, fileJob *FileJob) (int, []byte) {
	if currentByte & mask != currentByte {
		return 0, []byte{}
	}

	potentialMatch := true
	for i := 0; i < len(matches); i++ {
		// This check saves a lot of needless processing and speeds up the loop
		if currentByte == matches[i].Open[0] {
			potentialMatch = true

			for j := 0; j < len(matches[i].Open); j++ {
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

func checkComplexity(currentByte byte, index int, endPoint int, mask byte, matches [][]byte, complexityBytes []byte, fileJob *FileJob) int {
	// Special case if the thing we are matching is not the first thing in the file
	// then we need to check that there was a whitespace before it
	if index != 0 {
		// If the byte before our current position is not a whitespace then return false

		if !isWhitespace(fileJob.Content[index-1]) {
			return 0
		}
	}

	if currentByte & mask != currentByte {
		return 0
	}

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

func isBinary(currentByte byte) bool {
	if !DisableCheckBinary && currentByte == 0 {
		return true
	}

	return false
}

func resetState(currentState int64) int64 {
	if currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_CODE {
		currentState = S_MULTICOMMENT
	} else if currentState == S_STRING {
		currentState = S_STRING
	} else {
		currentState = S_BLANK
	}

	return currentState
}

func shouldProcess(currentByte, processBytesMask byte, processBytes []byte) bool {
	if currentByte & processBytesMask != currentByte {
		return false
	}
	// Determine if we should process the state at all based on what states we know about
	for i:=0; i<len(processBytes); i++ {
		if currentByte == processBytes[i] {
			return true
		}
	}

	return false
}

func stringState(fileJob *FileJob, index *int, endPoint int, stringMask byte, endString []byte, currentState int64) int64 {
	// Its not possible to enter this state without checking at least 1 byte so it is safe to check -1 here
	// without checking if it is out of bounds first
	for i:= *index; i < endPoint; i++ {
		*index = i
		if fileJob.Content[i] == '\n' {
			return currentState
		}

		if fileJob.Content[i-1] != '\\' && checkForMatchSingle(fileJob.Content[i], i, endPoint, stringMask, endString, fileJob) {
			return S_CODE
		}
	}

	return currentState
}

func codeState(
	fileJob *FileJob,
	index int,
	processMask byte,
	processBytes []byte,
	offsetJump int,
	endString []byte,
	endPoint int,
	stringMask byte,
	stringChecks []OpenClose,
	currentState int64,
	slCommentMask byte,
	singleLineCommentChecks [][]byte,
	endComments [][]byte,
	nested bool,
	mlCommentMask byte,
	multiLineCommentChecks []OpenClose,
	complexityMask byte,
	complexityChecks [][]byte,
	complexityBytes []byte,
) (int, int, []byte, int64, [][]byte) {

	for i := index; i < endPoint; i++ {
		index = i

		if fileJob.Content[i] == '\n' {
			return index, offsetJump, endString, currentState, endComments
		}

		if shouldProcess(fileJob.Content[i], processMask, processBytes) {
			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[i], i, endPoint, stringMask, stringChecks, fileJob)
			if offsetJump != 0 {
				currentState = S_STRING
				return i, offsetJump, endString, currentState, endComments
			}

			if checkForMatch(fileJob.Content[i], i, endPoint, slCommentMask, singleLineCommentChecks, fileJob) {
				currentState = S_COMMENT_CODE
				return i, offsetJump, endString, currentState, endComments
			}
			if len(endComments) == 0 || nested {
				offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[i], i, endPoint, mlCommentMask, multiLineCommentChecks, fileJob)
				if offsetJump != 0 {
					endComments = append(endComments, endString)
					currentState = S_MULTICOMMENT_CODE
					i += offsetJump - 1
					return i, offsetJump, endString, currentState, endComments
				}
			}
			if !Complexity {
				offsetJump = checkComplexity(fileJob.Content[i], i, endPoint, complexityMask, complexityChecks, complexityBytes, fileJob)
				if offsetJump != 0 {
					fileJob.Complexity++
				}
			}
		}
	}

	return index, offsetJump, endString, currentState, endComments
}

func commentState(fileJob *FileJob, index int, endPoint int, slCommentMask byte, endComments [][]byte, offsetJump int, currentState int64, nested bool, endString []byte, mlCommentMask byte, multiLineCommentChecks []OpenClose) (int, [][]byte, int, int64, []byte) {
	for i := index; i < endPoint; i++ {
		index = i

		if fileJob.Content[i] == '\n' {
			return index, endComments, offsetJump, currentState, endString
		}

		if checkForMatchSingle(fileJob.Content[index], index, endPoint, slCommentMask, endComments[len(endComments)-1], fileJob) {
			// set offset jump here
			offsetJump = len(endComments[len(endComments)-1])
			endComments = endComments[:len(endComments)-1]

			if len(endComments) == 0 {
				// If we started as multiline code switch back to code so we count correctly
				// IE i := 1 /* for the lols */
				// TODO is that required? Might still be required to count correctly
				if currentState == S_MULTICOMMENT_CODE {
					currentState = S_CODE // TODO pointless to change here, just set S_MULTICOMMENT_BLANK
				} else {
					currentState = S_MULTICOMMENT_BLANK
				}
			}

			index += offsetJump - 1
			return index, endComments, offsetJump, currentState, endString
		}
		// Check if we are entering another multiline comment
		// This should come below check for match single as it speeds up processing
		if nested || len(endComments) == 0 {
			offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, mlCommentMask, multiLineCommentChecks, fileJob)
			if offsetJump != 0 {
				endComments = append(endComments, endString)
				index += offsetJump - 1
				return index, endComments, offsetJump, currentState, endString
			}
		}
	}

	return index, endComments, offsetJump, currentState, endString
}

func blankState(
	nested bool,
	endComments [][]byte,
	offsetJump int,
	endString []byte,
	fileJob *FileJob,
	index int,
	endPoint int,
	mlCommentMask byte,
	multiLineCommentChecks []OpenClose,
	currentState int64,
	slCommentMask byte,
	singleLineCommentChecks [][]byte,
	stringMask byte,
	stringChecks []OpenClose,
	complexityMask byte,
	complexityChecks [][]byte,
	complexityBytes []byte,
) (int, int64, []byte, [][]byte) {
	if nested || len(endComments) == 0 {
		offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, mlCommentMask, multiLineCommentChecks, fileJob)
		if offsetJump != 0 {
			endComments = append(endComments, endString)
			currentState = S_MULTICOMMENT
			index += offsetJump - 1
			return index, currentState, endString, endComments
		}
	}
	// Moving this above nested comments slows performance to leave it below
	if checkForMatch(fileJob.Content[index], index, endPoint, slCommentMask, singleLineCommentChecks, fileJob) {
		currentState = S_COMMENT
		return index, currentState, endString, endComments
	}
	// Moving this above nested comments or single line comment checks slows performance so leave it below
	offsetJump, endString = checkForMatchMultiOpen(fileJob.Content[index], index, endPoint, stringMask, stringChecks, fileJob)
	if offsetJump != 0 {
		currentState = S_STRING
		return index, currentState, endString, endComments
	}
	currentState = S_CODE
	if !Complexity {
		offsetJump = checkComplexity(fileJob.Content[index], index, endPoint, complexityMask, complexityChecks, complexityBytes, fileJob)
		if offsetJump != 0 {
			fileJob.Complexity++
		}
	}

	return index, currentState, endString, endComments
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

	langFeatures := LanguageFeatures[fileJob.Language]

	complexityMask := langFeatures.ComplexityCheckMask
	complexityChecks := langFeatures.ComplexityChecks
	complexityBytes := langFeatures.ComplexityBytes
	slCommentMask := langFeatures.SingleLineCommentMask
	singleLineCommentChecks := langFeatures.SingleLineComment
	mlCommentMask := langFeatures.MultiLineCommentMask
	multiLineCommentChecks := langFeatures.MultiLineComment
	stringMask := langFeatures.StringCheckMask
	stringChecks := langFeatures.StringChecks
	nested := langFeatures.Nested
	processMask := langFeatures.ProcessMask
	processBytes := langFeatures.ProcessBytes

	endPoint := int(fileJob.Bytes - 1)
	currentState := S_BLANK
	endComments := [][]byte{}
	endString := []byte{}

	// If we have checked bytes ahead of where we are we can jump ahead and save time
	// this value stores that jump
	offsetJump := 0

	// For determining duplicates we need the below. The reason for creating
	// the byte array here is to avoid GC pressure. MD5 is in the standard library
	// and is fast enough to not warrant murmur3 hashing. No need to be
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
		if index < 10000 && isBinary(fileJob.Content[index]) {
			fileJob.Binary = true
			return
		}

		// Based on our current state determine if the state should change by checking
		// what the character is. The below is very CPU bound so need to be careful if
		// changing anything in here and profile/measure afterwards!
		// NB that the order of the if statements matters and has been set to what in benchmarks is most efficient
		if !isWhitespace(fileJob.Content[index]) {
			switch currentState {
			case S_CODE:
				index, offsetJump, endString, currentState, endComments = codeState(
					fileJob,
					index,
					processMask,
					processBytes,
					offsetJump,
					endString,
					endPoint,
					stringMask,
					stringChecks,
					currentState,
					slCommentMask,
					singleLineCommentChecks,
					endComments,
					nested,
					mlCommentMask,
					multiLineCommentChecks,
					complexityMask,
					complexityChecks,
					complexityBytes,
				)
			case S_STRING:
				currentState = stringState(fileJob, &index, endPoint, stringMask, endString, currentState)
			case S_MULTICOMMENT, S_MULTICOMMENT_CODE:
				index, endComments, offsetJump, currentState, endString = commentState(
					fileJob,
					index,
					endPoint,
					slCommentMask,
					endComments,
					offsetJump,
					currentState,
					nested,
					endString,
					mlCommentMask,
					multiLineCommentChecks,
				)
			case S_BLANK, S_MULTICOMMENT_BLANK:
				// From blank we can move into comment, move into a multiline comment
				// or move into code but we can only do one.
				index, currentState, endString, endComments = blankState(
					nested,
					endComments,
					offsetJump,
					endString,
					fileJob,
					index,
					endPoint,
					mlCommentMask,
					multiLineCommentChecks,
					currentState,
					slCommentMask,
					singleLineCommentChecks,
					stringMask,
					stringChecks,
					complexityMask,
					complexityChecks,
					complexityBytes,
				)
			}
		}

		// This means the end of processing the line so calculate the stats according to what state
		// we are currently in
		if fileJob.Content[index] == '\n' || index >= endPoint {
			fileJob.Lines++

			if Trace {
				printTrace(fmt.Sprintf("%s line %d ended with state: %d", fileJob.Location, fileJob.Lines, currentState))
			}

			switch {
			case currentState == S_CODE || currentState == S_STRING || currentState == S_COMMENT_CODE || currentState == S_MULTICOMMENT_CODE:
				{
					fileJob.Code++
					currentState = resetState(currentState)
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_CODE) {
							return
						}
					}
				}
			case currentState == S_COMMENT || currentState == S_MULTICOMMENT || currentState == S_MULTICOMMENT_BLANK:
				{
					fileJob.Comment++
					currentState = resetState(currentState)
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_COMMENT) {
							return
						}
					}
				}
			case currentState == S_BLANK:
				{
					fileJob.Blank++
					if fileJob.Callback != nil {
						if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_BLANK) {
							return
						}
					}
				}
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
