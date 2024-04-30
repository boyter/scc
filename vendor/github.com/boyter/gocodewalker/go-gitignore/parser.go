// SPDX-License-Identifier: MIT

package gitignore

import (
	"io"
)

// Parser is the interface for parsing .gitignore files and extracting the set
// of patterns specified in the .gitignore file.
type Parser interface {
	// Parse returns all well-formed .gitignore Patterns contained within the
	// parser stream. Parsing will terminate at the end of the stream, or if
	// the parser error handler returns false.
	Parse() []Pattern

	// Next returns the next well-formed .gitignore Pattern from the parser
	// stream.  If an error is encountered, and the error handler is either
	// not defined, or returns true, Next will skip to the end of the current
	// line and attempt to parse the next Pattern. If the error handler
	// returns false, or the parser reaches the end of the stream, Next
	// returns nil.
	Next() Pattern

	// Position returns the current position of the parser in the input stream.
	Position() Position
} // Parser{}

// parser is the implementation of the .gitignore parser
type parser struct {
	_lexer Lexer
	_undo  []*Token
	_error func(Error) bool
} // parser{}

// NewParser returns a new Parser instance for the given stream r.
// If err is not nil, it will be called for every error encountered during
// parsing. Parsing will terminate at the end of the stream, or if err
// returns false.
func NewParser(r io.Reader, err func(Error) bool) Parser {
	return &parser{_lexer: NewLexer(r), _error: err}
} // NewParser()

// Parse returns all well-formed .gitignore Patterns contained within the
// parser stream. Parsing will terminate at the end of the stream, or if
// the parser error handler returns false.
func (p *parser) Parse() []Pattern {
	// keep parsing until there's no more patterns
	_patterns := make([]Pattern, 0)
	for {
		_pattern := p.Next()
		if _pattern == nil {
			return _patterns
		}
		_patterns = append(_patterns, _pattern)
	}
} // Parse()

// Next returns the next well-formed .gitignore Pattern from the parser stream.
// If an error is encountered, and the error handler is either not defined, or
// returns true, Next will skip to the end of the current line and attempt to
// parse the next Pattern. If the error handler returns false, or the parser
// reaches the end of the stream, Next returns nil.
func (p *parser) Next() Pattern {
	// keep searching until we find the next pattern, or until we
	// reach the end of the file
	for {
		_token, _err := p.next()
		if _err != nil {
			if !p.errors(_err) {
				return nil
			}

			// we got an error from the lexer, so skip the remainder
			// of this line and try again from the next line
			for _err != nil {
				_err = p.skip()
				if _err != nil {
					if !p.errors(_err) {
						return nil
					}
				}
			}
			continue
		}

		switch _token.Type {
		// we're at the end of the file
		case EOF:
			return nil

		// we have a blank line or comment
		case EOL:
			continue
		case COMMENT:
			continue

		// otherwise, attempt to build the next pattern
		default:
			_pattern, _err := p.build(_token)
			if _err != nil {
				if !p.errors(_err) {
					return nil
				}

				// we encountered an error parsing the retrieved tokens
				//      - skip to the end of the line
				for _err != nil {
					_err = p.skip()
					if _err != nil {
						if !p.errors(_err) {
							return nil
						}
					}
				}

				// skip to the next token
				continue
			} else if _pattern != nil {
				return _pattern
			}
		}
	}
} // Next()

// Position returns the current position of the parser in the input stream.
func (p *parser) Position() Position {
	// if we have any previously read tokens, then the token at
	// the end of the "undo" list (most recently "undone") gives the
	// position of the parser
	_length := len(p._undo)
	if _length != 0 {
		return p._undo[_length-1].Position
	}

	// otherwise, return the position of the lexer
	return p._lexer.Position()
} // Position()

//
// private methods
//

// build attempts to build a well-formed .gitignore Pattern starting from the
// given Token t. An Error will be returned if the sequence of tokens returned
// by the Lexer does not represent a valid Pattern.
func (p *parser) build(t *Token) (Pattern, Error) {
	// attempt to create a valid pattern
	switch t.Type {
	// we have a negated pattern
	case NEGATION:
		return p.negation(t)

	// attempt to build a path specification
	default:
		return p.path(t)
	}
} // build()

