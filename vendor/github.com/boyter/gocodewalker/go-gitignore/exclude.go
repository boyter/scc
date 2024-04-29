// SPDX-License-Identifier: MIT

package gitignore

import (
	"os"
	"path/filepath"
)

// exclude attempts to return the GitIgnore instance for the
// $GIT_DIR/info/exclude from the working copy to which path belongs.
func exclude(path string) (GitIgnore, error) {
	// attempt to locate GIT_DIR
	_gitdir := os.Getenv("GIT_DIR")
	if _gitdir == "" {
		_gitdir = filepath.Join(path, ".git")
	}
	_info, _err := os.Stat(_gitdir)
	if _err != nil {
		if os.IsNotExist(_err) {
			return nil, nil
		} else {
			return nil, _err
		}
	} else if !_info.IsDir() {
		return nil, nil
	}

	// is there an info/exclude file within this directory?
	_file := filepath.Join(_gitdir, "info", "exclude")
	_, _err = os.Stat(_file)
	if _err != nil {
		if os.IsNotExist(_err) {
			return nil, nil
		} else {
			return nil, _err
		}
	}

	// attempt to load the exclude file
	return NewFromFile(_file)
} // exclude()
