package processor

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
)

// Flags set via the CLI which control how the output is displayed

// Files indicates if there should be file output or not when formatting
var Files = false

// Languages indicates if the command line should print out the supported languages
var Languages = false

// Verbose enables verbose logging output
var Verbose = false

// Debug enables debug logging output
var Debug = false

// Trace enables trace logging output which is extremely verbose
var Trace = false

// Duplicates enables duplicate file detection
var Duplicates = false

// Complexity toggles complexity calculation
var Complexity = false

// More enables wider output with more information in formatter
var More = false

// Cocomo toggles the COCOMO calculation
var Cocomo = false

// DisableCheckBinary toggles checking for binary files using NUL bytes
var DisableCheckBinary = false

// SortBy sets which column output in formatter should be sorted by
var SortBy = ""

// Exclude is a regular expression which is used to exclude files from being processed
var Exclude = ""

// Format sets the output format of the formatter
var Format = ""

// FileOutput sets the file that output should be written to
var FileOutput = ""

// PathBlacklist sets the paths that should be skipped
var PathBlacklist = []string{}

// FileListQueueSize is the queue of files found and ready to be read into memory
var FileListQueueSize = runtime.NumCPU()

// FileReadJobWorkers is the number of processes that read files off disk into memory
var FileReadJobWorkers = runtime.NumCPU() * 4

// FileReadContentJobQueueSize is a queue of files ready to be processed
var FileReadContentJobQueueSize = runtime.NumCPU()

// FileProcessJobWorkers is the number of workers that process the file collecting stats
var FileProcessJobWorkers = runtime.NumCPU() * 4

// FileSummaryJobQueueSize is the queue used to hold processed file statistics before formatting
var FileSummaryJobQueueSize = runtime.NumCPU()

// WhiteListExtensions is a list of extensions which are whitelisted to be processed
var WhiteListExtensions = []string{}

// AverageWage is the average wage in dollars used for the COCOMO cost estimate
var AverageWage int64 = 56286

// GcFileCount is the number of files to process before turning the GC back on
var GcFileCount = 10000
var gcPercent = -1
var isLazy = false

// DirFilePaths is not set via flags but by arguments following the flags for file or directory to process
var DirFilePaths = []string{}

// Raw languageDatabase loaded
var languageDatabase = map[string]Language{}

// ExtensionToLanguage is loaded from the JSON that is in constants.go
var ExtensionToLanguage = map[string][]string{}

// LanguageFeatures contains the processed languages from processLanguageFeature
var LanguageFeatures = map[string]LanguageFeature{}

// LanguageFeaturesMutex is the shared mutex used to control getting and setting of language features
// used rather than sync.Map because it turned out to be marginally faster
var LanguageFeaturesMutex = sync.Mutex{}

// ConfigureGc needs to be set outside of ProcessConstants because it should only be enabled in command line
// mode https://github.com/boyter/scc/issues/32
func ConfigureGc() {
	gcPercent = debug.SetGCPercent(gcPercent)
}

// ConfigureLazy is a simple setter used to turn on lazy loading used only by command line
func ConfigureLazy(lazy bool) {
	isLazy = lazy
}

// ProcessConstants is responsible for setting up the language features based on the JSON file that is stored in constants
// Needs to be called at least once in order for anything to actually happen
func ProcessConstants() {
	languageDatabase = loadDatabase()

	startTime := makeTimestampNano()
	for name, value := range languageDatabase {
		for _, ext := range value.Extensions {
			ExtensionToLanguage[ext] = append(ExtensionToLanguage[ext], name)
		}
	}

	if Trace {
		printTrace(fmt.Sprintf("nanoseconds build extension to language: %d", makeTimestampNano()-startTime))
	}

	// If lazy is set then we want to load in the features as we find them not in one go
	// however otherwise being used as a library so just load them all in
	if !isLazy {
		startTime = makeTimestampMilli()
		for name, value := range languageDatabase {
			processLanguageFeature(name, value)
		}

		if Trace {
			printTrace(fmt.Sprintf("milliseconds build language features: %d", makeTimestampMilli()-startTime))
		}
	} else {
		printTrace("configured to lazy load language features")
	}
}

