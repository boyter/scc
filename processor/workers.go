package processor

import (
	"bytes"
	"fmt"
	"hash"
	"sync"
	"sync/atomic"

	"github.com/minio/blake2b-simd"
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
	SDocString         int64 = 9
)

const SheBang string = "#!"

// LineType what type of line are are processing
type LineType int32

// These are not meant to be CAMEL_CASE but as it us used by an external project we cannot change it
const (
	LINE_BLANK LineType = iota
	LINE_CODE
	LINE_COMMENT
)

// Taken from https://en.wikipedia.org/wiki/Byte_order_mark#Byte_order_marks_by_encoding
// These indicate that we cannot count the file correctly so we can at least warn the user
var ByteOrderMarks = [][]byte{
	{254, 255},            // UTF-16 BE
	{255, 254},            // UTF-16 LE
	{0, 0, 254, 255},      // UTF-32 BE
	{255, 254, 0, 0},      // UTF-32 LE
	{43, 47, 118, 56},     // UTF-7
	{43, 47, 118, 57},     // UTF-7
	{43, 47, 118, 43},     // UTF-7
	{43, 47, 118, 47},     // UTF-7
	{43, 47, 118, 56, 45}, // UTF-7
	{247, 100, 76},        // UTF-1
	{221, 115, 102, 115},  // UTF-EBCDIC
	{14, 254, 255},        // SCSU
	{251, 238, 40},        // BOCU-1
	{132, 49, 149, 51},    // GB-18030
}

