// SPDX-License-Identifier: MIT

//go:build !unix

package processor

import (
	"bytes"
	"fmt"
	"os"
)

// readFile is the portable form, used where the system calls of
// filereader_unix.go are not the ones the platform has.
func (reader *FileReader) readFileInto(path string, buf []byte, _ int) ([]byte, error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("error opening %s: %v", path, err)
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(fd)

	reader.Buffer = bytes.NewBuffer(buf[:0])
	if _, err = reader.Buffer.ReadFrom(fd); err != nil {
		return nil, fmt.Errorf("error reading %s: %w", path, err)
	}

	return reader.Buffer.Bytes(), nil
}
