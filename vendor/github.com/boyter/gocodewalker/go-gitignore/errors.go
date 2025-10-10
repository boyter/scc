// SPDX-License-Identifier: MIT

package gitignore

import (
	"errors"
)

var (
	ErrCarriageReturnError   = errors.New("unexpected carriage return '\\r'")
	ErrInvalidPatternError   = errors.New("invalid pattern")
	ErrInvalidDirectoryError = errors.New("invalid directory")
)
