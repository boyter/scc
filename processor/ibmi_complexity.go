// SPDX-License-Identifier: MIT
package processor

import (
	"bytes"
	"regexp"
	"strings"
)

var embeddedSQLStart = regexp.MustCompile(`(?im)^[ \t]*(?:C[ \t]+)?/?EXEC[ \t]+SQL(?:[ \t\r\n]|$)`)

// Decision words must be opcodes, not operands, names or parameter values.
func ibmiComplexityToken(job *FileJob, index, length int) bool {
	rpg := job.Language == "RPG" || job.Language == "RPGLE" || job.Language == "SQLRPGLE"
	cl := job.Language == "IBM i CL"
	if !rpg && !cl {
		return true
	}
	// COMP is a fixed-format opcode; free-form identifiers must not count.
	if rpg && bytes.EqualFold(job.Content[index:index+length], []byte("COMP")) {
		return false
	}
	if index+length < len(job.Content) && isIdentifierContinue(job.Content[index+length]) {
		return false
	}
	if index > 0 && (job.Content[index-1] == '&' || job.Content[index-1] == '%') {
		return false
	}
	if job.Language == "SQLRPGLE" {
		statement := job.Content[:index]
		if i := bytes.LastIndexByte(statement, ';'); i >= 0 {
			statement = statement[i+1:]
		}
		if embeddedSQLStart.Match(statement) {
			return false
		}
	}
	bomSkip := 0
	if bytes.HasPrefix(job.Content, []byte{239, 187, 191}) {
		bomSkip = 3
	}
	start := bytes.LastIndexByte(job.Content[:index], '\n') + 1
	if start == 0 {
		start = bomSkip
	}
	line := job.Content[start:index]
	if rpg && !rpgFullyFree(job.Content[bomSkip:]) {
		if len(line) > 5 && (line[5] == 'C' || line[5] == 'c') {
			return false
		} // counted from the opcode field
		if len(line) >= 5 {
			line = line[5:]
		}
	}
	if rpg {
		if i := bytes.LastIndexByte(line, ';'); i >= 0 {
			line = line[i+1:]
		}
	}
	prefix := strings.TrimSpace(string(line))
	// A complete block comment may precede an executable command.
	for strings.HasSuffix(prefix, "*/") {
		i := strings.LastIndex(prefix, "/*")
		if i < 0 {
			break
		}
		prefix = strings.TrimSpace(prefix[:i])
	}
	if prefix == "" {
		return true
	}
	if cl {
		if strings.HasSuffix(prefix, ":") && !strings.ContainsAny(prefix, " \t()") {
			return true
		}
		prefix = strings.ToUpper(prefix)
		return strings.HasSuffix(prefix, "THEN(") || strings.HasSuffix(prefix, "EXEC(")
	}
	return false
}

func fixedRPGDecision(language string, line []byte) bool {
	if len(line) < 6 || (line[5] != 'C' && line[5] != 'c') {
		return false
	}
	start, end := 25, 35 // RPG IV
	if language == "RPG" {
		start, end = 27, 32
	} // RPG III
	if len(line) <= start {
		return false
	}
	op := strings.ToUpper(strings.TrimSpace(string(line[start:min(end, len(line))])))
	if i := strings.IndexByte(op, '('); i >= 0 {
		op = op[:i]
	}
	for _, decision := range languageDatabase[language].ComplexityChecks {
		if op == decision {
			return true
		}
	}
	return false
}
