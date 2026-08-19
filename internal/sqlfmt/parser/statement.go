package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseStatement parses a single top-level SQL statement — SELECT, INSERT,
// UPDATE, DELETE, a UNION of SELECT branches, a DDL statement (CREATE TABLE,
// ALTER TABLE, CREATE INDEX, DROP INDEX, DROP TABLE, TRUNCATE TABLE), or a
// transaction/session statement (START TRANSACTION, BEGIN, COMMIT, ROLLBACK,
// SAVEPOINT, RELEASE SAVEPOINT, SET), optionally preceded by a WITH clause
// (except before INSERT/REPLACE or any DDL/transaction/session statement,
// none of which MySQL allows WITH before) — dispatching on the statement's
// leading keyword. This is the formatter's entry point.
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
	case with == nil:
		if stmt, ok := p.parseDDLStatement(); ok {
			return stmt
		}

		if stmt, ok := p.parseSessionStatement(); ok {
			return stmt
		}
	}

	msg := "expected SELECT, INSERT, UPDATE, DELETE, REPLACE, CREATE, ALTER, DROP, TRUNCATE, " +
		"START, BEGIN, COMMIT, ROLLBACK, SAVEPOINT, RELEASE, or SET, got %s"

	return failReturn[sqlast.Statement](p, msg, p.tok.Type)
}

// parseDDLStatement dispatches a leading CREATE, ALTER, DROP, or TRUNCATE to
// its statement parser, split out of parseStatement to keep that switch's
// cyclomatic complexity down. It reports whether the current token started
// a recognized DDL statement.
func (p *Parser) parseDDLStatement() (sqlast.Statement, bool) {
	switch {
	case p.at(CREATE):
		return p.parseCreateStatement(), true
	case p.at(ALTER):
		return p.parseAlterTableStatement(), true
	case p.at(DROP):
		return p.parseDropStatement(), true
	case p.at(TRUNCATE):
		return p.parseTruncateTableStatement(), true
	default:
		return nil, false
	}
}
