// SPDX-License-Identifier: MIT
package processor

import (
	"bytes"
	"strings"
)

// Explicit source-format directives take effect after their own line.
func cobolSourceFormat(content []byte) (fixed, changed bool) {
	line, _, _ := bytes.Cut(content, []byte{'\n'})
	text := strings.ToUpper(strings.TrimSpace(string(line)))
	if !strings.HasPrefix(text, ">>") && len(line) >= 7 {
		if line[6] == '*' || line[6] == '/' {
			return false, false
		}
		text = strings.ToUpper(strings.TrimSpace(string(line[7:])))
	}
	text = strings.TrimSpace(strings.TrimPrefix(text, ">>"))
	fields := strings.Fields(text)
	if len(fields) < 3 || fields[0] != "SOURCE" || fields[1] != "FORMAT" {
		return false, false
	}
	format := fields[2]
	if format == "IS" && len(fields) > 3 {
		format = fields[3]
	}
	switch strings.Trim(format, "\"'.") {
	case "FREE":
		return false, true
	case "FIXED":
		return true, true
	}
	return false, false
}

func cobolCommentLine(content []byte) bool {
	line, _, _ := bytes.Cut(content, []byte{'\n'})
	return len(line) > 6 && (line[6] == '*' || line[6] == '/')
}
