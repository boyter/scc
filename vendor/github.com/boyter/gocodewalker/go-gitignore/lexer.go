// SPDX-License-Identifier: MIT

package gitignore

import (
	"bufio"
	"io"
)

//
// inspired by https://blog.gopheracademy.com/advent-2014/parsers-lexers/
//

// lexer is the implementation of the .gitignore lexical analyser
type lexer struct {
	_r        *bufio.Reader
	_unread   []rune
	_offset   int
	_line     int
	_column   int
	_previous []int
} // lexer{}

// Lexer is the interface to the lexical analyser for .gitignore files
type Lexer interface {
	// Next returns the next Token from the Lexer reader. If an error is
	// encountered, it will be returned as an Error instance, detailing the
	// error and its position within the stream.
	Next() (*Token, Error)

	// Position returns the current position of the Lexer.
	Position() Position

	// String returns the string representation of the current position of the
	// Lexer.
	String() string
}

// NewLexer returns a Lexer instance for the io.Reader r.
func NewLexer(r io.Reader) Lexer {
	return &lexer{_r: bufio.NewReader(r), _line: 1, _column: 1}
} // NewLexer()

// Next returns the next Token from the Lexer reader. If an error is
// encountered, it will be returned as an Error instance, detailing the error
// and its position within the stream.
func (l *lexer) Next() (*Token, Error) {
	// are we at the beginning of the line?
	_beginning := l.beginning()

	// read the next rune
	_r, _err := l.read()
	if _err != nil {
		return nil, _err
	}

	switch _r {
	// end of file
	case _EOF:
		return l.token(EOF, nil, nil)

	// whitespace ' ', '\t'
	case _SPACE:
		fallthrough
	case _TAB:
		l.unread(_r)
		_rtn, _err := l.whitespace()
		return l.token(WHITESPACE, _rtn, _err)

	// end of line '\n' or '\r\n'
	case _CR:
		fallthrough
	case _NEWLINE:
		l.unread(_r)
		_rtn, _err := l.eol()
		return l.token(EOL, _rtn, _err)

	// separator '/'
	case _SEPARATOR:
		return l.token(SEPARATOR, []rune{_r}, nil)

	// '*' or any '**'
	case _WILDCARD:
		// is the wildcard followed by another wildcard?
		//      - does this represent the "any" token (i.e. "**")
		_next, _err := l.peek()
		if _err != nil {
			return nil, _err
		} else if _next == _WILDCARD {
			// we know read() will succeed here since we used peek() above
			_, _ = l.read()
			return l.token(ANY, []rune{_WILDCARD, _WILDCARD}, nil)
		}

		// we have a single wildcard, so treat this as a pattern
		l.unread(_r)
		_rtn, _err := l.pattern()
		return l.token(PATTERN, _rtn, _err)

	// comment '#'
	case _COMMENT:
		l.unread(_r)

		// if we are at the start of the line, then we treat this as a comment
		if _beginning {
			_rtn, _err := l.comment()
			return l.token(COMMENT, _rtn, _err)
		}

		// otherwise, we regard this as a pattern
		_rtn, _err := l.pattern()
		return l.token(PATTERN, _rtn, _err)

	// negation '!'
	case _NEGATION:
		if _beginning {
			return l.token(NEGATION, []rune{_r}, nil)
		}
		fallthrough

	// pattern
	default:
		l.unread(_r)
		_rtn, _err := l.pattern()
		return l.token(PATTERN, _rtn, _err)
	}
} // Next()

// Position returns the current position of the Lexer.
func (l *lexer) Position() Position {
	return Position{"", l._line, l._column, l._offset}
} // Position()

// String returns the string representation of the current position of the
// Lexer.
func (l *lexer) String() string {
	return l.Position().String()
} // String()

//
// private methods
//

