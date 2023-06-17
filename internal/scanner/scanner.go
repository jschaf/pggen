package scanner

import (
	"bytes"
	"fmt"
	"github.com/jschaf/pggen/internal/token"
	gotok "go/token"
	"unicode"
	"unicode/utf8"
)

const (
	eof = -1
	bom = 0xFEFF // byte order mark, only permitted as very first character
)

// An ErrorHandler may be provided to Scanner.Init. If a syntax error is
// encountered and a handler was installed, the handler is called with a
// position and an error message. The position points to the beginning of
// the offending token.
type ErrorHandler func(pos gotok.Position, msg string)

// A Scanner holds the scanner's internal state while processing a given text.
// It can be allocated as part of another data structure but must be initialized
// via Init before use.
type Scanner struct {
	// immutable state
	file *gotok.File  // source file handle
	src  []byte       // source code
	err  ErrorHandler // error reporting; or nil

	// scanning state
	ch       rune        // current character
	offset   int         // character offset
	rdOffset int         // reading offset (position after current character)
	prev     token.Token // previous token
	prevCh   rune        // previous character
}

// TemplateQuery represents a single parsed SQL TemplateQuery.
type TemplateQuery struct {
	Comments []string
	// Name of the query, from the comment preceding the query.
	// Like 'FindAuthors' in:
	//     -- Name: FindAuthors :many
	Name string
	// The SQL as it appeared in the source query file.
	SQL string
}

// Read the next Unicode char into s.ch.
// s.ch < 0 means end-of-file.
func (s *Scanner) next() {
	if s.rdOffset < len(s.src) {
		s.offset = s.rdOffset
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		r, w := rune(s.src[s.rdOffset]), 1
		switch {
		case r == 0:
			s.error(s.offset, "illegal character NUL")
		case r >= utf8.RuneSelf:
			// not ASCII
			r, w = utf8.DecodeRune(s.src[s.rdOffset:])
			if r == utf8.RuneError && w == 1 {
				s.error(s.offset, "illegal UTF-8 encoding")
			} else if r == bom && s.offset > 0 {
				s.error(s.offset, "illegal byte order mark")
			}
		}
		s.rdOffset += w
		s.prevCh = s.ch
		s.ch = r
	} else {
		s.offset = len(s.src)
		if s.ch == '\n' {
			s.file.AddLine(s.offset)
		}
		s.prevCh = s.ch
		s.ch = eof
	}
}

func (s *Scanner) error(offs int, msg string) {
	if s.err != nil {
		s.err(s.file.Position(s.file.Pos(offs)), msg)
	}
}

func (s *Scanner) errorf(offset int, format string, args ...interface{}) {
	s.error(offset, fmt.Sprintf(format, args...))
}

// Init prepares the scanner s to tokenize the text src by setting the scanner
// at the beginning of src. The scanner uses the file set file for position
// information and it adds line information for each line. It is ok to re-use
// the same file when re-scanning the same file as line information which is
// already present is ignored. Init causes a panic if the file size does not
// match the src size.
//
// Calls to Scan will invoke the error handler err if they encounter a syntax
// error and err is not nil.
//
// Note that Init may call err if there is an error in the first character
// of the file.
func (s *Scanner) Init(file *gotok.File, src []byte, err ErrorHandler) {
	// Explicitly initialize all fields since a scanner may be reused.
	if file.Size() != len(src) {
		panic(fmt.Sprintf("file size (%d) does not match src len (%d)", file.Size(), len(src)))
	}
	s.file = file
	s.src = src
	s.err = err

	s.ch = ' '
	s.offset = 0
	s.rdOffset = 0

	s.next()
	if s.ch == bom {
		s.next() // ignore BOM at file beginning
	}
}

// peek returns the byte following the most recently read character without
// advancing the scanner. If the scanner is at EOF, peek returns 0.
func (s *Scanner) peek() byte {
	if s.rdOffset < len(s.src) {
		return s.src[s.rdOffset]
	}
	return 0
}

func (s *Scanner) skipWhitespace() {
	for isSpace(s.ch) {
		s.next()
	}
}

