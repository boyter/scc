// SPDX-License-Identifier: MIT

package processor

import (
	"slices"
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
