// SPDX-License-Identifier: MIT

package gitignore

import (
	"errors"
)

var (
	CarriageReturnError   = errors.New("unexpected carriage return '\\r'")
	InvalidPatternError   = errors.New("invalid pattern")
	InvalidDirectoryError = errors.New("invalid directory")
)
