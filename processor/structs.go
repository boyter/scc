// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"hash"
	"regexp"
	"slices"
	"sync"
)

// Used by trie structure to store the types
const (
	TString int = iota + 1
	TSlcomment
	TMlcomment
	TComplexity
	TComplexityPostfix
)

// ByteType constants for per-byte content classification.
// When FileJob.ClassifyContent is true, CountStats populates
// FileJob.ContentByteType with one of these values per byte.
const (
	ByteTypeBlank   byte = 0
	ByteTypeCode    byte = 1
	ByteTypeComment byte = 2
	ByteTypeString  byte = 3
)

// Quote is a struct which holds rules and start/end values for string quotes
type Quote struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	IgnoreEscape bool   `json:"ignoreEscape"` // To enable turning off the \ check for C# @"\" string examples https://github.com/boyter/scc/issues/71
	DocString    bool   `json:"docString"`    // To enable docstring check for Python where "If the triple quote string starts following a newline with only white-space characters in front and ends followed by only a newline or white-space characters it is a comment" https://github.com/boyter/scc/issues/62
}

// Heuristic is a regex pattern used to disambiguate shared file extensions (for
// example .h between C / C++ / Objective-C) along with a cheap set of necessary
// string literals. The expensive regex is only run when one of Literals is
// present in the content, which is a fast reject for the overwhelmingly common
// case where the file is not the language being guessed. See guessByHeuristics.
type Heuristic struct {
	// Pattern is the regex evaluated against the file content.
	Pattern string `json:"pattern"`
	// Literals is the set of substrings of which at least one must be present
	// (case sensitive) for Pattern to have any chance of matching. When empty
	// the regex is always run, so a pattern is never silently disabled.
	Literals []string `json:"literals"`
	// Anchored, when true, requires each literal to sit at the start of a line
	// preceded only by spaces or tabs. This mirrors the (?m)^[ \t]* prefix used
	// by the keyword patterns and avoids false positives such as the substring
	// "entry" satisfying a check for the "try" keyword.
	Anchored bool `json:"anchored"`
}

// Language is a struct which contains the values for each language stored in languages.json
type Language struct {
	LineComment                     []string    `json:"line_comment"`
	ComplexityChecks                []string    `json:"complexitychecks"`
	ComplexityChecksPostfix         []string    `json:"complexitychecks_postfix"`
	ComplexityChecksPostfixExcludes []string    `json:"complexitychecks_postfix_excludes"`
	Extensions                      []string    `json:"extensions"`
	MultiLine                       [][]string  `json:"multi_line"`
	Quotes                          []Quote     `json:"quotes"`
	Keywords                        []string    `json:"keywords"`
	Heuristics                      []Heuristic `json:"heuristics"`
	FileNames                       []string    `json:"filenames"`
	SheBangs                        []string    `json:"shebangs"`
	ExtensionFile                   bool        `json:"extensionFile"`
	NestedMultiLine                 bool        `json:"nestedmultiline"`
}

// LanguageFeature is a struct which represents the conversion from Language into what is used for matching
type LanguageFeature struct {
	Complexity            *Trie
	MultiLineComments     *Trie
	MultiLine             [][]string // in case someone needs the actual value
	SingleLineComments    *Trie
	LineComment           []string // in case someone needs the actual value
	Strings               *Trie
	Tokens                *Trie
	Nested                bool
	PostfixExcludes       [][]byte
	ComplexityCheckMask   byte
	SingleLineCommentMask byte
	MultiLineCommentMask  byte
	StringCheckMask       byte
	ProcessMask           byte
	Keywords              []string
	KeywordBytes          [][]byte
	Heuristics            []CompiledHeuristic
	Quotes                []Quote
}

// CompiledHeuristic is the runtime form of a Heuristic with its regex compiled
// and its literals pre-converted to bytes for matching.
type CompiledHeuristic struct {
	Re       *regexp.Regexp
	Literals [][]byte
	Anchored bool
}

// FileJobCallback is an interface that FileJobs can implement to get a per line callback with the line type
type FileJobCallback interface {
	// ProcessLine should return true to continue processing or false to stop further processing and return
	ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool
}

