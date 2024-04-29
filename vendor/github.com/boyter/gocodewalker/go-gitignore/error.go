// SPDX-License-Identifier: MIT

package gitignore

type Error interface {
	error

	// Position returns the position of the error within the .gitignore file
	// (if any)
	Position() Position

	// Underlying returns the underlying error, permitting direct comparison
	// against the wrapped error.
	Underlying() error
}

type err struct {
	error
	_position Position
} // err()

// NewError returns a new Error instance for the given error e and position p.
func NewError(e error, p Position) Error {
	return &err{error: e, _position: p}
} // NewError()

func (e *err) Position() Position { return e._position }

func (e *err) Underlying() error { return e.error }

// ensure err satisfies the Error interface
var _ Error = &err{}
