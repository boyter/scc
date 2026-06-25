// SPDX-License-Identifier: MIT

package processor

import (
	"fmt"
	"io"
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
var Version = "4.0.0 (beta)"

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

// IgnoreFiles are paths to additional ignore files supplied via --ignore-file.
// They are applied as a low priority base layer in the order supplied so a later
// file can override an earlier one, and any in-tree .gitignore/.ignore/.sccignore
// discovered while walking overrides all of them.
var IgnoreFiles = []string{}

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

type remapRule struct {
	pattern  []byte
	language string
}

type remapConfig struct {
	all     []remapRule
	unknown []remapRule
}

type processorContext struct {
	remap remapConfig
}

func parseRemapRules(value string) []remapRule {
	rules := []remapRule{}

	for s := range strings.SplitSeq(value, ",") {
		t := strings.Split(s, ":")
		if len(t) == 2 {
			rules = append(rules, remapRule{
				pattern:  []byte(t[0]),
				language: t[1],
			})
		}
	}

	return rules
}

func newRemapConfig(remapAll string, remapUnknown string) remapConfig {
	return remapConfig{
		all:     parseRemapRules(remapAll),
		unknown: parseRemapRules(remapUnknown),
	}
}

// MatchEngine selects how a CountRule pattern is interpreted. Glob is the
// default; regex is opt-in via the re: prefix.
type MatchEngine int

const (
	// MatchGlob is the default. The pattern is a glob ('*' and '?') translated
	// to an anchored regex and matched as a full match against the path.
	MatchGlob MatchEngine = iota
	// MatchRegex treats the pattern as a raw (unanchored) RE2 regex. Opt in
	// with the re: prefix.
	MatchRegex
)

// CountRule is the typed, library-facing form of a --count-as-pattern rule.
// It matches files by their path and relabels them to a new named category
// whose counting rules are cloned from an existing base language.
type CountRule struct {
	Engine       MatchEngine // MatchGlob (the default) or MatchRegex
	Pattern      string      // glob or regex source
	Name         string      // new category display name
	BaseLanguage string      // existing language whose counting rules are cloned
}

// CountRules is the typed input set either directly by library users or by the
// CLI after parsing CountAsPattern. Setup happens in setupCountRules.
var CountRules []CountRule

// CountAsPattern holds the raw repeatable --count-as-pattern flag values. Each
// is parsed into a CountRule at setup. Library users may set CountRules directly.
var CountAsPattern []string

// compiledCountRule is the runtime form scanned by newFileJob
type compiledCountRule struct {
	re   *regexp.Regexp
	name string
}

var compiledCountRules []compiledCountRule

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

// Locomo toggles the LOCOMO (LLM Output COst MOdel) calculation
var Locomo = false

// CostComparison enables both COCOMO and LOCOMO output for side-by-side comparison
var CostComparison = false

// LocomoPresetName is the LLM model preset for pricing and throughput defaults
var LocomoPresetName = "medium"

// LocomoInputPrice is the cost per 1M input tokens (overrides preset)
var LocomoInputPrice float64
var LocomoInputPriceSet = false

// LocomoOutputPrice is the cost per 1M output tokens (overrides preset)
var LocomoOutputPrice float64
var LocomoOutputPriceSet = false

// LocomoTPS is the output tokens per second (overrides preset)
var LocomoTPS float64
var LocomoTPSSet = false

// LocomoReviewMinutesPerLine is the human review time per line of code in minutes
var LocomoReviewMinutesPerLine float64 = 0.01

// LocomoConfig is the power-user config string "tokensPerLine,baseInputPerLine,complexityWeight,iterations,iterationWeight"
var LocomoConfig = ""

// LocomoTokensPerLine is the average number of output tokens per line of code
var LocomoTokensPerLine float64 = 10

// LocomoBaseInputPerLine is the base number of input tokens per output line
var LocomoBaseInputPerLine float64 = 20

// LocomoComplexityWeight is the scaling weight applied to sqrt(complexity density) for input tokens
var LocomoComplexityWeight float64 = 5

// LocomoIterations is the base number of iteration/retry attempts
var LocomoIterations float64 = 1.5

// LocomoIterationWeight is the scaling weight for complexity-driven retries
var LocomoIterationWeight float64 = 2

// LocomoCyclesOverride is the user-supplied iteration factor override (--locomo-cycles)
var LocomoCyclesOverride float64

// LocomoCyclesSet indicates whether --locomo-cycles was explicitly set
var LocomoCyclesSet = false

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

// Hotspots toggles the hotspots git-history report
var Hotspots = false

// ByAuthor toggles the author-rollup git-history report
var ByAuthor = false

// Timeline selects an over-time view. With ByAuthor, runs the author
// timeline report (plan 04); alone, runs the languages-over-time report
// (plan 05). With Hotspots set, the combination errors out.
var Timeline = false

// HistoryBuckets is the time-bucket resolution for the timeline reports.
// Wired to --buckets in main.go; default 60.
var HistoryBuckets = 60

// FoldAuthors enables the name+domain identity folding fallback applied
// after the mailmap. Toggled off via --no-fold-authors.
var FoldAuthors = true

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

// EnableGc restores the garbage collector to the percentage captured by ConfigureGc.
func EnableGc() {
	if gcPercent != -1 {
		debug.SetGCPercent(gcPercent)
	}
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

	// Set up any path pattern count rules, minting new categories backed by a
	// base language. The function clones the base language and builds its
	// features so counting works in both lazy and non-lazy modes.
	if len(CountAsPattern) != 0 || len(CountRules) != 0 {
		setupCountRules()
	}

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
		if len(t) != 2 {
			printError(fmt.Sprintf("ignoring malformed count-as rule %q: expected format <from>:<to>", s))
			continue
		}

		// There are two cases here.
		// first is they provide the name e.g. "Cargo Lock"
		// second is that the user supplies the extension EG wsdl
		// we should support BOTH cases
		// always remember we only need to validate t[1] as that's the one
		// that tells us where we are trying to map
		target, ok := resolveBaseLanguage(t[1])
		if ok {
			ExtensionToLanguage[strings.ToLower(t[0])] = []string{target}
			printDebugF("set to count extension: %s as language %s", t[0], target)
			continue
		}

		// The target t[1] matched neither a known language name nor a known
		// extension, so no mapping was registered. Warn rather than silently
		// ignoring the rule, since count-as cannot mint new categories yet.
		printError(fmt.Sprintf("ignoring count-as rule %q: target %q is not a known language or extension", s, t[1]))
	}
}

