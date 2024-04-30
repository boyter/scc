// SPDX-License-Identifier: MIT

package gitignore

import (
	"os"
	"path/filepath"
	"strings"
)

const File = ".gitignore"

// repository is the implementation of the set of .gitignore files within a
// repository hierarchy
type repository struct {
	ignore
	_errors  func(e Error) bool
	_cache   Cache
	_file    string
	_exclude GitIgnore
} // repository{}

// NewRepository returns a GitIgnore instance representing a git repository
// with root directory base. If base is not a directory, or base cannot be
// read, NewRepository will return an error.
//
// Internally, NewRepository uses NewRepositoryWithFile.
func NewRepository(base string) (GitIgnore, error) {
	return NewRepositoryWithFile(base, File)
} // NewRepository()

// NewRepositoryWithFile returns a GitIgnore instance representing a git
// repository with root directory base. The repository will use file as
// the name of the files within the repository from which to load the
// .gitignore patterns. If file is the empty string, NewRepositoryWithFile
// uses ".gitignore". If the ignore file name is ".gitignore", the returned
// GitIgnore instance will also consider patterns listed in
// $GIT_DIR/info/exclude when performing repository matching.
//
// Internally, NewRepositoryWithFile uses NewRepositoryWithErrors.
func NewRepositoryWithFile(base, file string) (GitIgnore, error) {
	// define an error handler to catch any file access errors
	//		- record the first encountered error
	var _error Error
	_errors := func(e Error) bool {
		if _error == nil {
			_error = e
		}
		return true
	}

	// attempt to retrieve the repository represented by this file
	_repository := NewRepositoryWithErrors(base, file, _errors)

	// did we encounter an error?
	//		- if the error has a zero Position then it was encountered
	//		  before parsing was attempted, so we return that error
	if _error != nil {
		if _error.Position().Zero() {
			return nil, _error.Underlying()
		}
	}

	// otherwise, we ignore the parser errors
	return _repository, nil
} // NewRepositoryWithFile()

// NewRepositoryWithErrors returns a GitIgnore instance representing a git
// repository with a root directory base. As with NewRepositoryWithFile, file
// specifies the name of the files within the repository containing the
// .gitignore patterns, and defaults to ".gitignore" if file is not specified.
// If the ignore file name is ".gitignore", the returned GitIgnore instance
// will also consider patterns listed in $GIT_DIR/info/exclude when performing
// repository matching.
//
// If errors is given, it will be invoked for each error encountered while
// matching a path against the repository GitIgnore (such as file permission
// denied, or errors during .gitignore parsing). See Match below.
//
// Internally, NewRepositoryWithErrors uses NewRepositoryWithCache.
func NewRepositoryWithErrors(base, file string, errors func(e Error) bool) GitIgnore {
	return NewRepositoryWithCache(base, file, NewCache(), errors)
} // NewRepositoryWithErrors()

// NewRepositoryWithCache returns a GitIgnore instance representing a git
// repository with a root directory base. As with NewRepositoryWithErrors,
// file specifies the name of the files within the repository containing the
// .gitignore patterns, and defaults to ".gitignore" if file is not specified.
// If the ignore file name is ".gitignore", the returned GitIgnore instance
// will also consider patterns listed in $GIT_DIR/info/exclude when performing
// repository matching.
//
// NewRepositoryWithCache will attempt to load each .gitignore within the
// repository only once, using NewWithCache to store the corresponding
// GitIgnore instance in cache. If cache is given as nil,
// NewRepositoryWithCache will create a Cache instance for this repository.
//
// If errors is given, it will be invoked for each error encountered while
// matching a path against the repository GitIgnore (such as file permission
// denied, or errors during .gitignore parsing). See Match below.
func NewRepositoryWithCache(base, file string, cache Cache, errors func(e Error) bool) GitIgnore {
	// do we have an error handler?
	_errors := errors
	if _errors == nil {
		_errors = func(e Error) bool { return true }
	}

	// extract the absolute path of the base directory
	_base, _err := filepath.Abs(base)
	if _err != nil {
		_errors(NewError(_err, Position{}))
		return nil
	}

	// ensure the given base is a directory
	_info, _err := os.Stat(_base)
	if _info != nil {
		if !_info.IsDir() {
			_err = InvalidDirectoryError
		}
	}
	if _err != nil {
		_errors(NewError(_err, Position{}))
		return nil
	}

	// if we haven't been given a base file name, use the default
	if file == "" {
		file = File
	}

	// are we matching .gitignore files?
	//		- if we are, we also consider $GIT_DIR/info/exclude
	var _exclude GitIgnore
	if file == File {
		_exclude, _err = exclude(_base)
		if _err != nil {
			_errors(NewError(_err, Position{}))
			return nil
		}
	}

	// create the repository instance
	_ignore := ignore{_base: _base}
	_repository := &repository{
		ignore:   _ignore,
		_errors:  _errors,
		_exclude: _exclude,
		_cache:   cache,
		_file:    file,
	}

	return _repository
} // NewRepositoryWithCache()

