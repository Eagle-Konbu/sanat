package parser

import (
	"fmt"
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
		return l.readIdentifierOrPrefixedLiteral(pos)
	case l.ch == '`':
		return l.readQuotedIdentToken(pos)
	case l.ch == '@':
		return l.readAtVariable(pos)
	case l.ch == '\'':
		lit, err := l.readString(pos)
		if err != nil {
			return Token{}, err
		}

		return Token{Type: STRING, Literal: lit, Pos: pos}, nil
	case l.startsNumber():
		tt, lit := l.readNumber()

		return Token{Type: tt, Literal: lit, Pos: pos}, nil
	default:
		return l.readOperator(pos)
	}
}

// readIdentifierOrPrefixedLiteral reads an unquoted identifier/keyword, or a
// x'..'/b'..' prefixed hex/bit string literal if l.ch starts one. l.ch must
// satisfy isIdentStart.
func (l *Lexer) readIdentifierOrPrefixedLiteral(pos Position) (Token, error) {
	if tok, ok, err := l.tryReadPrefixedStringLiteral(pos); err != nil || ok {
		return tok, err
	}

	lit := l.readIdentifier()

	return Token{Type: lookupIdent(lit), Literal: lit, Pos: pos}, nil
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

	// /*+ ... */ optimizer hints and /*! ... */ (or /*!50700 ... */
	// version-gated) executable comments carry semantic content that a
	// plain comment does not. Discarding them like an ordinary block
	// comment would silently drop meaning from the formatted output, so
	// this is reported as a lex error instead: callers fall back to
	// returning the input unchanged rather than mangling it.
	if l.ch == '+' || l.ch == '!' {
		return &LexError{Pos: startPos, Msg: "MySQL optimizer hints and executable comments are not supported"}
	}

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

// tryReadPrefixedStringLiteral reads MySQL's x'..'/b'..' hex- and
// bit-string literal forms. These are only recognized when the quote
// immediately follows the x/b letter with no space, which is also how they
// are distinguished from a plain identifier named x/b (always followed by
// whitespace or an operator before any string literal, in valid SQL). l.ch
// must satisfy isIdentStart; if it isn't 'x'/'X'/'b'/'B' immediately
// followed by a quote, this consumes nothing and reports ok=false.
func (l *Lexer) tryReadPrefixedStringLiteral(startPos Position) (Token, bool, error) {
	if l.peek() != '\'' {
		return Token{}, false, nil
	}

	var tt TokenType

	switch l.ch {
	case 'x', 'X':
		tt = HexStr
	case 'b', 'B':
		tt = BitStr
	default:
		return Token{}, false, nil
	}

	prefix := l.ch

	l.readChar()
	l.readChar()

	content, err := l.readPrefixedStringContent(startPos, tt)
	if err != nil {
		return Token{}, false, err
	}

	if tt == HexStr && len(content)%2 != 0 {
		return Token{}, false, &LexError{Pos: startPos, Msg: "hex literal must have an even number of digits"}
	}

	l.readChar() // consume closing '

	return Token{Type: tt, Literal: string(prefix) + "'" + content + "'", Pos: startPos}, true, nil
}

// readPrefixedStringContent reads the digit content of an x'..'/b'..'
// literal up to (not including) the closing quote, validating that every
// character is a legal digit for tt: hex digits for HexStr, 0/1 for BitStr.
// l.ch must be positioned just after the opening quote.
func (l *Lexer) readPrefixedStringContent(startPos Position, tt TokenType) (string, error) {
	start := l.pos

	isValidDigit := isHexDigit
	if tt == BitStr {
		isValidDigit = isBinDigit
	}

	for l.ch != '\'' {
		if l.ch == eof {
			return "", &LexError{Pos: startPos, Msg: unterminatedStringMsg}
		}

		if !isValidDigit(l.ch) {
			return "", &LexError{Pos: startPos, Msg: fmt.Sprintf("invalid digit %q in string literal", l.ch)}
		}

		l.readChar()
	}

	return l.input[start:l.pos], nil
}

// readQuotedIdentToken reads a backtick-quoted identifier and wraps it as a
// QuotedIdent token, extracted from Next to keep its cyclomatic complexity
// down.
func (l *Lexer) readQuotedIdentToken(pos Position) (Token, error) {
	lit, err := l.readQuotedIdent(pos)
	if err != nil {
		return Token{}, err
	}

	return Token{Type: QuotedIdent, Literal: lit, Pos: pos}, nil
}

// readAtVariable reads a user-defined variable reference: '@' followed by an
// identifier, e.g. @my_var. The Literal carries only the name, without the
// leading '@'. l.ch must be '@'.
func (l *Lexer) readAtVariable(startPos Position) (Token, error) {
	l.readChar() // consume '@'

	if !isIdentStart(l.ch) {
		return Token{}, &LexError{Pos: startPos, Msg: "expected identifier after '@'"}
	}

	return Token{Type: AtVariable, Literal: l.readIdentifier(), Pos: startPos}, nil
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
			return "", &LexError{Pos: startPos, Msg: unterminatedStringMsg}
		case l.ch == '\'' && l.peek() == '\'':
			sb.WriteRune('\'')
			l.readChar()
			l.readChar()
		case l.ch == '\'':
			l.readChar() // consume closing '

			return sb.String(), nil
		case l.ch == '\\':
			s, err := l.readEscapedString(startPos)
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

func (l *Lexer) readEscapedString(stringStart Position) (string, error) {
	l.readChar() // consume backslash

	if l.ch == eof {
		return "", &LexError{Pos: stringStart, Msg: unterminatedStringMsg}
	}

	s := unescape(l.ch)
	l.readChar()

	return s, nil
}

// unescape maps a character following a backslash to the string it represents,
// per MySQL's escape sequence rules. \% and \_ retain the backslash: they are
// only unescaped within LIKE pattern-matching, which this lexer does not
// interpret. Any other character not in the recognized set represents itself
// with the backslash dropped.
func unescape(ch rune) string {
	switch ch {
	case '0':
		return "\x00"
	case 'b':
		return "\b"
	case 'n':
		return "\n"
	case 'r':
		return "\r"
	case 't':
		return "\t"
	case 'Z':
		return "\x1a"
	case '%', '_':
		return "\\" + string(ch)
	default:
		return string(ch)
	}
}

// startsNumber reports whether l.ch begins a numeric literal: an ordinary
// leading digit, or a '.' immediately followed by one (a leading-dot literal
// like ".5", with no integer part).
func (l *Lexer) startsNumber() bool {
	return isDigit(l.ch) || (l.ch == '.' && isDigit(l.peek()))
}

func (l *Lexer) readNumber() (TokenType, string) {
	if l.ch == '0' {
		if lit, ok := l.tryReadPrefixedNumber(); ok {
			return INT, lit
		}
	}

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

// tryReadPrefixedNumber reads a 0x.../0b... hex or binary integer literal.
// l.ch must be '0'. Per MySQL, the x/b letter must be lowercase in this
// notation (0X.../0B... is not a valid integer literal, unlike the X'..'/B'..'
// string forms, which accept either case).
func (l *Lexer) tryReadPrefixedNumber() (string, bool) {
	start := l.pos

	switch {
	case l.peek() == 'x' && isHexDigit(l.peekAt(2)):
		l.readChar() // consume '0'
		l.readChar() // consume 'x'

		for isHexDigit(l.ch) {
			l.readChar()
		}

		return l.input[start:l.pos], true
	case l.peek() == 'b' && isBinDigit(l.peekAt(2)):
		l.readChar() // consume '0'
		l.readChar() // consume 'b'

		for isBinDigit(l.ch) {
			l.readChar()
		}

		return l.input[start:l.pos], true
	default:
		return "", false
	}
}

func (l *Lexer) readDigits() {
	for isDigit(l.ch) {
		l.readChar()
	}
}

// tryReadFraction reads a fractional part starting at a '.', if present.
// The '.' must be followed by a digit: this is what disambiguates a
// leading-dot literal like ".5" (see the matching case in Next()) from a
// number immediately followed by a DOT-qualified identifier, e.g. "123.col".
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

		if l.ch == '>' {
			l.readChar()

			return Token{Type: NSE, Literal: "<=>", Pos: pos}
		}

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
	';': SEMICOLON,
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

// isCommentBoundary reports whether ch is what MySQL requires immediately
// after "--" for it to start a comment: whitespace or a control character.
// EOF doesn't count — a bare trailing "--" with nothing after it is not a
// comment start, matching MySQL's stricter enforcement of this rule.
func isCommentBoundary(ch rune) bool {
	return isSpace(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch rune) bool {
	return isDigit(ch) || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isBinDigit(ch rune) bool {
	return ch == '0' || ch == '1'
}

func isIdentStart(ch rune) bool {
	return ch == '_' || ch == '$' || unicode.IsLetter(ch)
}

func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch)
}
