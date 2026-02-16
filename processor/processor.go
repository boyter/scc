// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/boyter/gocodewalker"
)

// Version indicates the version of the application
var Version = "3.7.0"

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

// MinifiedGenerated enables minified/generated file detection
var MinifiedGenerated = false

// IgnoreMinifiedGenerate printing counts for minified/generated files
var IgnoreMinifiedGenerate = false

// MinifiedGeneratedLineByteLength number of bytes per average line to determine file is minified/generated
var MinifiedGeneratedLineByteLength = 255

// Minified enables minified file detection
var Minified = false

// IgnoreMinified ignore printing counts for minified files
var IgnoreMinified = false

// Generated enables generated file detection
var Generated = false

// GeneratedMarkers defines head markers for generated file detection
var GeneratedMarkers []string

// IgnoreGenerated ignore printing counts for generated files
var IgnoreGenerated = false

// Complexity toggles complexity calculation
var Complexity = false

// More enables wider output with more information in formatter
var More = false

// Cocomo toggles the COCOMO calculation
var Cocomo = false

// SLOCCountFormat prints a more SLOCCount like COCOMO calculation
var SLOCCountFormat = false

// CocomoProjectType allows the flipping between project types which impacts the calculation
var CocomoProjectType = "organic"

// Size toggles the Size calculation
var Size = false

// Draw horizontal borders between sections.
var HBorder = false

// SizeUnit determines what size calculation is used for megabytes
var SizeUnit = "si"

// Ci indicates if running inside a CI so to disable box drawing characters
var Ci = false

// GitIgnore disables .gitignore checks
var GitIgnore = false

// GitModuleIgnore disables .gitmodules checks
var GitModuleIgnore = false

// Ignore disables ignore file checks
var Ignore = false

// SccIgnore disables sccignore file checks
var SccIgnore = false

// CountIgnore should we count ignore files?
var CountIgnore = false

// DisableCheckBinary toggles checking for binary files using NUL bytes
var DisableCheckBinary = false

// UlocMode toggles checking for binary files using NUL bytes
var UlocMode = false

// Percent toggles checking for binary files using NUL bytes
var Percent = false

// MaxMean sets the calculation of the max and mean line length
var MaxMean = false

// Dryness toggles checking for binary files using NUL bytes
var Dryness = false

// SortBy sets which column output in formatter should be sorted by
var SortBy = ""

// Exclude is a regular expression which is used to exclude files from being processed
var Exclude = []string{}

// CountAs is a rule for mapping known or new extensions to other rules
var CountAs = ""

// Format sets the output format of the formatter
var Format = ""

// FormatMulti is a rule for defining multiple output formats
var FormatMulti = ""

// SQLProject is used to store the name for the SQL insert formats but is optional
var SQLProject = ""

// RemapUnknown allows remapping of unknown files with a string to search the content for
var RemapUnknown = ""

// RemapAll allows remapping of all files with a string to search the content for
var RemapAll = ""

// CurrencySymbol allows setting the currency symbol for cocomo project cost estimation
var CurrencySymbol = ""

// FileOutput sets the file that output should be written to
var FileOutput = ""

// PathDenyList sets the paths that should be skipped
var PathDenyList = []string{}

// FileListQueueSize is the queue of files found and ready to be read into memory
var FileListQueueSize = runtime.NumCPU()

// FileProcessJobWorkers is the number of workers that process the file collecting stats
var FileProcessJobWorkers = runtime.NumCPU() * 4

// FileSummaryJobQueueSize is the queue used to hold processed file statistics before formatting
var FileSummaryJobQueueSize = runtime.NumCPU()

// DirectoryWalkerJobWorkers is the number of workers which will walk the directory tree
var DirectoryWalkerJobWorkers = 8

// AllowListExtensions is a list of extensions which are allowed to be processed
var AllowListExtensions = []string{}

// ExcludeListExtensions is a list of extensions which should be ignored
var ExcludeListExtensions = []string{}

