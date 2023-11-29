package main

import (
	"math"
	"sync"
	"time"
)

type cacheEntry struct {
	entry []byte
	hits  int
}

type SimpleCache struct {
	maxItems int
	items    map[string]cacheEntry
	lock     sync.Mutex
}

func NewSimpleCache(maxItems int) *SimpleCache {
	simpleCache := SimpleCache{
		maxItems: maxItems,
		items:    map[string]cacheEntry{},
		lock:     sync.Mutex{},
	}
	simpleCache.runAgeItems()
	return &simpleCache
}

func (cache *SimpleCache) runAgeItems() {
	go func() {
		for {
			// maps are randomly ordered, so only decrementing 50 at a time should be acceptable
			count := 50
			cache.lock.Lock()
			for k, v := range cache.items {
				if v.hits > 0 {
					v.hits--
					cache.items[k] = v
				}
				count--
				if count <= 0 {
					break
				}
			}
			cache.lock.Unlock()
			time.Sleep(10 * time.Second)
		}
	}()
}

func (cache *SimpleCache) Add(cacheKey string, entry []byte) {
	cache.expireItems()

	cache.lock.Lock()
	defer cache.lock.Unlock()

	cache.items[cacheKey] = cacheEntry{
		entry: entry,
		hits:  1,
	}
}

func (cache *SimpleCache) Get(cacheKey string) ([]byte, bool) {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	item, ok := cache.items[cacheKey]

	if ok {
		if item.hits < 100 {
			item.hits++
		}
		cache.items[cacheKey] = item
		return item.entry, true
	}

	return nil, false
}

// ExpireItems is called before any insert operation because we need to ensure we have less than
// the total number of items
func (cache *SimpleCache) expireItems() {
	cache.lock.Lock()
	defer cache.lock.Unlock()

	count := 10
	if len(cache.items) >= cache.maxItems {
		lfuKey := ""
		lfuLowestCount := math.MaxInt

		for k, v := range cache.items {
			v.hits--
			cache.items[k] = v
			if v.hits < lfuLowestCount {
				lfuKey = k
				lfuLowestCount = v.hits
			}

			// we only want to process X random elements so we don't spin forever
			count--
			if count <= 0 {
				break
			}
		}

		delete(cache.items, lfuKey)
	}
}