// read the next rune from the stream. Return an Error if there is a problem
// reading from the stream. If the end of stream is reached, return the EOF
// Token.
func (l *lexer) read() (rune, Error) {
	var _r rune
	var _err error

	// do we have any unread runes to read?
	_length := len(l._unread)
	if _length > 0 {
		_r = l._unread[_length-1]
		l._unread = l._unread[:_length-1]

		// otherwise, attempt to read a new rune
	} else {
		_r, _, _err = l._r.ReadRune()
		if _err == io.EOF {
			return _EOF, nil
		}
	}

	// increment the offset and column counts
	l._offset++
	l._column++

	return _r, l.err(_err)
} // read()

// unread returns the given runes to the stream, making them eligible to be
// read again. The runes are returned in the order given, so the last rune
// specified will be the next rune read from the stream.
func (l *lexer) unread(r ...rune) {
	// ignore EOF runes
	_r := make([]rune, 0)
	for _, _rune := range r {
		if _rune != _EOF {
			_r = append(_r, _rune)
		}
	}

	// initialise the unread rune list if necessary
	if l._unread == nil {
		l._unread = make([]rune, 0)
	}
	if len(_r) != 0 {
		l._unread = append(l._unread, _r...)
	}

	// decrement the offset and column counts
	//      - we have to take care of column being 0
	//      - at present we can only unwind across a single line boundary
	_length := len(_r)
	for ; _length > 0; _length-- {
		l._offset--
		if l._column == 1 {
			_length := len(l._previous)
			if _length > 0 {
				l._column = l._previous[_length-1]
				l._previous = l._previous[:_length-1]
				l._line--
			}
		} else {
			l._column--
		}
	}
} // unread()

// peek returns the next rune in the stream without consuming it (i.e. it will
// be returned by the next call to read or peek). peek will return an error if
// there is a problem reading from the stream.
func (l *lexer) peek() (rune, Error) {
	// read the next rune
	_r, _err := l.read()
	if _err != nil {
		return _r, _err
	}

	// unread & return the rune
	l.unread(_r)
	return _r, _err
} // peek()

// newline adjusts the positional counters when an end of line is reached
func (l *lexer) newline() {
	// adjust the counters for the new line
	if l._previous == nil {
		l._previous = make([]int, 0)
	}
	l._previous = append(l._previous, l._column)
	l._column = 1
	l._line++
} // newline()

// comment reads all runes until a newline or end of file is reached. An
// error is returned if an error is encountered reading from the stream.
func (l *lexer) comment() ([]rune, Error) {
	_comment := make([]rune, 0)

	// read until we reach end of line or end of file
	//		- as we are in a comment, we ignore escape characters
	for {
		_next, _err := l.read()
		if _err != nil {
			return _comment, _err
		}

		// read until we have end of line or end of file
		switch _next {
		case _CR:
			fallthrough
		case _NEWLINE:
			fallthrough
		case _EOF:
			// return the read run to the stream and stop
			l.unread(_next)
			return _comment, nil
		}

		// otherwise, add this run to the comment
		_comment = append(_comment, _next)
	}
} // comment()

// escape attempts to read an escape sequence (e.g. '\ ') form the input
// stream. An error will be returned if there is an error reading from the
// stream. escape returns just the escape rune if the following rune is either
// end of line or end of file (since .gitignore files do not support line
// continuations).
func (l *lexer) escape() ([]rune, Error) {
	// attempt to process the escape sequence
	_peek, _err := l.peek()
	if _err != nil {
		return nil, _err
	}

	// what is the next rune after the escape?
	switch _peek {
	// are we at the end of the line or file?
	//      - we return just the escape rune
	case _CR:
		fallthrough
	case _NEWLINE:
		fallthrough
	case _EOF:
		return []rune{_ESCAPE}, nil
	}

	// otherwise, return the escape and the next rune
	//      - we know read() will succeed here since we used peek() above
	_, _ = l.read()
	return []rune{_ESCAPE, _peek}, nil
} // escape()

