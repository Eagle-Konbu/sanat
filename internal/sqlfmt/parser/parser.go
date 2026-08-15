package parser

import (
	"fmt"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// ParseExpr parses a single SQL expression from input.
func ParseExpr(input string) (sqlast.Expr, error) {
	p, err := NewParser(input)
	if err != nil {
		return nil, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if !p.at(EOF) {
		return nil, p.errorf("unexpected token %s after expression", p.tok.Type)
	}

	return expr, nil
}

// ParseSelect parses a SELECT statement (optionally preceded by a WITH
// clause) from input.
func ParseSelect(input string) (*sqlast.Select, error) {
	p, err := NewParser(input)
	if err != nil {
		return nil, err
	}

	stmt, err := p.parseSelectStatement()
	if err != nil {
		return nil, err
	}

	if !p.at(EOF) {
		return nil, p.errorf("unexpected token %s after statement", p.tok.Type)
	}

	return stmt, nil
}

// ParseError describes malformed input encountered while parsing.
type ParseError struct {
	Pos Position
	Msg string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%d:%d: %s", e.Pos.Line, e.Pos.Column, e.Msg)
}

// Parser parses a token stream produced by a Lexer into sqlast nodes. It
// keeps a three-token lookahead buffer: the grammar needs two tokens of
// lookahead past the current one to tell `table.*` apart from `table.col`.
type Parser struct {
	lex      *Lexer
	tok      Token
	peekTok  Token
	peek2Tok Token
}

// NewParser creates a Parser over input. It returns an error if the input
// cannot be lexed at all.
func NewParser(input string) (*Parser, error) {
	p := &Parser{lex: New(input)}

	for range 3 {
		if err := p.advance(); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// advance consumes the current token and shifts the lookahead buffer,
// lexing one new token into the tail slot.
func (p *Parser) advance() error {
	p.tok = p.peekTok
	p.peekTok = p.peek2Tok

	tok, err := p.lex.Next()
	if err != nil {
		return err
	}

	p.peek2Tok = tok

	return nil
}

func (p *Parser) at(tt TokenType) bool      { return p.tok.Type == tt }
func (p *Parser) peekAt(tt TokenType) bool  { return p.peekTok.Type == tt }
func (p *Parser) peek2At(tt TokenType) bool { return p.peek2Tok.Type == tt }

func (p *Parser) errorf(format string, args ...any) error {
	return &ParseError{Pos: p.tok.Pos, Msg: fmt.Sprintf(format, args...)}
}

// expect verifies the current token has type tt and consumes it.
func (p *Parser) expect(tt TokenType) error {
	if !p.at(tt) {
		return p.errorf("expected %s, got %s", tt, p.tok.Type)
	}

	return p.advance()
}

// consume advances past the current token if it has type tt, reporting
// whether it did so.
func (p *Parser) consume(tt TokenType) (bool, error) {
	if !p.at(tt) {
		return false, nil
	}

	if err := p.advance(); err != nil {
		return false, err
	}

	return true, nil
}
