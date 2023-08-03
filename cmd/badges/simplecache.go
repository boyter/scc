package main

import (
	"math"
	"sync"
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
	cache := SimpleCache{
		maxItems: maxItems,
		items:    map[string]cacheEntry{},
		lock:     sync.Mutex{},
	}

	return &cache
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
		lfuLowestCount := math.MaxInt64

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
