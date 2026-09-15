// SPDX-License-Identifier: MIT

package gocodewalker

import (
	"io/fs"
	"os"
)

// readDirectoryPortable lists a directory through the standard library. It is
// what every platform other than Linux uses, and what Linux itself falls back
// to when the open hook has been replaced.
func (f *FileWalker) readDirectoryPortable(directory string) ([]fs.DirEntry, error) {
	d, err := f.osOpen(directory)
	if err != nil {
		return nil, err
	}
	defer func(d *os.File) {
		if err := d.Close(); err != nil {
			f.errorsHandler(err)
		}
	}(d)

	return d.ReadDir(-1)
}
