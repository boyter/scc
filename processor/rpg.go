// SPDX-License-Identifier: MIT

package processor

import "bytes"

// **FREE on the first source line removes the RPGLE column restrictions.
func rpgFullyFree(content []byte) bool {
	line, _, _ := bytes.Cut(content, []byte{'\n'})
	return bytes.EqualFold(bytes.TrimRight(line, " \t\r"), []byte("**FREE"))
}

func rpgCommentLine(content []byte) bool {
	line, _, _ := bytes.Cut(content, []byte{'\n'})
	return len(line) > 6 && line[6] == '*'
}
