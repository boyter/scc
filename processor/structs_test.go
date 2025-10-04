// SPDX-License-Identifier: MIT

package processor

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestCheckDuplicates(t *testing.T) {
	c := CheckDuplicates{
		hashes: make(map[int64][][]byte),
	}

	c.Add(1, []byte("hash"))
	c.Add(1, []byte("hash2"))

	if !c.Check(1, []byte("hash")) {
		t.Error("Expected match")
	}

	if !c.Check(1, []byte("hash2")) {
		t.Error("Expected match")
	}

	if c.Check(2, []byte("hash")) {
		t.Error("Expected no match")
	}

	if c.Check(1, []byte("hash3")) {
		t.Error("Expected no match")
	}
}

func TestMatch(t *testing.T) {
	trie := &Trie{}
	trie.InsertClose(TString, []byte("'"), []byte("'"))
	trie.InsertClose(TString, []byte("'''"), []byte("'''"))

	testCases := []struct {
		token        []byte
		expectType   int
		expectDepth  int
		expectClosed []byte
	}{
		{
			token:        []byte("'"),
			expectType:   TString,
			expectDepth:  0,
			expectClosed: []byte("'"),
		},
		{
			token:        []byte("-"),
			expectType:   0,
			expectDepth:  0,
			expectClosed: []byte{},
		},
		{
			token:        []byte("'-"),
			expectType:   TString,
			expectDepth:  1,
			expectClosed: []byte("'"),
		},
		{
			token:        []byte("''"),
			expectType:   TString,
			expectDepth:  0,
			expectClosed: []byte("'"),
		},
		{
			token:        []byte("'''"),
			expectType:   1,
			expectDepth:  2,
			expectClosed: []byte("'''"),
		},
		{
			token:        []byte("'''a'''"),
			expectType:   TString,
			expectDepth:  3,
			expectClosed: []byte("'''"),
		},
	}

	for _, tc := range testCases {
		typ, depth, closed := trie.Match(tc.token)
		if typ != tc.expectType {
			t.Errorf("\"%v\" matched wrong type, want: %v, got: %v", string(tc.token), tc.expectType, typ)
		}
		if depth != tc.expectDepth {
			t.Errorf("\"%v\" matched wrong depth, want: %v, got: %v", string(tc.token), tc.expectDepth, depth)
		}
		if !slices.Equal(tc.expectClosed, closed) {
			t.Errorf("\"%v\" matched wrong closed, want: %v, got: %v", string(tc.token), tc.expectClosed, closed)
		}
	}
}

func TestFileJobJSONIgnoreFields(t *testing.T) {
	job := FileJob{
		Language:           "Dockerfile",
		PossibleLanguages:  []string{"Dockerfile"},
		Filename:           "Dockerfile",
		Extension:          "Dockerfile",
		Location:           "Dockerfile",
		Symlocation:        "",
		Content:            []uint8{1, 2, 3},
		Bytes:              248,
		Lines:              14,
		Code:               11,
		Comment:            0,
		Blank:              3,
		Complexity:         1,
		ComplexityLine:     []int64{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		WeightedComplexity: 1,
		Hash:               nil,
		Callback:           &linecounter{},
		Binary:             false,
		Minified:           false,
		Generated:          false,
		EndPoint:           0,
		Uloc:               0,
		LineLength:         []int{1, 2, 3},
	}

	data, err := json.MarshalIndent(&job, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, `"Content": `) {
		t.Errorf("Content should be ignored")
	}
	if strings.Contains(jsonStr, `"ComplexityLine": `) {
		t.Errorf("ComplexityLine should be ignored")
	}
	if strings.Contains(jsonStr, `"Callback": `) {
		t.Errorf("Callback should be ignored")
	}
	if strings.Contains(jsonStr, `"LineLength": `) {
		t.Errorf("LineLength should be ignored")
	}
}