// LoadLanguageFeature will load a single feature as requested given the name
func LoadLanguageFeature(loadName string) {
	if !isLazy {
		return
	}

	// Check if already loaded and if so return because we don't need to do it again
	LanguageFeaturesMutex.Lock()
	_, ok := LanguageFeatures[loadName]
	LanguageFeaturesMutex.Unlock()
	if ok {
		return
	}

	var name string
	var value Language

	for name, value = range languageDatabase {
		if name == loadName {
			break
		}
	}

	startTime := makeTimestampNano()
	processLanguageFeature(loadName, value)
	if Trace {
		printTrace(fmt.Sprintf("nanoseconds to build language %s features: %d", loadName, makeTimestampNano()-startTime))
	}
}

func processLanguageFeature(name string, value Language) {
	complexityTrie := &Trie{}
	slCommentTrie := &Trie{}
	mlCommentTrie := &Trie{}
	stringTrie := &Trie{}
	tokenTrie := &Trie{}

	complexityMask := byte(0)
	singleLineCommentMask := byte(0)
	multiLineCommentMask := byte(0)
	stringMask := byte(0)
	processMask := byte(0)

	for _, v := range value.ComplexityChecks {
		complexityMask |= v[0]
		complexityTrie.Insert(TComplexity, []byte(v))
		if !Complexity {
			tokenTrie.Insert(TComplexity, []byte(v))
		}
	}
	if !Complexity {
		processMask |= complexityMask
	}

	for _, v := range value.LineComment {
		singleLineCommentMask |= v[0]
		slCommentTrie.Insert(TSlcomment, []byte(v))
		tokenTrie.Insert(TSlcomment, []byte(v))
	}
	processMask |= singleLineCommentMask

	for _, v := range value.MultiLine {
		multiLineCommentMask |= v[0][0]
		mlCommentTrie.InsertClose(TMlcomment, []byte(v[0]), []byte(v[1]))
		tokenTrie.InsertClose(TMlcomment, []byte(v[0]), []byte(v[1]))
	}
	processMask |= multiLineCommentMask

	for _, v := range value.Quotes {
		stringMask |= v[0][0]
		stringTrie.InsertClose(TString, []byte(v[0]), []byte(v[1]))
		tokenTrie.InsertClose(TString, []byte(v[0]), []byte(v[1]))
	}
	processMask |= stringMask

	LanguageFeaturesMutex.Lock()
	LanguageFeatures[name] = LanguageFeature{
		Complexity:            complexityTrie,
		MultiLineComments:     mlCommentTrie,
		SingleLineComments:    slCommentTrie,
		Strings:               stringTrie,
		Tokens:                tokenTrie,
		Nested:                value.NestedMultiLine,
		ComplexityCheckMask:   complexityMask,
		MultiLineCommentMask:  multiLineCommentMask,
		SingleLineCommentMask: singleLineCommentMask,
		StringCheckMask:       stringMask,
		ProcessMask:           processMask,
		Keywords:              value.Keywords,
	}
	LanguageFeaturesMutex.Unlock()
}

func processFlags() {
	// If wide/more mode is enabled we want the complexity calculation
	// to happen regardless as that is the only purpose of the flag
	if More && Complexity {
		Complexity = false
	}

	if Debug {
		printDebug(fmt.Sprintf("Path Black List: %v", PathBlacklist))
		printDebug(fmt.Sprintf("Sort By: %s", SortBy))
		printDebug(fmt.Sprintf("White List: %v", WhiteListExtensions))
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

// Process is the main entry point of the command line it sets everything up and starts running
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
	fpath := filepath.Clean(DirFilePaths[0])

	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		fmt.Println("file or directory does not exists: " + fpath)
		return
	}

	SortBy = strings.ToLower(SortBy)

	if Debug {
		printDebug(fmt.Sprintf("NumCPU: %d", runtime.NumCPU()))
		printDebug(fmt.Sprintf("SortBy: %s", SortBy))
		printDebug(fmt.Sprintf("PathBlacklist: %v", PathBlacklist))
	}

	fileListQueue := make(chan *FileJob, FileListQueueSize)                     // Files ready to be read from disk
	fileReadContentJobQueue := make(chan *FileJob, FileReadContentJobQueueSize) // Files ready to be processed
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)         // Files ready to be summarised

	go walkDirectoryParallel(fpath, fileListQueue)
	go fileReaderWorker(fileListQueue, fileReadContentJobQueue)
	go fileProcessorWorker(fileReadContentJobQueue, fileSummaryJobQueue)

	result := fileSummarize(fileSummaryJobQueue)

	if FileOutput == "" {
		fmt.Println(result)
	} else {
		_ = ioutil.WriteFile(FileOutput, []byte(result), 0600)
		fmt.Println("results written to " + FileOutput)
	}
}