// negation attempts to build a well-formed negated .gitignore Pattern starting
// from the negation Token t. As with build, negation returns an Error if the
// sequence of tokens returned by the Lexer does not represent a valid Pattern.
func (p *parser) negation(t *Token) (Pattern, Error) {
	// a negation appears before a path specification, so
	// skip the negation token
	_next, _err := p.next()
	if _err != nil {
		return nil, _err
	}

	// extract the sequence of tokens for this path
	_tokens, _err := p.sequence(_next)
	if _err != nil {
		return nil, _err
	}

	// include the "negation" token at the front of the sequence
	_tokens = append([]*Token{t}, _tokens...)

	// return the Pattern instance
	return NewPattern(_tokens), nil
} // negation()

// path attempts to build a well-formed .gitignore Pattern representing a path
// specification, starting with the Token t. If the sequence of tokens returned
// by the Lexer does not represent a valid Pattern, path returns an Error.
// Trailing whitespace is dropped from the sequence of pattern tokens.
func (p *parser) path(t *Token) (Pattern, Error) {
	// extract the sequence of tokens for this path
	_tokens, _err := p.sequence(t)
	if _err != nil {
		return nil, _err
	}

	// remove trailing whitespace tokens
	_length := len(_tokens)
	for _length > 0 {
		// if we have a non-whitespace token, we can stop
		_length--
		if _tokens[_length].Type != WHITESPACE {
			break
		}

		// otherwise, truncate the token list
		_tokens = _tokens[:_length]
	}

	// return the Pattern instance
	return NewPattern(_tokens), nil
} // path()

// sequence attempts to extract a well-formed Token sequence from the Lexer
// representing a .gitignore Pattern. sequence returns an Error if the
// retrieved sequence of tokens does not represent a valid Pattern.
func (p *parser) sequence(t *Token) ([]*Token, Error) {
	// extract the sequence of tokens for a valid path
	//      - this excludes the negation token, which is handled as
	//        a special case before sequence() is called
	switch t.Type {
	// the path starts with a separator
	case SEPARATOR:
		return p.separator(t)

	// the path starts with the "any" pattern ("**")
	case ANY:
		return p.any(t)

	// the path starts with whitespace, wildcard or a pattern
	case WHITESPACE:
		fallthrough
	case PATTERN:
		return p.pattern(t)
	}

	// otherwise, we have an invalid specification
	p.undo(t)
	return nil, p.err(InvalidPatternError)
} // sequence()

// separator attempts to retrieve a valid sequence of tokens that may appear
// after the path separator '/' Token t. An Error is returned if the sequence if
// tokens is not valid, or if there is an error extracting tokens from the
// input stream.
func (p *parser) separator(t *Token) ([]*Token, Error) {
	// build a list of tokens that may appear after a separator
	_tokens := []*Token{t}
	_token, _err := p.next()
	if _err != nil {
		return _tokens, _err
	}

	// what tokens are we allowed to have follow a separator?
	switch _token.Type {
	// a separator can be followed by a pattern or
	// an "any" pattern (i.e. "**")
	case ANY:
		_next, _err := p.any(_token)
		return append(_tokens, _next...), _err

	case WHITESPACE:
		fallthrough
	case PATTERN:
		_next, _err := p.pattern(_token)
		return append(_tokens, _next...), _err

	// if we encounter end of line or file we are done
	case EOL:
		fallthrough
	case EOF:
		return _tokens, nil

	// a separator can be followed by another separator
	//      - it's not ideal, and not very useful, but it's interpreted
	//        as a single separator
	//      - we could clean it up here, but instead we pass
	//        everything down to the matching later on
	case SEPARATOR:
		_next, _err := p.separator(_token)
		return append(_tokens, _next...), _err
	}

	// any other token is invalid
	p.undo(_token)
	return _tokens, p.err(InvalidPatternError)
} // separator()

