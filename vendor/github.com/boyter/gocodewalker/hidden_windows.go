// SPDX-License-Identifier: MIT OR Unlicense
//go:build windows
// +build windows

package gocodewalker

import (
	"os"
	"path"
	"syscall"
)

// IsHidden Returns true if file is hidden
func IsHidden(file os.FileInfo, directory string) (bool, error) {
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
