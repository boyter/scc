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

// TestReadFileWrongSize covers the cases where the size the caller passes is not
// the number of bytes the file holds, which is what the read loop's early exit
// has to get right: a file that grew since it was stat'd, one that shrank, and
// the synthetic files whose stat size is zero or a fixed page whatever they
// hold. Every one of them has to come back whole.
func TestReadFileWrongSize(t *testing.T) {
	dir := t.TempDir()

	sizes := []int{0, 1, 100, readBufferFloor - 1, readBufferFloor, readBufferFloor + 1, 3 * readBufferFloor}
	claims := []struct {
		name string
		of   func(int) int
	}{
		{"truthful", func(n int) int { return n }},
		{"zero, as /proc reports", func(n int) int { return 0 }},
		{"half, the file grew", func(n int) int { return n / 2 }},
		{"one short, the file grew by a byte", func(n int) int { return n - 1 }},
		{"one long, the file shrank by a byte", func(n int) int { return n + 1 }},
		{"double, the file shrank", func(n int) int { return n * 2 }},
		{"a fixed page, as sysfs reports", func(int) int { return 4096 }},
	}

	reader := NewFileReader()
	for _, n := range sizes {
		want := make([]byte, n)
		for i := range want {
			want[i] = byte('a' + i%26)
		}
		path := filepath.Join(dir, "input.go")
		if err := os.WriteFile(path, want, 0600); err != nil {
			t.Fatal(err)
		}

		for _, claim := range claims {
			size := claim.of(n)
			if size < 0 {
				continue
			}

			got, err := reader.ReadFile(path, size)
			if err != nil {
				t.Fatalf("ReadFile on %d bytes told %d (%s): %v", n, size, claim.name, err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("ReadFile on %d bytes told %d (%s) returned %d bytes, want %d",
					n, size, claim.name, len(got), n)
			}
		}
	}
}

// TestReadFileSyntheticFile reads a file that stats as zero bytes, is not empty,
// and hands back a page at a time however much is asked for. /proc/kallsyms is
// the example: it stats as nothing, runs to tens of megabytes, and answers a
// 64KB read with 4,092 bytes. A short read there is not the end of the file, and
// a reader that takes it for one stops after the first page.
func TestReadFileSyntheticFile(t *testing.T) {
	const path = "/proc/kallsyms"

	info, err := os.Stat(path)
	if err != nil {
		t.Skipf("no %s on this system", path)
	}
	if info.Size() != 0 {
		t.Skipf("%s stats as %d bytes, not the zero this test is about", path, info.Size())
	}

	// os.ReadFile is the reference. It has the same problem to solve and solves
	// it by reading until a zero length read.
	want, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("cannot read %s: %v", path, err)
	}
	if len(want) < 1<<20 {
		t.Skipf("%s is only %d bytes here, too small to catch a truncation", path, len(want))
	}

	reader := NewFileReader()
	got, err := reader.ReadFile(path, int(info.Size()))
	if err != nil {
		t.Fatal(err)
	}

	// The symbol table does not move under us the way /proc/self/maps would, but
	// a module load between the two reads would, so allow a little slack rather
	// than demanding the same byte count.
	if len(got) < len(want)/2 {
		t.Fatalf("read %d bytes of %s where os.ReadFile got %d: truncated", len(got), path, len(want))
	}
}

func TestReadBufferSize(t *testing.T) {
	for _, tc := range []struct{ in, want int }{
		{0, readBufferFloor},
		{1, readBufferFloor},
		{readBufferFloor, readBufferFloor},
		{readBufferFloor + 1, readBufferFloor * 2},
		{readBufferFloor * 2, readBufferFloor * 2},
		{readBufferFloor*2 + 1, readBufferFloor * 4},
	} {
		if got := readBufferSize(tc.in); got != tc.want {
			t.Errorf("readBufferSize(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
