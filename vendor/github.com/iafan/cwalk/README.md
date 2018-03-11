### cwalk = Concurrent filepath.Walk

A concurrent version of https://golang.org/pkg/path/filepath/#Walk function
that scans files in a directory tree and runs a callback for each file.

Since scanning (and callback execution) is done from within goroutines,
this may result in a significant performance boost on multicore systems
in cases when the bottleneck is the CPU, not the I/O.

My tests showed ~3.5x average speed increase on an 8-core CPU and 8 workers.
For measurements, I used the provided `bin/traversaltime.go` utility that measures
directory traversal time for both concurrent (`cwalk.Walk()`) and standard
(`filepath.Walk()`) functions.

Here are two common use cases when `cwalk` might be useful:

  1. You're doing subsequent scans of the same directory
     (e.g. monitoring it for changes), which means that the directory structure
     is likely cached in memory by OS;

  2. You're doing some CPU-heavy processing for each file in the callback.

### Installation

```shell
$ go get github.com/iafan/cwalk
```

### Usage

```go
import "github.com/iafan/cwalk"

...

func walkFunc(path string, info os.FileInfo, err error) error {
    ...
}

...

cwalk.Walk("/path/to/dir", walkFunc)
```
