package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseStatement parses a single top-level SQL statement — SELECT, INSERT,
// UPDATE, DELETE, or a UNION of SELECT branches, optionally preceded by a
// WITH clause (except before INSERT, which MySQL doesn't allow) — dispatching
// on the statement's leading keyword. This is the formatter's entry point.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseStatement(input string) (stmt sqlast.Statement, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	stmt = p.parseStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return stmt, nil
}

func (p *Parser) parseStatement() sqlast.Statement {
	with := p.parseOptionalWith()

	switch {
	case p.at(SELECT):
		return p.parseSelectOrUnionAfterWith(with)
	case p.at(UPDATE):
		return p.parseUpdateStatementAfterWith(with)
	case p.at(DELETE):
		return p.parseDeleteStatementAfterWith(with)
	case with == nil && (p.at(INSERT) || p.at(REPLACE)):
		return p.parseInsertStatement()
	default:
		return failReturn[sqlast.Statement](p, "expected SELECT, INSERT, UPDATE, DELETE, or REPLACE, got %s", p.tok.Type)
	}
}
