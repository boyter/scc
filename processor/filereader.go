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

	// Reset contents, but retain the underlying memory that's already been
	// allocated
	reader.Buffer.Reset()

	// stat, err := fd.Stat()
	// if err != nil {
	// 	return nil, fmt.Errorf("error opening %s: %v", path, err)
	// }

	//reader.Buffer.Grow(int(stat.Size()))
	reader.Buffer.Grow(size)

	_, err = io.Copy(reader.Buffer, fd)
	if err != nil {
		return nil, fmt.Errorf("error reading %s: %v", path, err)
	}

	return reader.Buffer.Bytes(), nil
}
