package processor

import (
	"bytes"
	"sync"
)

type Language struct {
	LineComment      []string   `json:"line_comment"`
	ComplexityChecks []string   `json:"complexitychecks"`
	Extensions       []string   `json:"extensions"`
	ExtensionFile    bool       `json:"extensionFile"`
	MultiLine        [][]string `json:"multi_line"`
	Quotes           [][]string `json:"quotes"`
	NestedMultiLine  bool       `json:"nestedmultiline"`
}

type LanguageFeature struct {
	ComplexityChecks  [][]byte
	ComplexityCheckMask byte
	ComplexityBytes   []byte
	SingleLineComment [][]byte
	SingleLineCommentMask byte
	MultiLineComment  []OpenClose
	MultiLineCommentMask byte
	StringChecks      []OpenClose
	StringCheckMask byte
	Nested            bool
	ProcessBytes      []byte
	ProcessMask       byte
}

// FileJobCallback is an interface that FileJobs can implement to get a per line callback with the line type
type FileJobCallback interface {
	// ProcessLine should return true to continue processing or false to stop further processing and return
	ProcessLine(job *FileJob, currentLine int64, lineType LineType) bool
}

type FileJob struct {
	Language           string
	Filename           string
	Extension          string
	Location           string
	Content            []byte
	Bytes              int64
	Lines              int64
	Code               int64
	Comment            int64
	Blank              int64
	Complexity         int64
	WeightedComplexity float64
	Hash               []byte
	Callback           FileJobCallback
	Binary             bool
}

type LanguageSummary struct {
	Name               string
	Bytes              int64
	Lines              int64
	Code               int64
	Comment            int64
	Blank              int64
	Complexity         int64
	Count              int64
	WeightedComplexity float64
	Files              []*FileJob
}

type OpenClose struct {
	Open  []byte
	Close []byte
}

type CheckDuplicates struct {
	hashes map[int64][][]byte
	mux    sync.Mutex
}

func (c *CheckDuplicates) Add(key int64, hash []byte) {
	c.mux.Lock()
	defer c.mux.Unlock()

	hashes, ok := c.hashes[key]
	if ok {
		c.hashes[key] = append(hashes, hash)
	} else {
		c.hashes[key] = [][]byte{hash}
	}
}

func (c *CheckDuplicates) Check(key int64, hash []byte) bool {
	c.mux.Lock()
	defer c.mux.Unlock()

	hashes, ok := c.hashes[key]
	if ok {
		for _, h := range hashes {
			if bytes.Equal(h, hash) {
				return true
			}
		}
	}

	return false
}
