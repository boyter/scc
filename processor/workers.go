// SPDX-License-Identifier: MIT OR Unlicense

package processor

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"strings"
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

// SheBang is a global constant for indicating a shebang file header
const SheBang string = "#!"

// LineType what type of line are are processing
type LineType int32

func (lt LineType) String() string {
	switch lt {
	case LINE_BLANK:
		return "blank"
	case LINE_CODE:
		return "code"
	case LINE_COMMENT:
		return "comment"
	default:
		return fmt.Sprintf("%d", lt)
	}
}

// These are not meant to be CAMEL_CASE but as it us used by an external project we cannot change it
const (
	LINE_BLANK LineType = iota
	LINE_CODE
	LINE_COMMENT
)

// ByteOrderMarks are taken from https://en.wikipedia.org/wiki/Byte_order_mark#Byte_order_marks_by_encoding
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

func shouldProcess(currentByte byte, processBytesMask uint64) bool {
	k := BloomTable[currentByte]
	if k&processBytesMask != k {
		return false
	}
	return true
}

// Some languages such as C# have quoted strings like @"\" where no escape character is required
// this checks if there is one so we can cater for these cases
func verifyIgnoreEscape(langFeatures *LanguageFeature, fileJob *FileJob, index int) (int, bool, bool) {
	docString := false
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
				docString = langFeatures.Quotes[i].DocString
				ignoreEscape = true
				index = index + len(langFeatures.Quotes[i].Start)
			}
		}
	}

	return index, docString, ignoreEscape
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

	var lineStart int
	var lineType LineType
	var currentState State = &StateBlank{}

	// For determining duplicates we need the below. The reason for creating
	// the byte array here is to avoid GC pressure. MD5 is in the standard library
	// and is fast enough to not warrant murmur3 hashing. No need to be
	// crypto secure here either so no need to eat the performance cost of a better
	// hash method
	if Duplicates {
		fileJob.Hash = blake2b.New256()
	}

	for index := checkBomSkip(fileJob); index < int(fileJob.Bytes); index++ {
		// Based on our current state determine if the state should change by checking
		// what the character is. The below is very CPU bound so need to be careful if
		// changing anything in here and profile/measure afterwards!
		// NB that the order of the if statements matters and has been set to what in benchmarks is most efficient
		if !isWhitespace(fileJob.Content[index]) {
			index, lineType, currentState = currentState.Process(fileJob, &langFeatures, index, lineType)
			if Trace {
				printTrace(fmt.Sprintf("state transition @ %d:%d: line=%s state=%s", fileJob.Lines+1, index-lineStart, lineType, currentState))
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
		if fileJob.Content[index] == '\n' || index >= fileJob.EndPoint {
			fileJob.Lines++

			if NoLarge && fileJob.Lines >= LargeLineCount {
				// Save memory by unsetting the content as we no longer require it
				fileJob.Content = nil
				return
			}

			switch lineType {
			case LINE_CODE:
				fileJob.Code++
			case LINE_COMMENT:
				fileJob.Comment++
			case LINE_BLANK:
				fileJob.Blank++
			}

			if Trace {
				printTrace(fmt.Sprintf(
					"%s line %d [%s] ended with state: %v: counted as %v",
					fileJob.Location,
					fileJob.Lines,
					string(fileJob.Content[lineStart:index]),
					currentState,
					lineType,
				))
				//printTrace(fmt.Sprintf(`line %d: "%s"`, fileJob.Lines, string(fileJob.Content[lineStart:index])))

				// lineStart is only used to produce the line trace, so it's
				// safe to update it inside the condition
				lineStart = index + 1
			}

			if fileJob.Callback != nil {
				if !fileJob.Callback.ProcessLine(fileJob, fileJob.Lines, lineType) {
					return
				}
			}

			lineType, currentState = currentState.Reset()
		}
	}

	isGenerated := false

	if Generated {
		headLen := 1000
		if headLen >= len(fileJob.Content) {
			headLen = len(fileJob.Content) - 1
		}
		head := bytes.ToLower(fileJob.Content[0:headLen])
		for _, marker := range GeneratedMarkers {
			if bytes.Contains(head, bytes.ToLower([]byte(marker))) {
				fileJob.Generated = true
				fileJob.Language = fileJob.Language + " (gen)"
				isGenerated = true

				if Verbose {
					printWarn(fmt.Sprintf("%s identified as isGenerated with heading comment", fileJob.Filename))
				}

				break
			}
		}
	}

	// check if 0 as well to avoid divide by zero https://github.com/boyter/scc/issues/223
	if !isGenerated && Minified && fileJob.Lines != 0 {
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
	var fileCount int64
	var gcEnabled int64
	var wg sync.WaitGroup

	for i := 0; i < FileProcessJobWorkers; i++ {
		wg.Add(1)
		go func() {
			reader := NewFileReader()

			for job := range input {
				atomic.CompareAndSwapInt64(&startTime, 0, makeTimestampMilli())

				loc := job.Location
				if job.Symlocation != "" {
					loc = job.Symlocation
				}

				fileStartTime := makeTimestampNano()
				content, err := reader.ReadFile(loc, int(job.Bytes))
				atomic.AddInt64(&fileCount, 1)

				if atomic.LoadInt64(&gcEnabled) == 0 && atomic.LoadInt64(&fileCount) >= int64(GcFileCount) {
					debug.SetGCPercent(gcPercent)
					atomic.AddInt64(&gcEnabled, 1)
					if Verbose {
						printWarn("read file limit exceeded GC re-enabled")
					}
				}

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

	remapped := false
	if RemapAll != "" {
		hardRemapLanguage(job)
	}

	// If the type is #! we should check to see if we can identify
	if job.Language == SheBang {
		if RemapUnknown != "" {
			remapped = unknownRemapLanguage(job)
		}

		// if we didn't remap we then want to see if its a #! map
		if remapped == false {
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
	}

	CountStats(job)

	if Duplicates {
		duplicates.mux.Lock()
		jobHash := job.Hash.Sum(nil)
		if duplicates.Check(job.Bytes, jobHash) {
			if Verbose {
				printWarn(fmt.Sprintf("skipping duplicate file: %s", job.Location))
			}

			duplicates.mux.Unlock()
			return false
		}

		duplicates.Add(job.Bytes, jobHash)
		duplicates.mux.Unlock()
	}

	if IgnoreMinified && job.Minified {
		if Verbose {
			printWarn(fmt.Sprintf("skipping minified file: %s", job.Location))
		}
		return false
	}

	if IgnoreGenerated && job.Generated {
		if Verbose {
			printWarn(fmt.Sprintf("skipping generated file: %s", job.Location))
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

	if job.Binary {
		if Verbose {
			printWarn(fmt.Sprintf("skipping file identified as binary: %s", job.Location))
		}
		return false
	}

	return true
}

func hardRemapLanguage(job *FileJob) bool {
	remapped := false
	for _, s := range strings.Split(RemapAll, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {
			cutoff := 1000 // 1000 bytes into the file to look

			// To avoid runtime panic check if the content we are cutting is smaller than 1000
			if len(job.Content) < cutoff {
				cutoff = len(job.Content)
			}

			if strings.Contains(string(job.Content[:cutoff]), t[0]) {
				job.Language = t[1]
				remapped = true

				if Verbose {
					printWarn(fmt.Sprintf("hard remapping: %s to %s", job.Location, job.Language))
				}
			}
		}
	}

	return remapped
}

func unknownRemapLanguage(job *FileJob) bool {
	remapped := false
	for _, s := range strings.Split(RemapUnknown, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {
			cutoff := 1000 // 1000 bytes into the file to look

			// To avoid runtime panic check if the content we are cutting is smaller than 1000
			if len(job.Content) < cutoff {
				cutoff = len(job.Content)
			}

			if strings.Contains(string(job.Content[:cutoff]), t[0]) {
				if Verbose {
					printWarn(fmt.Sprintf("unknown remapping: %s to %s", job.Location, job.Language))
				}

				job.Language = t[1]
				remapped = true
			}
		}
	}

	return remapped
}
