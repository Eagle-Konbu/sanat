package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eof = rune(-1)

// LexError describes malformed input encountered while lexing.
type LexError struct {
	Pos Position
	Msg string
}

func (e *LexError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// Lexer tokenizes a MySQL SQL string.
type Lexer struct {
	input   string
	pos     int // byte offset of ch
	readPos int // byte offset of the next rune to read
	ch      rune
	line    int
	col     int // rune column of ch, 1-based
}

// New creates a Lexer over input.
func New(input string) *Lexer {
	l := &Lexer{input: input, line: 1}
	l.readChar()

	return l
}

// Next returns the next token in the input, or an error if the input is malformed.
// The final token before an error-free end of input has Type EOF.
func (l *Lexer) Next() (Token, error) {
	if err := l.skipWhitespaceAndComments(); err != nil {
		return Token{}, err
	}

	pos := l.currentPos()

	switch {
	case l.ch == eof:
		return Token{Type: EOF, Pos: pos}, nil
	case isIdentStart(l.ch):
		lit := l.readIdentifier()

		return Token{Type: lookupIdent(lit), Literal: lit, Pos: pos}, nil
	case l.ch == '`':
		lit, err := l.readQuotedIdent(pos)
		if err != nil {
			return Token{}, err
		}

		return Token{Type: QuotedIdent, Literal: lit, Pos: pos}, nil
	case l.ch == '\'':
		lit, err := l.readString(pos)
		if err != nil {
			return Token{}, err
		}

		return Token{Type: STRING, Literal: lit, Pos: pos}, nil
	case isDigit(l.ch):
		tt, lit := l.readNumber()

		return Token{Type: tt, Literal: lit, Pos: pos}, nil
	default:
		return l.readOperator(pos)
	}
}

func (l *Lexer) currentPos() Position {
	return Position{Offset: l.pos, Line: l.line, Column: l.col}
}

func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}

	if l.readPos >= len(l.input) {
		l.pos = l.readPos
		l.ch = eof
		l.col++

		return
	}

	r, w := utf8.DecodeRuneInString(l.input[l.readPos:])
	l.pos = l.readPos
	l.readPos += w
	l.ch = r
	l.col++
}

// peekAt returns the nth rune after ch (n=1 is the immediate next rune)
// without consuming any input.
func (l *Lexer) peekAt(n int) rune {
	offset := l.readPos

	var r rune

	for range n {
		if offset >= len(l.input) {
			return eof
		}

		var w int

		r, w = utf8.DecodeRuneInString(l.input[offset:])
		offset += w
	}

	return r
}

func (l *Lexer) peek() rune { return l.peekAt(1) }

func (l *Lexer) skipWhitespaceAndComments() error {
	for {
		switch {
		case isSpace(l.ch):
			l.readChar()
		case l.ch == '#':
			l.skipLineComment()
		case l.isDashComment():
			l.readChar()
			l.readChar()
			l.skipLineComment()
		case l.ch == '/' && l.peek() == '*':
			if err := l.skipBlockComment(); err != nil {
				return err
			}
		default:
			return nil
		}
	}
}