var duplicates = CheckDuplicates{
	hashes: make(map[int64][][]byte),
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

func stringState(fileJob *FileJob, index int, endPoint int, stringTrie *Trie, endString []byte, currentState int64, ignoreEscape bool) (int, int64) {
	// Its not possible to enter this state without checking at least 1 byte so it is safe to check -1 here
	// without checking if it is out of bounds first
	for i := index; i < endPoint; i++ {
		index = i

		// If we hit a newline, return because we want to count the stats but keep
		// the current state so we end up back in this loop when the outer
		// one calls again
		if fileJob.Content[i] == '\n' {
			return i, currentState
		}

		// If we are in a literal string we want to ignore the \ check OR we aren't checking for special ones
		if ignoreEscape || fileJob.Content[i-1] != '\\' {
			if checkForMatchSingle(fileJob.Content[i], index, endPoint, endString, fileJob) {
				return i, SCode
			}
		}
	}

	return index, currentState
}

// This is a special state check pretty much only ever used by Python codebases
// but potentially it could be expanded to deal with other types
func docStringState(fileJob *FileJob, index int, endPoint int, stringTrie *Trie, endString []byte, currentState int64) (int, int64) {
	// Its not possible to enter this state without checking at least 1 byte so it is safe to check -1 here
	// without checking if it is out of bounds first
	for i := index; i < endPoint; i++ {
		index = i

		if fileJob.Content[i] == '\n' {
			return i, currentState
		}

		if fileJob.Content[i-1] != '\\' {
			if ok, _, _ := stringTrie.Match(fileJob.Content[i:]); ok != 0 {
				// So we have hit end of docstring at this point in which case check if only whitespace characters till the next
				// newline and if so we change to a comment otherwise to code
				// need to start the loop after ending definition of docstring, therefore adding the length of the string to
				// the index
				for j := index + len(endString); j <= endPoint; j++ {
					if fileJob.Content[j] == '\n' {
						if Debug {
							printDebug("Found newline so docstring is comment")
						}
						return i, SComment
					}

					if !isWhitespace(fileJob.Content[j]) {
						if Debug {
							printDebug(fmt.Sprintf("Found something not whitespace so is code: %s", string(fileJob.Content[j])))
						}
						return i, SCode
					}
				}

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
) (int, int64, []byte, [][]byte, bool) {
	for i := index; i < endPoint; i++ {
		curByte := fileJob.Content[i]
		index = i

		if curByte == '\n' {
			return i, currentState, endString, endComments, false
		}

		if isBinary(i, curByte) {
			fileJob.Binary = true
			return i, currentState, endString, endComments, false
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
				// If we are in string state then check what sort of string so we know if docstring OR ignoreescape string
				i, ignoreEscape := verifyIgnoreEscape(langFeatures, fileJob, index)

				// It is safe to -1 here as to enter the code state we need to have
				// transitioned from blank to here hence i should always be >= 1
				// This check is to ensure we aren't in a character declaration
				// TODO this should use language features
				if fileJob.Content[i-1] != '\\' {
					currentState = SString
				}

				return i, currentState, endString, endComments, ignoreEscape

			case TSlcomment:
				currentState = SCommentCode
				return i, currentState, endString, endComments, false

			case TMlcomment:
				if langFeatures.Nested || len(endComments) == 0 {
					endComments = append(endComments, endString)
					currentState = SMulticommentCode
					i += offsetJump - 1

					return i, currentState, endString, endComments, false
				}

			case TComplexity:
				if index == 0 || isWhitespace(fileJob.Content[index-1]) {
					fileJob.Complexity++
				}
			}
		}
	}

	return index, currentState, endString, endComments, false
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
) (int, int64, []byte, [][]byte, bool) {
	switch tokenType, offsetJump, endString := langFeatures.Tokens.Match(fileJob.Content[index:]); tokenType {
	case TMlcomment:
		if langFeatures.Nested || len(endComments) == 0 {
			endComments = append(endComments, endString)
			currentState = SMulticomment
			index += offsetJump - 1
			return index, currentState, endString, endComments, false
		}

	case TSlcomment:
		currentState = SComment
		return index, currentState, endString, endComments, false

	case TString:
		index, ignoreEscape := verifyIgnoreEscape(langFeatures, fileJob, index)

		for _, v := range langFeatures.Quotes {
			if v.End == string(endString) && v.DocString {
				currentState = SDocString
				return index, currentState, endString, endComments, ignoreEscape
			}
		}
		currentState = SString
		return index, currentState, endString, endComments, ignoreEscape

	case TComplexity:
		currentState = SCode
		if index == 0 || isWhitespace(fileJob.Content[index-1]) {
			fileJob.Complexity++
		}

	default:
		currentState = SCode
	}

	return index, currentState, endString, endComments, false
}

// Some languages such as C# have quoted strings like @"\" where no escape character is required
// this checks if there is one so we can cater for these cases
func verifyIgnoreEscape(langFeatures LanguageFeature, fileJob *FileJob, index int) (int, bool) {
	ignoreEscape := false

	// loop over the string states and if we have the special flag match, and if so we need to ensure we can handle them
	for i := 0; i < len(langFeatures.Quotes); i++ {
		if langFeatures.Quotes[i].DocString || langFeatures.Quotes[i].IgnoreEscape {
			// If so we need to check if where we are falls into these conditions
			isMatch := true
			for j := 0; j < len(langFeatures.Quotes[i].Start); j++ {
				if len(fileJob.Content) <= index+j || fileJob.Content[index+j] != langFeatures.Quotes[i].Start[j] {
					isMatch = false
					break
				}
			}

			// If we have a match then jump ahead enough so we don't pick it up again for cases like @"
			if isMatch {
				ignoreEscape = true
				index = index + len(langFeatures.Quotes[i].Start)
			}
		}
	}

	return index, ignoreEscape
}

// CountStats will process the fileJob
// If the file contains anything even just a newline its line count should be >= 1.
// If the file has a size of 0 its line count should be 0.
// Newlines belong to the line they started on so a file of \n means only 1 line
// This is the 'hot' path for the application and needs to be as fast as possible
func CountStats(fileJob *FileJob) {
	// If the file has a length of 0 it is is empty then we say it has no lines
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

	// TODO needs to be set via langFeatures.Quotes[0].IgnoreEscape for the matching feature
	ignoreEscape := false

	// For determining duplicates we need the below. The reason for creating
	// the byte array here is to avoid GC pressure. MD5 is in the standard library
	// and is fast enough to not warrant murmur3 hashing. No need to be
	// crypto secure here either so no need to eat the performance cost of a better
	// hash method
	var digest hash.Hash
	if Duplicates {
		digest = blake2b.New256()
	}

	for index := checkBomSkip(fileJob); index < len(fileJob.Content); index++ {
		// Based on our current state determine if the state should change by checking
		// what the character is. The below is very CPU bound so need to be careful if
		// changing anything in here and profile/measure afterwards!
		// NB that the order of the if statements matters and has been set to what in benchmarks is most efficient
		if !isWhitespace(fileJob.Content[index]) {

			switch currentState {
			case SCode:
				index, currentState, endString, endComments, ignoreEscape = codeState(
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
				index, currentState = stringState(fileJob, index, endPoint, langFeatures.Strings, endString, currentState, ignoreEscape)
			case SDocString:
				// For a docstring we can either move into blank in which case we count it as a docstring
				// or back into code in which case it should be counted as code
				index, currentState = docStringState(fileJob, index, endPoint, langFeatures.Strings, endString, currentState)
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
				index, currentState, endString, endComments, ignoreEscape = blankState(
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

		// We shouldn't normally need this, but unclosed strings or comments
		// might leave the index past the end of the file when we reach this
		// point.
		if index >= len(fileJob.Content) {
			return
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

			if NoLarge && fileJob.Lines >= LargeLineCount {
				// Save memory by unsetting the content as we no longer require it
				fileJob.Content = nil
				return
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
				if Trace {
					printTrace(fmt.Sprintf("%s line %d ended with state: %d: counted as code", fileJob.Location, fileJob.Lines, currentState))
				}
			case SComment, SMulticomment, SMulticommentBlank:
				fileJob.Comment++
				currentState = resetState(currentState)
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_COMMENT) {
						return
					}
				}
				if Trace {
					printTrace(fmt.Sprintf("%s line %d ended with state: %d: counted as comment", fileJob.Location, fileJob.Lines, currentState))
				}
			case SBlank:
				fileJob.Blank++
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_BLANK) {
						return
					}
				}
				if Trace {
					printTrace(fmt.Sprintf("%s line %d ended with state: %d: counted as blank", fileJob.Location, fileJob.Lines, currentState))
				}
			case SDocString:
				fileJob.Comment++
				if fileJob.Callback != nil {
					if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, LINE_COMMENT) {
						return
					}
				}
				if Trace {
					printTrace(fmt.Sprintf("%s line %d ended with state: %d: counted as comment", fileJob.Location, fileJob.Lines, currentState))
				}
			}
		}
	}

	if Duplicates {
		fileJob.Hash = digest.Sum(nil)
	}

	if MinifiedGenerated {
		avgLineByteCount := len(fileJob.Content) / int(fileJob.Lines)
		minifiedGeneratedCheck(avgLineByteCount, fileJob)
	}
}

func minifiedGeneratedCheck(avgLineByteCount int, fileJob *FileJob) {
	if avgLineByteCount >= MinifiedGeneratedLineByteLength {
		fileJob.Minified = true
		fileJob.Language = fileJob.Language + " (min)"

		if Verbose {
			printWarn(fmt.Sprintf("%s identified as minified/generated with average line byte length of %d >= %d", fileJob.Filename, avgLineByteCount, MinifiedGeneratedLineByteLength))
		}
	} else {
		if Debug {
			printDebug(fmt.Sprintf("%s not identified as minified/generated with average line byte length of %d < %d", fileJob.Filename, avgLineByteCount, MinifiedGeneratedLineByteLength))
		}
	}
}

// Check if we have any Byte Order Marks (BOM) in front of the file
func checkBomSkip(fileJob *FileJob) int {
	// UTF-8 BOM which if detected we should skip the BOM as we can then count correctly
	// []byte is UTF-8 BOM taken from https://en.wikipedia.org/wiki/Byte_order_mark#Byte_order_marks_by_encoding
	if bytes.HasPrefix(fileJob.Content, []byte{239, 187, 191}) {
		if Verbose {
			printWarn(fmt.Sprintf("UTF-8 BOM found for file %s skipping 3 bytes", fileJob.Filename))
		}
		return 3
	}

	// If we have one of the other BOM then we might not be able to count correctly so if verbose let the user know
	if Verbose {
		for _, v := range ByteOrderMarks {
			if bytes.HasPrefix(fileJob.Content, v) {
				printWarn(fmt.Sprintf("BOM found for file %s indicating it is not ASCII/UTF-8 and may be counted incorrectly or ignored as a binary file", fileJob.Filename))
			}
		}
	}

	return 0
}

// Reads and processes files from input chan in parallel, and sends results to
// output chan
func fileProcessorWorker(input chan *FileJob, output chan *FileJob) {
	var startTime int64
	var wg sync.WaitGroup

	for i := 0; i < FileProcessJobWorkers; i++ {
		wg.Add(1)
		go func() {
			reader := NewFileReader()

			for job := range input {
				atomic.CompareAndSwapInt64(&startTime, 0, makeTimestampMilli())

				fileStartTime := makeTimestampNano()

				content, err := reader.ReadFile(job.Location, int(job.Bytes))

				if Trace {
					printTrace(fmt.Sprintf("nanoseconds read into memory: %s: %d", job.Location, makeTimestampNano()-fileStartTime))
				}

				if err == nil {
					job.Content = content
					if processFile(job) {
						output <- job
					}
				} else {
					if Verbose {
						printWarn(fmt.Sprintf("error reading: %s %s", job.Location, err))
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

// Process a single file
// File must have been read to job.Content already
func processFile(job *FileJob) bool {
	fileStartTime := makeTimestampNano()

	contents := job.Content

	// Needs to always run to ensure the language is set
	job.Language = DetermineLanguage(job.Filename, job.Language, job.PossibleLanguages, job.Content)

	// If the type is #! we should check to see if we can identify
	if job.Language == SheBang {
		cutoff := 200

		// To avoid runtime panic check if the content we are cutting is smaller than 200
		if len(contents) < cutoff {
			cutoff = len(contents)
		}

		lang, err := DetectSheBang(string(contents[:cutoff]))
		if err != nil {
			if Verbose {
				printWarn(fmt.Sprintf("unable to determine #! language for %s", job.Location))
			}
			return false
		}
		if Verbose {
			printWarn(fmt.Sprintf("detected #! %s for %s", lang, job.Location))
		}

		job.Language = lang
		LoadLanguageFeature(lang)
	}

	CountStats(job)

	if Duplicates {
		duplicates.mux.Lock()
		if duplicates.Check(job.Bytes, job.Hash) {
			if Verbose {
				printWarn(fmt.Sprintf("skipping duplicate file: %s", job.Location))
			}

			duplicates.mux.Unlock()
			return false
		}

		duplicates.Add(job.Bytes, job.Hash)
		duplicates.mux.Unlock()
	}

	if IgnoreMinifiedGenerate && job.Minified {
		if Verbose {
			printWarn(fmt.Sprintf("skipping minified/generated file: %s", job.Location))
		}
		return false
	}

	if NoLarge && job.Lines >= LargeLineCount {
		if Verbose {
			printWarn(fmt.Sprintf("skipping large file due to line length: %s", job.Location))
		}
		return false
	}

	if Trace {
		printTrace(fmt.Sprintf("nanoseconds process: %s: %d", job.Location, makeTimestampNano()-fileStartTime))
	}

	if !job.Binary {
		return true
	} else {
		if Verbose {
			printWarn(fmt.Sprintf("skipping file identified as binary: %s", job.Location))
		}
		return false
	}
}
