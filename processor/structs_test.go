package processor

import (
	"testing"
)

func CheckDuplicatesTest(t *testing.T) {
	c := CheckDuplicates{
		hashes: make(map[int][][]byte),
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
