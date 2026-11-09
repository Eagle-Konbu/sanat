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
		return l.readNumberToken(pos)
	default:
		return l.readOperator(pos)
	}
}

// readIdentifierOrEscapeString reads an unquoted identifier/keyword, or a
// prefixed string constant (E'...', B'...', X'...', or N'...') if l.ch starts
// one. l.ch must satisfy isIdentStart. Unquoted identifiers are folded to
// lower case, matching PostgreSQL's default identifier-folding behavior
// (QuotedIdent preserves case instead).
func (l *Lexer) readIdentifierOrEscapeString(pos Position) (Token, error) {
	if l.peek() == '\'' {
		if tok, ok, err := l.tryReadPrefixedString(pos); ok || err != nil {
			return tok, err
		}
	}

	if l.startsUnicodeEscapeConstant() {
		return Token{}, &LexError{Pos: pos, Msg: "Unicode escape strings (U&'...') are not supported"}
	}

	lit, err := l.readIdentifier()
	if err != nil {
		return Token{}, err
	}

	tt := lookupIdent(lit)
	if tt == IDENT {
		lit = asciiLower(lit)
	}

	return Token{Type: tt, Literal: lit, Pos: pos}, nil
}

// tryReadPrefixedString reads a string constant whose one-letter prefix is
// l.ch and whose opening quote follows it, reporting ok=false without
// consuming anything if l.ch is not such a prefix.
func (l *Lexer) tryReadPrefixedString(pos Position) (Token, bool, error) {
	prefix := unicode.ToUpper(l.ch)

	var (
		tt      TokenType
		escapes bool
	)

	switch prefix {
	case 'E':
		tt, escapes = STRING, true
	case 'N':
		tt = NationalString
	case 'B', 'X':
		tt = BitString
	default:
		return Token{}, false, nil
	}

	l.readChar() // consume the prefix

	lit, err := l.readString(pos, escapes)
	if err != nil {
		return Token{}, true, err
	}

	if tt == BitString {
		if !isBitStringContent(prefix, lit) {
			return Token{}, true, &LexError{Pos: pos, Msg: fmt.Sprintf("invalid %c'...' bit-string constant", prefix)}
		}

		lit = string(prefix) + "'" + lit + "'"
	}

	return Token{Type: tt, Literal: lit, Pos: pos}, true, nil
}

// isBitStringContent reports whether content holds only binary digits for a
// B'...' constant, or only hexadecimal digits for an X'...' one.
func isBitStringContent(prefix rune, content string) bool {
	for _, ch := range content {
		if prefix == 'B' && ch != '0' && ch != '1' || prefix == 'X' && !isHexDigit(ch) {
			return false
		}
	}

	return true
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

// skipBlockComment skips a block comment, counting nested "/*" ... "*/"
// pairs, since PostgreSQL ends the comment only at the "*/" matching the
// outermost "/*". An unterminated comment is reported at its opening "/*".
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

// readIdentifier reads an unquoted identifier, rejecting bytes that are not
// valid UTF-8, which isIdentStart would otherwise accept as non-ASCII.
func (l *Lexer) readIdentifier() (string, error) {
	start := l.pos

	for isIdentPart(l.ch) {
		if l.atInvalidUTF8() {
			return "", &LexError{Pos: l.currentPos(), Msg: "invalid UTF-8 in identifier"}
		}

		l.readChar()
	}

	return l.input[start:l.pos], nil
}

// atInvalidUTF8 reports whether ch was decoded from an invalid UTF-8 byte,
// as opposed to a literal U+FFFD in the input.
func (l *Lexer) atInvalidUTF8() bool {
	return l.ch == utf8.RuneError && l.readPos-l.pos == 1
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
		return l.readOctalEscape(stringStart)
	default:
		return l.readSimpleEscape(), nil
	}
}

// readSimpleEscape decodes a single-character escape. A backslash before
// any character without a special meaning yields that character, so
// E'it\'s' is "it's" and E'\q' is "q", as in PostgreSQL.
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

// readOctalEscape decodes a \o, \oo, or \ooo escape: one to three octal
// digits.
func (l *Lexer) readOctalEscape(stringStart Position) (string, error) {
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
	return escapedByte(stringStart, byte(n&0xFF))
}

// escapedByte returns the byte an octal or hex escape denotes, rejecting a
// zero byte, which PostgreSQL cannot store in a text value.
func escapedByte(stringStart Position, b byte) (string, error) {
	if b == 0 {
		return "", &LexError{Pos: stringStart, Msg: "invalid escape: string constants cannot contain a zero byte"}
	}

	return string([]byte{b}), nil
}

// readHexEscape decodes a \xh or \xhh escape: one or two hex digits, which
// denote a raw byte like an octal escape does.
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

	return escapedByte(stringStart, byte(n))
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

// readLowSurrogateEscape reads the escape that must immediately follow a
// high surrogate. As in PostgreSQL, either the \u or the \U form may supply
// the low surrogate, regardless of which form supplied the high one.
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
// unicode escape, rejecting zero and values beyond U+10FFFF. Surrogates are
// left for the caller to validate.
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
	if err != nil || n == 0 || n > unicode.MaxRune {
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

		if isIdentStart(l.ch) {
			return Token{}, &LexError{Pos: startPos, Msg: "trailing junk after parameter"}
		}

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

// readNumberToken reads a numeric literal. Like PostgreSQL 15+, it rejects
// an identifier character directly after the number ("123abc", "0x1F"),
// instead of reading it as the number followed by an alias.
func (l *Lexer) readNumberToken(pos Position) (Token, error) {
	tt, lit := l.readNumber()

	if isIdentStart(l.ch) {
		return Token{}, &LexError{Pos: pos, Msg: "trailing junk after numeric literal"}
	}

	return Token{Type: tt, Literal: lit, Pos: pos}, nil
}

// readNumber reads an unsigned numeric literal, typed FLOAT if it has a
// fractional part or an exponent and INT otherwise. A leading sign is never
// part of the literal: it lexes as a separate operator.
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
// numeric), except that "1..2" leaves ".." intact for the parser.
func (l *Lexer) tryReadFraction() bool {
	if l.ch != '.' || l.peek() == '.' {
		return false
	}

	l.readChar() // consume '.'
	l.readDigits()

	return true
}

// tryReadExponent reads an exponent ("e5", "E-3") only when a digit follows
// the 'e' and optional sign, so in "1e" or "1ea" the 'e' is left to lex as
// the start of an identifier.
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

// readColon reads ':' and the operators beginning with it, which are not
// op_chars in PostgreSQL: "::" (cast), ":=" (named argument), or a lone ':'
// (array slice bound separator).
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
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v'
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

// isIdentStart reports whether ch can start an unquoted identifier: an ASCII
// letter, an underscore, or any non-ASCII character. PostgreSQL's scanner
// accepts every byte from 0x80 up, so combining marks and symbols are
// identifier characters too. Unlike sqlfmt/parser's MySQL lexer, '$' is
// deliberately excluded — PostgreSQL reserves a leading '$' for placeholders
// and dollar-quote delimiters.
func isIdentStart(ch rune) bool {
	return ch == '_' || ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= utf8.RuneSelf
}

// isIdentPart reports whether ch can appear after the first character of an
// unquoted identifier. PostgreSQL allows '$' here (just not as the leading
// character), unlike sqlfmt/parser's MySQL lexer, which allows it anywhere.
func isIdentPart(ch rune) bool {
	return isIdentStart(ch) || isDigit(ch) || ch == '$'
}
