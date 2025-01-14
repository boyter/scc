# SimpleCache

[![Scc Count Badge](https://sloc.xyz/github/boyter/simplecache/)](https://github.com/boyter/simplecache/)

A simple thread safe cache implementation using Go generics.

## Why?

While many excellent cache solutions exist, what I often want for smaller projects is a map, with some expiration 
abilities over it. That is intended to fill that role. This is because different types can have
different caching needs, such as a small group of items that should never expire, items that should exist in cache
forever only being removed when the cache is full. Or some combination.

Most operations should be `o(1)` as well as all being thread safe.

### What isn't it

1. A generic cache for anything E.G. redis/memcached
2. Aiming for extreme performance under load
3. Implementing any sort of persistence

# Usage

Import `github.com/boyter/simplecache`

```go
sc := simplecache.New[string]()

_ = sc.Set("key-1", "some value")

v, ok := sc.Get("key-1")
if ok {
	fmt.Println(v) // prints "some value"
}
v, ok = sc.Get("key-99")
if ok {
	fmt.Println(v) // not run "key-99" was never added
}
```

Note that a default cache has an limit of 100,000 items, once the next item is added beyond this limit 5 random 
entries will be checked, and one of them removed based on the default LFU algorithm. 

You can configure this through the use of options, as indicated below

```go
oMi := 1000
oEp := simplecache.LRU
oEs := 5
oMA := time.Second * 60

sc := simplecache.New[string](simplecache.Option{
    MaxItems:        &oMi, // max number of items the cache will hold, evicting on Set, nil for no limit
    EvictionPolicy:  &oEp, // Which eviction policy should be applied LRU or LFU
    EvictionSamples: &oEs, // How many random samples to take from the items to find the best to expire
    MaxAge:          &oMA, // Max age an item can live on Get when past this will be deleted, nil for no expiry
})
```

# Benchmarks?

I don't have any. It's a Go map with some locking. It should be fine. Being 5% faster or slower than any other
cache isn't the point here.