// Match attempts to match the path against this repository. Matching proceeds
// according to normal gitignore rules, where .gtignore files in the same
// directory as path, take precedence over .gitignore files higher up the
// path hierarchy, and child files and directories are ignored if the parent
// is ignored. If the path is matched by a gitignore pattern in the repository,
// a Match is returned detailing the matched pattern. The returned Match
// can be used to determine if the path should be ignored or included according
// to the repository.
//
// If an error is encountered during matching, the repository error handler
// (if configured via NewRepositoryWithErrors or NewRepositoryWithCache), will
// be called. If the error handler returns false, matching will terminate and
// Match will return nil. If handler returns true, Match will continue
// processing in an attempt to match path.
//
// Match will raise an error and return nil if the absolute path cannot be
// determined, or if its not possible to determine if path represents a file
// or a directory.
//
// If path is not located under the root of this repository, Match returns nil.
func (r *repository) Match(path string) Match {
	// ensure we have the absolute path for the given file
	_path, _err := filepath.Abs(path)
	if _err != nil {
		r._errors(NewError(_err, Position{}))
		return nil
	}

	// is the path a file or a directory?
	_info, _err := os.Stat(_path)
	if _err != nil {
		r._errors(NewError(_err, Position{}))
		return nil
	}
	_isdir := _info.IsDir()

	// attempt to match the absolute path
	return r.Absolute(_path, _isdir)
} // Match()

// Absolute attempts to match an absolute path against this repository. If the
// path is not located under the base directory of this repository, or is not
// matched by this repository, nil is returned.
func (r *repository) Absolute(path string, isdir bool) Match {
	// does the file share the same directory as this ignore file?
	if !strings.HasPrefix(path, r.Base()) {
		return nil
	}

	// extract the relative path of this file
	_prefix := len(r.Base()) + 1
	_rel := string(path[_prefix:])
	return r.Relative(_rel, isdir)
} // Absolute()

// Relative attempts to match a path relative to the repository base directory.
// If the path is not matched by the repository, nil is returned.
func (r *repository) Relative(path string, isdir bool) Match {
	// if there's no path, then there's nothing to match
	_path := filepath.Clean(path)
	if _path == "." {
		return nil
	}

	// repository matching:
	//		- a child path cannot be considered if its parent is ignored
	//		- a .gitignore in a lower directory overrides a .gitignore in a
	//		  higher directory

	// first, is the parent directory ignored?
	//		- extract the parent directory from the current path
	_parent, _local := filepath.Split(_path)
	_match := r.Relative(_parent, true)
	if _match != nil {
		if _match.Ignore() {
			return _match
		}
	}
	_parent = filepath.Clean(_parent)

	// the parent directory isn't ignored, so we now look at the original path
	//		- we consider .gitignore files in the current directory first, then
	//		  move up the path hierarchy
	var _last string
	for {
		_file := filepath.Join(r._base, _parent, r._file)
		_ignore := NewWithCache(_file, r._cache, r._errors)
		if _ignore != nil {
			_match := _ignore.Relative(_local, isdir)
			if _match != nil {
				return _match
			}
		}

		// if there's no parent, then we're done
		//		- since we use filepath.Clean() we look for "."
		if _parent == "." {
			break
		}

		// we don't have a match for this file, so we progress up the
		// path hierarchy
		//		- we are manually building _local using the .gitignore
		//		  separator "/", which is how we handle operating system
		//		  file system differences
		_parent, _last = filepath.Split(_parent)
		_parent = filepath.Clean(_parent)
		_local = _last + string(_SEPARATOR) + _local
	}

	// do we have a global exclude file? (i.e. GIT_DIR/info/exclude)
	if r._exclude != nil {
		return r._exclude.Relative(path, isdir)
	}

	// we have no match
	return nil
} // Relative()

// ensure repository satisfies the GitIgnore interface
var _ GitIgnore = &repository{}
