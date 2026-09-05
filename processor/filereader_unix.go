// SPDX-License-Identifier: MIT

//go:build unix

package processor

import (
	"fmt"
	"syscall"
)

// readFile opens and reads with the system calls rather than through os.Open.
//
// os.Open hands the descriptor to the runtime poller, which for a regular file
// means an epoll_ctl that always fails with EPERM and a handful of fcntl calls
// to put the descriptor back into blocking mode. None of it does anything for a
// file being read start to finish, and over a tree the size of the Linux kernel
// it came to 344,000 fcntl and 86,000 failing epoll_ctl calls, roughly five
// wasted system calls for every file counted. Reading the same 671MB of C with
// the calls below rather than os.Open takes 25ms where it took 33ms.
//
// size is what the caller's stat said the file holds. It is only ever a hint for
// how large a buffer to take: a short read does not establish the end of a file,
// on a filesystem that chops reads up or on one whose size was a lie, so the
// loop reads until it is told zero. Stopping at the size instead saved half the
// read calls, 179,668 down to 90,247 over the Linux kernel, and about five
// milliseconds of system time in a five second run. That is not worth a count
// that depends on the filesystem underneath it.
//
// Windows and anything else keeps os.Open, in filereader_other.go.
func (reader *FileReader) readFileInto(path string, buf []byte, size int) ([]byte, error) {
	// O_CLOEXEC because os.Open sets it and this replaced os.Open. Without it a
	// descriptor held while a file is counted would survive into any child
	// process started from another goroutine.
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", path, err)
	}
	defer func() {
		_ = syscall.Close(fd)
	}()

	total := 0
	for {
		if total == len(buf) {
			// the file grew since it was sized, or the size was a lie
			buf = append(buf, 0)
			buf = buf[:cap(buf)]
		}

		n, err := syscall.Read(fd, buf[total:])
		if n > 0 {
			total += n

			continue
		}
		if err == syscall.EINTR {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("error reading %s: %w", path, err)
		}

		break
	}

	return buf[:total], nil
}
