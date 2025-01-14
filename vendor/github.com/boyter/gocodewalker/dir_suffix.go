// SPDX-License-Identifier: MIT

package gocodewalker

import (
	"path/filepath"
	"strings"
)

// isSuffixDir returns true if base ends with suffix. Suffix "/" will be trimmed.
// suffix must be a valid sub dir of base.
// For examples:
//   - isSuffixDir("a", "a") returns true
//   - isSuffixDir("a/b/c", "c") returns true
//   - isSuffixDir("a/b/c", "b/c") returns true
//   - isSuffixDir("a/b/c", "b") returns false
//   - isSuffixDir("a/b/c", "a/b") returns false, "a/b" is a valid sub dir but not at the end of "a/b/c"
//   - isSuffixDir("a/bb/c", "b/c") returns false
func isSuffixDir(base string, suffix string) bool {
	if base == "" || suffix == "" {
		return false
	}
	base = strings.TrimSuffix(filepath.ToSlash(base), "/")
	suffix = strings.TrimSuffix(filepath.ToSlash(suffix), "/")
	newBase := strings.TrimSuffix(base, suffix)
	if newBase == base {
		return false
	}
	return strings.HasSuffix(newBase, "/") || newBase == ""
}