func lower(ch rune) rune     { return ('a' - 'A') | ch } // returns lower-case ch iff ch is ASCII letter
func isDecimal(ch rune) bool { return '0' <= ch && ch <= '9' }
func isSpace(ch rune) bool   { return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' }

func isLetter(ch rune) bool {
	return 'a' <= lower(ch) && lower(ch) <= 'z' || ch == '_' || ch >= utf8.RuneSelf && unicode.IsLetter(ch)
}

func (s *Scanner) scanLineComment() string {
	offs := s.offset
	for s.ch != '\n' && s.ch >= 0 {
		s.next()
	}
	return string(s.src[offs:s.offset])
}

func (s *Scanner) scanBlockComment() (token.Token, string) {
	offs := s.offset
	s.next() // consume '/'
	s.next() // consume '*'
	depth := 1

	for depth > 0 {
		if s.ch == eof {
			s.error(offs, "unterminated block comment")
			return token.Illegal, ""
		}
		if s.ch == '*' && s.peek() == '/' {
			s.next()
			s.next()
			depth--
			continue
		}
		if s.ch == '/' && s.peek() == '*' {
			s.next()
			s.next()
			depth++
			continue
		}
		s.next()
	}
	return token.BlockComment, string(s.src[offs:s.offset])
}

func (s *Scanner) scanSingleQuoteString() (token.Token, string) {
	offs := s.offset
	s.next() // consume the opening single quote
	for s.ch > 0 {
		if s.ch == '\'' {
			if s.peek() == '\'' {
				// Consecutive single quotes is a literal single quote.
				// https://www.postgresql.org/docs/13/sql-syntax-lexical.html#SQL-SYNTAX-CONSTANTS
				s.next()
				s.next()
				continue
			} else {
				s.next() // consume closing single quote
				return token.String, string(s.src[offs:s.offset])
			}
		}
		s.next()
	}
	s.errorf(offs, "unterminated single-quote string literal: %s", string(s.src[offs:s.offset]))
	return token.Illegal, ""
}

func (s *Scanner) scanDollarQuoteString() (token.Token, string) {
	offs := s.offset
	// opening tag
	s.next() // consume opening dollar sign of start tag
	for s.ch != '$' {
		if s.ch == eof {
			s.errorf(offs, "unterminated dollar-quoted string: %s", string(s.src[offs:s.offset]))
			return token.Illegal, ""
		}
		if !isLetter(s.ch) && !isDecimal(s.ch) {
			s.errorf(offs, "invalid dollar quoted tag: %s", string(s.src[offs:s.offset]))
			return token.Illegal, ""
		}
		s.next()
	}
	s.next() // consume closing dollar sign for start tag
	tag := s.src[offs:s.offset]

	// string contents
	idx := bytes.Index(s.src[s.offset:], tag)
	if idx == -1 {
		s.errorf(offs, "no closing delimiter found for dollar quoted string: %s", string(s.src[offs:s.offset]))
		return token.Illegal, ""
	}
	for i := 0; i < idx; i++ {
		s.next()
	}

	// closing tag
	for i := 0; i < len(tag); i++ {
		s.next()
	}
	return token.String, string(s.src[offs:s.offset])
}

func (s *Scanner) scanDoubleQuoteString() (token.Token, string) {
	offs := s.offset
	s.next() // consume the opening double quote
	for s.ch > 0 {
		if s.ch == '"' {
			if s.peek() == '"' {
				// Consecutive double quotes is a literal double quote.
				// https://www.postgresql.org/docs/13/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS
				s.next()
				s.next()
				continue
			} else {
				s.next() // consume closing double quote
				return token.QuotedIdent, string(s.src[offs:s.offset])
			}
		}
		s.next()
	}
	s.errorf(offs, "unterminated double-quote string literal: %s", string(s.src[offs:s.offset]))
	return token.Illegal, ""
}

// scanQueryFragment scans any piece of a query that's not a string or comment.
func (s *Scanner) scanQueryFragment() (token.Token, string) {
	offs := s.offset
	for s.ch > 0 {
		switch {
		case s.ch == eof:
			str := string(s.src[offs:s.offset])
			s.error(offs, "unterminated query: "+str)
			return token.Illegal, str
		case s.ch == ';':
			return token.QueryFragment, string(s.src[offs:s.offset])
		case s.ch == '-' && s.peek() == '-':
			return token.QueryFragment, string(s.src[offs:s.offset])
		case s.ch == '/' && s.peek() == '*':
			return token.QueryFragment, string(s.src[offs:s.offset])
		case s.ch == '\'' || s.ch == '"':
			return token.QueryFragment, string(s.src[offs:s.offset])
		case s.ch == '$':
			// A dollar sign can be part of an identifier. Consume the identifier
			// here for cases like 'select 1 as foo$$$$bar'.
			if isLetter(s.prevCh) || isDecimal(s.prevCh) {
				for isLetter(s.ch) || isDecimal(s.ch) || s.ch == '$' {
					s.next()
				}
				continue
			} else {
				return token.QueryFragment, string(s.src[offs:s.offset])
			}
		}
		s.next()
	}
	return token.QueryFragment, string(s.src[offs:s.offset])
}

// Scan scans the next token and returns the token position, the token, and its
// literal string if applicable. The source end is indicated by token.EOF.
//
// If the returned token is a literal, or token.TexComment, the literal string
// has the corresponding value.
//
// If the returned token is token.Illegal, the literal string is the offending
// character.
//
// In all other cases, Scan returns an empty literal string.
//
// For more tolerant parsing, Scan will return a valid token if possible even
// if a syntax error was encountered. Thus, even if the resulting token sequence
// contains no illegal tokens, a client may not assume that no error has
// occurred. Instead, the client must check the scanner's ErrorCount or the
// number of calls of the error handler, if there was one installed.
//
// Token positions are relative to the file.
func (s *Scanner) Scan() (pos gotok.Pos, tok token.Token, lit string) {
	s.skipWhitespace()
	pos = s.file.Pos(s.offset)

	switch s.ch {
	case eof:
		tok = token.EOF
	case '-':
		if s.peek() == '-' {
			tok = token.LineComment
			lit = s.scanLineComment()
		} else {
			tok, lit = s.scanQueryFragment()
		}
	case '/':
		if s.peek() == '*' {
			tok, lit = s.scanBlockComment()
		} else {
			tok, lit = s.scanQueryFragment()
		}
	case '\'':
		tok, lit = s.scanSingleQuoteString()
	case '$':
		tok, lit = s.scanDollarQuoteString()
	case '"':
		tok, lit = s.scanDoubleQuoteString()
	case ';':
		s.next()
		tok = token.Semicolon
	default:
		tok, lit = s.scanQueryFragment()
	}

	s.prev = tok
	return
}
