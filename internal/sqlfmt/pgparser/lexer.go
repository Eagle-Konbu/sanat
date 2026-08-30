package pgparser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const eof = rune(-1)

const unterminatedStringMsg = "unterminated string literal"

// LexError describes malformed input encountered while lexing.
type LexError struct {
	Pos Position
	Msg string
}

// Error implements the error interface, formatting the error's position and message.
func (e *LexError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// Lexer tokenizes a PostgreSQL SQL string, assuming standard_conforming_strings
// = on (the default since PostgreSQL 9.1, and the only mode this lexer
// supports): a plain '...' string has no backslash escapes, matching the
// opposite default from internal/sqlfmt/parser's MySQL lexer.
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
		return l.readIdentifierOrEscapeString(pos)
	case l.ch == '"':
		return l.readQuotedIdentToken(pos)
	case l.ch == '\'':
		lit, err := l.readString(pos, false)
		if err != nil {
			return Token{}, err
		}

		return Token{Type: STRING, Literal: lit, Pos: pos}, nil
	case l.ch == '$':
		return l.readDollarSigil(pos)
	case l.startsNumber():
		tt, lit := l.readNumber()

		return Token{Type: tt, Literal: lit, Pos: pos}, nil
	default:
		return l.readOperator(pos)
	}
}

