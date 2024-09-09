package main

import (
	"math"
	"sync"
	"time"
)

type cacheEntry struct {
	entry []byte
	hits  int
	age   int64
}

type SimpleCache struct {
	maxItems          int
	items             map[string]cacheEntry
	lock              sync.Mutex
	getUnix           func() int64
	ageOutTimeSeconds int64
}

func NewSimpleCache(maxItems int, ageOutTimeSeconds int64) *SimpleCache {
	simpleCache := SimpleCache{
		maxItems: maxItems,
		items:    map[string]cacheEntry{},
		lock:     sync.Mutex{},
		getUnix: func() int64 {
			return time.Now().Unix()
		},
		ageOutTimeSeconds: ageOutTimeSeconds,
	}
	simpleCache.runAgeItems()
	return &simpleCache
}

func (c *SimpleCache) runAgeItems() {
	go func() {
		time.Sleep(10 * time.Second)
		c.adjustLfu()
		c.ageOut()
	}()
}

func (c *SimpleCache) adjustLfu() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// maps are randomly ordered, so only decrementing 50 at a time should be acceptable
	count := 50

	for k, v := range c.items {
		if v.hits > 0 {
			v.hits--
			c.items[k] = v
		}
		count--
		if count <= 0 {
			break
		}
	}
}

func (c *SimpleCache) Add(cacheKey string, entry []byte) {
	c.evictItems()

	c.lock.Lock()
	defer c.lock.Unlock()

	c.items[cacheKey] = cacheEntry{
		entry: entry,
		hits:  1,
		age:   c.getUnix(),
	}
}

func (c *SimpleCache) Get(cacheKey string) ([]byte, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	item, ok := c.items[cacheKey]

	if ok {
		if item.hits < 100 {
			item.hits++
		}
		c.items[cacheKey] = item
		return item.entry, true
	}

	return nil, false
}

// evictItems is called before any insert operation because we need to ensure we have less than
// the total number of items
func (c *SimpleCache) evictItems() {
	c.lock.Lock()
	defer c.lock.Unlock()

	// insert process only needs to expire if we have too much
	// as such if we haven't hit the limit return
	if len(c.items) < c.maxItems {
		return
	}

	count := 10

	lfuKey := ""
	lfuLowestCount := math.MaxInt

	for k, v := range c.items {
		v.hits--
		c.items[k] = v
		if v.hits < lfuLowestCount {
			lfuKey = k
			lfuLowestCount = v.hits
		}

		// we only want to process X random elements so we don't spin forever
		// however we also exit if the count is <= 0
		count--
		if count <= 0 || lfuLowestCount <= 0 {
			break
		}
	}

	delete(c.items, lfuKey)
}

// ageOut is called on a schedule to evict the oldest entry so long as
// its older than the configured cache time
func (c *SimpleCache) ageOut() {
	// we also want to age out things eventually to avoid https://github.com/boyter/scc/discussions/435
	// as such loop though and the first one that's older than a day with 0 hits is removed

	c.lock.Lock()
	defer c.lock.Unlock()

	count := 10
	lfuKey := ""
	lfuOldest := int64(math.MaxInt)

	// maps are un-ordered so this is acceptable
	for k, v := range c.items {
		c.items[k] = v
		if v.age < lfuOldest {
			lfuKey = k
			lfuOldest = v.age
		}

		count--
		if count <= 0 {
			break
		}
	}

	// evict the oldest but only if its older than it should be
	if lfuOldest <= c.getUnix()-c.ageOutTimeSeconds {
		delete(c.items, lfuKey)
	}
}
