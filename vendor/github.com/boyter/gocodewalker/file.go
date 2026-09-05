// Package file provides file operations specific to code repositories
// such as walking the file tree obeying .ignore and .gitignore files
// or looking for the root directory assuming already in a git project

// SPDX-License-Identifier: MIT

package gocodewalker

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/boyter/gocodewalker/go-gitignore"
	"golang.org/x/sync/errgroup"
)

const (
	GitIgnore             = ".gitignore"
	Ignore                = ".ignore"
	GitModules            = ".gitmodules"
	IgnoreBinaryFileBytes = 1000

	// gitDirName is the entry that marks a repository root, either as a
	// directory for a normal checkout or as a regular file for a worktree or
	// submodule. Kept unexported because it is an implementation detail of
	// finding $GIT_DIR/info/exclude rather than something callers configure.
	gitDirName = ".git"
)

// ErrTerminateWalk error which indicates that the walker was terminated
var ErrTerminateWalk = errors.New("gocodewalker terminated")

// SkipReason indicates why a file or directory was skipped during walking
type SkipReason string

const (
	SkipReasonGitignore              SkipReason = "gitignore"
	SkipReasonIgnoreFile             SkipReason = "ignore_file"
	SkipReasonCustomIgnore           SkipReason = "custom_ignore"
	SkipReasonGlobalIgnore           SkipReason = "global_ignore"
	SkipReasonModuleIgnore           SkipReason = "module_ignore"
	SkipReasonIncludeFilename        SkipReason = "include_filename"
	SkipReasonExcludeFilename        SkipReason = "exclude_filename"
	SkipReasonIncludeFilenameRegex   SkipReason = "include_filename_regex"
	SkipReasonExcludeFilenameRegex   SkipReason = "exclude_filename_regex"
	SkipReasonHidden                 SkipReason = "hidden"
	SkipReasonAllowListExtension     SkipReason = "allow_list_extension"
	SkipReasonExcludeListExtension   SkipReason = "exclude_list_extension"
	SkipReasonLocationExcludePattern SkipReason = "location_exclude_pattern"
	SkipReasonBinary                 SkipReason = "binary"
	SkipReasonIncludeDirectory       SkipReason = "include_directory"
	SkipReasonExcludeDirectory       SkipReason = "exclude_directory"
	SkipReasonIncludeDirectoryRegex  SkipReason = "include_directory_regex"
	SkipReasonExcludeDirectoryRegex  SkipReason = "exclude_directory_regex"
)

// File is a struct returned which contains the location and the filename of the file that passed all exclusion rules
type File struct {
	Location string
	Filename string
}

var semaphoreCount = 8

type FileWalker struct {
	fileListQueue          chan<- *File
	fileBatchQueue         chan<- []*File   // set by SetFileBatchQueue, takes precedence over fileListQueue
	errorsHandler          func(error) bool // If returns true will continue to process where possible, otherwise returns if possible
	skipHandler            func(path string, name string, isDir bool, reason SkipReason)
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
	stopWalking            atomic.Bool    // mirrors terminateWalking plus any fatal error, read on the hot path
	walkErr                error          // first error seen by any walking goroutine, guarded by walkMutex
	walkWg                 sync.WaitGroup // tracks every detached subdirectory walk for the whole walk
	isWalking              bool
	IgnoreIgnoreFile       bool     // Should .ignore files be respected?
	IgnoreGitIgnore        bool     // Should .gitignore files be respected?
	IgnoreGitModules       bool     // Should .gitmodules files be respected?
	CustomIgnore           []string // Custom ignore filenames discovered while walking
	CustomIgnorePatterns   []string // Custom ignore patterns re-anchored at every directory
	CustomIgnoreFiles      []string // Paths to ignore files read once and anchored at the walk root (lowest priority; any discovered ignore file overrides them)
	IncludeHidden          bool     // Should hidden files and directories be included/walked
	osOpen                 func(name string) (*os.File, error)
	osReadFile             func(name string) ([]byte, error)
	gitDirFromEnv          bool   // was $GIT_DIR set when this walk started
	gitExcludeFromEnv      []byte // contents of $GIT_DIR/info/exclude, read once per walk
	countingSemaphore      chan bool
	semaphoreCount         int
	MaxDepth               int
	IgnoreBinaryFiles      bool // Should we open the file and try to determine if it is binary?
	IgnoreBinaryFileBytes  int  // How many bytes should be used
}

