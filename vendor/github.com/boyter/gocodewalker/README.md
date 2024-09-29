# gocodewalker

[![Go Report Card](https://goreportcard.com/badge/github.com/boyter/gocodewalker)](https://goreportcard.com/report/github.com/boyter/gocodewalker)
[![Str Count Badge](https://sloc.xyz/github/boyter/gocodewalker/)](https://github.com/boyter/gocodewalker/)

Library to help with walking of code directories in Go. 

The problem. You want to walk the directories of a code repository. You want to respect .gitignore and .ignore files, and 
some are nested. This library is the answer.

 - Designed to walk code repositories or find the root of them.
 - By default, respects both .gitignore and .ignore files (can be disabled) and nested ones for accuracy
 - Has configurable options for skipping files based on regex, extension or general match
 - Uses readdir to provide as fast as possible file walking

NB this was moved from go-code-walker due to the name being annoying and to ensure it has a unique package name. Should still be drop in replaceable
so long as you refer to the new package name.

https://pkg.go.dev/github.com/boyter/gocodewalker

Package provides file operations specific to code repositories such as walking the file tree obeying .ignore and .gitignore files
or looking for the root directory assuming already in a git project.

Example of usage,

```go
fileListQueue := make(chan *gocodewalker.File, 100)

fileWalker := gocodewalker.NewFileWalker(".", fileListQueue)

// restrict to only process files that have the .go extension
fileWalker.AllowListExtensions = append(fileWalker.AllowListExtensions, "go")

// handle the errors by printing them out and then ignore
errorHandler := func(e error) bool {
    fmt.Println("ERR", e.Error())
    return true
}
fileWalker.SetErrorHandler(errorHandler)

go fileWalker.Start()

for f := range fileListQueue {
    fmt.Println(f.Location)
}
```

The above by default will recursively add files to the fileListQueue respecting both .ignore and .gitignore files if found, and
only adding files with the go extension into the queue.

You can also run the walker in parallel with the results intermixed if required,

```go
fileListQueue := make(chan *gocodewalker.File, 100)

fileWalker := gocodewalker.NewParallelFileWalker([]string{".", "someotherdir"}, fileListQueue)
go fileWalker.Start()

for f := range fileListQueue {
    fmt.Println(f.Location)
}
```

All code is licenced as MIT.

### Error Handler

You can supply your own error handler when walking. This allows you to perform an action when there is an error
and decide if the walker should continue to process, or return.

The simplest handler is the below, which if set will swallow all errors and continue as best it can.

```go
errorHandler := func(e error) bool {
    return true
}
fileWalker.SetErrorHandler(errorHandler)
```

If you wanted to return on errors you could use the following.

```go
errorHandler := func(e error) bool {
    return false
}
fileWalker.SetErrorHandler(errorHandler)
```

If you wanted to terminate walking on an error you could use the following, which would cause it to return the error,
and then terminate all walking. This might be desirable where any error indicates a total failure.

```go
errorHandler := func(e error) bool {
    fileWalker.Terminate()
    return false
}
fileWalker.SetErrorHandler(errorHandler)
```

### Testing

Done through unit/integration tests. Otherwise see https://github.com/svent/gitignore-test

See `./cmd/gocodewalker/main.go` for an example of how to implement and validate 

### Info

Details on how gitignores work

https://stackoverflow.com/questions/71735516/proper-way-to-setup-multiple-gitignore-files-in-nested-folders-of-a-repository
