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
	Complexity            *Trie
	MultiLineComments     *Trie
	SingleLineComments    *Trie
	Strings               *Trie
	Nested                bool
	ComplexityCheckMask   byte
	SingleLineCommentMask byte
	MultiLineCommentMask  byte
	StringCheckMask       byte
	ProcessMask           byte
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

type Trie struct {
	Exists bool
	Close  []byte
	Table  [256]*Trie
}

func (t *Trie) Insert(str []byte) {
	var x, y *Trie

	x = t
	for _, c := range str {
		y = x.Table[byte(c)]
		if y == nil {
			y = &Trie{}
			x.Table[byte(c)] = y
		}
		x = y
	}
	x.Exists = true
}

func (t *Trie) InsertClose(str, suffix []byte) {
	var x, y *Trie

	x = t
	for _, c := range str {
		y = x.Table[byte(c)]
		if y == nil {
			y = &Trie{}
			x.Table[byte(c)] = y
		}
		x = y
	}
	x.Exists = true
	x.Close = suffix
}

func (t *Trie) Match(str []byte) (bool, int, []byte) {
	var x *Trie
	var depth int
	var c byte

	x = t
	for depth, c = range str {
		if x.Table[c] == nil {
			return x.Exists, depth, x.Close
		}
		x = x.Table[c]
	}
	return x.Exists, depth, x.Close
}
