// SPDX-License-Identifier: MIT
//go:build !linux

package gocodewalker

import (
	"io/fs"
	"os"
)

// readDirectory lists a directory, returning its entries in the order the
// filesystem reports them.
func (f *FileWalker) readDirectory(directory string) ([]fs.DirEntry, error) {
	return f.readDirectoryPortable(directory)
}

// rawDirentsUsable is only ever true on Linux, where directories are listed
// with getdents64 directly.
func rawDirentsUsable(_ func(name string) (*os.File, error)) bool {
	return false
}
