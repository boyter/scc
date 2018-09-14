package processor

import (
	"bytes"
	"sync"
)

const (
	T_STRING int = iota + 1
	T_SLCOMMENT
	T_MLCOMMENT
	T_COMPLEXITY
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
	Tokens                *Trie
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
	Type  int
	Close []byte
	Table [256]*Trie
}

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

func (root *Trie) Match(token []byte) (int, int, []byte) {
	var node *Trie
	var depth int
	var c byte

	node = root
	for depth, c = range token {
		if node.Table[c] == nil {
			return node.Type, depth, node.Close
		}
		node = node.Table[c]
	}
	return node.Type, depth, node.Close
}
