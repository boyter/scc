package processor

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// The below are used as identifiers for the code state machine
const (
	SBlank             int64 = 1
	SCode              int64 = 2
	SComment           int64 = 3
	SCommentCode       int64 = 4 // Indicates comment after code
	SMulticomment      int64 = 5
	SMulticommentCode  int64 = 6 // Indicates multi comment after code
	SMulticommentBlank int64 = 7 // Indicates multi comment ended with blank afterwards
	SString            int64 = 8
)

// LineType what type of line are are processing
type LineType int32

// These are not meant to be CAMEL_CASE but as it us used by an external project we cannot change it
const (
	LINE_BLANK LineType = iota
	LINE_CODE
	LINE_COMMENT
)

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

func isWhitespace(currentByte byte) bool {
	if currentByte != ' ' && currentByte != '\t' && currentByte != '\n' && currentByte != '\r' {
		return false
	}

	return true
}

// Check if this file is binary by checking for nul byte and if so bail out
// this is how GNU Grep, git and ripgrep check for binary files
func isBinary(index int, currentByte byte) bool {
	if index < 10000 && !DisableCheckBinary && currentByte == 0 {
		return true
	}

	return false
}

func shouldProcess(currentByte, processBytesMask byte) bool {
	if currentByte&processBytesMask != currentByte {
		return false
	}
	return true
}

func resetState(currentState int64) int64 {
	if currentState == SMulticomment || currentState == SMulticommentCode {
		currentState = SMulticomment
	} else if currentState == SString {
		currentState = SString
	} else {
		currentState = SBlank
	}

	return currentState
}

func stringState(fileJob *FileJob, index int, endPoint int, stringTrie *Trie, endString []byte, currentState int64) (int, int64) {
	// Its not possible to enter this state without checking at least 1 byte so it is safe to check -1 here
	// without checking if it is out of bounds first
	for i := index; i < endPoint; i++ {
		index = i

		if fileJob.Content[i] == '\n' {
			return i, currentState
		}

		if fileJob.Content[i-1] != '\\' {
			if ok, _, _ := stringTrie.Match(fileJob.Content[i:]); ok != 0 {
				return i, SCode
			}
		}
	}

	return index, currentState
}

func codeState(
	fileJob *FileJob,
	index int,
	endPoint int,
	currentState int64,
	endString []byte,
	endComments [][]byte,
	langFeatures LanguageFeature,
	digest *hash.Hash,
) (int, int64, []byte, [][]byte) {
	for i := index; i < endPoint; i++ {
		curByte := fileJob.Content[i]
		index = i

		if curByte == '\n' {
			return i, currentState, endString, endComments
		}

		if isBinary(i, curByte) {
			fileJob.Binary = true
			return i, currentState, endString, endComments
		}

		if shouldProcess(curByte, langFeatures.ProcessMask) {
			if Duplicates {
				// Technically this is wrong because we skip bytes so this is not a true
				// hash of the file contents, but for duplicate files it shouldn't matter
				// as both will skip the same way
				digestible := []byte{fileJob.Content[index]}
				(*digest).Write(digestible)
			}

			switch tokenType, offsetJump, endString := langFeatures.Tokens.Match(fileJob.Content[i:]); tokenType {
			case TString:
				currentState = SString
				return i, currentState, endString, endComments

			case TSlcomment:
				currentState = SCommentCode
				return i, currentState, endString, endComments

			case TMlcomment:
				if langFeatures.Nested || len(endComments) == 0 {
					endComments = append(endComments, endString)
					currentState = SMulticommentCode
					i += offsetJump - 1
					return i, currentState, endString, endComments
				}

			case TComplexity:
				if index == 0 || isWhitespace(fileJob.Content[index-1]) {
					fileJob.Complexity++
				}
			}
		}
	}

	return index, currentState, endString, endComments
}

func commentState(fileJob *FileJob, index int, endPoint int, currentState int64, endComments [][]byte, endString []byte, langFeatures LanguageFeature) (int, int64, []byte, [][]byte) {
	for i := index; i < endPoint; i++ {
		curByte := fileJob.Content[i]
		index = i

		if curByte == '\n' {
			return i, currentState, endString, endComments
		}

		if checkForMatchSingle(curByte, index, endPoint, endComments[len(endComments)-1], fileJob) {
			// set offset jump here
			offsetJump := len(endComments[len(endComments)-1])
			endComments = endComments[:len(endComments)-1]

			if len(endComments) == 0 {
				// If we started as multiline code switch back to code so we count correctly
				// IE i := 1 /* for the lols */
				// TODO is that required? Might still be required to count correctly
				if currentState == SMulticommentCode {
					currentState = SCode // TODO pointless to change here, just set S_MULTICOMMENT_BLANK
				} else {
					currentState = SMulticommentBlank
				}
			}

			i += offsetJump - 1
			return i, currentState, endString, endComments
		}
		// Check if we are entering another multiline comment
		// This should come below check for match single as it speeds up processing
		if langFeatures.Nested || len(endComments) == 0 {
			if ok, offsetJump, endString := langFeatures.MultiLineComments.Match(fileJob.Content[i:]); ok != 0 {
				endComments = append(endComments, endString)
				i += offsetJump - 1
				return i, currentState, endString, endComments
			}
		}
	}

	return index, currentState, endString, endComments
}

