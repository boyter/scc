package processor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
)

// Flags set via the CLI which control how the output is displayed
var Files = false
var Languages = false
var Verbose = false
var Debug = false
var Trace = false
var Duplicates = false
var Complexity = false
var More = false
var Cocomo = false
var DisableCheckBinary = false
var SortBy = ""
var Format = ""
var FileOutput = ""
var PathBlacklist = ""
var FileListQueueSize = runtime.NumCPU()
var FileReadJobQueueSize = runtime.NumCPU()
var FileReadContentJobQueueSize = runtime.NumCPU()
var FileProcessJobQueueSize = runtime.NumCPU()
var FileSummaryJobQueueSize = runtime.NumCPU()
var WhiteListExtensions = ""
var AverageWage int64 = 56286
var GcFileCount = 10000
var gcPercent = -1

// Not set via flags but by arguments following the the flags
var DirFilePaths = []string{}

// Loaded from the JSON that is in constants.go
var ExtensionToLanguage = map[string]string{}
var LanguageFeatures = map[string]LanguageFeature{}

// ProcessConstants is responsible for setting up the language features based on the JSON file that is stored in constants
// Needs to be called at least once in order for anything to actually happen
func ProcessConstants() {
	var database = loadDatabase()
	gcPercent = debug.SetGCPercent(gcPercent)

	startTime := makeTimestampNano()
	for name, value := range database {
		for _, ext := range value.Extensions {
			ExtensionToLanguage[ext] = name
		}
	}

	if Trace {
		printTrace(fmt.Sprintf("nanoseconds build extension to language: %d", makeTimestampNano()-startTime))
	}

	startTime = makeTimestampMilli()
	for name, value := range database {
		complexityBytes := []byte{}
		complexityChecks := [][]byte{}
		singleLineComment := [][]byte{}
		multiLineComment := []OpenClose{}
		stringChecks := []OpenClose{}

		for _, v := range value.ComplexityChecks {
			complexityBytes = append(complexityBytes, v[0])
			complexityChecks = append(complexityChecks, []byte(v))
		}

		for _, v := range value.LineComment {
			singleLineComment = append(singleLineComment, []byte(v))
		}

		for _, v := range value.MultiLine {
			multiLineComment = append(multiLineComment, OpenClose{
				Open:  []byte(v[0]),
				Close: []byte(v[1]),
			})
		}

		for _, v := range value.Quotes {
			stringChecks = append(stringChecks, OpenClose{
				Open:  []byte(v[0]),
				Close: []byte(v[1]),
			})
		}

		LanguageFeatures[name] = LanguageFeature{
			ComplexityBytes:   complexityBytes,
			ComplexityChecks:  complexityChecks,
			MultiLineComment:  multiLineComment,
			SingleLineComment: singleLineComment,
			StringChecks:      stringChecks,
		}
	}

	if Trace {
		printTrace(fmt.Sprintf("milliseconds build language features: %d", makeTimestampMilli()-startTime))
	}
}

func processFlags() {
	// If wide/more mode is enabled we want the complexity calculation
	// to happen regardless as thats the only purpose of the flag
	if More && Complexity {
		Complexity = false
	}

	if Debug {
		printDebug(fmt.Sprintf("Path Black List: %s", PathBlacklist))
		printDebug(fmt.Sprintf("Sort By: %s", SortBy))
		printDebug(fmt.Sprintf("White List: %s", WhiteListExtensions))
		printDebug(fmt.Sprintf("Files Output: %t", Files))
		printDebug(fmt.Sprintf("Verbose: %t", Verbose))
		printDebug(fmt.Sprintf("Duplicates Detection: %t", Duplicates))
		printDebug(fmt.Sprintf("Complexity Calculation: %t", !Complexity))
		printDebug(fmt.Sprintf("Wide: %t", More))
		printDebug(fmt.Sprintf("Average Wage: %d", AverageWage))
		printDebug(fmt.Sprintf("Cocomo: %t", !Cocomo))
	}
}

func loadDatabase() map[string]Language {
	var database map[string]Language
	startTime := makeTimestampMilli()

	data, err := base64.StdEncoding.DecodeString(languages)
	if err != nil {
		panic(fmt.Sprintf("failed to base64 decode languages: %v", err))
	}

	if err := json.Unmarshal(data, &database); err != nil {
		panic(fmt.Sprintf("languages json invalid: %v", err))
	}

	if Trace {
		printTrace(fmt.Sprintf("milliseconds unmarshal: %d", makeTimestampMilli()-startTime))
	}

	return database
}

func printLanguages() {
	database := loadDatabase()
	var names []string

	for key := range database {
		names = append(names, key)
	}

	sort.Slice(names, func(i, j int) bool {
		return strings.Compare(strings.ToLower(names[i]), strings.ToLower(names[j])) < 0
	})

	for _, name := range names {
		fmt.Println(fmt.Sprintf("%s (%s)", name, strings.Join(database[name].Extensions, ",")))
	}
}

func Process() {
	if Languages {
		printLanguages()
		return
	}

	ProcessConstants()
	processFlags()

	// Clean up any invalid arguments before setting everything up
	if len(DirFilePaths) == 0 {
		DirFilePaths = append(DirFilePaths, ".")
	}

	SortBy = strings.ToLower(SortBy)

	if Debug {
		printDebug(fmt.Sprintf("NumCPU: %d", runtime.NumCPU()))
		printDebug(fmt.Sprintf("SortBy: %s", SortBy))
		printDebug(fmt.Sprintf("PathBlacklist: %s", PathBlacklist))
	}

	fileListQueue := make(chan *FileJob, FileListQueueSize)                     // Files ready to be read from disk
	fileReadContentJobQueue := make(chan *FileJob, FileReadContentJobQueueSize) // Files ready to be processed
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)         // Files ready to be summerised

	go walkDirectoryParallel(DirFilePaths[0], &fileListQueue)
	go fileReaderWorker(&fileListQueue, &fileReadContentJobQueue)
	go fileProcessorWorker(&fileReadContentJobQueue, &fileSummaryJobQueue)

	result := fileSummerize(&fileSummaryJobQueue)

	if FileOutput == "" {
		fmt.Println(result)
	} else {
		ioutil.WriteFile(FileOutput, []byte(result), 0600)
		fmt.Println("results written to " + FileOutput)
	}
}
