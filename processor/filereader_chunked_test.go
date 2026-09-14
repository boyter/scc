// SPDX-License-Identifier: MIT

//go:build unix

package processor

import (
	"bytes"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A read that comes up short is only the end of a file once the stat's promise
// has been met, and readFileInto leans on that to finish most files in one read.
// These are the inputs where the short read is not the end.
//
// TestReadFileSyntheticFile covers the same ground with /proc/kallsyms, which
// exists on Linux and nowhere else, so it skips on the machine this was written
// on. A fifo is the portable stand-in: it stats as zero bytes, it hands back
// only what has been written so far, and every read of it is short.

// writeFifoInChunks opens the fifo for writing and feeds it a chunk at a time,
// so every read on the other end comes back short.
func writeFifoInChunks(t *testing.T, path string, content []byte, chunk int) {
	t.Helper()

	go func() {
		// Opening for write blocks until the reader opens, which is what
		// synchronises the two ends. Nothing here fails the test directly,
		// since a test goroutine may not call Fatal.
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err != nil {
			return
		}
		defer func() { _ = file.Close() }()

		for start := 0; start < len(content); start += chunk {
			end := min(start+chunk, len(content))
			if _, err := file.Write(content[start:end]); err != nil {
				return
			}
		}
	}()
}

func TestReadFileShortReadsAreNotTheEnd(t *testing.T) {
	for _, test := range []struct {
		name string
		// told is what the reader is told the file holds, standing in for what
		// a stat returned.
		told func(n int) int
		// mayTruncate marks the input where a short read genuinely cannot be
		// told from the end of the file, so a prefix is all that is promised.
		mayTruncate bool
	}{
		{name: "stats as zero, the way a fifo and /proc do", told: func(int) int { return 0 }},
		{name: "stats truthfully, but the reads are chopped up", told: func(n int) int { return n }},
		{name: "stats long, the file shrank and the reads are chopped", told: func(n int) int { return n * 2 }},
		{
			// The one input the first-read rule cannot cover. A source that
			// reports a nonzero size and also chops its reads can hand back a
			// chop landing anywhere at or past that size, and nothing in the
			// read says whether the file ended there or merely reached the size
			// it used to be. Telling the two apart costs the second read the
			// whole change exists to avoid, so what is asserted here is what is
			// actually promised: whatever comes back is a correct prefix, never
			// wrong bytes. scc cannot reach this - a fifo stats as zero, and
			// size > 0 gates the shortcut - so the case is here to pin the
			// limit down, not to describe a file scc will meet.
			name:        "stats short, the file grew and the reads are chopped",
			told:        func(n int) int { return n / 2 },
			mayTruncate: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "chunked.go")
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Skipf("cannot make a fifo here: %v", err)
			}

			want := make([]byte, 32*1024)
			for i := range want {
				want[i] = byte('a' + i%26)
			}

			writeFifoInChunks(t, path, want, 4096)

			done := make(chan []byte, 1)
			go func() {
				reader := NewFileReader()
				got, err := reader.ReadFile(path, test.told(len(want)))
				if err != nil {
					done <- nil

					return
				}
				done <- got
			}()

			select {
			case got := <-done:
				if got == nil {
					t.Fatal("reading the fifo failed")
				}
				if test.mayTruncate {
					if !bytes.HasPrefix(want, got) {
						t.Fatalf("read %d bytes of a %d byte fifo and they are not a prefix of it",
							len(got), len(want))
					}

					return
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("read %d bytes of a %d byte fifo, so a short read was taken for the end",
						len(got), len(want))
				}
			case <-time.After(30 * time.Second):
				t.Fatal("timed out reading the fifo")
			}
		})
	}
}
