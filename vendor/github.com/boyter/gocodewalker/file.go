// Package file provides file operations specific to code repositories
// such as walking the file tree obeying .ignore and .gitignore files
// or looking for the root directory assuming already in a git project

// SPDX-License-Identifier: MIT

package gocodewalker

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/boyter/gocodewalker/go-gitignore"
	"golang.org/x/sync/errgroup"
)

const (
	GitIgnore  = ".gitignore"
	Ignore     = ".ignore"
	GitModules = ".gitmodules"
)

// ErrTerminateWalk error which indicates that the walker was terminated
var ErrTerminateWalk = errors.New("gocodewalker terminated")

// File is a struct returned which contains the location and the filename of the file that passed all exclusion rules
type File struct {
	Location string
	Filename string
}

var semaphoreCount = 8

type FileWalker struct {
	fileListQueue          chan *File
	errorsHandler          func(error) bool // If returns true will continue to process where possible, otherwise returns if possible
	directory              string
	directories            []string
	LocationExcludePattern []string // Case-sensitive patterns which exclude directory/file matches
	IncludeDirectory       []string
	ExcludeDirectory       []string // Paths to always ignore such as .git,.svn and .hg
	IncludeFilename        []string
	ExcludeFilename        []string
	IncludeDirectoryRegex  []*regexp.Regexp // Must match regex as logical OR IE can match any of them
	ExcludeDirectoryRegex  []*regexp.Regexp
	IncludeFilenameRegex   []*regexp.Regexp
	ExcludeFilenameRegex   []*regexp.Regexp
	AllowListExtensions    []string // Which extensions should be allowed case sensitive
	ExcludeListExtensions  []string // Which extensions should be excluded case sensitive
	walkMutex              sync.Mutex
	terminateWalking       bool
	isWalking              bool
	IgnoreIgnoreFile       bool     // Should .ignore files be respected?
	IgnoreGitIgnore        bool     // Should .gitignore files be respected?
	IgnoreGitModules       bool     // Should .gitmodules files be respected?
	CustomIgnore           []string // Custom ignore files
	IncludeHidden          bool     // Should hidden files and directories be included/walked
	osOpen                 func(name string) (*os.File, error)
	osReadFile             func(name string) ([]byte, error)
	countingSemaphore      chan bool
	semaphoreCount         int
	MaxDepth               int
}

// NewFileWalker constructs a filewalker, which will walk the supplied directory
// and output File results to the supplied queue as it finds them
func NewFileWalker(directory string, fileListQueue chan *File) *FileWalker {
	return &FileWalker{
		fileListQueue:          fileListQueue,
		errorsHandler:          func(e error) bool { return true }, // a generic one that just swallows everything
		directory:              directory,
		LocationExcludePattern: nil,
		IncludeDirectory:       nil,
		ExcludeDirectory:       nil,
		IncludeFilename:        nil,
		ExcludeFilename:        nil,
		IncludeDirectoryRegex:  nil,
		ExcludeDirectoryRegex:  nil,
		IncludeFilenameRegex:   nil,
		ExcludeFilenameRegex:   nil,
		AllowListExtensions:    nil,
		ExcludeListExtensions:  nil,
		walkMutex:              sync.Mutex{},
		terminateWalking:       false,
		isWalking:              false,
		IgnoreIgnoreFile:       false,
		IgnoreGitIgnore:        false,
		CustomIgnore:           []string{},
		IgnoreGitModules:       false,
		IncludeHidden:          false,
		osOpen:                 os.Open,
		osReadFile:             os.ReadFile,
		countingSemaphore:      make(chan bool, semaphoreCount),
		semaphoreCount:         semaphoreCount,
		MaxDepth:               -1,
	}
}

// NewParallelFileWalker constructs a filewalker, which will walk the supplied directories in parallel
// and output File results to the supplied queue as it finds them
func NewParallelFileWalker(directories []string, fileListQueue chan *File) *FileWalker {
	return &FileWalker{
		fileListQueue:          fileListQueue,
		errorsHandler:          func(e error) bool { return true }, // a generic one that just swallows everything
		directories:            directories,
		LocationExcludePattern: nil,
		IncludeDirectory:       nil,
		ExcludeDirectory:       nil,
		IncludeFilename:        nil,
		ExcludeFilename:        nil,
		IncludeDirectoryRegex:  nil,
		ExcludeDirectoryRegex:  nil,
		IncludeFilenameRegex:   nil,
		ExcludeFilenameRegex:   nil,
		AllowListExtensions:    nil,
		ExcludeListExtensions:  nil,
		walkMutex:              sync.Mutex{},
		terminateWalking:       false,
		isWalking:              false,
		IgnoreIgnoreFile:       false,
		IgnoreGitIgnore:        false,
		CustomIgnore:           []string{},
		IgnoreGitModules:       false,
		IncludeHidden:          false,
		osOpen:                 os.Open,
		osReadFile:             os.ReadFile,
		countingSemaphore:      make(chan bool, semaphoreCount),
		semaphoreCount:         semaphoreCount,
		MaxDepth:               -1,
	}
}