// any attempts to retrieve a valid sequence of tokens that may appear
// after the any '**' Token t. An Error is returned if the sequence if
// tokens is not valid, or if there is an error extracting tokens from the
// input stream.
func (p *parser) any(t *Token) ([]*Token, Error) {
	// build the list of tokens that may appear after "any" (i.e. "**")
	_tokens := []*Token{t}
	_token, _err := p.next()
	if _err != nil {
		return _tokens, _err
	}

	// what tokens are we allowed to have follow an "any" symbol?
	switch _token.Type {
	// an "any" token may only be followed by a separator
	case SEPARATOR:
		_next, _err := p.separator(_token)
		return append(_tokens, _next...), _err

	// whitespace is acceptable if it takes us to the end of the line
	case WHITESPACE:
		return _tokens, p.eol()

	// if we encounter end of line or file we are done
	case EOL:
		fallthrough
	case EOF:
		return _tokens, nil
	}

	// any other token is invalid
	p.undo(_token)
	return _tokens, p.err(InvalidPatternError)
} // any()

// pattern attempts to retrieve a valid sequence of tokens that may appear
// after the path pattern Token t. An Error is returned if the sequence if
// tokens is not valid, or if there is an error extracting tokens from the
// input stream.
func (p *parser) pattern(t *Token) ([]*Token, Error) {
	// build the list of tokens that may appear after a pattern
	_tokens := []*Token{t}
	_token, _err := p.next()
	if _err != nil {
		return _tokens, _err
	}

	// what tokens are we allowed to have follow a pattern?
	var _next []*Token
	switch _token.Type {
	case SEPARATOR:
		_next, _err = p.separator(_token)
		return append(_tokens, _next...), _err

	case WHITESPACE:
		fallthrough
	case PATTERN:
		_next, _err = p.pattern(_token)
		return append(_tokens, _next...), _err

	// if we encounter end of line or file we are done
	case EOL:
		fallthrough
	case EOF:
		return _tokens, nil
	}

	// any other token is invalid
	p.undo(_token)
	return _tokens, p.err(InvalidPatternError)
} // pattern()

// eol attempts to consume the next Lexer token to read the end of line or end
// of file. If a EOL or EOF is not reached , eol will return an error.
func (p *parser) eol() Error {
	// are we at the end of the line?
	_token, _err := p.next()
	if _err != nil {
		return _err
	}

	// have we encountered whitespace only?
	switch _token.Type {
	// if we're at the end of the line or file, we're done
	case EOL:
		fallthrough
	case EOF:
		p.undo(_token)
		return nil
	}

	// otherwise, we have an invalid pattern
	p.undo(_token)
	return p.err(InvalidPatternError)
} // eol()

// next returns the next token from the Lexer, or an error if there is a
// problem reading from the input stream.
func (p *parser) next() (*Token, Error) {
	// do we have any previously read tokens?
	_length := len(p._undo)
	if _length > 0 {
		_token := p._undo[_length-1]
		p._undo = p._undo[:_length-1]
		return _token, nil
	}

	// otherwise, attempt to retrieve the next token from the lexer
	return p._lexer.Next()
} // next()

// skip reads Tokens from the input until the end of line or end of file is
// reached. If there is a problem reading tokens, an Error is returned.
func (p *parser) skip() Error {
	// skip to the next end of line or end of file token
	for {
		_token, _err := p.next()
		if _err != nil {
			return _err
		}

		// if we have an end of line or file token, then we can stop
		switch _token.Type {
		case EOL:
			fallthrough
		case EOF:
			return nil
		}
	}
} // skip()

// undo returns the given Token t to the parser input stream to be retrieved
// again on a subsequent call to next.
func (p *parser) undo(t *Token) {
	// add this token to the list of previously read tokens
	//      - initialise the undo list if required
	if p._undo == nil {
		p._undo = make([]*Token, 0, 1)
	}
	p._undo = append(p._undo, t)
} // undo()

// err returns an Error for the error e, capturing the current parser Position.
func (p *parser) err(e error) Error {
	// convert the error to include the parser position
	return NewError(e, p.Position())
} // err()

// errors returns the response from the parser error handler to the Error e. If
// no error handler has been configured for this parser, errors returns true.
func (p *parser) errors(e Error) bool {
	// do we have an error handler?
	if p._error == nil {
		return true
	}

	// pass the error through to the error handler
	//      - if this returns false, parsing will stop
	return p._error(e)
} // errors()