// NewFileWalker constructs a filewalker, which will walk the supplied directory
// and output File results to the supplied queue as it finds them
func NewFileWalker(directory string, fileListQueue chan<- *File) *FileWalker {
	return &FileWalker{
		fileListQueue:          fileListQueue,
		errorsHandler:          func(e error) bool { return true }, // a generic one that just swallows everything
		skipHandler:            func(path string, name string, isDir bool, reason SkipReason) {},
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
		CustomIgnorePatterns:   []string{},
		CustomIgnoreFiles:      []string{},
		IgnoreGitModules:       false,
		IncludeHidden:          false,
		osOpen:                 os.Open,
		osReadFile:             os.ReadFile,
		countingSemaphore:      make(chan bool, semaphoreCount),
		semaphoreCount:         semaphoreCount,
		MaxDepth:               -1,
		IgnoreBinaryFiles:      false,
		IgnoreBinaryFileBytes:  IgnoreBinaryFileBytes,
	}
}

// NewParallelFileWalker constructs a filewalker, which will walk the supplied directories in parallel
// and output File results to the supplied queue as it finds them
func NewParallelFileWalker(directories []string, fileListQueue chan<- *File) *FileWalker {
	return &FileWalker{
		fileListQueue:          fileListQueue,
		errorsHandler:          func(e error) bool { return true }, // a generic one that just swallows everything
		skipHandler:            func(path string, name string, isDir bool, reason SkipReason) {},
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
		CustomIgnorePatterns:   []string{},
		CustomIgnoreFiles:      []string{},
		IgnoreGitModules:       false,
		IncludeHidden:          false,
		osOpen:                 os.Open,
		osReadFile:             os.ReadFile,
		countingSemaphore:      make(chan bool, semaphoreCount),
		semaphoreCount:         semaphoreCount,
		MaxDepth:               -1,
		IgnoreBinaryFiles:      false,
		IgnoreBinaryFileBytes:  IgnoreBinaryFileBytes,
	}
}

// SetFileBatchQueue switches the walker over to handing its results out one
// directory at a time rather than one file at a time. When it is set the
// supplied channel receives a slice of the files each directory yielded, and it
// rather than the per file queue is the one closed when the walk finishes.
//
// This exists because the handover, not the walking, is what limits the walk.
// Measured on the Linux kernel (90,000 files in 6,000 directories) the walking
// goroutines spent 44% of their time blocked in the per file channel send at
// the default concurrency, and 72% of it at a concurrency of 64 — which is why
// raising the concurrency did not make the walk any faster, the extra
// goroutines simply queued at the channel. The cost is per send rather than per
// file, and does not come from the buffer being full: a queue 3,000 times
// deeper did not change it. Handing over a directory at a time turns roughly
// 90,000 sends into roughly 6,000.
//
// Must be called before Start. Passing nil restores the per file behaviour.
func (f *FileWalker) SetFileBatchQueue(queue chan<- []*File) {
	f.walkMutex.Lock()
	defer f.walkMutex.Unlock()
	f.fileBatchQueue = queue
}

// SetConcurrency sets the concurrency when walking
// which controls the number of goroutines that
// walk directories concurrently
// by default it is set to 8
// must be a whole integer greater than 0
//
// It has no effect once Start has been called, because the value is read
// once at the beginning of the walk and cannot change while walking.
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
	f.terminateWalking = true
	if f.walkErr == nil {
		f.walkErr = ErrTerminateWalk
	}
	f.walkMutex.Unlock()
	// set last so any goroutine that observes the stop flag is guaranteed to
	// see the error that goes with it
	f.stopWalking.Store(true)
}

// stop records the first error seen by any walking goroutine and asks every
// other goroutine to unwind as soon as it can. Later errors are discarded so
// the reason the walk ended is the first thing that went wrong, which matches
// the old serial walk returning on the first error it hit.
// The supplied error is returned so callers can write `return f.stop(err)`.
func (f *FileWalker) stop(err error) error {
	f.walkMutex.Lock()
	if f.walkErr == nil {
		f.walkErr = err
	}
	f.walkMutex.Unlock()
	f.stopWalking.Store(true)
	return err
}

// firstError returns the error the walk ended with, if any
func (f *FileWalker) firstError() error {
	f.walkMutex.Lock()
	defer f.walkMutex.Unlock()
	return f.walkErr
}

// SetErrorHandler sets the function that is called on processing any error
// where if you return true it will attempt to continue processing, and if false
// will return the error instantly
//
// The handler is called from the goroutines doing the walking and so may be
// called concurrently by several of them at once. It must be safe for
// concurrent use.
func (f *FileWalker) SetErrorHandler(errors func(error) bool) {
	if errors != nil {
		f.errorsHandler = errors
	}
}

