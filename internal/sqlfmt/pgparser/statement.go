package pgparser

import (
	"fmt"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// ParseStatement parses a single SQL statement from input: a query (a SELECT
// or a set operation) or an INSERT, UPDATE, or DELETE, any of which may be
// led by a WITH clause.
func ParseStatement(input string) (pgast.Statement, error) {
	return parseStatementInput(input, (*Parser).parseStatementAfterWith)
}

// parseStatementInput parses input as one statement led by an optional WITH
// clause, with parseAfterWith parsing the rest of it, then allows a single
// trailing semicolon. It backs each exported statement entry point.
//
//nolint:nonamedreturns // the named error is mutated by the deferred recover
func parseStatementInput[T pgast.Statement](input string, parseAfterWith func(*Parser) T) (_ T, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	result := parseWithPrefixed(p, parseAfterWith)
	p.expectStatementEnd()

	return result, nil
}

// expectStatementEnd consumes an optional trailing semicolon and fails unless
// the input ends there.
func (p *Parser) expectStatementEnd() {
	p.consume(SEMICOLON)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}
}

// parseStatement parses a statement in a nested position that admits any
// statement kind, such as a CTE body.
func (p *Parser) parseStatement() pgast.Statement {
	return parseWithPrefixed(p, (*Parser).parseStatementAfterWith)
}

// parseStatementAfterWith parses a statement whose optional WITH clause has
// already been consumed, dispatching on its leading keyword.
func (p *Parser) parseStatementAfterWith() pgast.Statement {
	switch p.tok.Type {
	case INSERT:
		return p.parseInsertAfterWith()
	case UPDATE:
		return p.parseUpdateAfterWith()
	case DELETE:
		return p.parseDeleteAfterWith()
	default:
		return p.parseQueryAfterWith()
	}
}

// parseWithPrefixed parses an optional WITH clause followed by the statement
// parseAfterWith parses, and attaches the clause to that statement. Every
// statement form reads its WITH clause through here, so the clause is
// attached in one place regardless of the statement kind.
func parseWithPrefixed[T pgast.Statement](p *Parser, parseAfterWith func(*Parser) T) T {
	pos := p.tok.Pos
	with := p.parseOptionalWith()
	stmt := parseAfterWith(p)
	attachWith(stmt, with, pos)

	return stmt
}

// attachWith attaches with, parsed at pos, to stmt. stmt may already carry a
// WITH clause of its own when it is a parenthesized query, as in
// "WITH a AS (...) (WITH b AS (...) SELECT ...)"; like PostgreSQL, this
// rejects it rather than merging the two.
func attachWith(stmt pgast.Statement, with *pgast.With, pos Position) {
	if with == nil {
		return
	}

	var target **pgast.With

	switch s := stmt.(type) {
	case *pgast.Select:
		target = &s.With
	case *pgast.SetOp:
		target = &s.With
	case *pgast.Insert:
		target = &s.With
	case *pgast.Update:
		target = &s.With
	case *pgast.Delete:
		target = &s.With
	default:
		panic(&ParseError{Pos: pos, Msg: fmt.Sprintf("WITH is not supported before %T", stmt)})
	}

	if *target != nil {
		panic(&ParseError{Pos: pos, Msg: "multiple WITH clauses not allowed"})
	}

	*target = with
}
