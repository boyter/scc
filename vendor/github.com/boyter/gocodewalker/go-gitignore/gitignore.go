// SPDX-License-Identifier: MIT

package gitignore

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// use an empty GitIgnore for cached lookups
var empty = &ignore{}

// GitIgnore is the interface to .gitignore files and repositories. It defines
// methods for testing files for matching the .gitignore file, and then
// determining whether a file should be ignored or included.
type GitIgnore interface {
	// Base returns the directory containing the .gitignore file.
	Base() string

	// Match attempts to match the path against this GitIgnore, and will
	// return its Match if successful. Match will invoke the GitIgnore error
	// handler (if defined) if it is not possible to determine the absolute
	// path of the given path, or if its not possible to determine if the
	// path represents a file or a directory. If an error occurs, Match
	// returns nil and the error handler (if defined via New, NewWithErrors
	// or NewWithCache) will be invoked.
	Match(path string) Match

	MatchIsDir(path string, _isdir bool) Match

	// Absolute attempts to match an absolute path against this GitIgnore. If
	// the path is not located under the base directory of this GitIgnore, or
	// is not matched by this GitIgnore, nil is returned.
	Absolute(string, bool) Match

	// Relative attempts to match a path relative to the GitIgnore base
	// directory. isdir is used to indicate whether the path represents a file
	// or a directory. If the path is not matched by the GitIgnore, nil is
	// returned.
	Relative(path string, isdir bool) Match

	// Ignore returns true if the path is ignored by this GitIgnore. Paths
	// that are not matched by this GitIgnore are not ignored. Internally,
	// Ignore uses Match, and will return false if Match() returns nil for path.
	Ignore(path string) bool

	// Include returns true if the path is included by this GitIgnore. Paths
	// that are not matched by this GitIgnore are always included. Internally,
	// Include uses Match, and will return true if Match() returns nil for path.
	Include(path string) bool
}

// ignore is the implementation of a .gitignore file.
type ignore struct {
	_base    string
	_pattern []Pattern
	_errors  func(Error) bool
}

// NewGitIgnore creates a new GitIgnore instance from the patterns listed in t,
// representing a .gitignore file in the base directory. If errors is given, it
// will be invoked for every error encountered when parsing the .gitignore
// patterns. Parsing will terminate if errors is called and returns false,
// otherwise, parsing will continue until end of file has been reached.
func New(r io.Reader, base string, errors func(Error) bool) GitIgnore {
	// do we have an error handler?
	_errors := errors
	if _errors == nil {
		_errors = func(e Error) bool { return true }
	}

	// extract the patterns from the reader
	_parser := NewParser(r, _errors)
	_patterns := _parser.Parse()

	return &ignore{_base: base, _pattern: _patterns, _errors: _errors}
} // New()

// NewFromFile creates a GitIgnore instance from the given file. An error
// will be returned if file cannot be opened or its absolute path determined.
func NewFromFile(file string) (GitIgnore, error) {
	// define an error handler to catch any file access errors
	//		- record the first encountered error
	var _error Error
	_errors := func(e Error) bool {
		if _error == nil {
			_error = e
		}
		return true
	}

	// attempt to retrieve the GitIgnore represented by this file
	_ignore := NewWithErrors(file, _errors)

	// did we encounter an error?
	//		- if the error has a zero Position then it was encountered
	//		  before parsing was attempted, so we return that error
	if _error != nil {
		if _error.Position().Zero() {
			return nil, _error.Underlying()
		}
	}

	// otherwise, we ignore the parser errors
	return _ignore, nil
} // NewFromFile()

// NewWithErrors creates a GitIgnore instance from the given file.
// If errors is given, it will be invoked for every error encountered when
// parsing the .gitignore patterns. Parsing will terminate if errors is called
// and returns false, otherwise, parsing will continue until end of file has
// been reached. NewWithErrors returns nil if the .gitignore could not be read.
func NewWithErrors(file string, errors func(Error) bool) GitIgnore {
	var _err error

	// do we have an error handler?
	_file := file
	_errors := errors
	if _errors == nil {
		_errors = func(e Error) bool { return true }
	} else {
		// augment the error handler to include the .gitignore file name
		//		- we do this here since the parser and lexer interfaces are
		//		  not aware of file names
		_errors = func(e Error) bool {
			// augment the position with the file name
			_position := e.Position()
			_position.File = _file

			// create a new error with the updated Position
			_error := NewError(e.Underlying(), _position)

			// invoke the original error handler
			return errors(_error)
		}
	}

	// we need the absolute path for the GitIgnore base
	_file, _err = filepath.Abs(file)
	if _err != nil {
		_errors(NewError(_err, Position{}))
		return nil
	}
	_base := filepath.Dir(_file)

	// attempt to open the ignore file to create the io.Reader
	_fh, _err := os.Open(_file)
	if _err != nil {
		_errors(NewError(_err, Position{}))
		return nil
	}

	// return the GitIgnore instance
	return New(_fh, _base, _errors)
} // NewWithErrors()