func blankState(
	fileJob *FileJob,
	index int,
	endPoint int,
	currentState int64,
	endComments [][]byte,
	endString []byte,
	langFeatures LanguageFeature,
) (int, int64, []byte, [][]byte) {
	switch tokenType, offsetJump, endString := langFeatures.Tokens.Match(fileJob.Content[index:]); tokenType {
	case TMlcomment:
		if langFeatures.Nested || len(endComments) == 0 {
			endComments = append(endComments, endString)
			currentState = SMulticomment
			index += offsetJump - 1
			return index, currentState, endString, endComments
		}

	case TSlcomment:
		currentState = SComment
		return index, currentState, endString, endComments

	case TString:
		currentState = SString
		return index, currentState, endString, endComments

	case TComplexity:
		currentState = SCode
		if index == 0 || isWhitespace(fileJob.Content[index-1]) {
			fileJob.Complexity++
		}

	default:
		currentState = SCode
	}

	return index, currentState, endString, endComments
}

// CountStats will process the fileJob
// If the file contains anything even just a newline its line count should be >= 1.
// If the file has a size of 0 its line count should be 0.
// Newlines belong to the line they started on so a file of \n means only 1 line
// This is the 'hot' path for the application and needs to be as fast as possible
func CountStats(fileJob *FileJob) {

	// Needs to always run to ensure the language is set
	determineLanguage(fileJob)

	// If the file has a length of 0 it is is empty then we say it has no lines
	fileJob.Bytes = int64(len(fileJob.Content))
	if fileJob.Bytes == 0 {
		fileJob.Lines = 0
		return
	}

	LanguageFeaturesMutex.Lock()
	langFeatures := LanguageFeatures[fileJob.Language]
	LanguageFeaturesMutex.Unlock()

	if langFeatures.Complexity == nil {
		langFeatures.Complexity = &Trie{}
	}
	if langFeatures.SingleLineComments == nil {
		langFeatures.SingleLineComments = &Trie{}
	}
	if langFeatures.MultiLineComments == nil {
		langFeatures.MultiLineComments = &Trie{}
	}
	if langFeatures.Strings == nil {
		langFeatures.Strings = &Trie{}
	}
	if langFeatures.Tokens == nil {
		langFeatures.Tokens = &Trie{}
	}

	endPoint := int(fileJob.Bytes - 1)
	currentState := SBlank
	endComments := [][]byte{}
	endString := []byte{}

	// For determining duplicates we need the below. The reason for creating
	// the byte array here is to avoid GC pressure. MD5 is in the standard library
	// and is fast enough to not warrant murmur3 hashing. No need to be
	// crypto secure here either so no need to eat the performance cost of a better
	// hash method
	var digest hash.Hash
	if Duplicates {
		digest = md5.New()
	}

	for index := 0; index < len(fileJob.Content); index++ {

		// Based on our current state determine if the state should change by checking
		// what the character is. The below is very CPU bound so need to be careful if
		// changing anything in here and profile/measure afterwards!
		// NB that the order of the if statements matters and has been set to what in benchmarks is most efficient
		if !isWhitespace(fileJob.Content[index]) {

			switch currentState {
			case SCode:
				index, currentState, endString, endComments = codeState(
					fileJob,
					index,
					endPoint,
					currentState,
					endString,
					endComments,
					langFeatures,
					&digest,
				)
			case SString:
				index, currentState = stringState(fileJob, index, endPoint, langFeatures.Strings, endString, currentState)
			case SMulticomment, SMulticommentCode:
				index, currentState, endString, endComments = commentState(
					fileJob,
					index,
					endPoint,
					currentState,
					endComments,
					endString,
					langFeatures,
				)
			case SBlank, SMulticommentBlank:
				// From blank we can move into comment, move into a multiline comment
				// or move into code but we can only do one.
				index, currentState, endString, endComments = blankState(
					fileJob,
					index,
					endPoint,
					currentState,
					endComments,
					endString,
					langFeatures,
				)
			}
		}

		// Only check the first 10000 characters for null bytes indicating a binary file
		// and if we find it then we return otherwise carry on and ignore binary markers
		if index < 10000 && fileJob.Binary {
			return
		}

		// This means the end of processing the line so calculate the stats according to what state
		// we are currently in
		if fileJob.Content[index] == '\n' || index >= endPoint {
			fileJob.Lines++

			if Trace {
				printTrace(fmt.Sprintf("%s line %d ended with state: %d", fileJob.Location, fileJob.Lines, currentState))
			}

			switch currentState {
			case SCode, SString, SCommentCode, SMulticommentCode:
				fileJob.Code++
				currentState = resetState(currentState)
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_CODE) {
						return
					}
				}
			case SComment, SMulticomment, SMulticommentBlank:
				fileJob.Comment++
				currentState = resetState(currentState)
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_COMMENT) {
						return
					}
				}
			case SBlank:
				fileJob.Blank++
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_BLANK) {
						return
					}
				}
			}
		}
	}

	if Duplicates {
		fileJob.Hash = digest.Sum(nil)
	}

	// Save memory by unsetting the content as we no longer require it
	fileJob.Content = nil
}

