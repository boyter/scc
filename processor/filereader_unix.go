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
// size is what the caller's stat said the file holds, and the loop uses it to
// decide when a short read is the end of the file.
//
// A short read on its own proves nothing: a filesystem may chop a read at any
// point, so stopping at the first one would make the count depend on the
// filesystem underneath. What does prove it is the FIRST read coming up short
// of the buffer while still delivering everything the stat promised, because a
// file read whole in one call has plainly ended. Anything after that first read
// is read until a zero-length read says stop, since a file arriving in pieces
// is a file whose pieces say nothing about where it ends.
//
// Reading every file that way cost a second read on all of them, 127,454
// against mezura's 63,918 over the Linux kernel, which is a syscall per file to
// cover a case the first-read rule already covers.
//
// A stat size of zero is not a promise of anything, so it buys no shortcut and
// the loop reads until it is told zero. That is /proc and sysfs, which report
// nothing and hand back a page at a time; TestReadFileSyntheticFile reads
// /proc/kallsyms and would truncate at the first page without it.
//
// The approach is mezura's, from the syscall table in boyter/scc#769.
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
	first := true
	for {
		if total == len(buf) {
			// the file grew since it was sized, or the size was a lie
			buf = append(buf, 0)
			buf = buf[:cap(buf)]
		}

		wanted := len(buf) - total
		n, err := syscall.Read(fd, buf[total:])
		if n > 0 {
			total += n

			// A first read that came up short of what was asked for and still
			// met the stat's promise has reached the end of the file, because
			// a regular file only ever returns less than was asked for at the
			// end of it. That is the whole of the argument, and it rests on the
			// file being regular rather than on any arithmetic about the
			// buffer: ReadFile hands us a pooled buffer rounded up to a power
			// of two, so len(buf) can be close to twice the size and n < wanted
			// is a far weaker statement than "short of size + readSlack".
			//
			// Only the first read gets that treatment. Once a read has been
			// chopped the reads are being chopped, and after that nothing short
			// of a zero-length read proves anything.
			//
			// What this does NOT cover is a source that both reports a nonzero
			// size and chops its reads, where a chop can land anywhere at or
			// past the size of a file that has since grown and is then taken
			// for the end. No read can tell that apart from the file simply
			// having ended, so no arrangement of this test can; confirming it
			// costs the second read the whole change exists to avoid. scc does
			// not meet it, because size comes from a stat of a regular file and
			// anything that chops - a fifo, /proc, sysfs - stats as zero and is
			// excluded by size > 0 below. A network filesystem serving a file
			// that is growing is the case that would.
			// TestReadFileShortReadsAreNotTheEnd holds it to all of that.
			if first && n < wanted && size > 0 && total >= size {
				break
			}
			first = false

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