// readIdentifierOrEscapeString reads an unquoted identifier/keyword, or an
// E'...' escape string if l.ch starts one. l.ch must satisfy isIdentStart.
// Unquoted identifiers are folded to lower case, matching PostgreSQL's
// default identifier-folding behavior (QuotedIdent preserves case instead).
func (l *Lexer) readIdentifierOrEscapeString(pos Position) (Token, error) {
	if (l.ch == 'e' || l.ch == 'E') && l.peek() == '\'' {
		l.readChar() // consume 'e'/'E'

		lit, err := l.readString(pos, true)
		if err != nil {
			return Token{}, err
		}

		return Token{Type: STRING, Literal: lit, Pos: pos}, nil
	}

	lit := l.readIdentifier()

	tt := lookupIdent(lit)
	if tt == IDENT {
		lit = strings.ToLower(lit)
	}

	return Token{Type: tt, Literal: lit, Pos: pos}, nil
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

// skipWhitespaceAndComments skips spaces, "--" line comments (PostgreSQL
// requires no boundary character after the dashes, unlike MySQL), and
// "/* ... */" block comments, which nest in PostgreSQL.
func (l *Lexer) skipWhitespaceAndComments() error {
	for {
		switch {
		case isSpace(l.ch):
			l.readChar()
		case l.ch == '-' && l.peek() == '-':
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

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != eof {
		l.readChar()
	}
}

func (l *Lexer) skipBlockComment() error {
	startPos := l.currentPos()
	l.readChar() // consume '/'
	l.readChar() // consume '*'

	depth := 1

	for depth > 0 {
		switch {
		case l.ch == eof:
			return &LexError{Pos: startPos, Msg: "unterminated block comment"}
		case l.ch == '/' && l.peek() == '*':
			l.readChar()
			l.readChar()

			depth++
		case l.ch == '*' && l.peek() == '/':
			l.readChar()
			l.readChar()

			depth--
		default:
			l.readChar()
		}
	}

	return nil
}

// readQuotedIdentToken reads a double-quoted identifier and wraps it as a
// QuotedIdent token, extracted from Next to keep its cyclomatic complexity down.
func (l *Lexer) readQuotedIdentToken(pos Position) (Token, error) {
	lit, err := l.readQuotedIdent(pos)
	if err != nil {
		return Token{}, err
	}

	return Token{Type: QuotedIdent, Literal: lit, Pos: pos}, nil
}

func (l *Lexer) readIdentifier() string {
	start := l.pos

	for isIdentPart(l.ch) {
		l.readChar()
	}

	return l.input[start:l.pos]
}

// readQuotedIdent reads a double-quoted identifier. A doubled double-quote
// represents a literal double-quote within the identifier.
func (l *Lexer) readQuotedIdent(startPos Position) (string, error) {
	l.readChar() // consume opening "

	var sb strings.Builder

	for {
		switch {
		case l.ch == eof:
			return "", &LexError{Pos: startPos, Msg: "unterminated quoted identifier"}
		case l.ch == '"' && l.peek() == '"':
			sb.WriteRune('"')
			l.readChar()
			l.readChar()
		case l.ch == '"':
			l.readChar() // consume closing "

			return sb.String(), nil
		default:
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}
}

// readString reads a single-quoted string literal, starting at the opening
// quote. Doubled-quote escaping (two consecutive ' characters) always
// applies; escapes reads PostgreSQL's E'...' backslash escape sequences
// (\n, \xHH, \uXXXX, octal, ...) in addition. l.ch must be the opening '.
func (l *Lexer) readString(startPos Position, escapes bool) (string, error) {
	l.readChar() // consume opening '

	var sb strings.Builder

	for {
		switch {
		case l.ch == eof:
			return "", &LexError{Pos: startPos, Msg: unterminatedStringMsg}
		case l.ch == '\'' && l.peek() == '\'':
			sb.WriteRune('\'')
			l.readChar()
			l.readChar()
		case l.ch == '\'':
			l.readChar() // consume closing '

			return sb.String(), nil
		case l.ch == '\\' && escapes:
			s, err := l.readEscapeSequence(startPos)
			if err != nil {
				return "", err
			}

			sb.WriteString(s)
		default:
			sb.WriteRune(l.ch)
			l.readChar()
		}
	}
}

// readEscapeSequence decodes one backslash escape sequence within an E'...'
// string. l.ch must be '\\'.
func (l *Lexer) readEscapeSequence(stringStart Position) (string, error) {
	l.readChar() // consume backslash

	switch {
	case l.ch == eof:
		return "", &LexError{Pos: stringStart, Msg: unterminatedStringMsg}
	case l.ch == 'x':
		return l.readHexEscape(stringStart)
	case l.ch == 'u':
		return l.readUnicodeEscape(stringStart, 4)
	case l.ch == 'U':
		return l.readUnicodeEscape(stringStart, 8)
	case isOctalDigit(l.ch):
		return l.readOctalEscape(), nil
	default:
		return l.readSimpleEscape(), nil
	}
}

func (l *Lexer) readSimpleEscape() string {
	ch := l.ch
	l.readChar()

	switch ch {
	case 'b':
		return "\b"
	case 'f':
		return "\f"
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 't':
		return "\t"
	default:
		return string(ch)
	}
}

func (l *Lexer) readOctalEscape() string {
	start := l.pos

	for range 3 {
		if !isOctalDigit(l.ch) {
			break
		}

		l.readChar()
	}

	n, _ := strconv.ParseInt(l.input[start:l.pos], 8, 32)

	return string(rune(n))
}

func (l *Lexer) readHexEscape(stringStart Position) (string, error) {
	l.readChar() // consume 'x'

	start := l.pos

	for range 2 {
		if !isHexDigit(l.ch) {
			break
		}

		l.readChar()
	}

	if l.pos == start {
		return "", &LexError{Pos: stringStart, Msg: `invalid hex escape: expected at least one hex digit after \x`}
	}

	n, _ := strconv.ParseInt(l.input[start:l.pos], 16, 32)

	return string(rune(n)), nil
}

func (l *Lexer) readUnicodeEscape(stringStart Position, digits int) (string, error) {
	l.readChar() // consume 'u'/'U'

	start := l.pos

	for range digits {
		if !isHexDigit(l.ch) {
			break
		}

		l.readChar()
	}

	if l.pos-start != digits {
		return "", &LexError{Pos: stringStart, Msg: fmt.Sprintf("invalid unicode escape: expected %d hex digits", digits)}
	}

	n, _ := strconv.ParseInt(l.input[start:l.pos], 16, 32)

	return string(rune(n)), nil
}

// readDollarSigil reads whatever follows a '$': a positional placeholder
// ($1, $2, ...) when a digit immediately follows, otherwise a dollar-quoted
// string ($$...$$ or $tag$...$tag$). l.ch must be '$'.
func (l *Lexer) readDollarSigil(startPos Position) (Token, error) {
	if isDigit(l.peek()) {
		l.readChar() // consume '$'

		start := l.pos
		l.readDigits()

		return Token{Type: Placeholder, Literal: l.input[start:l.pos], Pos: startPos}, nil
	}

	lit, err := l.readDollarQuotedString(startPos)
	if err != nil {
		return Token{}, err
	}

	return Token{Type: STRING, Literal: lit, Pos: startPos}, nil
}

// readDollarQuotedString reads a $tag$...$tag$ (or $$...$$) dollar-quoted
// string. No escaping is interpreted inside it. l.ch must be '$'.
func (l *Lexer) readDollarQuotedString(startPos Position) (string, error) {
	l.readChar() // consume opening '$'

	tagStart := l.pos

	for l.ch != '$' && l.ch != eof {
		l.readChar()
	}

	if l.ch == eof {
		return "", &LexError{Pos: startPos, Msg: "unterminated dollar-quoted string tag"}
	}

	tag := l.input[tagStart:l.pos]
	l.readChar() // consume closing '$' of the opening delimiter

	delimiter := "$" + tag + "$"
	contentStart := l.pos

	for {
		if l.ch == eof {
			return "", &LexError{Pos: startPos, Msg: "unterminated dollar-quoted string"}
		}

		if l.ch == '$' && strings.HasPrefix(l.input[l.pos:], delimiter) {
			content := l.input[contentStart:l.pos]

			for range len(delimiter) {
				l.readChar()
			}

			return content, nil
		}

		l.readChar()
	}
}

// startsNumber reports whether l.ch begins a numeric literal: an ordinary
// leading digit, or a '.' immediately followed by one (a leading-dot literal
// like ".5", with no integer part).
func (l *Lexer) startsNumber() bool {
	return isDigit(l.ch) || (l.ch == '.' && isDigit(l.peek()))
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

// tryReadFraction reads a fractional part starting at a '.', if present.
// The '.' must be followed by a digit: this is what disambiguates a
// leading-dot literal like ".5" from a number immediately followed by a
// DOT-qualified identifier, e.g. "123.col".
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
	case '-':
		return l.readMinusOrArrow(pos), nil
	case '#':
		return l.readHash(pos)
	case '@':
		return l.readAt(pos)
	case '?':
		return l.readQuestion(pos), nil
	case ':':
		return l.readColon(pos), nil
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
	case '@':
		l.readChar()

		return Token{Type: ContainedBy, Literal: "<@", Pos: pos}
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

func (l *Lexer) readMinusOrArrow(pos Position) Token {
	l.readChar() // consume '-'

	if l.ch != '>' {
		return Token{Type: MINUS, Literal: "-", Pos: pos}
	}

	l.readChar() // consume '>'

	if l.ch == '>' {
		l.readChar()

		return Token{Type: ArrowText, Literal: "->>", Pos: pos}
	}

	return Token{Type: Arrow, Literal: "->", Pos: pos}
}

func (l *Lexer) readHash(pos Position) (Token, error) {
	if l.peek() != '>' {
		ch := l.ch
		l.readChar()

		return Token{}, &LexError{Pos: pos, Msg: fmt.Sprintf("unexpected character %q", ch)}
	}

	l.readChar() // consume '#'
	l.readChar() // consume '>'

	if l.ch == '>' {
		l.readChar()

		return Token{Type: HashArrowText, Literal: "#>>", Pos: pos}, nil
	}

	return Token{Type: HashArrow, Literal: "#>", Pos: pos}, nil
}

func (l *Lexer) readAt(pos Position) (Token, error) {
	if l.peek() != '>' {
		ch := l.ch
		l.readChar()

		return Token{}, &LexError{Pos: pos, Msg: fmt.Sprintf("unexpected character %q", ch)}
	}

	l.readChar() // consume '@'
	l.readChar() // consume '>'

	return Token{Type: Contains, Literal: "@>", Pos: pos}, nil
}

func (l *Lexer) readQuestion(pos Position) Token {
	l.readChar() // consume '?'

	switch l.ch {
	case '|':
		l.readChar()

		return Token{Type: JSONExistsAny, Literal: "?|", Pos: pos}
	case '&':
		l.readChar()

		return Token{Type: JSONExistsAll, Literal: "?&", Pos: pos}
	default:
		return Token{Type: JSONExists, Literal: "?", Pos: pos}
	}
}

func (l *Lexer) readColon(pos Position) Token {
	l.readChar() // consume ':'

	if l.ch == ':' {
		l.readChar()

		return Token{Type: DoubleColon, Literal: "::", Pos: pos}
	}

	return Token{Type: COLON, Literal: ":", Pos: pos}
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
	'*': STAR,
	'/': SLASH,
	'%': PERCENT,
	'(': LPAREN,
	')': RPAREN,
	'[': LBRACKET,
	']': RBRACKET,
	',': COMMA,
	'.': DOT,
	';': SEMICOLON,
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isOctalDigit(ch rune) bool {
	return ch >= '0' && ch <= '7'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

// isIdentStart reports whether ch can start an unquoted identifier: a letter
// or underscore. Unlike sqlfmt/parser's MySQL lexer, '$' is deliberately
// excluded — PostgreSQL reserves a leading '$' for placeholders and
// dollar-quote delimiters.
func isIdentStart(ch rune) bool {
	return ch == '_' || unicode.IsLetter(ch)
}

// isIdentPart reports whether ch can appear after the first character of an
// unquoted identifier. PostgreSQL allows '$' here (just not as the leading
// character), unlike sqlfmt/parser's MySQL lexer, which allows it anywhere.
func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == '$'
}