type languageGuess struct {
	Name  string
	Count int
}

// Given a filejob which could have multiple language types make a guess to the type
// based on keywords supplied, which is similar to how https://github.com/vmchale/polyglot does it
// If however there is only a single language we
func determineLanguage(fileJob *FileJob) {

	// If being called through an API its possible nothing is set here and as
	// such should just return as the Language value should have already been set
	if len(fileJob.PossibleLanguages) == 0 {
		return
	}

	// There should only be two possibilities now, either we have a single language
	// in which case we set it and return
	// or we have multiple in which case we try to determine it heuristically
	if len(fileJob.PossibleLanguages) == 1 {
		fileJob.Language = fileJob.PossibleLanguages[0]
		return
	}

	startTime := makeTimestampNano()

	var toCheck string
	if len(fileJob.Content) > 2000 {
		toCheck = string(fileJob.Content)[:2000]
	} else {
		toCheck = string(fileJob.Content)
	}

	toSort := []languageGuess{}
	for _, lan := range fileJob.PossibleLanguages {
		LanguageFeaturesMutex.Lock()
		langFeatures := LanguageFeatures[lan]
		LanguageFeaturesMutex.Unlock()

		count := 0
		for _, key := range langFeatures.Keywords {
			if strings.Contains(toCheck, key) {
				fileJob.Language = lan
				count++
			}
		}

		toSort = append(toSort, languageGuess{Name: lan, Count: count})
	}

	sort.Slice(toSort, func(i, j int) bool {
		return toSort[i].Count > toSort[j].Count
	})

	if Verbose {
		printWarn(fmt.Sprintf("guessing language %s for file %s", toSort[0].Name, fileJob.Filename))
	}

	if Trace {
		printTrace(fmt.Sprintf("nanoseconds to guess language: %s: %d", fileJob.Filename, makeTimestampNano()-startTime))
	}

	if len(toSort) != 0 {
		fileJob.Language = toSort[0].Name
	}
}

// Reads entire file into memory and then pushes it onto the next queue
func fileReaderWorker(input chan *FileJob, output chan *FileJob) {
	var startTime int64
	var wg sync.WaitGroup

	for i := 0; i < FileReadJobWorkers; i++ {
		wg.Add(1)
		go func() {
			for res := range input {
				atomic.CompareAndSwapInt64(&startTime, 0, makeTimestampMilli())

				fileStartTime := makeTimestampNano()
				content, err := ioutil.ReadFile(res.Location)

				if Trace {
					printTrace(fmt.Sprintf("nanoseconds read into memory: %s: %d", res.Location, makeTimestampNano()-fileStartTime))
				}

				if err == nil {
					res.Content = content
					output <- res
				} else {
					if Verbose {
						printWarn(fmt.Sprintf("error reading: %s %s", res.Location, err))
					}
				}
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)

		if Debug {
			printDebug(fmt.Sprintf("milliseconds reading files into memory: %d", makeTimestampMilli()-startTime))
		}
	}()
}

var duplicates = CheckDuplicates{
	hashes: make(map[int64][][]byte),
}

// Does the actual processing of stats and as such contains the hot path CPU call
func fileProcessorWorker(input chan *FileJob, output chan *FileJob) {
	var startTime int64
	var wg sync.WaitGroup
	for i := 0; i < FileProcessJobWorkers; i++ {
		wg.Add(1)
		go func() {
			for res := range input {
				atomic.CompareAndSwapInt64(&startTime, 0, makeTimestampMilli())

				fileStartTime := makeTimestampNano()
				CountStats(res)

				if Duplicates {
					duplicates.mux.Lock()
					if duplicates.Check(res.Bytes, res.Hash) {
						if Verbose {
							printWarn(fmt.Sprintf("skipping duplicate file: %s", res.Location))
						}

						duplicates.mux.Unlock()
						continue
					}

					duplicates.Add(res.Bytes, res.Hash)
					duplicates.mux.Unlock()
				}

				if Trace {
					printTrace(fmt.Sprintf("nanoseconds process: %s: %d", res.Location, makeTimestampNano()-fileStartTime))
				}

				if !res.Binary {
					output <- res
				} else {
					if Verbose {
						printWarn(fmt.Sprintf("skipping file identified as binary: %s", res.Location))
					}
				}
			}

			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(output)
	}()

	if Debug {
		printDebug(fmt.Sprintf("milliseconds processing files: %d", makeTimestampMilli()-startTime))
	}
}