// SetConcurrency sets the concurrency when walking
// which controls the number of goroutines that
// walk directories concurrently
// by default it is set to 8
// must be a whole integer greater than 0
func (f *FileWalker) SetConcurrency(i int) {
	f.walkMutex.Lock()
	defer f.walkMutex.Unlock()
	if i >= 1 {
		f.semaphoreCount = i
	}
}

// Walking gets the state of the file walker and determine
// if we are walking or not
func (f *FileWalker) Walking() bool {
	f.walkMutex.Lock()
	defer f.walkMutex.Unlock()
	return f.isWalking
}

// Terminate have the walker break out of walking and return as
// soon as it possibly can. This is needed because
// this walker needs to work in a TUI interactive mode and
// as such we need to be able to end old processes
func (f *FileWalker) Terminate() {
	f.walkMutex.Lock()
	defer f.walkMutex.Unlock()
	f.terminateWalking = true
}

// SetErrorHandler sets the function that is called on processing any error
// where if you return true it will attempt to continue processing, and if false
// will return the error instantly
func (f *FileWalker) SetErrorHandler(errors func(error) bool) {
	if errors != nil {
		f.errorsHandler = errors
	}
}

// Start will start walking the supplied directory with the supplied settings
// and putting files that mach into the supplied channel.
// Returns usual ioutil errors if there is a file issue
// and a ErrTerminateWalk if terminate is called while walking
func (f *FileWalker) Start() error {
	f.walkMutex.Lock()
	f.isWalking = true
	f.walkMutex.Unlock()

	// we now set the counting semaphore based on the count
	// done here because it should not change while walking
	f.countingSemaphore = make(chan bool, semaphoreCount)

	var err error
	if len(f.directories) != 0 {
		eg := errgroup.Group{}
		for _, directory := range f.directories {
			d := directory // capture var
			eg.Go(func() error {
				return f.walkDirectoryRecursive(0, d, []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{})
			})
		}

		err = eg.Wait()
	} else {
		if f.directory != "" {
			err = f.walkDirectoryRecursive(0, f.directory, []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{})
		}
	}

	close(f.fileListQueue)

	f.walkMutex.Lock()
	f.isWalking = false
	f.walkMutex.Unlock()

	return err
}

