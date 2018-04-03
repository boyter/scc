package processor

import (
	"bytes"
	"sync"
)

type Language struct {
	LineComment      []string   `json:"line_comment"`
	ComplexityChecks []string   `json:"complexitychecks"`
	Extensions       []string   `json:"extensions"`
	MultiLine        [][]string `json:"multi_line"`
	Quotes           [][]string `json:"quotes"`
}

type LanguageFeature struct {
	ComplexityChecks  [][]byte
	ComplexityBytes   []byte
	SingleLineComment [][]byte
	MultiLineComment  []OpenClose
	StringChecks      []OpenClose
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
