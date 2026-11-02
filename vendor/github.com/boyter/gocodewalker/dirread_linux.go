// SPDX-License-Identifier: MIT
//go:build linux

package gocodewalker

import (
	"bytes"
	"encoding/binary"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"syscall"
)

// Listing a directory through os.Open costs more than the openat it needs. The
// runtime tries to hand every descriptor it opens to the netpoller, which for a
// directory always fails with EPERM, so each directory paid four fcntl calls
// and a pair of epoll_ctl calls purely to discover that a directory cannot be
// polled. Opening the directory with the raw open syscall and reading it with
// getdents64 skips all of that, which on a large tree removes tens of thousands
// of syscalls that never did anything.
//
// The entries this produces are the same as (*os.File).ReadDir(-1): the same
// order, the same treatment of "." and "..", the same lstat fallback when the
// filesystem does not report a type, and symlinks reported as symlinks rather
// than as whatever they point at.

// The same buffer size the standard library reads directories with. A larger
// buffer does mean fewer getdents64 calls in a directory big enough to need
// more than one, but it was measured and it loses: at 64 KiB the walk allocated
// 70% more and ran 8% slower than the standard library on a source tree,
// because a pool is emptied by every garbage collection and the buffer is then
// allocated and zeroed again, and almost every directory in a real tree is
// answered by a single call whatever the size. Buffers are pooled rather than
// allocated per directory since the walk runs several goroutines at once.
const direntBufferSize = 8 * 1024

var direntBufferPool = sync.Pool{
	New: func() any {
		b := make([]byte, direntBufferSize)
		return &b
	},
}

// Field offsets of struct linux_dirent64, which are the same on every Linux
// port: d_ino (8), d_off (8), d_reclen (2), d_type (1), then the name.
const (
	direntOffsetReclen = 16
	direntOffsetType   = 18
	direntOffsetName   = 19
)

// readDirectory lists a directory, returning its entries in the order the
// filesystem reports them.
func (f *FileWalker) readDirectory(directory string) ([]fs.DirEntry, error) {
	if !f.useRawDirents {
		return f.readDirectoryPortable(directory)
	}

	return f.readDirectoryGetdents(directory)
}

// rawDirentsUsable reports whether this walk can list directories with
// getdents64 directly. It cannot when the open hook has been replaced, because
// then the caller wants their own opener used.
func rawDirentsUsable(open func(name string) (*os.File, error)) bool {
	return open != nil && reflect.ValueOf(open).Pointer() == reflect.ValueOf(os.Open).Pointer()
}

func (f *FileWalker) readDirectoryGetdents(directory string) ([]fs.DirEntry, error) {
	fd, err := syscall.Open(directory, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_CLOEXEC, 0)
	if err != nil {
		// the same error the standard library would have returned, so callers
		// cannot tell which of the two listed the directory
		return nil, &os.PathError{Op: "open", Path: directory, Err: err}
	}
	defer func() {
		if err := syscall.Close(fd); err != nil {
			f.errorsHandler(&os.PathError{Op: "close", Path: directory, Err: err})
		}
	}()

	bufp := direntBufferPool.Get().(*[]byte)
	defer direntBufferPool.Put(bufp)
	buf := *bufp

	var entries []fs.DirEntry
	for {
		n, err := syscall.Getdents(fd, buf)
		if err != nil {
			return entries, &os.PathError{Op: "readdirent", Path: directory, Err: err}
		}
		if n <= 0 {
			return entries, nil
		}

		for offset := 0; offset+direntOffsetName <= n; {
			// the record length is written by the kernel in native byte order
			reclen := int(binary.NativeEndian.Uint16(buf[offset+direntOffsetReclen:]))
			if reclen < direntOffsetName || offset+reclen > n {
				// a record that does not fit what the kernel said it wrote, so
				// there is nothing sensible left to read out of this buffer
				break
			}

			record := buf[offset : offset+reclen]
			offset += reclen

			// a zero inode marks an entry deleted while the directory was open
			if binary.NativeEndian.Uint64(record) == 0 {
				continue
			}

			name := record[direntOffsetName:]
			if end := bytes.IndexByte(name, 0); end >= 0 {
				name = name[:end]
			}
			if len(name) == 0 || isDotOrDotDot(name) {
				continue
			}

			mode, known := direntFileMode(record[direntOffsetType])
			if !known {
				// some filesystems answer DT_UNKNOWN and leave the type to be
				// asked for, so ask, exactly as the standard library does
				resolved, exists, err := lstatFileMode(directory, string(name))
				if err != nil {
					return entries, err
				}
				if !exists {
					// vanished between being listed and being asked about
					continue
				}
				mode = resolved
			}

			entries = append(entries, &dirent{parent: directory, name: string(name), mode: mode})
		}
	}
}

// lstatFileMode asks the filesystem what an entry is, for the filesystems that
// answer DT_UNKNOWN rather than saying so up front. It reports exists as false
// when the entry went away between being listed and being asked about, which is
// not an error, just a directory that changed under the walk.
func lstatFileMode(directory, name string) (fs.FileMode, bool, error) {
	info, err := os.Lstat(filepath.Join(directory, name))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, false, nil
		}

		return 0, false, err
	}

	return info.Mode().Type(), true, nil
}

func isDotOrDotDot(name []byte) bool {
	if name[0] != '.' {
		return false
	}

	return len(name) == 1 || (len(name) == 2 && name[1] == '.')
}

// direntFileMode turns a d_type into the mode bits fs.DirEntry reports,
// returning false when the filesystem did not say what the entry is.
func direntFileMode(dtype uint8) (fs.FileMode, bool) {
	switch dtype {
	case syscall.DT_REG:
		return 0, true
	case syscall.DT_DIR:
		return fs.ModeDir, true
	case syscall.DT_LNK:
		return fs.ModeSymlink, true
	case syscall.DT_BLK:
		return fs.ModeDevice, true
	case syscall.DT_CHR:
		return fs.ModeDevice | fs.ModeCharDevice, true
	case syscall.DT_FIFO:
		return fs.ModeNamedPipe, true
	case syscall.DT_SOCK:
		return fs.ModeSocket, true
	}

	return 0, false
}

// dirent is the fs.DirEntry a getdents64 record yields. It holds the directory
// it came from so that Info can answer the way the standard library does, by
// lstat of the entry itself rather than of whatever a symlink points at.
type dirent struct {
	parent string
	name   string
	mode   fs.FileMode
}

func (d *dirent) Name() string      { return d.name }
func (d *dirent) IsDir() bool       { return d.mode.IsDir() }
func (d *dirent) Type() fs.FileMode { return d.mode }
func (d *dirent) Info() (fs.FileInfo, error) {
	return os.Lstat(filepath.Join(d.parent, d.name))
}