// FileJob is a struct used to hold all of the results of processing internally before sent to the formatter
type FileJob struct {
	Language             string
	PossibleLanguages    []string // Used to hold potentially more than one language which populates language when determined
	Filename             string
	Extension            string
	Location             string
	Symlocation          string
	Content              []byte `json:"-"`
	Bytes                int64
	Lines                int64
	Code                 int64
	Comment              int64
	Blank                int64
	Complexity           int64
	ComplexityLine       []int64 `json:"-"`
	WeightedComplexity   float64
	Hash                 hash.Hash
	Callback             FileJobCallback `json:"-"`
	Binary               bool
	Minified             bool
	Generated            bool
	EndPoint             int
	Uloc                 int
	LineLength           []int  `json:"-"`
	ClassifyContent      bool   `json:"-"` // When true, CountStats populates ContentByteType
	ContentByteType      []byte `json:"-"` // Per-byte classification, allocated by CountStats when ClassifyContent is true
	TrackComplexityLines bool   `json:"-"` // When true, CountStats populates ComplexityLine
}

// FilterContentByType returns a copy of Content with bytes not matching any of
// the given types replaced by spaces. Newlines are always preserved regardless
// of type. Returns nil if ContentByteType is nil.
func (fj *FileJob) FilterContentByType(keepTypes ...byte) []byte {
	if fj.ContentByteType == nil {
		return nil
	}

	keep := make(map[byte]bool, len(keepTypes))
	for _, t := range keepTypes {
		keep[t] = true
	}

	result := make([]byte, len(fj.Content))
	for i, b := range fj.Content {
		if b == '\n' || keep[fj.ContentByteType[i]] {
			result[i] = b
		} else {
			result[i] = ' '
		}
	}
	return result
}

// LanguageSummary is used to hold summarized results for a single language
type LanguageSummary struct {
	Name               string
	Bytes              int64
	CodeBytes          int64
	Lines              int64
	Code               int64
	Comment            int64
	Blank              int64
	Complexity         int64
	Count              int64
	WeightedComplexity float64
	Files              []*FileJob
	LineLength         []int
	ULOC               int
	CodePercent        *float64 `json:",omitempty"`
	CommentPercent     *float64 `json:",omitempty"`
	BlankPercent       *float64 `json:",omitempty"`
	LinePercent        *float64 `json:",omitempty"`
	ComplexityPercent  *float64 `json:",omitempty"`
	BytePercent        *float64 `json:",omitempty"`
	FilePercent        *float64 `json:",omitempty"`
}

// OpenClose is used to hold an open/close pair for matching such as multi line comments
type OpenClose struct {
	Open  []byte
	Close []byte
}

// CheckDuplicates is used to hold hashes if duplicate detection is enabled it comes with a mutex
// that should be locked while a check is being performed then added
type CheckDuplicates struct {
	hashes map[int64][][]byte
	mux    sync.Mutex
}

// Add is a non thread safe add a key into the duplicates check need to use mutex inside struct before calling this
func (c *CheckDuplicates) Add(key int64, hash []byte) {
	hashes, ok := c.hashes[key]
	if ok {
		c.hashes[key] = append(hashes, hash)
	} else {
		c.hashes[key] = [][]byte{hash}
	}
}

// Check is a non thread safe check to see if the key exists already need to use mutex inside struct before calling this
func (c *CheckDuplicates) Check(key int64, hash []byte) bool {
	hashes, ok := c.hashes[key]

	return ok && slices.ContainsFunc(hashes, func(h []byte) bool {
		return bytes.Equal(h, hash)
	})
}

// Trie is a structure used to store matches efficiently
type Trie struct {
	Type  int
	Close []byte
	Table [256]*Trie
}

// Insert inserts a string into the trie for matching
func (root *Trie) Insert(tokenType int, token []byte) {
	var node *Trie

	node = root
	for _, c := range token {
		if node.Table[c] == nil {
			node.Table[c] = &Trie{}
		}
		node = node.Table[c]
	}
	node.Type = tokenType
}

// InsertClose closes off a string in the trie
func (root *Trie) InsertClose(tokenType int, openToken, closeToken []byte) {
	var node *Trie

	node = root
	for _, c := range openToken {
		if node.Table[c] == nil {
			node.Table[c] = &Trie{}
		}
		node = node.Table[c]
	}
	node.Type = tokenType
	node.Close = closeToken
}

// Match checks the created trie structure for a match
func (root *Trie) Match(token []byte) (int, int, []byte) {
	var node *Trie
	var depth int
	var c byte

	node = root
	var prevClosedNode *Trie
	var prevClosedDepth int
	for depth, c = range token {
		if node.Table[c] == nil {
			break
		}
		node = node.Table[c]
		if len(node.Close) > 0 {
			prevClosedNode = node
			prevClosedDepth = depth
		}
	}
	if len(node.Close) == 0 && prevClosedNode != nil {
		return prevClosedNode.Type, prevClosedDepth, prevClosedNode.Close
	}
	return node.Type, depth, node.Close
}