// resolveBaseLanguage resolves a user supplied target to a canonical language
// name. It first tries to match a language name (most reliable as names are
// unique) and falls back to matching a known extension. Returns the canonical
// language name and whether it was resolved.
func resolveBaseLanguage(target string) (string, bool) {
	// Match by language name which is the most reliable as the name is unique
	for name := range languageDatabase {
		if strings.EqualFold(name, target) {
			return name, true
		}
	}

	// Fall back to extension match, note this is less reliable as some
	// languages share extensions so we take the first registered language
	langs, ok := ExtensionToLanguage[strings.ToLower(target)]
	if ok && len(langs) != 0 {
		return langs[0], true
	}

	return "", false
}

// parseCountAsPattern parses a single --count-as-pattern rule of the form
// [engine:]pattern:name:baselang into a CountRule.
//
// The engine prefix is optional and the pattern is treated as a GLOB BY
// DEFAULT; prefix with re: to opt into a regex (or glob: to be explicit). We
// keep glob and regex as distinct modes rather than inferring, because the same
// string is valid in both engines with different meaning (e.g. "foo.rb" matches
// only foo.rb as a glob but also fooXrb as a regex), so guessing would silently
// match the wrong files.
//
// Because regex patterns and paths legitimately contain ':', name and baselang
// are peeled from the right and the pattern is whatever remains in between.
func parseCountAsPattern(s string) (CountRule, error) {
	engine := MatchGlob
	rest := s

	switch {
	case strings.HasPrefix(rest, "re:"):
		engine = MatchRegex
		rest = rest[len("re:"):]
	case strings.HasPrefix(rest, "glob:"):
		engine = MatchGlob
		rest = rest[len("glob:"):]
	}

	// baselang = after the last ':', name = between the 2nd-last and last ':'
	lastColon := strings.LastIndex(rest, ":")
	if lastColon == -1 {
		return CountRule{}, fmt.Errorf("expected format [engine:]pattern:name:baselang")
	}
	baseLanguage := rest[lastColon+1:]

	nameColon := strings.LastIndex(rest[:lastColon], ":")
	if nameColon == -1 {
		return CountRule{}, fmt.Errorf("expected format [engine:]pattern:name:baselang")
	}
	name := rest[nameColon+1 : lastColon]
	pattern := rest[:nameColon]

	if pattern == "" || name == "" || baseLanguage == "" {
		return CountRule{}, fmt.Errorf("pattern, name and baselang must all be non-empty")
	}

	return CountRule{Engine: engine, Pattern: pattern, Name: name, BaseLanguage: baseLanguage}, nil
}

