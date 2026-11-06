package pgparser

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
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

	// signRunEnd is the byte offset where the trailing '+'/'-' characters
	// split off the last operator end. Each of them is a one-character
	// operator, so the run is not scanned again for each of them.
	signRunEnd int
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

	if l.startsUnicodeEscapeConstant() {
		return Token{}, &LexError{Pos: pos, Msg: "Unicode escape strings (U&'...') are not supported"}
	}

	lit := l.readIdentifier()

	tt := lookupIdent(lit)
	if tt == IDENT {
		lit = asciiLower(lit)
	}

	return Token{Type: tt, Literal: lit, Pos: pos}, nil
}

// startsUnicodeEscapeConstant reports whether l.ch starts an unsupported
// U&'...' string or U&"..." identifier, which would otherwise lex as the
// identifier u, the & operator, and a string, silently changing the
// statement's meaning.
func (l *Lexer) startsUnicodeEscapeConstant() bool {
	next := l.peekAt(2)

	return (l.ch == 'u' || l.ch == 'U') && l.peek() == '&' && (next == '\'' || next == '"')
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

			if sb.Len() == 0 {
				return "", &LexError{Pos: startPos, Msg: "zero-length quoted identifier"}
			}

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
//
// Adjacent string constants separated only by whitespace containing a newline
// are joined into one, per PostgreSQL's string-continuation rule.
func (l *Lexer) readString(startPos Position, escapes bool) (string, error) {
	l.readChar() // consume opening '

	var sb strings.Builder

	for {
		switch {
		case l.ch == eof:
			return "", &LexError{Pos: startPos, Msg: unterminatedStringMsg}
		case l.ch == '\'':
			if l.readQuoteInString(&sb) {
				return sb.String(), nil
			}
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

// readQuoteInString consumes a ' inside a string body and reports whether it
// ended the string: a doubled quote writes a literal ' to sb, and a closing
// quote followed by a continuation string resumes at the continuation's body.
func (l *Lexer) readQuoteInString(sb *strings.Builder) bool {
	if l.peek() == '\'' {
		sb.WriteByte('\'')
		l.readChar()
		l.readChar()

		return false
	}

	l.readChar() // consume closing '

	if !l.skipToStringContinuation() {
		return true
	}

	l.readChar() // consume the continuation's opening '

	return false
}

// skipToStringContinuation advances to the opening quote of a continuation
// string constant and reports true, or leaves the lexer unchanged and reports
// false. Like PostgreSQL's scanner, only whitespace and "--" comments may
// separate the two parts, and at least one newline is required; block
// comments end the string.
func (l *Lexer) skipToStringContinuation() bool {
	saved := *l
	sawNewline := false

	for {
		switch {
		case l.ch == '\n' || l.ch == '\r':
			sawNewline = true

			l.readChar()
		case isSpace(l.ch):
			l.readChar()
		case l.ch == '-' && l.peek() == '-':
			l.skipLineComment()
		case l.ch == '\'' && sawNewline:
			return true
		default:
			*l = saved

			return false
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

	n, _ := strconv.ParseUint(l.input[start:l.pos], 8, 16)

	// Octal and hex escapes denote raw bytes, not code points: E'\303\251'
	// is the two-byte UTF-8 encoding of 'é'. Like PostgreSQL, values above
	// \377 are truncated to their low byte.
	return string([]byte{byte(n & 0xFF)})
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

	n, _ := strconv.ParseUint(l.input[start:l.pos], 16, 8)

	return string([]byte{byte(n)}), nil
}

// readUnicodeEscape decodes a \uXXXX or \UXXXXXXXX escape. A UTF-16 high
// surrogate must be immediately followed by a low-surrogate escape, and the
// pair is combined into one code point, as PostgreSQL does.
func (l *Lexer) readUnicodeEscape(stringStart Position, digits int) (string, error) {
	cp, err := l.readUnicodeCodePoint(stringStart, digits)
	if err != nil {
		return "", err
	}

	switch {
	case isLowSurrogate(cp):
		return "", &LexError{Pos: stringStart, Msg: "invalid unicode surrogate pair: unexpected low surrogate"}
	case isHighSurrogate(cp):
		low, err := l.readLowSurrogateEscape(stringStart)
		if err != nil {
			return "", err
		}

		cp = utf16.DecodeRune(cp, low)
	}

	return string(cp), nil
}

func (l *Lexer) readLowSurrogateEscape(stringStart Position) (rune, error) {
	errMissing := &LexError{Pos: stringStart, Msg: "invalid unicode surrogate pair: expected low surrogate escape"}

	if l.ch != '\\' {
		return 0, errMissing
	}

	var digits int

	switch l.peek() {
	case 'u':
		digits = 4
	case 'U':
		digits = 8
	default:
		return 0, errMissing
	}

	l.readChar() // consume backslash

	low, err := l.readUnicodeCodePoint(stringStart, digits)
	if err != nil {
		return 0, err
	}

	if !isLowSurrogate(low) {
		return 0, errMissing
	}

	return low, nil
}

// readUnicodeCodePoint reads the 'u'/'U' and exactly digits hex digits of a
// unicode escape, rejecting values beyond U+10FFFF. Surrogates are left for
// the caller to validate.
func (l *Lexer) readUnicodeCodePoint(stringStart Position, digits int) (rune, error) {
	l.readChar() // consume 'u'/'U'

	start := l.pos

	for range digits {
		if !isHexDigit(l.ch) {
			break
		}

		l.readChar()
	}

	if l.pos-start != digits {
		return 0, &LexError{Pos: stringStart, Msg: fmt.Sprintf("invalid unicode escape: expected %d hex digits", digits)}
	}

	n, err := strconv.ParseUint(l.input[start:l.pos], 16, 32)
	if err != nil || n > unicode.MaxRune {
		return 0, &LexError{Pos: stringStart, Msg: "invalid unicode escape value"}
	}

	return rune(n), nil
}

func isHighSurrogate(r rune) bool { return r >= 0xD800 && r <= 0xDBFF }

func isLowSurrogate(r rune) bool { return r >= 0xDC00 && r <= 0xDFFF }

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

	// A tag follows unquoted-identifier rules, except '$' cannot appear in it.
	if isIdentStart(l.ch) {
		for isIdentPart(l.ch) && l.ch != '$' {
			l.readChar()
		}
	}

	if l.ch != '$' {
		return "", &LexError{Pos: startPos, Msg: "invalid or unterminated dollar-quote tag"}
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

			for range utf8.RuneCountInString(delimiter) {
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
// As in PostgreSQL, the digits after the '.' are optional ("1." is a valid
// numeric, so "1.col" lexes as "1." followed by "col"), except that "1..2"
// leaves ".." intact for the parser.
func (l *Lexer) tryReadFraction() bool {
	if l.ch != '.' || l.peek() == '.' {
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

// readOperator reads an operator or punctuation token. A run of operator
// characters is one operator, as in PostgreSQL's scanner (scan.l), so any
// operator, including a user-defined one, lexes as a single token.
func (l *Lexer) readOperator(pos Position) (Token, error) {
	switch {
	case l.ch == ':':
		return l.readColon(pos), nil
	case l.pos < l.signRunEnd:
		text := string(l.ch)
		l.readChar()

		return Token{Type: operatorTokenType(text), Literal: text, Pos: pos}, nil
	case isOperatorChar(l.ch):
		text, runLen := operatorText(l.input[l.pos:])
		l.signRunEnd = l.pos + runLen

		for range len(text) {
			l.readChar()
		}

		return Token{Type: operatorTokenType(text), Literal: text, Pos: pos}, nil
	default:
		return l.readSingleCharToken(pos)
	}
}

// isOperatorChar reports whether ch can be part of an operator
// (PostgreSQL's op_chars). All of them are ASCII.
func isOperatorChar(ch rune) bool {
	return strings.ContainsRune("~!@#^&|`?+-*/%<>=", ch)
}

// operatorText returns the operator at the start of input, following
// scan.l: the longest run of operator characters, cut short where a "--" or
// "/*" comment starts. For SQL compatibility a multi-character operator may
// not end in '+' or '-' unless it also contains a character SQL's own
// operators never use, so "=-1" lexes as "=" then "-1" but "?-" stays whole.
// It also returns the length of the run before that trailing '+'/'-' is
// dropped; what was dropped lexes as one-character operators.
func operatorText(input string) (string, int) {
	n := 0
	for n < len(input) && isOperatorChar(rune(input[n])) {
		n++
	}

	run := input[:n]
	if i := commentStart(run); i >= 0 {
		run = run[:i]
	}

	if strings.ContainsAny(run, "~!@#^&|`?%") {
		return run, len(run)
	}

	return run[:max(1, len(strings.TrimRight(run, "+-")))], len(run)
}

// commentStart returns the index of the first "--" or "/*" in text, or -1.
func commentStart(text string) int {
	dashes, slashStar := strings.Index(text, "--"), strings.Index(text, "/*")

	switch {
	case dashes < 0:
		return slashStar
	case slashStar < 0:
		return dashes
	default:
		return min(dashes, slashStar)
	}
}

// fixedOperators are the operators that keep their own token type, because
// the grammar gives them a precedence level or meaning of their own. Every
// other operator is an Op.
var fixedOperators = map[string]TokenType{
	"=":  EQ,
	"<>": NE,
	"!=": NE,
	"<":  LT,
	">":  GT,
	"<=": LE,
	">=": GE,
	"+":  PLUS,
	"-":  MINUS,
	"*":  STAR,
	"/":  SLASH,
	"%":  PERCENT,
	"^":  CARET,
	"=>": EqualsGreater,
}

func operatorTokenType(text string) TokenType {
	if tt, ok := fixedOperators[text]; ok {
		return tt
	}

	return Op
}

func (l *Lexer) readColon(pos Position) Token {
	l.readChar() // consume ':'

	switch l.ch {
	case ':':
		l.readChar()

		return Token{Type: DoubleColon, Literal: "::", Pos: pos}
	case '=':
		l.readChar()

		return Token{Type: ColonEquals, Literal: ":=", Pos: pos}
	default:
		return Token{Type: COLON, Literal: ":", Pos: pos}
	}
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
	'(': LPAREN,
	')': RPAREN,
	'[': LBRACKET,
	']': RBRACKET,
	',': COMMA,
	'.': DOT,
	';': SEMICOLON,
}

func isSpace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f'
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
