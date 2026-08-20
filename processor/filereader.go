// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"fmt"
	"os"
)

// FileReader is a struct responsible for reading files into its buffer
type FileReader struct {
	Buffer *bytes.Buffer
}

// NewFileReader creates a new file reader responsible for reading a file
func NewFileReader() FileReader {
	return FileReader{
		Buffer: &bytes.Buffer{},
	}
}

// ReadFile actually reads the file into a buffer size controlled by LargeByteCount
func (reader *FileReader) ReadFile(path string, size int) ([]byte, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", path, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(fd)

	// Generally, re-using the buffer is best. But, if we end up reading a huge
	// file we would allocate an equally huge buffer. Rather than keep the huge
	// buffer around forever, it's probably worth eating the GC cost of
	// replacing it so that we can release the memory.
	if int64(reader.Buffer.Cap()) > LargeByteCount {
		reader.Buffer = &bytes.Buffer{}
	}

	// Reset contents, but retain the underlying memory that's already been allocated.
	reader.Buffer.Reset()
	// Leave room for ReadFrom's final EOF probe to avoid an extra buffer growth.
	reader.Buffer.Grow(size + bytes.MinRead)

	_, err = reader.Buffer.ReadFrom(fd)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return reader.Buffer.Bytes(), nil
}
