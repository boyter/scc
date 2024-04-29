// SPDX-License-Identifier: MIT

package gitignore

// tokenset represents an ordered list of Tokens
type tokenset []*Token

// String() returns a concatenated string of all runes represented by the
// list of tokens.
func (t tokenset) String() string {
	// concatenate the tokens into a single string
	_rtn := ""
	for _, _t := range []*Token(t) {
		_rtn = _rtn + _t.Token()
	}
	return _rtn
} // String()