// ExcludeFilename is a list of filenames which should be ignored
var ExcludeFilename = []string{}

// AverageWage is the average wage in dollars used for the COCOMO cost estimate
var AverageWage int64 = 56286

// Overhead is the overhead multiplier for corporate overhead (facilities, equipment, accounting, etc.)
var Overhead float64 = 2.4

// EAF is the effort adjustment factor derived from the cost drivers, i.e. 1.0 if rated nominal
var EAF float64 = 1.0

// GcFileCount is the number of files to process before turning the GC back on
var GcFileCount = 10000
var gcPercent = -1
var isLazy = false

// NoLarge if set true will ignore files over a certain number of lines or bytes
var NoLarge = false

// IncludeSymLinks if set true will count symlink files
var IncludeSymLinks = false

// LargeLineCount number of lines before being counted as a large file based on https://github.com/pinpt/ripsrc/blob/master/ripsrc/fileinfo/fileinfo.go#L44
var LargeLineCount int64 = 40000

// LargeByteCount number of bytes before being counted as a large file based on https://github.com/pinpt/ripsrc/blob/master/ripsrc/fileinfo/fileinfo.go#L44
var LargeByteCount int64 = 1000000

// DirFilePaths is not set via flags but by arguments following the flags for file or directory to process
var DirFilePaths = []string{}

// ExtensionToLanguage is loaded from the JSON that is in constants.go
var ExtensionToLanguage = map[string][]string{}

// ShebangLookup loaded from the JSON in constants.go contains shebang lookups
var ShebangLookup = map[string][]string{}

// FilenameToLanguage similar to ExtensionToLanguage loaded from the JSON in constants.go
var FilenameToLanguage = map[string]string{}

// LanguageFeatures contains the processed languages from processLanguageFeature
var LanguageFeatures = map[string]LanguageFeature{}

// LanguageFeaturesMutex is the shared mutex used to control getting and setting of language features
// used rather than sync.Map because it turned out to be marginally faster
var LanguageFeaturesMutex = sync.Mutex{}

// Start time in milli seconds in case we want the total time
var startTimeMilli = makeTimestampMilli()

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
	startTime := makeTimestampNano()
	for name, value := range languageDatabase {
		for _, ext := range value.Extensions {
			ExtensionToLanguage[ext] = append(ExtensionToLanguage[ext], name)
		}

		for _, fname := range value.FileNames {
			FilenameToLanguage[fname] = name
		}

		if len(value.SheBangs) != 0 {
			ShebangLookup[name] = value.SheBangs
		}
	}

	// If we have anything in CountAs set it up now
	if len(CountAs) != 0 {
		setupCountAs()
	}

	printTraceF("nanoseconds build extension to language: %d", makeTimestampNano()-startTime)

	// Configure COCOMO setting
	_, ok := projectType[strings.ToLower(CocomoProjectType)]
	if !ok {
		// let's see if we can turn it into a custom one
		spl := strings.Split(CocomoProjectType, ",")
		val := []float64{}
		if len(spl) == 5 {
			// let's try to convert to float if we can
			for i := 1; i < 5; i++ {
				f, err := strconv.ParseFloat(spl[i], 64)
				if err == nil {
					val = append(val, f)
				}
			}
		}

		if len(val) == 4 {
			projectType[CocomoProjectType] = val
		} else {
			// if nothing matches fall back to organic
			CocomoProjectType = "organic"
		}
	}

	// If lazy is set then we want to load in the features as we find them not in one go
	// however otherwise being used as a library so just load them all in
	if !isLazy {
		startTime = makeTimestampMilli()
		for name, value := range languageDatabase {
			processLanguageFeature(name, value)
		}

		printTraceF("milliseconds build language features: %d", makeTimestampMilli()-startTime)
	} else {
		printTrace("configured to lazy load language features")
	}

	// Fix for https://github.com/boyter/scc/issues/250
	fixedPath := make([]string, 0, len(PathDenyList))
	for _, path := range PathDenyList {
		fixedPath = append(fixedPath, strings.TrimRight(path, "/"))
	}
	PathDenyList = fixedPath
}