// NewWithCache returns a GitIgnore instance (using NewWithErrors)
// for the given file. If the file has been loaded before, its GitIgnore
// instance will be returned from the cache rather than being reloaded. If
// cache is not defined, NewWithCache will behave as NewWithErrors
//
// If NewWithErrors returns nil, NewWithCache will store an empty
// GitIgnore (i.e. no patterns) against the file to prevent repeated parse
// attempts on subsequent requests for the same file. Subsequent calls to
// NewWithCache for a file that could not be loaded due to an error will
// return nil.
//
// If errors is given, it will be invoked for every error encountered when
// parsing the .gitignore patterns. Parsing will terminate if errors is called
// and returns false, otherwise, parsing will continue until end of file has
// been reached.
func NewWithCache(file string, cache Cache, errors func(Error) bool) GitIgnore {
	// do we have an error handler?
	_errors := errors
	if _errors == nil {
		_errors = func(e Error) bool { return true }
	}

	// use the file absolute path as its key into the cache
	_abs, _err := filepath.Abs(file)
	if _err != nil {
		_errors(NewError(_err, Position{}))
		return nil
	}

	var _ignore GitIgnore
	if cache != nil {
		_ignore = cache.Get(_abs)
	}
	if _ignore == nil {
		_ignore = NewWithErrors(file, _errors)
		if _ignore == nil {
			// if the load failed, cache an empty GitIgnore to prevent
			// further attempts to load this file
			_ignore = empty
		}
		if cache != nil {
			cache.Set(_abs, _ignore)
		}
	}

	// return the ignore (if we have it)
	if _ignore == empty {
		return nil
	} else {
		return _ignore
	}
} // NewWithCache()

// Base returns the directory containing the .gitignore file for this GitIgnore.
func (i *ignore) Base() string {
	return i._base
} // Base()

// Match attempts to match the path against this GitIgnore, and will
// return its Match if successful. Match will invoke the GitIgnore error
// handler (if defined) if it is not possible to determine the absolute
// path of the given path, or if its not possible to determine if the
// path represents a file or a directory. If an error occurs, Match
// returns nil and the error handler (if defined via New, NewWithErrors
// or NewWithCache) will be invoked.
func (i *ignore) Match(path string) Match {
	// ensure we have the absolute path for the given file
	_path, _err := filepath.Abs(path)
	if _err != nil {
		i._errors(NewError(_err, Position{}))
		return nil
	}

	// is the path a file or a directory?
	_info, _err := os.Stat(_path)
	if _err != nil {
		i._errors(NewError(_err, Position{}))
		return nil
	}
	_isdir := _info.IsDir()

	// attempt to match the absolute path
	return i.Absolute(_path, _isdir)
} // Match()

func (i *ignore) MatchIsDir(path string, _isdir bool) Match {
	// ensure we have the absolute path for the given file
	_path, _err := filepath.Abs(path)
	if _err != nil {
		i._errors(NewError(_err, Position{}))
		return nil
	}

	// attempt to match the absolute path
	return i.Absolute(_path, _isdir)
} // Match()

// Absolute attempts to match an absolute path against this GitIgnore. If
// the path is not located under the base directory of this GitIgnore, or
// is not matched by this GitIgnore, nil is returned.
func (i *ignore) Absolute(path string, isdir bool) Match {
	// does the file share the same directory as this ignore file?
	if !strings.HasPrefix(path, i._base) {
		return nil
	}

	// extract the relative path of this file
	_prefix := len(i._base) + 1 // BOYTERWASHERE
	//_prefix := len(i._base)
	_rel := string(path[_prefix:])
	return i.Relative(_rel, isdir)
} // Absolute()

// Relative attempts to match a path relative to the GitIgnore base
// directory. isdir is used to indicate whether the path represents a file
// or a directory. If the path is not matched by the GitIgnore, nil is
// returned.
func (i *ignore) Relative(path string, isdir bool) Match {
	// if we are on Windows, then translate the path to Unix form
	_rel := path
	if runtime.GOOS == "windows" {
		_rel = filepath.ToSlash(_rel)
	}

	// iterate over the patterns for this ignore file
	//      - iterate in reverse, since later patterns overwrite earlier
	for _i := len(i._pattern) - 1; _i >= 0; _i-- {
		_pattern := i._pattern[_i]
		if _pattern.Match(_rel, isdir) {
			return _pattern
		}
	}

	// we don't match this file
	return nil
} // Relative()

// Ignore returns true if the path is ignored by this GitIgnore. Paths
// that are not matched by this GitIgnore are not ignored. Internally,
// Ignore uses Match, and will return false if Match() returns nil for path.
func (i *ignore) Ignore(path string) bool {
	_match := i.Match(path)
	if _match != nil {
		return _match.Ignore()
	}

	// we didn't match this path, so we don't ignore it
	return false
} // Ignore()

// Include returns true if the path is included by this GitIgnore. Paths
// that are not matched by this GitIgnore are always included. Internally,
// Include uses Match, and will return true if Match() returns nil for path.
func (i *ignore) Include(path string) bool {
	_match := i.Match(path)
	if _match != nil {
		return _match.Include()
	}

	// we didn't match this path, so we include it
	return true
} // Include()

// ensure Ignore satisfies the GitIgnore interface
var _ GitIgnore = &ignore{}
