package parser

import (
	"fmt"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// ParseError describes malformed input encountered while parsing.
type ParseError struct {
	Pos Position
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// Parser parses a token stream produced by a Lexer into sqlast nodes.
//
// A lexer failure or a syntax error is signaled by panicking with an error
// value (a *LexError or *ParseError), recovered at the top-level ParseExpr/
// ParseSelect entry points via recoverParseError. Internal parseXxx methods
// therefore build and return sqlast nodes directly rather than the usual
// (T, error) pair — the alternative would bury nearly every one of the many
// grammar productions in this file under a repeated `if err != nil { return
// nil, err }` that can only ever fire on a lexer-level failure.
type Parser struct {
	lex      *Lexer
	tok      Token
	peekTok  Token
	peek2Tok Token
}

// NewParser creates a Parser over input, priming its three-token lookahead
// buffer. It panics (with a *LexError) if the input cannot be lexed at all.
func NewParser(input string) *Parser {
	p := &Parser{lex: New(input)}

	for range 3 {
		p.advance()
	}

	return p
}

// advance consumes the current token and shifts the lookahead buffer,
// lexing one new token into the tail slot. It panics on a lexer failure.
func (p *Parser) advance() {
	p.tok = p.peekTok
	p.peekTok = p.peek2Tok

	tok, err := p.lex.Next()
	if err != nil {
		panic(err)
	}

	p.peek2Tok = tok
}

func (p *Parser) at(tt TokenType) bool      { return p.tok.Type == tt }
func (p *Parser) peekAt(tt TokenType) bool  { return p.peekTok.Type == tt }
func (p *Parser) peek2At(tt TokenType) bool { return p.peek2Tok.Type == tt }

func (p *Parser) errorf(format string, args ...any) *ParseError {
	return &ParseError{Pos: p.tok.Pos, Msg: fmt.Sprintf(format, args...)}
}

// failf aborts parsing with a syntax error at the current position.
func (p *Parser) failf(format string, args ...any) {
	panic(p.errorf(format, args...))
}

// expect verifies the current token has type tt and consumes it, failing if
// it doesn't match.
func (p *Parser) expect(tt TokenType) {
	if !p.at(tt) {
		p.failf("expected %s, got %s", tt, p.tok.Type)
	}

	p.advance()
}

// failReturn aborts parsing like failf, but is for use in a `return`
// statement from a function whose result type isn't void: fail's panic
// means the zero value it returns is never actually reached, but Go still
// requires a value of the right type to satisfy the return statement.
func failReturn[T any](p *Parser, format string, args ...any) T {
	p.failf(format, args...)

	var zero T

	return zero
}

// consume advances past the current token if it has type tt, reporting
// whether it did so.
func (p *Parser) consume(tt TokenType) bool {
	if !p.at(tt) {
		return false
	}

	p.advance()

	return true
}

// recoverParseError recovers a panic raised by this package and assigns it
// to *err; any other panic (a real bug, not a parse error) keeps propagating.
func recoverParseError(err *error) {
	r := recover()
	if r == nil {
		return
	}

	e, ok := r.(error)
	if !ok {
		panic(r)
	}

	*err = e
}

// ParseExpr parses a single SQL expression from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseExpr(input string) (expr sqlast.Expr, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	expr = p.parseExpr()

	if !p.at(EOF) {
		p.failf("unexpected token %s after expression", p.tok.Type)
	}

	return expr, nil
}

// ParseSelect parses a SELECT statement (optionally preceded by a WITH
// clause) from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseSelect(input string) (sel *sqlast.Select, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	sel = p.parseSelectStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return sel, nil
}
