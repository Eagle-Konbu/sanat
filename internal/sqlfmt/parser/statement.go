package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseStatement parses a single top-level SQL statement — SELECT, INSERT,
// UPDATE, DELETE, a UNION of SELECT branches, a DDL statement (CREATE TABLE,
// ALTER TABLE, CREATE INDEX, DROP INDEX, DROP TABLE, TRUNCATE TABLE), a
// transaction/session statement (START TRANSACTION, BEGIN, COMMIT, ROLLBACK,
// SAVEPOINT, RELEASE SAVEPOINT, SET), or an admin/utility statement (SHOW
// TABLES/CREATE TABLE/COLUMNS/INDEX/DATABASES/VARIABLES/STATUS, DESCRIBE,
// EXPLAIN, USE), optionally preceded by a WITH clause (except before
// INSERT/REPLACE or any DDL/transaction/session/admin statement, none of
// which MySQL allows WITH before — EXPLAIN's wrapped select_stmt accepts its
// own WITH clause instead) — dispatching on the statement's leading keyword.
// This is the formatter's entry point.
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
		if stmt, ok := p.parseStatementWithoutWith(); ok {
			return stmt
		}
	}

	msg := "expected SELECT, INSERT, UPDATE, DELETE, REPLACE, CREATE, ALTER, DROP, TRUNCATE, " +
		"START, BEGIN, COMMIT, ROLLBACK, SAVEPOINT, RELEASE, SET, SHOW, DESCRIBE, EXPLAIN, or USE, got %s"

	return failReturn[sqlast.Statement](p, msg, p.tok.Type)
}

// parseStatementWithoutWith dispatches to the DDL, transaction/session, or
// admin/utility statement parsers, split out of parseStatement to keep that
// switch's cyclomatic complexity down — none of these statement kinds accept
// a leading WITH clause. It reports whether the current token started a
// recognized statement of one of those kinds.
func (p *Parser) parseStatementWithoutWith() (sqlast.Statement, bool) {
	if stmt, ok := p.parseDDLStatement(); ok {
		return stmt, true
	}

	if stmt, ok := p.parseSessionStatement(); ok {
		return stmt, true
	}

	return p.parseAdminStatement()
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
