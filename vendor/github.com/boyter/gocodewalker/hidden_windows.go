// SPDX-License-Identifier: MIT
//go:build windows
// +build windows

package gocodewalker

import (
	"io/fs"
	"os"
	"path"
	"syscall"
)

// IsHidden Returns true if file is hidden
func IsHidden(file os.FileInfo, directory string) (bool, error) {
	return IsHiddenDirEntry(fs.FileInfoToDirEntry(file), directory)
}

// IsHiddenDirEntry is similar to [IsHidden], excepts it accepts [fs.DirEntry] as its argument
func IsHiddenDirEntry(file fs.DirEntry, directory string) (bool, error) {
	fullpath := path.Join(directory, file.Name())
	pointer, err := syscall.UTF16PtrFromString(fullpath)
	if err != nil {
		return false, err
	}
	attributes, err := syscall.GetFileAttributes(pointer)
	if err != nil {
		return false, err
	}
	return attributes&syscall.FILE_ATTRIBUTE_HIDDEN != 0, nil
}
