Project Cuba
------------

Experiment in allowing workers to own the means of production.

Go makes many parallel cases easy to implement. Cuba aims to simplify some of
the cases that aren't as easy to implement.

If your algorithm can handle unbounded parallelism, then just spawn thousands
of goroutines and let the Go runtime figure out how to make it work. However,
this doesn't work if you have limits on external resources like open file
descriptors or database connection handles. It also may not be the most memory
efficient since each goroutine needs its own call stack.

For bounded parallelism, Go's `chan`s allow splitting a task into a sequence of
steps that happen in parallel. This model supports fanning in and out to
increase the parallelism beyond the number of sequential steps.

The limitation of using `chan`s is that they only work so long as the pipeline
of work is unidirectional. Simple linear sequences are easiest, but you can
construct any acyclical graph.

Cuba aims to support parallelism where the dataflow may be cyclical.

One example is a crawler style algorithm which involves both taking a node to
process, and pushing newly discovered nodes back onto the queue.

Another example is backing off and retrying without head-of-line blocking. Work
items could be pushed to the back of the queue to be retried when they come
around again.

Usage
=====

First, define a worker function:

```go
func doWork(handle *cuba.Handle) {
	item := handle.Item().(myItemType)

  // Do something with item

  for _, newItem := range newItemsFound {
    handle.Push(newItem)
    // Optionally: handle.Sync()
  }
}
```

Normally, `handle.Push()` buffers the new items before releasing them back to
the work pool. When the function returns, the pool mutex will be aquired once
and the items pushed as a batch. This means that other threads may sit idle
until the function returns.

Calling `handle.Sync()` will immediately aquire the lock and push any items in
the buffer. This will increase lock contention, but may improve parallelism if
you have long running workers.

Then initialize a pool, seed with initial items, and wait for processing to complete.

```go
  // Initialize a new pool.
  // Swap `cuba.NewQueue()` for `cuba.NewStack()` for LIFO order
	pool := cuba.New(doWork, cuba.NewQueue())

  // Optionally: pool.SetMaxWorkers(n)

  // Seed the pool with initial work
  // Workers are started as soon as something is available to process
  pool.Push(myFirstItem)

  // Wait for the workers to finish processing all work and terminate
  pool.Finish()
```

By default, `cuba` pools have a maximum thread count equal to `runtime.NumCPU()`. This can be changed by calling `pool.SetMaxWorkers`.

