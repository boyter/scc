// SPDX-License-Identifier: MIT
//go:build !windows
// +build !windows

package gocodewalker

import (
	"io/fs"
	"os"
)

// IsHidden Returns true if file is hidden
func IsHidden(file os.FileInfo, directory string) (bool, error) {
	return IsHiddenDirEntry(fs.FileInfoToDirEntry(file), directory)
}

// IsHiddenDirEntry is similar to [IsHidden], excepts it accepts [fs.DirEntry] as its argument
func IsHiddenDirEntry(file fs.DirEntry, directory string) (bool, error) {
	return file.Name()[0:1] == ".", nil
}
