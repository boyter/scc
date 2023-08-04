package main

import (
	"fmt"
	"testing"
)

func TestSimpleCache_Add(t *testing.T) {
	simpleCache := NewSimpleCache(5)

	for i := 0; i < 5; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
	}

	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			simpleCache.Get(fmt.Sprintf("%d", i))
		}
	}

	simpleCache.Add("10", []byte{})
}

func TestSimpleCache_Multiple(t *testing.T) {
	simpleCache := NewSimpleCache(10)

	for i := 0; i < 500; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
	}

	simpleCache.Add("10", []byte{})

	if len(simpleCache.items) != 10 {
		t.Errorf("expected 10 items got %v", len(simpleCache.items))
	}
}

func TestSimpleCache_MultipleLarge(t *testing.T) {
	simpleCache := NewSimpleCache(1000)

	for i := 0; i < 500000; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
		simpleCache.Add("10", []byte{})
		simpleCache.Get(fmt.Sprintf("%d", i))
		simpleCache.Get("10")
		simpleCache.Get("10")
	}

	if len(simpleCache.items) != 999 {
		t.Errorf("expected 999 items got %v", len(simpleCache.items))
	}
}