// SetSkipHandler sets the function that is called whenever a file or directory is skipped
// by the filter pipeline. The handler receives the full path, the entry name, whether it is
// a directory, and the reason it was skipped. By default it is a no-op.
//
// The handler is called from the goroutines doing the walking and so may be
// called concurrently by several of them at once. It must be safe for
// concurrent use.
func (f *FileWalker) SetSkipHandler(handler func(path string, name string, isDir bool, reason SkipReason)) {
	if handler != nil {
		f.skipHandler = handler
	}
}

// Start will start walking the supplied directory with the supplied settings
// and putting files that mach into the supplied channel.
// Returns usual ioutil errors if there is a file issue
// and a ErrTerminateWalk if terminate is called while walking
func (f *FileWalker) Start() error {
	f.walkMutex.Lock()
	f.isWalking = true
	// clear anything left over from a previous walk, but keep a Terminate that
	// was called before we started, since that has always stopped the walk
	if f.terminateWalking {
		f.walkErr = ErrTerminateWalk
	} else {
		f.walkErr = nil
	}
	// the concurrency is read once here because it must not change while
	// walking, and this is what SetConcurrency has been setting all along
	concurrency := f.semaphoreCount
	// a FileWalker built as a bare struct literal rather than through one of the
	// constructors has no concurrency set, and a zero sized semaphore would make
	// the root block forever, so fall back to the package default
	if concurrency < 1 {
		concurrency = semaphoreCount
	}
	terminated := f.terminateWalking
	f.walkMutex.Unlock()
	f.stopWalking.Store(terminated)

	// we now set the counting semaphore based on the count
	// done here because it should not change while walking
	f.countingSemaphore = make(chan bool, concurrency)

	// $GIT_DIR cannot change while walking, so it is read once here rather than
	// once per directory, and when it is set the exclude file it names is read
	// once here as well rather than re-read in every directory below.
	f.readEnvGitExclude()

	if len(f.directories) != 0 {
		eg := errgroup.Group{}
		for _, directory := range f.directories {
			d := directory // capture var
			eg.Go(func() error {
				// each root takes a slot for the whole of its own walk, so that
				// the number of directories being read at once never exceeds the
				// configured concurrency however many roots were supplied. This
				// cannot deadlock: a root holds exactly one slot and never blocks
				// on another, and its subdirectories only ever take a slot when
				// one is already free (see walkDirectoryRecursive).
				f.countingSemaphore <- true
				defer func() { <-f.countingSemaphore }()

				globalIgnores, gerr := f.buildGlobalIgnores(d)
				if gerr != nil {
					return f.stop(gerr)
				}
				return f.walkDirectoryRecursive(0, d, absoluteWalkRoot(d), globalIgnores, f.rootGitIgnores(d), []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{})
			})
		}

		_ = eg.Wait()
	} else {
		if f.directory != "" {
			f.countingSemaphore <- true
			globalIgnores, gerr := f.buildGlobalIgnores(f.directory)
			if gerr != nil {
				_ = f.stop(gerr)
			} else {
				_ = f.walkDirectoryRecursive(0, f.directory, absoluteWalkRoot(f.directory), globalIgnores, f.rootGitIgnores(f.directory), []gitignore.GitIgnore{}, []gitignore.GitIgnore{}, []gitignore.GitIgnore{})
			}
			<-f.countingSemaphore
		}
	}

	// subdirectory walks are detached from the frame that forked them, so that a
	// goroutine holds its semaphore slot only while it is actually reading a
	// directory rather than while it waits on its children. That means the roots
	// finishing does not mean the walk has finished, and everything still running
	// has to be waited on here before the queue can be closed.
	f.walkWg.Wait()

	if f.fileBatchQueue != nil {
		close(f.fileBatchQueue)
	} else {
		close(f.fileListQueue)
	}

	f.walkMutex.Lock()
	f.isWalking = false
	err := f.walkErr
	f.walkMutex.Unlock()

	return err
}

