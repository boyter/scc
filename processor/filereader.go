// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"math/bits"
)

// readSlack is the spare room left past the size the caller gave us. The read
// loop needs at least one byte beyond the file to tell a read that finished the
// file from one that merely filled the buffer, and a file that grew between the
// stat and the open should not force a reallocation, so leave a little more than
// one byte. Taking this to zero would cost a second read on every file.
const readSlack = 512

// readBufferFloor is the smallest buffer we bother to keep. Most source files
// are a few kilobytes, so starting here means the common case allocates once
// per worker and never again, rather than climbing a doubling ladder from 512
// bytes and zeroing every rung on the way up.
const readBufferFloor = 64 * 1024

// readBufferRetain is the largest buffer a reader will hold on to between
// files. Anything past it is read into a one-shot buffer that is dropped when
// the file is done, so a single enormous file does not leave every worker
// sitting on an enormous buffer for the rest of the run.
const readBufferRetain = 8 * 1024 * 1024

// FileReader is a struct responsible for reading files into its buffer
type FileReader struct {
	// Buffer stays for the non-unix read path, which reads through the
	// bytes.Buffer methods.
	Buffer *bytes.Buffer
	buf    []byte
}

// NewFileReader creates a new file reader responsible for reading a file
func NewFileReader() FileReader {
	return FileReader{
		Buffer: &bytes.Buffer{},
	}
}

// readBufferSize rounds the wanted size up to a power of two, no smaller than
// the floor, so a reader reallocates a handful of times over a whole tree
// instead of once for every file that is bigger than the last.
func readBufferSize(need int) int {
	if need <= readBufferFloor {
		return readBufferFloor
	}
	if bits.OnesCount(uint(need)) == 1 {
		return need
	}
	return 1 << bits.Len(uint(need))
}

// ReadFile reads a file into a buffer that is reused between calls, growing it
// to the size the caller says the file is.
//
// The buffer is only ever grown, and grown in power-of-two steps, because every
// growth is a fresh allocation that the runtime zeroes before we immediately
// overwrite it with the file. Reading the Linux kernel used to allocate 203MB
// here — the reader was thrown away whenever it had grown past LargeByteCount,
// so every worker that met a megabyte file climbed the whole doubling ladder
// again from nothing, and with a worker per four cores that happened a lot.
//
// A file too big to be worth keeping a buffer for is read into a one-shot
// buffer instead, so one enormous file in a tree does not leave every worker
// holding an enormous buffer for the rest of the run.
func (reader *FileReader) ReadFile(path string, size int) ([]byte, error) {
	need := size + readSlack

	if need > readBufferRetain {
		return reader.readFileInto(path, make([]byte, need), size)
	}

	if cap(reader.buf) < need {
		reader.buf = make([]byte, readBufferSize(need))
	}

	return reader.readFileInto(path, reader.buf[:cap(reader.buf)], size)
}