// globToRegex converts a simple glob into an anchored regex. Glob is the
// default --count-as-pattern engine. Only '*' (any run of characters) and '?'
// (single character) are special, everything else is matched literally. The
// result is anchored as a full match.
func globToRegex(glob string) string {
	var b strings.Builder
	b.WriteByte('^')
	for _, r := range glob {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteByte('.')
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteByte('$')
	return b.String()
}

// setupCountRules parses CountAsPattern into CountRules, compiles each rule and
// registers a cloned language under its new name so counting works. Invalid
// rules are reported to stderr and skipped, consistent with --count-as.
func setupCountRules() {
	for _, s := range CountAsPattern {
		rule, err := parseCountAsPattern(s)
		if err != nil {
			printError(fmt.Sprintf("ignoring malformed count-as-pattern rule %q: %s", s, err))
			continue
		}
		CountRules = append(CountRules, rule)
	}

	for _, rule := range CountRules {
		base, ok := resolveBaseLanguage(rule.BaseLanguage)
		if !ok {
			printError(fmt.Sprintf("ignoring count-as-pattern rule for %q: base language %q is not a known language or extension", rule.Name, rule.BaseLanguage))
			continue
		}

		source := rule.Pattern
		if rule.Engine == MatchGlob {
			source = globToRegex(rule.Pattern)
		}

		re, err := regexp.Compile(source)
		if err != nil {
			printError(fmt.Sprintf("ignoring count-as-pattern rule for %q: invalid pattern %q: %s", rule.Name, rule.Pattern, err))
			continue
		}

		// Clone the base language under the new name so it has counting rules,
		// clearing the matchers so the minted category never participates in
		// normal extension/filename/shebang detection.
		cloned := languageDatabase[base]
		cloned.Extensions = nil
		cloned.FileNames = nil
		cloned.SheBangs = nil
		languageDatabase[rule.Name] = cloned

		// Populate features now in non-lazy mode, otherwise LoadLanguageFeature
		// will build them on first use since the name is in languageDatabase.
		if !isLazy {
			processLanguageFeature(rule.Name, cloned)
		}

		compiledCountRules = append(compiledCountRules, compiledCountRule{re: re, name: rule.Name})
		printDebugF("set to count path matching %q as new language %s based on %s", rule.Pattern, rule.Name, base)
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
	keywordBytes := make([][]byte, 0, len(value.Keywords))
	postfixExcludes := make([][]byte, 0, len(value.ComplexityChecksPostfixExcludes))

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

	for _, v := range value.ComplexityChecksPostfix {
		if !Complexity {
			tokenTrie.Insert(TComplexityPostfix, []byte(v))
			processMask |= v[0]
		}
	}

	for _, v := range value.ComplexityChecksPostfixExcludes {
		postfixExcludes = append(postfixExcludes, []byte(v))
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

	for _, v := range value.Keywords {
		keywordBytes = append(keywordBytes, []byte(v))
	}

	// Compile any regex heuristics used to disambiguate shared extensions such
	// as .h between C / C++ / Objective-C. The patterns are validated at
	// generation time (scripts/include.go) so MustCompile is safe here, but we
	// guard with Compile anyway to honour the no-panics policy.
	heuristics := make([]*regexp.Regexp, 0, len(value.Heuristics))
	for _, v := range value.Heuristics {
		re, err := regexp.Compile(v)
		if err != nil {
			printWarnF("failed to compile heuristic %q for language %s: %v", v, name, err)
			continue
		}
		heuristics = append(heuristics, re)
	}

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
		PostfixExcludes:       postfixExcludes,
		ComplexityCheckMask:   complexityMask,
		MultiLineCommentMask:  multiLineCommentMask,
		SingleLineCommentMask: singleLineCommentMask,
		StringCheckMask:       stringMask,
		ProcessMask:           processMask,
		Keywords:              value.Keywords,
		KeywordBytes:          keywordBytes,
		Heuristics:            heuristics,
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
	// If cost-comparison is enabled, turn on both COCOMO and LOCOMO
	if CostComparison {
		Cocomo = false
		Locomo = true
	}

	// LOCOMO needs complexity data to produce accurate estimates.
	// If complexity was disabled via --no-complexity, force it back on.
	if Locomo && Complexity {
		Complexity = false
	}

	printDebugF("Average Wage: %d", AverageWage)
	printDebugF("Cocomo: %t", !Cocomo)
	printDebugF("Locomo: %t", Locomo)
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

func PrintLanguages(dst io.Writer) {
	names := make([]string, 0, len(languageDatabase))
	for key := range languageDatabase {
		names = append(names, key)
	}

	slices.SortFunc(names, func(a, b string) int {
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})

	for _, name := range names {
		_, _ = fmt.Fprintf(dst, "%s (%s)\n", name, strings.Join(append(languageDatabase[name].Extensions, languageDatabase[name].FileNames...), ","))
	}
}

// global variables to deal with ULOC calculations
var ulocMutex = sync.Mutex{}
var ulocGlobalCount = map[string]struct{}{}
var ulocLanguageCount = map[string]map[string]struct{}{}

// Process is the main entry point of the command line it sets everything up and starts running
func Process() {
	if Languages {
		PrintLanguages(os.Stdout)
		return
	}

	ProcessConstants()
	processFlags()
	cleanVisitedPaths()

	// Clean up any invalid arguments before setting everything up
	if len(DirFilePaths) == 0 {
		DirFilePaths = append(DirFilePaths, ".")
	}

	// --report mode short-circuits the normal format dispatch and writes a
	// self-contained HTML report. Mutually exclusive with --format / -f: if
	// the user passed both, warn on stderr and let --report win.
	if ReportOut != "" {
		if Format != "" && Format != "tabular" {
			fmt.Fprintf(os.Stderr, "warning: --report overrides --format=%s\n", Format)
		}
		parseReportSkip(ReportSkip)
		if len(DirFilePaths) > 1 {
			fmt.Fprintf(os.Stderr, "warning: --report only analyses the first positional path (%s); other paths ignored\n", DirFilePaths[0])
		}
		if err := runReport(DirFilePaths); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if Hotspots && (ByAuthor || Timeline) {
		fmt.Println("--hotspots is mutually exclusive with --by-author / --timeline; pick one report")
		os.Exit(1)
	}

	if Hotspots || ByAuthor || Timeline {
		if err := validateHistoryFlags(os.Stderr); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if Hotspots {
		if err := runHotspotsReport(DirFilePaths[0]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if ByAuthor && Timeline {
		if err := runAuthorTimelineReport(DirFilePaths[0]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if ByAuthor {
		if err := runAuthorsReport(DirFilePaths[0]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if Timeline {
		if err := runLanguagesTimelineReport(DirFilePaths[0]); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
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
	ctx := processorContext{remap: newRemapConfig(RemapAll, RemapUnknown)}

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
	fileWalker.CustomIgnoreFiles = IgnoreFiles

	var excludePathRegexes []*regexp.Regexp
	for _, exclude := range Exclude {
		regexpResult, err := regexp.Compile(exclude)
		if err == nil {
			fileWalker.ExcludeFilenameRegex = append(fileWalker.ExcludeFilenameRegex, regexpResult)
			fileWalker.ExcludeDirectoryRegex = append(fileWalker.ExcludeDirectoryRegex, regexpResult)
			excludePathRegexes = append(excludePathRegexes, regexpResult)
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
			shouldExclude := false
			for _, re := range excludePathRegexes {
				if re.MatchString(fi.Location) {
					shouldExclude = true
					break
				}
			}
			if shouldExclude {
				continue
			}

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

	go ctx.fileProcessorWorker(fileListQueue, fileSummaryJobQueue)

	result := fileSummarize(fileSummaryJobQueue)
	if FileOutput == "" {
		fmt.Print(result)
	} else {
		_ = os.WriteFile(FileOutput, []byte(result), 0644)
		fmt.Println("results written to " + FileOutput)
	}
}
