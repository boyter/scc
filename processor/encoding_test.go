// SPDX-License-Identifier: MIT

package processor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestProcessFileTextEncodings runs the same two-line SQL file, saved in each
// encoding a Windows editor commonly writes, through processFile. SSMS and
// Visual Studio save stored procedures as UTF-16 LE with a BOM by default, and
// those files were classed as binary on the nul between every ASCII byte and
// silently dropped from the count. A file with a UTF-16 BOM should count the
// same as its UTF-8 twin, and report its size on disk.
//
// UTF-16 with no BOM is out of scope, as it is for ripgrep: nothing cheap and
// reliable tells it apart from binary, so it stays classed as binary.
func TestProcessFileTextEncodings(t *testing.T) {
	ProcessConstants()

	cases := []struct {
		file   string
		binary bool
	}{
		{"utf8.sql", false},
		{"utf8_bom.sql", false},
		{"utf16le_bom.sql", false},
		{"utf16be_bom.sql", false},
		{"utf16le_nobom.sql", true},
	}

	ctx := processorContext{remap: newRemapConfig("", "")}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("..", "examples", "encoding", tc.file))
			if err != nil {
				t.Fatal(err)
			}

			possible, extension := DetectLanguage(tc.file)
			fileJob := FileJob{
				Filename:          tc.file,
				Extension:         extension,
				PossibleLanguages: possible,
				Content:           content,
				Bytes:             int64(len(content)),
			}

			counted := ctx.processFile(&fileJob)

			if fileJob.Binary != tc.binary {
				t.Fatalf("binary: want %v got %v", tc.binary, fileJob.Binary)
			}
			if tc.binary {
				if counted {
					t.Fatal("a binary file should not be counted")
				}
				return
			}

			if !counted {
				t.Fatal("file was not counted")
			}
			if fileJob.Language != "SQL" {
				t.Errorf("language: want SQL got %q", fileJob.Language)
			}
			if fileJob.Bytes != int64(len(content)) {
				t.Errorf("bytes: want size on disk %d got %d", len(content), fileJob.Bytes)
			}
			if fileJob.Lines != 2 || fileJob.Code != 2 || fileJob.Comment != 0 || fileJob.Blank != 0 || fileJob.Complexity != 0 {
				t.Errorf("want lines=2 code=2 comment=0 blank=0 complexity=0, got lines=%d code=%d comment=%d blank=%d complexity=%d",
					fileJob.Lines, fileJob.Code, fileJob.Comment, fileJob.Blank, fileJob.Complexity)
			}
		})
	}
}

func TestDecodeUTF16BOM(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want []byte
		ok   bool
	}{
		{"empty", nil, nil, false},
		{"one byte", []byte{0xFF}, nil, false},
		{"utf8", []byte("abc"), nil, false},
		{"utf8 bom", []byte{0xEF, 0xBB, 0xBF, 'a'}, nil, false},
		{"utf32 le bom", []byte{0xFF, 0xFE, 0, 0, 'a', 0, 0, 0}, nil, false},
		{"le bom only", []byte{0xFF, 0xFE}, []byte{}, true},
		{"le ascii", []byte{0xFF, 0xFE, 'h', 0, 'i', 0}, []byte("hi"), true},
		{"be ascii", []byte{0xFE, 0xFF, 0, 'h', 0, 'i'}, []byte("hi"), true},
		{"le bmp", []byte{0xFF, 0xFE, 0xE9, 0x00}, []byte("é"), true},
		// U+1F600 is the surrogate pair D83D DE00.
		{"le surrogate pair", []byte{0xFF, 0xFE, 0x3D, 0xD8, 0x00, 0xDE}, []byte("😀"), true},
		{"be surrogate pair", []byte{0xFE, 0xFF, 0xD8, 0x3D, 0xDE, 0x00}, []byte("😀"), true},
		{"lone high surrogate at end", []byte{0xFF, 0xFE, 0x3D, 0xD8}, []byte("\uFFFD"), true},
		{"high surrogate then ascii", []byte{0xFF, 0xFE, 0x3D, 0xD8, 'a', 0}, []byte("\uFFFDa"), true},
		{"lone low surrogate", []byte{0xFF, 0xFE, 0x00, 0xDE, 'a', 0}, []byte("\uFFFDa"), true},
		{"odd trailing byte", []byte{0xFF, 0xFE, 'a', 0, 'b'}, []byte("a"), true},
		{"odd byte after newline", []byte{0xFF, 0xFE, 'a', 0, '\n', 0, 'b'}, []byte("a\n"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := decodeUTF16BOM(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok: want %v got %v", tc.ok, ok)
			}
			if ok && !bytes.Equal(got, tc.want) {
				t.Errorf("want %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkDecodeUTF16BOMMiss(b *testing.B) {
	content := bytes.Repeat([]byte("SELECT * FROM t;\n"), 1000)
	for b.Loop() {
		decodeUTF16BOM(content)
	}
}