// eol returns all runes from the current position to the end of the line. An
// error is returned if there is a problem reading from the stream, or if a
// carriage return character '\r' is encountered that is not followed by a
// newline '\n'.
func (l *lexer) eol() ([]rune, Error) {
	// read the to the end of the line
	//      - we should only be called here when we encounter an end of line
	//        sequence
	_line := make([]rune, 0, 1)

	// loop until there's nothing more to do
	for {
		_next, _err := l.read()
		if _err != nil {
			return _line, _err
		}

		// read until we have a newline or we're at end of file
		switch _next {
		// end of file
		case _EOF:
			return _line, nil

		// carriage return - we expect to see a newline next
		case _CR:
			_line = append(_line, _next)
			_next, _err = l.read()
			if _err != nil {
				return _line, _err
			} else if _next != _NEWLINE {
				l.unread(_next)
				return _line, l.err(CarriageReturnError)
			}
			fallthrough

		// newline
		case _NEWLINE:
			_line = append(_line, _next)
			return _line, nil
		}
	}
} // eol()

// whitespace returns all whitespace (i.e. ' ' and '\t') runes in a sequence,
// or an error if there is a problem reading the next runes.
func (l *lexer) whitespace() ([]rune, Error) {
	// read until we hit the first non-whitespace rune
	_ws := make([]rune, 0, 1)

	// loop until there's nothing more to do
	for {
		_next, _err := l.read()
		if _err != nil {
			return _ws, _err
		}

		// what is this next rune?
		switch _next {
		// space or tab is consumed
		case _SPACE:
			fallthrough
		case _TAB:
			break

		// non-whitespace rune
		default:
			// return the rune to the buffer and we're done
			l.unread(_next)
			return _ws, nil
		}

		// add this rune to the whitespace
		_ws = append(_ws, _next)
	}
} // whitespace()

// pattern returns all runes representing a file or path pattern, delimited
// either by unescaped whitespace, a path separator '/' or enf of file. An
// error is returned if a problem is encountered reading from the stream.
func (l *lexer) pattern() ([]rune, Error) {
	// read until we hit the first whitespace/end of line/eof rune
	_pattern := make([]rune, 0, 1)

	// loop until there's nothing more to do
	for {
		_r, _err := l.read()
		if _err != nil {
			return _pattern, _err
		}

		// what is the next rune?
		switch _r {
		// whitespace, newline, end of file, separator
		//		- this is the end of the pattern
		case _SPACE:
			fallthrough
		case _TAB:
			fallthrough
		case _CR:
			fallthrough
		case _NEWLINE:
			fallthrough
		case _SEPARATOR:
			fallthrough
		case _EOF:
			// return what we have
			l.unread(_r)
			return _pattern, nil

		// a wildcard is the end of the pattern if it is part of any '**'
		case _WILDCARD:
			_next, _err := l.peek()
			if _err != nil {
				return _pattern, _err
			} else if _next == _WILDCARD {
				l.unread(_r)
				return _pattern, _err
			} else {
				_pattern = append(_pattern, _r)
			}

		// escape sequence - consume the next rune
		case _ESCAPE:
			_escape, _err := l.escape()
			if _err != nil {
				return _pattern, _err
			}

			// add the escape sequence as part of the pattern
			_pattern = append(_pattern, _escape...)

		// any other character, we add to the pattern
		default:
			_pattern = append(_pattern, _r)
		}
	}
} // pattern()

// token returns a Token instance of the given type_ represented by word runes.
func (l *lexer) token(type_ TokenType, word []rune, e Error) (*Token, Error) {
	// if we have an error, then we return a BAD token
	if e != nil {
		type_ = BAD
	}

	// extract the lexer position
	//      - the column is taken from the current column position
	//        minus the length of the consumed "word"
	_word := len(word)
	_column := l._column - _word
	_offset := l._offset - _word
	position := Position{"", l._line, _column, _offset}

	// if this is a newline token, we adjust the line & column counts
	if type_ == EOL {
		l.newline()
	}

	// return the Token
	return NewToken(type_, word, position), e
} // token()

// err returns an Error encapsulating the error e and the current Lexer
// position.
func (l *lexer) err(e error) Error {
	// do we have an error?
	if e == nil {
		return nil
	} else {
		return NewError(e, l.Position())
	}
} // err()

// beginning returns true if the Lexer is at the start of a new line.
func (l *lexer) beginning() bool {
	return l._column == 1
} // beginning()

// ensure the lexer conforms to the lexer interface
var _ Lexer = &lexer{}
