package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestSimpleCache_Add(t *testing.T) {
	simpleCache := NewSimpleCache(5, 60)

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
	simpleCache := NewSimpleCache(10, 60)

	for i := 0; i < 500; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
	}

	simpleCache.Add("10", []byte{})

	if len(simpleCache.items) != 10 {
		t.Errorf("expected 10 items got %v", len(simpleCache.items))
	}
}

func TestSimpleCache_MultipleLarge(t *testing.T) {
	simpleCache := NewSimpleCache(1000, 60)

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

func TestSimpleCache_AgeOut(t *testing.T) {
	simpleCache := &SimpleCache{
		maxItems: 100,
		items:    map[string]cacheEntry{},
		lock:     sync.Mutex{},
		getUnix: func() int64 {
			return 0
		},
		ageOutTimeSeconds: 10,
	}

	for i := 0; i < 10; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
	}

	// advance time
	simpleCache.getUnix = func() int64 {
		return 10000
	}
	// simulate eviction over time
	for i := 0; i < 10; i++ {
		simpleCache.ageOut()
	}

	if len(simpleCache.items) != 0 {
		t.Errorf("expected 0 items got %v", len(simpleCache.items))
	}
}

func TestSimpleCache_AgeOutTime(t *testing.T) {
	simpleCache := NewSimpleCache(100, 1)

	for i := 0; i < 10; i++ {
		simpleCache.Add(fmt.Sprintf("%d", i), []byte{})
	}

	time.Sleep(1 * time.Second)

	// simulate eviction over time
	for i := 0; i < 10; i++ {
		simpleCache.ageOut()
	}

	if len(simpleCache.items) != 0 {
		t.Errorf("expected 0 items got %v", len(simpleCache.items))
	}
}