// Configure and setup any count-as params the use has supplied
func setupCountAs() {
	for s := range strings.SplitSeq(CountAs, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {

			identified := false

			// There are two cases here.
			// first is they provide the name e.g. "Cargo Lock"
			// second is that the user supplies the extension EG wsdl
			// we should support BOTH cases
			// always remember we only need to validate t[1] as that's the one
			// that tells us where we are trying to map

			// See if we can identify based on language name which is the most
			// reliable as the name should be unique
			for name := range languageDatabase {
				if strings.EqualFold(name, t[1]) {
					ExtensionToLanguage[strings.ToLower(t[0])] = []string{name}
					identified = true
					printDebugF("set to count extension: %s as language %s by language", t[0], name)
				}
			}

			// If the above did not work, its a matter of extension match
			// note that this is less reliable as some languages share extensions
			if !identified {
				target, ok := ExtensionToLanguage[strings.ToLower(t[1])]

				if ok {
					ExtensionToLanguage[strings.ToLower(t[0])] = target
					printDebugF("set to count extension: %s as language %s by extension", t[0], target)
				}
			}
		}
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
	printTraceF("nanoseconds to build language %s features: %d", loadName, makeTimestampNano()-startTime)
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
		stringMask |= v.Start[0]
		stringTrie.InsertClose(TString, []byte(v.Start), []byte(v.End))
		tokenTrie.InsertClose(TString, []byte(v.Start), []byte(v.End))
	}
	processMask |= stringMask

	LanguageFeaturesMutex.Lock()
	LanguageFeatures[name] = LanguageFeature{
		Complexity:            complexityTrie,
		MultiLineComments:     mlCommentTrie,
		MultiLine:             value.MultiLine,
		SingleLineComments:    slCommentTrie,
		LineComment:           value.LineComment,
		Strings:               stringTrie,
		Tokens:                tokenTrie,
		Nested:                value.NestedMultiLine,
		ComplexityCheckMask:   complexityMask,
		MultiLineCommentMask:  multiLineCommentMask,
		SingleLineCommentMask: singleLineCommentMask,
		StringCheckMask:       stringMask,
		ProcessMask:           processMask,
		Keywords:              value.Keywords,
		Quotes:                value.Quotes,
	}
	LanguageFeaturesMutex.Unlock()
}

func processFlags() {
	// If wide/more mode is enabled we want the complexity calculation
	// to happen regardless as that is the only purpose of the flag
	if More && Complexity {
		Complexity = false
	}

	// If ignore minified/generated is on ensure we turn on the code to calculate that
	if IgnoreMinifiedGenerate {
		MinifiedGenerated = true
		IgnoreMinified = true
		IgnoreGenerated = true
	}

	if MinifiedGenerated {
		Minified = true
		Generated = true
	}

	if IgnoreMinified {
		Minified = true
	}

	if IgnoreGenerated {
		Generated = true
	}

	if Dryness {
		UlocMode = true
	}

	printDebugF("Path Deny List: %v", PathDenyList)
	printDebugF("Sort By: %s", SortBy)
	printDebugF("White List: %v", AllowListExtensions)
	printDebugF("Files Output: %t", Files)
	printDebugF("Verbose: %t", Verbose)
	printDebugF("Duplicates Detection: %t", Duplicates)
	printDebugF("Complexity Calculation: %t", !Complexity)
	printDebugF("Wide: %t", More)
	printDebugF("Average Wage: %d", AverageWage)
	printDebugF("Cocomo: %t", !Cocomo)
	printDebugF("Minified/Generated Detection: %t/%t", Minified, Generated)
	printDebugF("Ignore Minified/Generated: %t/%t", IgnoreMinified, IgnoreGenerated)
	printDebugF("IncludeSymLinks: %t", IncludeSymLinks)
	printDebugF("Uloc: %t", UlocMode)
	printDebugF("Dryness: %t", Dryness)
}