func (l *Lexer) isDashComment() bool {
	return l.ch == '-' && l.peek() == '-' && isCommentBoundary(l.peekAt(2))
}

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != eof {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() error {
	startPos := l.currentPos()
	l.readChar() // consume '/'
	l.readChar() // consume '*'

	for {
		switch {
		case l.ch == eof:
			return &LexError{Pos: startPos, Msg: "unterminated block comment"}
		case l.ch == '*' && l.peek() == '/':
			l.readChar()
			l.readChar()

			return nil
		default:
			l.readChar()
		}
	}
}

func (l *Lexer) readIdentifier() string {
	start := l.pos

	for isIdentPart(l.ch) {
		l.readChar()
	}

	return l.input[start:l.pos]
}

// readQuotedIdent reads a backtick-quoted identifier. A doubled backtick
// represents a literal backtick within the identifier.
func (l *Lexer) readQuotedIdent(startPos Position) (string, error) {
	l.readChar() // consume opening `

	var sb strings.Builder

	for {
		switch {
		case l.ch == eof:
			return "", &LexError{Pos: startPos, Msg: "unterminated quoted identifier"}
		case l.ch == '`' && l.peek() == '`':
			sb.WriteRune('`')
			l.readChar()
			l.readChar()
		case l.ch == '`':
			l.readChar() // consume closing `

			return sb.String(), nil
		default:
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}
}

// readString reads a single-quoted string literal. It supports MySQL's
// doubled-quote escaping as well as backslash escape sequences.
func (l *Lexer) readString(startPos Position) (string, error) {
	l.readChar() // consume opening '

	var sb strings.Builder

	for {
		switch {
		case l.ch == eof:
			return "", &LexError{Pos: startPos, Msg: "unterminated string literal"}
		case l.ch == '\'' && l.peek() == '\'':
			sb.WriteRune('\'')
			l.readChar()
			l.readChar()
		case l.ch == '\'':
			l.readChar() // consume closing '

			return sb.String(), nil
		case l.ch == '\\':
			r, err := l.readEscapedRune(startPos)
			if err != nil {
				return "", err
			}

			sb.WriteRune(r)
		default:
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}
}

func (l *Lexer) readEscapedRune(stringStart Position) (rune, error) {
	l.readChar() // consume backslash

	if l.ch == eof {
		return 0, &LexError{Pos: stringStart, Msg: "unterminated string literal"}
	}

	r := unescape(l.ch)
	l.readChar()

	return r, nil
}

// unescape maps a character following a backslash to the rune it represents,
// per MySQL's escape sequence rules. Any character not in the recognized set
// (e.g. \\, \', \%, \_) represents itself.
func unescape(ch rune) rune {
	switch ch {
	case '0':
		return '\x00'
	case 'b':
		return '\b'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case 'Z':
		return '\x1a'
	default:
		return ch
	}
}

func (l *Lexer) readNumber() (TokenType, string) {
	start := l.pos

	l.readDigits()

	tt := INT
	if l.tryReadFraction() {
		tt = FLOAT
	}

	if l.tryReadExponent() {
		tt = FLOAT
	}

	return tt, l.input[start:l.pos]
}

func (l *Lexer) readDigits() {
	for isDigit(l.ch) {
		l.readChar()
	}
}

func (l *Lexer) tryReadFraction() bool {
	if l.ch != '.' || !isDigit(l.peek()) {
		return false
	}

	l.readChar() // consume '.'
	l.readDigits()

	return true
}

func (l *Lexer) tryReadExponent() bool {
	if l.ch != 'e' && l.ch != 'E' {
		return false
	}

	expDigitsOffset := 1
	if sign := l.peekAt(1); sign == '+' || sign == '-' {
		expDigitsOffset = 2
	}

	if !isDigit(l.peekAt(expDigitsOffset)) {
		return false
	}

	l.readChar() // consume 'e'/'E'

	if l.ch == '+' || l.ch == '-' {
		l.readChar()
	}

	l.readDigits()

	return true
}

func (l *Lexer) readOperator(pos Position) (Token, error) {
	switch l.ch {
	case '<':
		return l.readLess(pos), nil
	case '>':
		return l.readGreater(pos), nil
	case '!':
		return l.readBang(pos)
	default:
		return l.readSingleCharToken(pos)
	}
}

func (l *Lexer) readLess(pos Position) Token {
	l.readChar() // consume '<'

	switch l.ch {
	case '>':
		l.readChar()

		return Token{Type: NE, Literal: "<>", Pos: pos}
	case '=':
		l.readChar()

		return Token{Type: LE, Literal: "<=", Pos: pos}
	default:
		return Token{Type: LT, Literal: "<", Pos: pos}
	}
}

func (l *Lexer) readGreater(pos Position) Token {
	l.readChar() // consume '>'

	if l.ch == '=' {
		l.readChar()

		return Token{Type: GE, Literal: ">=", Pos: pos}
	}

	return Token{Type: GT, Literal: ">", Pos: pos}
}

func (l *Lexer) readBang(pos Position) (Token, error) {
	if l.peek() == '=' {
		l.readChar()
		l.readChar()

		return Token{Type: NE, Literal: "!=", Pos: pos}, nil
	}

	ch := l.ch
	l.readChar()

	return Token{}, &LexError{Pos: pos, Msg: fmt.Sprintf("unexpected character %q", ch)}
}

func (l *Lexer) readSingleCharToken(pos Position) (Token, error) {
	ch := l.ch

	tt, ok := singleCharTokens[ch]
	if !ok {
		l.readChar()

		return Token{}, &LexError{Pos: pos, Msg: fmt.Sprintf("unexpected character %q", ch)}
	}

	l.readChar()

	return Token{Type: tt, Literal: string(ch), Pos: pos}, nil
}

var singleCharTokens = map[rune]TokenType{
	'=': EQ,
	'+': PLUS,
	'-': MINUS,
	'*': STAR,
	'/': SLASH,
	'%': PERCENT,
	'(': LPAREN,
	')': RPAREN,
	',': COMMA,
	'.': DOT,
	':': COLON,
	'?': QUESTION,
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isCommentBoundary(ch rune) bool {
	return ch == eof || isSpace(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch rune) bool {
	return ch == '_' || ch == '$' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch)
}
