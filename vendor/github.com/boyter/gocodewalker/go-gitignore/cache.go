// SPDX-License-Identifier: MIT

package gitignore

import (
	"sync"
)

// Cache is the interface for the GitIgnore cache
type Cache interface {
	// Set stores the GitIgnore ignore against its path.
	Set(path string, ig GitIgnore)

	// Get attempts to retrieve an GitIgnore instance associated with the given
	// path. If the path is not known nil is returned.
	Get(path string) GitIgnore
}

// cache is the default thread-safe cache implementation
type cache struct {
	_i    map[string]GitIgnore
	_lock sync.Mutex
}

// NewCache returns a Cache instance. This is a thread-safe, in-memory cache
// for GitIgnore instances.
func NewCache() Cache {
	return &cache{}
} // Cache()

// Set stores the GitIgnore ignore against its path.
func (c *cache) Set(path string, ignore GitIgnore) {
	if ignore == nil {
		return
	}

	// ensure the map is defined
	if c._i == nil {
		c._i = make(map[string]GitIgnore)
	}

	// set the cache item
	c._lock.Lock()
	c._i[path] = ignore
	c._lock.Unlock()
} // Set()

// Get attempts to retrieve an GitIgnore instance associated with the given
// path. If the path is not known nil is returned.
func (c *cache) Get(path string) GitIgnore {
	c._lock.Lock()
	_ignore, _ok := c._i[path]
	c._lock.Unlock()
	if _ok {
		return _ignore
	} else {
		return nil
	}
} // Get()

// ensure cache supports the Cache interface
var _ Cache = &cache{}
