// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkFileReaderReadFile(b *testing.B) {
	originalLargeByteCount := LargeByteCount
	LargeByteCount = 1_000_000
	b.Cleanup(func() {
		LargeByteCount = originalLargeByteCount
	})

	tests := []struct {
		name string
		size int
	}{
		{name: "empty_0B", size: 0},
		{name: "small_4KiB", size: 4 * 1024},
		{name: "medium_256KiB", size: 256 * 1024},
		{name: "large_4MiB", size: 4 * 1024 * 1024},
	}

	for _, tc := range tests {
		b.Run(tc.name, func(b *testing.B) {
			content := bytes.Repeat([]byte{'x'}, tc.size)
			path := filepath.Join(b.TempDir(), "input.go")
			if err := os.WriteFile(path, content, 0600); err != nil {
				b.Fatal(err)
			}

			reader := NewFileReader()
			b.ReportAllocs()
			b.SetBytes(int64(tc.size))
			b.ResetTimer()
			for b.Loop() {
				got, err := reader.ReadFile(path, tc.size)
				if err != nil {
					b.Fatal(err)
				}
				if len(got) != tc.size {
					b.Fatalf("read %d bytes, want %d", len(got), tc.size)
				}
			}
		})
	}
}