func (f *FileWalker) walkDirectoryRecursive(iteration int,
	directory string,
	gitignores []gitignore.GitIgnore,
	ignores []gitignore.GitIgnore,
	moduleIgnores []gitignore.GitIgnore,
	customIgnores []gitignore.GitIgnore) error {

	// implement max depth option
	if f.MaxDepth != -1 && iteration >= f.MaxDepth {
		return nil
	}

	if iteration == 1 {
		f.countingSemaphore <- true
		defer func() {
			<-f.countingSemaphore
		}()
	}

	// NB have to call unlock not using defer because method is recursive
	// and will deadlock if not done manually
	f.walkMutex.Lock()
	if f.terminateWalking {
		f.walkMutex.Unlock()
		return ErrTerminateWalk
	}
	f.walkMutex.Unlock()

	d, err := f.osOpen(directory)
	if err != nil {
		// nothing we can do with this so return nil and process as best we can
		if f.errorsHandler(err) {
			return nil
		}
		return err
	}
	defer d.Close()

	foundFiles, err := d.ReadDir(-1)
	if err != nil {
		// nothing we can do with this so return nil and process as best we can
		if f.errorsHandler(err) {
			return nil
		}
		return err
	}

	files := []fs.DirEntry{}
	dirs := []fs.DirEntry{}

	// We want to break apart the files and directories from the
	// return as we loop over them differently and this avoids some
	// nested if logic at the expense of a "redundant" loop
	for _, file := range foundFiles {
		if file.IsDir() {
			dirs = append(dirs, file)
		} else {
			files = append(files, file)
		}
	}

	// Pull out all ignore, gitignore and gitmodule files and add them
	// to out collection of gitignores to be applied for this pass
	// and any subdirectories
	// Since they can apply to the current list of files we need to ensure
	// we do this before processing files themselves
	for _, file := range files {
		if !f.IgnoreGitIgnore {
			if file.Name() == GitIgnore {
				c, err := f.osReadFile(filepath.Join(directory, file.Name()))
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				gitIgnore := gitignore.New(bytes.NewReader(c), abs, nil)
				gitignores = append(gitignores, gitIgnore)
			}
		}

		if !f.IgnoreIgnoreFile {
			if file.Name() == Ignore {
				c, err := f.osReadFile(filepath.Join(directory, file.Name()))
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				gitIgnore := gitignore.New(bytes.NewReader(c), abs, nil)
				ignores = append(ignores, gitIgnore)
			}
		}

		// this should only happen on the first iteration
		// because there should be one .gitmodules file per repository
		// however we also need to support someone running in a directory of
		// projects that have multiple repositories or in a go vendor
		// repository etc... hence check every time
		if !f.IgnoreGitModules {
			if file.Name() == GitModules {
				// now we need to open and parse the file
				c, err := f.osReadFile(filepath.Join(directory, file.Name()))
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				for _, gm := range extractGitModuleFolders(string(c)) {
					gitIgnore := gitignore.New(strings.NewReader(gm), abs, nil)
					moduleIgnores = append(moduleIgnores, gitIgnore)
				}
			}
		}

		for _, ci := range f.CustomIgnore {
			if file.Name() == ci {
				c, err := f.osReadFile(filepath.Join(directory, file.Name()))
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return err
				}

				gitIgnore := gitignore.New(bytes.NewReader(c), abs, nil)
				customIgnores = append(customIgnores, gitIgnore)
			}
		}
	}

	// Process files first to start feeding whatever process is consuming
	// the output before traversing into directories for more files
	for _, file := range files {
		shouldIgnore := false
		joined := filepath.Join(directory, file.Name())

		for _, ignore := range gitignores {
			// we have the following situations
			// 1. none of the gitignores match
			// 2. one or more match
			// for #1 this means we should include the file
			// for #2 this means the last one wins since it should be the most correct
			if ignore.MatchIsDir(joined, false) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}

		for _, ignore := range ignores {
			// same rules as above
			if ignore.MatchIsDir(joined, false) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}

		for _, ignore := range customIgnores {
			// same rules as above
			if ignore.MatchIsDir(joined, false) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}

		if len(f.IncludeFilename) != 0 {
			// include files
			found := false
			for _, allow := range f.IncludeFilename {
				if file.Name() == allow {
					found = true
				}
			}
			if !found {
				shouldIgnore = true
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeFilename {
			if file.Name() == deny {
				shouldIgnore = true
			}
		}

		if len(f.IncludeFilenameRegex) != 0 {
			found := false
			for _, allow := range f.IncludeFilenameRegex {
				if allow.Match([]byte(file.Name())) {
					found = true
				}
			}
			if !found {
				shouldIgnore = true
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeFilenameRegex {
			if deny.Match([]byte(file.Name())) {
				shouldIgnore = true
			}
		}

		// Ignore hidden files
		if !f.IncludeHidden {
			s, err := IsHiddenDirEntry(file, directory)
			if err != nil {
				if !f.errorsHandler(err) {
					return err
				}
			}

			if s {
				shouldIgnore = true
			}
		}

		// Check against extensions
		if len(f.AllowListExtensions) != 0 {
			ext := GetExtension(file.Name())

			a := false
			for _, v := range f.AllowListExtensions {
				if v == ext {
					a = true
				}
			}

			// try again because we could have one of those pesky ones such as something.spec.tsx
			// but only if we didn't already find something to save on a bit of processing
			if !a {
				ext = GetExtension(ext)
				for _, v := range f.AllowListExtensions {
					if v == ext {
						a = true
					}
				}
			}

			if !a {
				shouldIgnore = true
			}
		}

		for _, deny := range f.ExcludeListExtensions {
			ext := GetExtension(file.Name())
			if ext == deny {
				shouldIgnore = true
			}

			if !shouldIgnore {
				ext = GetExtension(ext)
				if ext == deny {
					shouldIgnore = true
				}
			}
		}

		for _, p := range f.LocationExcludePattern {
			if strings.Contains(joined, p) {
				shouldIgnore = true
			}
		}

		if !shouldIgnore {
			f.fileListQueue <- &File{
				Location: joined,
				Filename: file.Name(),
			}
		}
	}

	// if we are the 1st iteration IE not the root, we run in parallel
	wg := sync.WaitGroup{}

	// Now we process the directories after hopefully giving the
	// channel some files to process
	for _, dir := range dirs {
		var shouldIgnore bool
		joined := filepath.Join(directory, dir.Name())

		// Check against the ignore files we have if the file we are looking at
		// should be ignored
		// It is safe to always call this because the gitignores will not be added
		// in previous steps
		for _, ignore := range gitignores {
			// we have the following situations
			// 1. none of the gitignores match
			// 2. one or more match
			// for #1 this means we should include the file
			// for #2 this means the last one wins since it should be the most correct
			if ignore.MatchIsDir(joined, true) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}
		for _, ignore := range ignores {
			// same rules as above
			if ignore.MatchIsDir(joined, true) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}
		for _, ignore := range customIgnores {
			// same rules as above
			if ignore.MatchIsDir(joined, true) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}
		for _, ignore := range moduleIgnores {
			// same rules as above
			if ignore.MatchIsDir(joined, true) != nil {
				shouldIgnore = ignore.Ignore(joined)
			}
		}

		// start by saying we didn't find it then check each possible
		// choice to see if we did find it
		// if we didn't find it then we should ignore
		if len(f.IncludeDirectory) != 0 {
			found := false
			for _, allow := range f.IncludeDirectory {
				if dir.Name() == allow {
					found = true
				}
			}
			if !found {
				shouldIgnore = true
			}
		}
		// Confirm if there are any files in the path deny list which usually includes
		// things like .git .hg and .svn
		// Comes after include as it takes precedence
		for _, deny := range f.ExcludeDirectory {
			if isSuffixDir(joined, deny) {
				shouldIgnore = true
			}
		}

		if len(f.IncludeDirectoryRegex) != 0 {
			found := false
			for _, allow := range f.IncludeDirectoryRegex {
				if allow.Match([]byte(dir.Name())) {
					found = true
				}
			}
			if !found {
				shouldIgnore = true
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeDirectoryRegex {
			if deny.Match([]byte(dir.Name())) {
				shouldIgnore = true
			}
		}

		// Ignore hidden directories
		if !f.IncludeHidden {
			s, err := IsHiddenDirEntry(dir, directory)
			if err != nil {
				if !f.errorsHandler(err) {
					return err
				}
			}

			if s {
				shouldIgnore = true
			}
		}

		if !shouldIgnore {
			for _, p := range f.LocationExcludePattern {
				if strings.Contains(joined, p) {
					shouldIgnore = true
				}
			}

			if iteration == 0 {
				wg.Add(1)
				go func(iteration int, directory string, gitignores []gitignore.GitIgnore, ignores []gitignore.GitIgnore) {
					_ = f.walkDirectoryRecursive(iteration+1, joined, gitignores, ignores, moduleIgnores, customIgnores)
					wg.Done()
				}(iteration, joined, gitignores, ignores)
			} else {
				err = f.walkDirectoryRecursive(iteration+1, joined, gitignores, ignores, moduleIgnores, customIgnores)
				if err != nil {
					return err
				}
			}
		}
	}

	wg.Wait()

	return nil
}

// FindRepositoryRoot given the supplied directory backwards looking for .git or .hg
// directories indicating we should start our search from that
// location as it's the root.
// Returns the first directory below supplied with .git or .hg in it
// otherwise the supplied directory
func FindRepositoryRoot(startDirectory string) string {
	// Firstly try to determine our real location
	curdir, err := os.Getwd()
	if err != nil {
		return startDirectory
	}

	// Check if we have .git or .hg where we are and if
	// so just return because we are already there
	if checkForGitOrMercurial(curdir) {
		return startDirectory
	}

	// We did not find something, so now we need to walk the file tree
	// backwards in a cross platform way and if we find
	// a match we return that
	lastIndex := strings.LastIndex(curdir, string(os.PathSeparator))
	for lastIndex != -1 {
		curdir = curdir[:lastIndex]

		if checkForGitOrMercurial(curdir) {
			return curdir
		}

		lastIndex = strings.LastIndex(curdir, string(os.PathSeparator))
	}

	// If we didn't find a good match return the supplied directory
	// so that we start the search from where we started at least
	// rather than the root
	return startDirectory
}

// Check if there is a .git or .hg folder in the supplied directory
func checkForGitOrMercurial(curdir string) bool {
	if stat, err := os.Stat(filepath.Join(curdir, ".git")); err == nil && stat.IsDir() {
		return true
	}

	if stat, err := os.Stat(filepath.Join(curdir, ".hg")); err == nil && stat.IsDir() {
		return true
	}

	return false
}

// GetExtension is a custom version of extracting extensions for a file
// which deals with extensions specific to code such as
// .travis.yml and the like
func GetExtension(name string) string {
	name = strings.ToLower(name)
	if !strings.Contains(name, ".") {
		return name
	}

	if strings.LastIndex(name, ".") == 0 {
		return name
	}

	return path.Ext(name)[1:]
}
