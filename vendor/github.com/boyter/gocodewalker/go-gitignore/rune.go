// SPDX-License-Identifier: MIT

package gitignore

const (
	// define the sentinel runes of the lexer
	_EOF       = rune(0)
	_CR        = rune('\r')
	_NEWLINE   = rune('\n')
	_COMMENT   = rune('#')
	_SEPARATOR = rune('/')
	_ESCAPE    = rune('\\')
	_SPACE     = rune(' ')
	_TAB       = rune('\t')
	_NEGATION  = rune('!')
	_WILDCARD  = rune('*')
)