// LanguageDatabase provides access to the internal language database
// useful for consuming applications wanting to consume and use
func LanguageDatabase() map[string]Language {
	return languageDatabase
}

func printLanguages() {
	names := make([]string, 0, len(languageDatabase))
	for key := range languageDatabase {
		names = append(names, key)
	}

	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	for _, name := range names {
		fmt.Printf("%s (%s)\n", name, strings.Join(append(languageDatabase[name].Extensions, languageDatabase[name].FileNames...), ","))
	}
}

// global variables to deal with ULOC calculations
var ulocMutex = sync.Mutex{}
var ulocGlobalCount = map[string]struct{}{}
var ulocLanguageCount = map[string]map[string]struct{}{}

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

	filePaths := []string{}
	dirPaths := []string{}

	// Check if the paths or files added exist and exit if not
	for _, f := range DirFilePaths {
		fpath := filepath.Clean(f)

		s, err := os.Stat(fpath)
		if err != nil {
			fmt.Println("file or directory could not be read: " + fpath)
			os.Exit(1)
		}

		if s.IsDir() {
			dirPaths = append(dirPaths, fpath)
		} else {
			filePaths = append(filePaths, fpath)
		}
	}

	SortBy = strings.ToLower(SortBy)

	printDebugF("NumCPU: %d", runtime.NumCPU())
	printDebugF("SortBy: %s", SortBy)
	printDebugF("PathDenyList: %v", PathDenyList)

	potentialFilesQueue := make(chan *gocodewalker.File, FileListQueueSize) // files that pass the .gitignore checks
	fileListQueue := make(chan *FileJob, FileListQueueSize)                 // Files ready to be read from disk
	fileSummaryJobQueue := make(chan *FileJob, FileSummaryJobQueueSize)     // Files ready to be summarised

	fileWalker := gocodewalker.NewParallelFileWalker(dirPaths, potentialFilesQueue)
	fileWalker.SetErrorHandler(func(e error) bool {
		printError(e.Error())
		return true
	})
	fileWalker.IgnoreGitIgnore = GitIgnore
	fileWalker.IgnoreIgnoreFile = Ignore
	fileWalker.IgnoreGitModules = GitModuleIgnore
	fileWalker.IncludeHidden = true
	fileWalker.ExcludeDirectory = PathDenyList
	fileWalker.SetConcurrency(DirectoryWalkerJobWorkers)

	if !SccIgnore {
		fileWalker.CustomIgnore = []string{".sccignore"}
	}

	for _, exclude := range Exclude {
		regexpResult, err := regexp.Compile(exclude)
		if err == nil {
			fileWalker.ExcludeFilenameRegex = append(fileWalker.ExcludeFilenameRegex, regexpResult)
			fileWalker.ExcludeDirectoryRegex = append(fileWalker.ExcludeDirectoryRegex, regexpResult)
		} else {
			printError(err.Error())
		}
	}

	go func() {
		err := fileWalker.Start()
		if err != nil {
			printError(err.Error())
		}
	}()

	go func() {
		for _, f := range filePaths {
			fileInfo, err := os.Lstat(f)
			if err != nil {
				continue
			}

			fileJob := newFileJob(f, f, fileInfo)
			if fileJob != nil {
				fileListQueue <- fileJob
			}
		}

		for fi := range potentialFilesQueue {
			fileInfo, err := os.Lstat(fi.Location)
			if err != nil {
				continue
			}

			if !fileInfo.IsDir() {
				fileJob := newFileJob(fi.Location, fi.Filename, fileInfo)
				if fileJob != nil {
					fileListQueue <- fileJob
				}
			}
		}
		close(fileListQueue)
	}()

	go fileProcessorWorker(fileListQueue, fileSummaryJobQueue)

	result := fileSummarize(fileSummaryJobQueue)
	if FileOutput == "" {
		fmt.Print(result)
	} else {
		_ = os.WriteFile(FileOutput, []byte(result), 0644)
		fmt.Println("results written to " + FileOutput)
	}
}
