// SPDX-License-Identifier: MIT

package simplecache

import (
	"errors"
	"math"
	"sync"
	"time"
)

type CacheEvictionPolicy int

const (
	LRU CacheEvictionPolicy = iota
	LFU
)

var ErrInvalidEvictionPolicy = errors.New("invalid eviction policy")

// CacheStatus represents the freshness state of a cache lookup
type CacheStatus int

const (
	CacheMiss  CacheStatus = iota
	CacheHit
	CacheStale
)

// CacheResult holds both the value and its freshness status
type CacheResult[T any] struct {
	Value  T
	Status CacheStatus
}

// cacheEntry is a generic structure to hold cached items
type cacheEntry[T any] struct {
	entry   T
	hits    int64
	age     time.Time
	lastUse time.Time
}

// Cache is a generic cache implementation using a map
// not designed for raw performance but to be simple to configure
type Cache[T any] struct {
	items           map[string]cacheEntry[T]
	mutex           sync.Mutex
	maxItems        int // 0 indicates no limits IE never expires unless age limits kick in
	evictionPolicy  CacheEvictionPolicy
	evictionSamples int           // how many random samples do we look for when expiring
	maxAge          time.Duration // at what point should it be evicted no matter what
	staleAge        time.Duration // at what point should the entry be considered stale
}

// CacheInterface is an interface for the Cache type mostly for mocking purposes
type CacheInterface[T any] interface {
	// Set adds or updates an item in the cache
	Set(key string, value T) error

	// Get retrieves an item from the cache by key, returning the value and a boolean indicating if the value was found
	Get(key string) (T, bool)

	// Delete removes an item from the cache by key
	Delete(key string)

	// Clear removes all items from the cache
	Clear()

	// GetWithStatus retrieves an item and its freshness status (CacheMiss, CacheHit, or CacheStale)
	GetWithStatus(key string) CacheResult[T]

	// Peek retrieves an item and its freshness status without ever evicting the entry
	Peek(key string) CacheResult[T]

	// Sum returns the count of items in the cache
	Sum() int
}

type Option struct {
	MaxItems        *int
	EvictionPolicy  *CacheEvictionPolicy
	EvictionSamples *int
	MaxAge          *time.Duration
	StaleAge        *time.Duration
}

// New creates and returns a new Cache
func New[T any](opts ...Option) *Cache[T] {
	sc := &Cache[T]{
		items:           make(map[string]cacheEntry[T]),
		mutex:           sync.Mutex{},
		maxItems:        100_000, // 0 indicates no limits
		evictionPolicy:  LFU,
		evictionSamples: 5,
		maxAge:          0, // by default no expiration
	}

	for _, opt := range opts {
		if opt.MaxItems != nil {
			sc.maxItems = *opt.MaxItems
		}
		if opt.EvictionPolicy != nil {
			sc.evictionPolicy = *opt.EvictionPolicy
		}
		if opt.EvictionSamples != nil {
			sc.evictionSamples = *opt.EvictionSamples
		}
		if opt.MaxAge != nil {
			sc.maxAge = *opt.MaxAge
		}
		if opt.StaleAge != nil {
			sc.staleAge = *opt.StaleAge
		}
	}

	return sc
}

// Set adds or updates an item in the cache with a given key
func (c *Cache[T]) Set(key string, value T) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.maxItems > 0 && len(c.items) >= c.maxItems {
		// now we need to expire things based on our settings
		switch c.evictionPolicy {
		case LRU:
			c.evictLRU()
		case LFU:
			c.evictLFU()
		default:
			return ErrInvalidEvictionPolicy
		}
	}

	c.items[key] = cacheEntry[T]{
		entry:   value,
		hits:    0,
		age:     time.Now(),
		lastUse: time.Now(),
	}

	return nil
}

// evictLFU is a simple LRU eviction where we check which item was last used
// from the random sample we iterate over and then remove it
func (c *Cache[T]) evictLRU() {
	count := 0
	pKey := ""
	oldest := time.Now()

	for k, v := range c.items { // iterating over a map is random in Go
		count++
		if v.age.Before(oldest) {
			oldest = v.age
			pKey = k
		}

		if count >= c.evictionSamples {
			break
		}
	}

	delete(c.items, pKey)
}

// evictLFU is a simple LFU eviction where we check which item has the least number of cache hits
// from the random sample we iterate over and remove it
func (c *Cache[T]) evictLFU() {
	count := 0
	pKey := ""
	pHit := int64(math.MaxInt64)

	for k, v := range c.items { // iterating over a map is random in Go
		count++
		if v.hits < pHit {
			pKey = k
			pHit = v.hits
		}

		if count > c.evictionSamples {
			break
		}
	}
	delete(c.items, pKey)
}

// Get retrieves an item from the cache by key, also incrementing the hit count
func (c *Cache[T]) Get(key string) (T, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, found := c.items[key]
	if found {
		if c.maxAge > 0 { // should it be expired?
			if entry.lastUse.Before(time.Now().Add(-c.maxAge)) {
				delete(c.items, key)
				var zero T
				return zero, false
			}
		}

		entry.hits++
		entry.lastUse = time.Now()
		c.items[key] = entry // Update the hit count in the cache
		return entry.entry, true
	}

	var zero T
	return zero, false
}

// GetWithStatus retrieves an item from the cache by key, returning a CacheResult
// that includes the value and its freshness status (CacheMiss, CacheHit, or CacheStale).
// Unlike Get, staleness is checked against the entry's creation time (age), not lastUse.
func (c *Cache[T]) GetWithStatus(key string) CacheResult[T] {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, found := c.items[key]
	if !found {
		var zero T
		return CacheResult[T]{Value: zero, Status: CacheMiss}
	}

	age := time.Since(entry.age)

	// Hard expiry
	if c.maxAge > 0 && age > c.maxAge {
		delete(c.items, key)
		var zero T
		return CacheResult[T]{Value: zero, Status: CacheMiss}
	}

	// Update hits/lastUse
	entry.hits++
	entry.lastUse = time.Now()
	c.items[key] = entry

	// Stale check
	if c.staleAge > 0 && age > c.staleAge {
		return CacheResult[T]{Value: entry.entry, Status: CacheStale}
	}

	return CacheResult[T]{Value: entry.entry, Status: CacheHit}
}

// Peek retrieves an item and its freshness status without ever evicting the entry.
// Returns CacheMiss only when the key does not exist. Returns CacheStale when the
// entry is past staleAge or maxAge, but still returns the value — it is up to the
// caller to decide what to do (e.g. trigger a background refresh, delete the key).
func (c *Cache[T]) Peek(key string) CacheResult[T] {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, found := c.items[key]
	if !found {
		var zero T
		return CacheResult[T]{Value: zero, Status: CacheMiss}
	}

	age := time.Since(entry.age)

	// Past maxAge → stale (but not deleted)
	if c.maxAge > 0 && age > c.maxAge {
		return CacheResult[T]{Value: entry.entry, Status: CacheStale}
	}

	// Past staleAge → stale
	if c.staleAge > 0 && age > c.staleAge {
		return CacheResult[T]{Value: entry.entry, Status: CacheStale}
	}

	return CacheResult[T]{Value: entry.entry, Status: CacheHit}
}

// Delete removes an item from the cache by key
func (c *Cache[T]) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.items, key)
}

// Clear removes all entries from the cache
func (c *Cache[T]) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]cacheEntry[T])
}

// Sum returns the count of items in the cache
func (c *Cache[T]) Sum() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return len(c.items)
}