// readEnvGitExclude reads $GIT_DIR once for the whole walk and, when it is set,
// reads the exclude file it names once. Previously both happened in every
// directory: the environment lookup was repeated for a value that cannot change
// while walking, and the same absolute file was read again for each directory
// and then anchored at that directory, so a root anchored pattern such as
// /build applied at every level instead of only at the root. When $GIT_DIR is
// set it names one git directory for the whole walk, so the file it holds is
// read once here and anchored at each walk root by rootGitIgnores.
func (f *FileWalker) readEnvGitExclude() {
	f.gitDirFromEnv = false
	f.gitExcludeFromEnv = nil

	if f.IgnoreGitIgnore {
		return
	}

	gitdir := os.Getenv("GIT_DIR")
	if gitdir == "" {
		return
	}

	// record that it was set even when the file is missing, because $GIT_DIR
	// overrides looking for a .git entry while walking either way
	f.gitDirFromEnv = true
	if content, err := os.ReadFile(filepath.Join(gitdir, "info", "exclude")); err == nil {
		f.gitExcludeFromEnv = content
	}
}

// rootGitIgnores returns the seed gitignores for a walk root, which is the
// exclude file named by $GIT_DIR anchored at that root when there is one, and
// nothing otherwise.
func (f *FileWalker) rootGitIgnores(directory string) []gitignore.GitIgnore {
	if len(f.gitExcludeFromEnv) == 0 {
		return []gitignore.GitIgnore{}
	}

	abs, err := filepath.Abs(directory)
	if err != nil {
		return []gitignore.GitIgnore{}
	}

	return []gitignore.GitIgnore{gitignore.New(bytes.NewReader(f.gitExcludeFromEnv), abs, nil)}
}

// gitInfoExcludePath returns the path of the info/exclude file belonging to the
// .git entry found in a directory.
//
// A normal checkout has .git as a directory and the file simply sits inside it.
// A submodule, or a checkout made by git worktree add, instead has .git as a
// regular file holding a "gitdir: <path>" line naming the real git directory,
// and for a worktree that directory holds a commondir file naming the shared
// directory which owns info/ (see gitrepository-layout). Following both is what
// lets a worktree honour the exclude file of the repository it belongs to, the
// way git itself does.
//
// A symlinked .git also reports as not being a directory, and reading it as a
// file fails, so anything that does not parse falls back to joining onto the
// entry as before, which resolves the link exactly as it always has.
func gitInfoExcludePath(directory string, entry fs.DirEntry) string {
	gitdir := filepath.Join(directory, gitDirName)

	if !entry.IsDir() {
		if resolved := resolveGitDirFile(directory, gitdir); resolved != "" {
			gitdir = resolved
		}
	}

	return filepath.Join(gitdir, "info", "exclude")
}

// resolveGitDirFile reads a .git file and returns the git directory that owns
// info/, or an empty string if the file is not a readable "gitdir:" pointer.
func resolveGitDirFile(directory string, gitFile string) string {
	content, err := os.ReadFile(gitFile)
	if err != nil {
		return ""
	}

	line, _, _ := strings.Cut(string(content), "\n")
	if !strings.HasPrefix(line, "gitdir:") {
		return ""
	}

	gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if gitdir == "" {
		return ""
	}
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(directory, gitdir)
	}

	// a worktree git directory is per worktree, but info/ lives in the common
	// directory it points at, which is the git directory of the main checkout
	if content, err := os.ReadFile(filepath.Join(gitdir, "commondir")); err == nil {
		if common := strings.TrimSpace(string(content)); common != "" {
			if filepath.IsAbs(common) {
				gitdir = common
			} else {
				gitdir = filepath.Join(gitdir, common)
			}
		}
	}

	return gitdir
}

// buildGlobalIgnores reads each path in CustomIgnoreFiles, parses it as gitignore
// syntax and anchors it at the supplied walk root directory so that root-anchored
// patterns (such as /build) resolve relative to the root rather than at every
// subdirectory. They are appended in supplied order, so a later supplied file wins
// over an earlier one. The resulting slice is seeded as the lowest priority set of
// ignores, meaning any ignore file discovered while walking overrides them.
// Missing or unreadable files are passed through errorsHandler and skipped when it
// returns true, consistent with how the other ignore file reads behave.
func (f *FileWalker) buildGlobalIgnores(directory string) ([]gitignore.GitIgnore, error) {
	if len(f.CustomIgnoreFiles) == 0 {
		return []gitignore.GitIgnore{}, nil
	}

	abs, err := filepath.Abs(directory)
	if err != nil {
		if f.errorsHandler(err) {
			return []gitignore.GitIgnore{}, nil
		}
		return nil, err
	}

	globalIgnores := []gitignore.GitIgnore{}
	for _, ignoreFile := range f.CustomIgnoreFiles {
		c, err := f.osReadFile(ignoreFile)
		if err != nil {
			if f.errorsHandler(err) {
				continue // if asked to ignore it lets continue
			}
			return nil, err
		}

		gitIgnore := gitignore.New(bytes.NewReader(c), filepath.ToSlash(abs), nil)
		globalIgnores = append(globalIgnores, gitIgnore)
	}

	return globalIgnores, nil
}

