// SPDX-License-Identifier: MIT

package gitignore

// Match represents the interface of successful matches against a .gitignore
// pattern set. A Match can be queried to determine whether the matched path
// should be ignored or included (i.e. was the path matched by a negated
// pattern), and to extract the position of the pattern within the .gitignore,
// and a string representation of the pattern.
type Match interface {
	// Ignore returns true if the match pattern describes files or paths that
	// should be ignored.
	Ignore() bool

	// Include returns true if the match pattern describes files or paths that
	// should be included.
	Include() bool

	// String returns a string representation of the matched pattern.
	String() string

	// Position returns the position in the .gitignore file at which the
	// matching pattern was defined.
	Position() Position
}
