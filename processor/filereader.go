package processor

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

type FileReader struct {
	Buffer *bytes.Buffer
}

func NewFileReader() FileReader {
	return FileReader{
		Buffer: &bytes.Buffer{},
	}
}

func (reader *FileReader) ReadFile(path string, size int) ([]byte, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", path, err)
	}
	defer fd.Close()

	// Generally, re-using the buffer is best. But, if we end up reading a huge
	// file we would allocate an equally huge buffer. Rather than keep the huge
	// buffer around forever, it's probably worth eating the GC cost of
	// replacing it so that we can release the memory.
	if int64(reader.Buffer.Cap()) > LargeByteCount {
		reader.Buffer = &bytes.Buffer{}
	}

	// Reset contents, but retain the underlying memory that's already been
	// allocated
	reader.Buffer.Reset()

	reader.Buffer.Grow(size)

	_, err = io.Copy(reader.Buffer, fd)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", path, err)
	}

	return reader.Buffer.Bytes(), nil
}