// absoluteWalkRoot resolves a walk root to slash separated absolute form, once,
// so that every path tested against an ignore file below it can be built by
// concatenation rather than resolved individually.
//
// Matching needs an absolute path because an ignore file's base is absolute.
// Resolving one per path meant a filepath.Abs per file per ignore file in
// scope, which is a working directory lookup and a Clean apiece, and the cache
// that existed to blunt that cost a map lookup and a retained entry for every
// path walked. A root resolved once and extended by concatenation gives the
// same answer for free.
//
// A root that cannot be resolved is returned unchanged, which leaves the walk
// passing the relative path it always did and MatchIsDir resolving it the old
// way, so a failure here costs speed rather than correctness.
func absoluteWalkRoot(directory string) string {
	abs, err := filepath.Abs(directory)
	if err != nil {
		return directory
	}

	// A trailing separator would make the concatenation below produce "//name".
	// Abs only leaves one on a volume root, "/" or "C:\\", so this trims exactly
	// that case and turns it into the empty prefix the concatenation wants.
	return strings.TrimSuffix(filepath.ToSlash(abs), "/")
}

func (f *FileWalker) walkDirectoryRecursive(iteration int,
	directory string,
	absDirectory string,
	globalIgnores []gitignore.GitIgnore,
	gitignores []gitignore.GitIgnore,
	ignores []gitignore.GitIgnore,
	moduleIgnores []gitignore.GitIgnore,
	customIgnores []gitignore.GitIgnore) error {

	// implement max depth option
	if f.MaxDepth != -1 && iteration >= f.MaxDepth {
		return nil
	}

	// A single atomic load rather than taking the walk mutex, because this runs
	// once per directory on every walking goroutine and the mutex became a point
	// of contention once the walk forks below the top level. It is set by
	// Terminate and by any error the error handler refused to continue past, so
	// checking it here stops the walk promptly from any depth.
	if f.stopWalking.Load() {
		return f.firstError()
	}

	d, err := f.osOpen(directory)
	if err != nil {
		// nothing we can do with this so return nil and process as best we can
		if f.errorsHandler(err) {
			return nil
		}
		return f.stop(err)
	}
	defer func(d *os.File) {
		err := d.Close()
		if err != nil {
			f.errorsHandler(err)
		}
	}(d)

	foundFiles, err := d.ReadDir(-1)
	if err != nil {
		// nothing we can do with this so return nil and process as best we can
		if f.errorsHandler(err) {
			return nil
		}
		return f.stop(err)
	}

	files := []fs.DirEntry{}
	dirs := []fs.DirEntry{}
	// the .git entry if this directory has one, which marks it as a repository
	// root and is the only place an info/exclude file can be found. It is picked
	// up here because the listing is already in hand, so no extra syscall is
	// needed to know whether looking for that file is worth it at all
	var gitEntry fs.DirEntry

	// We want to break apart the files and directories from the
	// return as we loop over them differently and this avoids some
	// nested if logic at the expense of a "redundant" loop
	for _, file := range foundFiles {
		if file.Name() == gitDirName {
			gitEntry = file
		}

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
					return f.stop(err)
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return f.stop(err)
				}

				gitIgnore := gitignore.New(bytes.NewReader(c), filepath.ToSlash(abs), nil)
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
					return f.stop(err)
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return f.stop(err)
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
					return f.stop(err)
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return f.stop(err)
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
					return f.stop(err)
				}

				abs, err := filepath.Abs(directory)
				if err != nil {
					if f.errorsHandler(err) {
						continue // if asked to ignore it lets continue
					}
					return f.stop(err)
				}

				gitIgnore := gitignore.New(bytes.NewReader(c), abs, nil)
				customIgnores = append(customIgnores, gitIgnore)
			}
		}
	}
	// info/exclude only exists at a repository root, so blindly trying to read it
	// in every directory was one guaranteed failed open per directory on any large
	// tree, 6,052 of 6,053 of them on the linux kernel. The directory listing is
	// already in hand and already tells us whether this is a repository root, the
	// same way .gitignore is found above, so only look when there is a .git entry
	// to look inside. When $GIT_DIR is set it overrides the .git entry entirely,
	// as it always has, and has already been read once by readEnvGitExclude.
	if !f.IgnoreGitIgnore && !f.gitDirFromEnv && gitEntry != nil {
		if content, err := os.ReadFile(gitInfoExcludePath(directory, gitEntry)); err == nil {
			abs, err := filepath.Abs(directory)
			if err == nil {
				gitExclude := gitignore.New(bytes.NewReader(content), abs, nil)
				if gitExclude != nil {
					gitignores = append(gitignores, gitExclude)
				}
			}
		}
	}

	// If we have custom ignore patterns defined we should concatenate them and treat them as a single gitignore file
	if len(f.CustomIgnorePatterns) > 0 {
		customIgnorePatternsCombined := strings.Join(f.CustomIgnorePatterns, "\n")

		abs, err := filepath.Abs(directory)
		if err != nil {
			if !f.errorsHandler(err) {
				return f.stop(err)
			}
		}

		gitIgnore := gitignore.New(bytes.NewReader([]byte(customIgnorePatternsCombined)), abs, nil)
		customIgnores = append(customIgnores, gitIgnore)
	}

	// When a batch queue is in use the files this directory yields are collected
	// here and handed over in one channel operation at the end of the loop. See
	// SetFileBatchQueue for why that matters.
	var batch []*File

	// Process files first to start feeding whatever process is consuming
	// the output before traversing into directories for more files
	for _, file := range files {
		shouldIgnore := false
		var skipReason SkipReason
		joined := filepath.ToSlash(filepath.Join(directory, file.Name()))
		matchPath := joined
		if absDirectory != directory {
			matchPath = absDirectory + "/" + file.Name()
		}

		// Global ignore files supplied by path are the lowest priority, so they
		// are checked first and anything discovered while walking can override them
		for _, ignore := range globalIgnores {
			if m := ignore.MatchIsDir(matchPath, false); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonGlobalIgnore
				} else {
					skipReason = ""
				}
			}
		}

		for _, ignore := range gitignores {
			// we have the following situations
			// 1. none of the gitignores match
			// 2. one or more match
			// for #1 this means we should include the file
			// for #2 this means the last one wins since it should be the most correct
			if m := ignore.MatchIsDir(matchPath, false); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonGitignore
				} else {
					skipReason = ""
				}
			}
		}

		for _, ignore := range ignores {
			// same rules as above
			if m := ignore.MatchIsDir(matchPath, false); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonIgnoreFile
				} else {
					skipReason = ""
				}
			}
		}

		for _, ignore := range customIgnores {
			// same rules as above
			if m := ignore.MatchIsDir(matchPath, false); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonCustomIgnore
				} else {
					skipReason = ""
				}
			}
		}

		if len(f.IncludeFilename) != 0 {
			// include files
			shouldIgnore = !slices.ContainsFunc(f.IncludeFilename, func(allow string) bool {
				return file.Name() == allow
			})
			if shouldIgnore {
				skipReason = SkipReasonIncludeFilename
			} else {
				skipReason = ""
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeFilename {
			if file.Name() == deny {
				shouldIgnore = true
				skipReason = SkipReasonExcludeFilename
				break
			}
		}

		if len(f.IncludeFilenameRegex) != 0 {
			shouldIgnore = !slices.ContainsFunc(f.IncludeFilenameRegex, func(allow *regexp.Regexp) bool {
				return allow.MatchString(file.Name())
			})
			if shouldIgnore {
				skipReason = SkipReasonIncludeFilenameRegex
			} else {
				skipReason = ""
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeFilenameRegex {
			if deny.MatchString(file.Name()) {
				shouldIgnore = true
				skipReason = SkipReasonExcludeFilenameRegex
				break
			}
		}

		// Ignore hidden files
		if !f.IncludeHidden {
			s, err := IsHiddenDirEntry(file, directory)
			if err != nil {
				if !f.errorsHandler(err) {
					return f.stop(err)
				}
			}

			if s {
				shouldIgnore = true
				skipReason = SkipReasonHidden
			}
		}

		// Check against extensions
		if len(f.AllowListExtensions) != 0 {
			ext := GetExtension(file.Name())
			// try again because we could have one of those pesky ones such as something.spec.tsx
			// but only if we didn't already find something to save on a bit of processing
			if !slices.Contains(f.AllowListExtensions, ext) && !slices.Contains(f.AllowListExtensions, GetExtension(ext)) {
				shouldIgnore = true
				skipReason = SkipReasonAllowListExtension
			}
		}

		if len(f.ExcludeListExtensions) != 0 {
			ext := GetExtension(file.Name())
			shouldIgnore = slices.ContainsFunc(f.ExcludeListExtensions, func(deny string) bool {
				return ext == deny || GetExtension(ext) == deny
			})
			if shouldIgnore {
				skipReason = SkipReasonExcludeListExtension
			} else {
				skipReason = ""
			}
		}

		for _, p := range f.LocationExcludePattern {
			if strings.Contains(joined, p) {
				shouldIgnore = true
				skipReason = SkipReasonLocationExcludePattern
				break
			}
		}

		if f.IgnoreBinaryFiles {
			fi, err := os.Open(filepath.Join(directory, file.Name()))
			if err != nil {
				if !f.errorsHandler(err) {
					return f.stop(err)
				}
			}
			defer func(fi *os.File) {
				_ = fi.Close()
			}(fi)

			buffer := make([]byte, f.IgnoreBinaryFileBytes)

			// Read up to buffer size
			_, err = io.ReadFull(fi, buffer)
			if err != nil && err != io.EOF && !errors.Is(err, io.ErrUnexpectedEOF) {
				if !f.errorsHandler(err) {
					return f.stop(err)
				}
			}

			// cheaply check if is binary file by checking for null byte.
			// note that this could be improved later on by checking for magic numbers and the like
			// but that should probably be its own package
			for _, b := range buffer {
				if b == 0 {
					shouldIgnore = true
					skipReason = SkipReasonBinary
					break
				}
			}
		}

		if shouldIgnore {
			f.skipHandler(joined, file.Name(), false, skipReason)
		} else {
			if f.fileBatchQueue != nil {
				if batch == nil {
					// one backing array for the whole directory, since the
					// number of files that survive the filters above is usually
					// close to the number of entries it holds
					batch = make([]*File, 0, len(files))
				}
				batch = append(batch, &File{
					Location: joined,
					Filename: file.Name(),
				})
			} else {
				f.fileListQueue <- &File{
					Location: joined,
					Filename: file.Name(),
				}
			}
		}
	}

	// hand this directory's files over in a single channel operation
	if len(batch) != 0 {
		f.fileBatchQueue <- batch
	}

	// The ignore slices are about to be handed to subdirectories, which each
	// append their own discovered ignore files to them. Appending to a slice
	// that has spare capacity writes into the shared backing array, so two
	// sibling directories would otherwise append into the same slot: a data race
	// now that siblings run concurrently, and before that a silent correctness
	// bug where a directory could be matched against a sibling's rules rather
	// than only its ancestors'. Clipping sets cap to len, which forces every
	// child's append to allocate its own array. It is O(1) and only reslices.
	gitignores = slices.Clip(gitignores)
	ignores = slices.Clip(ignores)
	moduleIgnores = slices.Clip(moduleIgnores)
	customIgnores = slices.Clip(customIgnores)

	// Now we process the directories after hopefully giving the
	// channel some files to process
	for _, dir := range dirs {
		var shouldIgnore bool
		var skipReason SkipReason
		joined := filepath.ToSlash(filepath.Join(directory, dir.Name()))
		matchPath := joined
		if absDirectory != directory {
			matchPath = absDirectory + "/" + dir.Name()
		}

		// Check against the ignore files we have if the file we are looking at
		// should be ignored
		// It is safe to always call this because the gitignores will not be added
		// in previous steps
		for _, ignore := range globalIgnores {
			if m := ignore.MatchIsDir(matchPath, true); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonGlobalIgnore
				} else {
					skipReason = ""
				}
			}
		}
		for _, ignore := range gitignores {
			// we have the following situations
			// 1. none of the gitignores match
			// 2. one or more match
			// for #1 this means we should include the file
			// for #2 this means the last one wins since it should be the most correct
			if m := ignore.MatchIsDir(matchPath, true); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonGitignore
				} else {
					skipReason = ""
				}
			}
		}
		for _, ignore := range ignores {
			// same rules as above
			if m := ignore.MatchIsDir(matchPath, true); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonIgnoreFile
				} else {
					skipReason = ""
				}
			}
		}
		for _, ignore := range customIgnores {
			// same rules as above
			if m := ignore.MatchIsDir(matchPath, true); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonCustomIgnore
				} else {
					skipReason = ""
				}
			}
		}
		for _, ignore := range moduleIgnores {
			// same rules as above
			if m := ignore.MatchIsDir(matchPath, true); m != nil {
				shouldIgnore = m.Ignore()
				if shouldIgnore {
					skipReason = SkipReasonModuleIgnore
				} else {
					skipReason = ""
				}
			}
		}

		// start by saying we didn't find it then check each possible
		// choice to see if we did find it
		// if we didn't find it then we should ignore
		if len(f.IncludeDirectory) != 0 {
			shouldIgnore = !slices.ContainsFunc(f.IncludeDirectory, func(allow string) bool {
				return dir.Name() == allow
			})
			if shouldIgnore {
				skipReason = SkipReasonIncludeDirectory
			} else {
				skipReason = ""
			}
		}
		// Confirm if there are any files in the path deny list which usually includes
		// things like .git .hg and .svn
		// Comes after include as it takes precedence
		for _, deny := range f.ExcludeDirectory {
			if isSuffixDir(joined, deny) {
				shouldIgnore = true
				skipReason = SkipReasonExcludeDirectory
				break
			}
		}

		if len(f.IncludeDirectoryRegex) != 0 {
			shouldIgnore = !slices.ContainsFunc(f.IncludeDirectoryRegex, func(allow *regexp.Regexp) bool {
				return allow.MatchString(dir.Name())
			})
			if shouldIgnore {
				skipReason = SkipReasonIncludeDirectoryRegex
			} else {
				skipReason = ""
			}
		}
		// Exclude comes after include as it takes precedence
		for _, deny := range f.ExcludeDirectoryRegex {
			if deny.MatchString(dir.Name()) {
				shouldIgnore = true
				skipReason = SkipReasonExcludeDirectoryRegex
				break
			}
		}

		// Ignore hidden directories
		if !f.IncludeHidden {
			s, err := IsHiddenDirEntry(dir, directory)
			if err != nil {
				if !f.errorsHandler(err) {
					return f.stop(err)
				}
			}

			if s {
				shouldIgnore = true
				skipReason = SkipReasonHidden
			}
		}

		for _, p := range f.LocationExcludePattern {
			if strings.Contains(joined, p) {
				shouldIgnore = true
				skipReason = SkipReasonLocationExcludePattern
				break
			}
		}

		if shouldIgnore {
			f.skipHandler(joined, dir.Name(), true, skipReason)
		}

		if !shouldIgnore {
			// Walk this subdirectory on its own goroutine if the semaphore has a
			// slot going spare, otherwise walk it inline on this one. The take is
			// deliberately non-blocking: a goroutine that already holds a slot
			// never waits for another, so there is no way for the walk to deadlock
			// on itself, and the number of live walking goroutines can never
			// exceed the configured concurrency however deep or wide the tree is.
			// Contrast the old behaviour, which forked only at iteration 0 and so
			// walked everything below a root's immediate children serially.
			select {
			case f.countingSemaphore <- true:
				f.walkWg.Add(1)
				go func(joined, matchPath string, gitignores, ignores, moduleIgnores, customIgnores []gitignore.GitIgnore) {
					defer f.walkWg.Done()
					defer func() { <-f.countingSemaphore }()
					// the error is recorded by stop rather than returned, since
					// there is nowhere to return it to from here
					_ = f.walkDirectoryRecursive(iteration+1, joined, matchPath, globalIgnores, gitignores, ignores, moduleIgnores, customIgnores)
				}(joined, matchPath, gitignores, ignores, moduleIgnores, customIgnores)
			default:
				if err := f.walkDirectoryRecursive(iteration+1, joined, matchPath, globalIgnores, gitignores, ignores, moduleIgnores, customIgnores); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// FindRepositoryRoot given the supplied directory walks backwards looking for a
// .git or .hg entry indicating we should start our search from that location as
// it's the root.
// Returns the first matching ancestor (inclusive of startDirectory) otherwise
// the supplied directory.
// This recognizes git worktrees and submodules, where .git is a regular file
// (containing "gitdir: …") rather than a directory — so a nested worktree
// resolves to its own root instead of the enclosing main repo.
func FindRepositoryRoot(startDirectory string) string {
	// Firstly try to determine our real location so the upward walk is
	// anchored to an absolute path
	abs, err := filepath.Abs(startDirectory)
	if err != nil {
		return startDirectory
	}

	// Walk the file tree backwards in a cross platform way and if we find
	// a match we return that
	dir := abs
	for {
		if checkForGitOrMercurial(dir) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// We didn't find a good match so return the supplied directory
			// so that we start the search from where we started at least
			// rather than the root
			return startDirectory
		}
		dir = parent
	}
}

// Check if there is a .git or .hg entry in the supplied directory.
// .git is accepted as either a directory (normal repo) or a regular file
// (git worktree, submodule).
func checkForGitOrMercurial(curdir string) bool {
	if stat, err := os.Stat(filepath.Join(curdir, ".git")); err == nil {
		if stat.IsDir() || stat.Mode().IsRegular() {
			return true
		}
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
