// SPDX-License-Identifier: MIT

package processor

import (
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// decodeUTF16BOM returns content transcoded to UTF-8 when it opens with a
// UTF-16 byte order mark, and false otherwise. SSMS and Visual Studio save
// stored procedures as UTF-16 LE with a BOM by default, and read raw those
// files are a nul between every ASCII byte, which the binary check rejects.
//
// Only the BOM is trusted. UTF-16 without one is left alone, as ripgrep does:
// nothing cheap and reliable tells it apart from binary. The check is two
// bytes, so a UTF-8 file pays nothing for it.
//
// A lone surrogate becomes U+FFFD rather than an error, so a damaged file
// still counts instead of stopping the run. A trailing odd byte is dropped:
// as U+FFFD after the last newline it would count as a line of code.
func decodeUTF16BOM(content []byte) ([]byte, bool) {
	if len(content) < 2 {
		return nil, false
	}

	var bigEndian bool
	switch {
	case content[0] == 0xFF && content[1] == 0xFE:
		// FF FE 00 00 is the UTF-32 LE mark, which this does not handle.
		if len(content) >= 4 && content[2] == 0 && content[3] == 0 {
			return nil, false
		}
		bigEndian = false
	case content[0] == 0xFE && content[1] == 0xFF:
		bigEndian = true
	default:
		return nil, false
	}

	body := content[2:]
	unit := func(i int) rune {
		if bigEndian {
			return rune(body[i])<<8 | rune(body[i+1])
		}
		return rune(body[i+1])<<8 | rune(body[i])
	}

	// Source text is mostly ASCII, which halves going from UTF-16 to UTF-8.
	out := make([]byte, 0, len(body)/2+8)
	for i := 0; i+1 < len(body); i += 2 {
		r := unit(i)
		if utf16.IsSurrogate(r) {
			decoded := unicode.ReplacementChar
			if i+3 < len(body) {
				decoded = utf16.DecodeRune(r, unit(i+2))
			}
			if decoded != unicode.ReplacementChar {
				i += 2
			}
			r = decoded
		}
		out = utf8.AppendRune(out, r)
	}

	return out, true
}
