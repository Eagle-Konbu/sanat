package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseStartTransaction parses a START TRANSACTION statement from input.
func ParseStartTransaction(input string) (*sqlast.StartTransaction, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseStartTransactionStatement)
}

// ParseBegin parses a BEGIN statement from input.
func ParseBegin(input string) (*sqlast.Begin, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseBeginStatement)
}

// ParseCommit parses a COMMIT statement from input.
func ParseCommit(input string) (*sqlast.Commit, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseCommitStatement)
}

// ParseRollback parses a ROLLBACK or ROLLBACK TO [SAVEPOINT] statement from input.
func ParseRollback(input string) (*sqlast.Rollback, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseRollbackStatement)
}

// ParseSavepoint parses a SAVEPOINT statement from input.
func ParseSavepoint(input string) (*sqlast.Savepoint, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseSavepointStatement)
}

// ParseReleaseSavepoint parses a RELEASE SAVEPOINT statement from input.
func ParseReleaseSavepoint(input string) (*sqlast.ReleaseSavepoint, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseReleaseSavepointStatement)
}

// ParseSetVariable parses a SET statement assigning a user-defined or
// session/global system variable from input, using ModeDefault.
func ParseSetVariable(input string) (*sqlast.SetVariable, error) {
	return ParseSetVariableWithMode(input, ModeDefault)
}

// ParseSetVariableWithMode parses a SET statement assigning a user-defined
// or session/global system variable from input, decoding string literals
// per mode.
func ParseSetVariableWithMode(input string, mode SQLMode) (*sqlast.SetVariable, error) {
	return parseDDLEntry(input, mode, (*Parser).parseSetVariableStatement)
}

// ParseSetNames parses a SET NAMES statement from input, using ModeDefault.
func ParseSetNames(input string) (*sqlast.SetNames, error) {
	return ParseSetNamesWithMode(input, ModeDefault)
}

// ParseSetNamesWithMode parses a SET NAMES statement from input, decoding
// string literals per mode.
func ParseSetNamesWithMode(input string, mode SQLMode) (*sqlast.SetNames, error) {
	return parseDDLEntry(input, mode, (*Parser).parseSetNamesStatement)
}

// parseSessionStatement dispatches a leading START, BEGIN, COMMIT, ROLLBACK,
// SAVEPOINT, RELEASE, or SET to its statement parser, split out of
// parseStatement to keep that switch's cyclomatic complexity down. It
// reports whether the current token started a recognized transaction/session
// statement.
func (p *Parser) parseSessionStatement() (sqlast.Statement, bool) {
	switch {
	case p.at(START):
		return p.parseStartTransactionStatement(), true
	case p.at(BEGIN):
		return p.parseBeginStatement(), true
	case p.at(COMMIT):
		return p.parseCommitStatement(), true
	case p.at(ROLLBACK):
		return p.parseRollbackStatement(), true
	case p.at(SAVEPOINT):
		return p.parseSavepointStatement(), true
	case p.at(RELEASE):
		return p.parseReleaseSavepointStatement(), true
	case p.at(SET):
		return p.parseSetStatement(), true
	default:
		return nil, false
	}
}

// parseStartTransactionStatement parses a START TRANSACTION statement,
// optionally followed by READ ONLY or READ WRITE. The current token must be
// START.
func (p *Parser) parseStartTransactionStatement() *sqlast.StartTransaction {
	p.expect(START)
	p.expect(TRANSACTION)

	st := &sqlast.StartTransaction{}

	if p.consume(READ) {
		switch {
		case p.consume(ONLY):
			st.Mode = sqlast.ReadOnlyMode
		case p.consume(WRITE):
			st.Mode = sqlast.ReadWriteMode
		default:
			p.failf("expected ONLY or WRITE after READ, got %s", p.tok.Type)
		}
	}

	return st
}

// parseBeginStatement parses a BEGIN [WORK] statement. The current token
// must be BEGIN.
func (p *Parser) parseBeginStatement() *sqlast.Begin {
	p.expect(BEGIN)
	p.consume(WORK)

	return &sqlast.Begin{}
}

// parseCommitStatement parses a COMMIT [WORK] statement. The current token
// must be COMMIT.
func (p *Parser) parseCommitStatement() *sqlast.Commit {
	p.expect(COMMIT)
	p.consume(WORK)

	return &sqlast.Commit{}
}

// parseRollbackStatement parses a ROLLBACK [WORK] statement, or a
// ROLLBACK [WORK] TO [SAVEPOINT] identifier statement. The current token
// must be ROLLBACK.
func (p *Parser) parseRollbackStatement() *sqlast.Rollback {
	p.expect(ROLLBACK)
	p.consume(WORK)

	if p.consume(TO) {
		p.consume(SAVEPOINT)

		return &sqlast.Rollback{SavepointName: sqlast.TableIdent(p.readIdent())}
	}

	return &sqlast.Rollback{}
}

// parseSavepointStatement parses a SAVEPOINT identifier statement. The
// current token must be SAVEPOINT.
func (p *Parser) parseSavepointStatement() *sqlast.Savepoint {
	p.expect(SAVEPOINT)

	return &sqlast.Savepoint{Name: sqlast.TableIdent(p.readIdent())}
}

// parseReleaseSavepointStatement parses a RELEASE SAVEPOINT identifier
// statement. The current token must be RELEASE.
func (p *Parser) parseReleaseSavepointStatement() *sqlast.ReleaseSavepoint {
	p.expect(RELEASE)
	p.expect(SAVEPOINT)

	return &sqlast.ReleaseSavepoint{Name: sqlast.TableIdent(p.readIdent())}
}

// parseSetStatement dispatches a leading SET to SET NAMES or a variable
// assignment. The current token must be SET.
func (p *Parser) parseSetStatement() sqlast.Statement {
	p.expect(SET)

	if p.consume(NAMES) {
		return p.parseSetNamesRest()
	}

	return p.parseSetVariableRest()
}

// parseSetNamesStatement parses a full SET NAMES statement. The current
// token must be SET.
func (p *Parser) parseSetNamesStatement() *sqlast.SetNames {
	p.expect(SET)
	p.expect(NAMES)

	return p.parseSetNamesRest()
}

// parseSetNamesRest parses the remainder of a SET NAMES statement after SET
// NAMES has been consumed: a charset, and an optional COLLATE collation.
func (p *Parser) parseSetNamesRest() *sqlast.SetNames {
	sn := &sqlast.SetNames{Charset: p.parseCharsetOrCollationName()}

	if p.consume(COLLATE) {
		sn.Collate = p.parseCharsetOrCollationName()
	}

	return sn
}

// parseCharsetOrCollationName parses a SET NAMES charset or COLLATE
// collation value: the DEFAULT keyword, a bare identifier, or a quoted
// string. Unlike DDL's DEFAULT/COMMENT values, this is deliberately not the
// full expression grammar — MySQL doesn't accept any other expression shape
// (arithmetic, function calls, qualified names, ...) in either position.
func (p *Parser) parseCharsetOrCollationName() sqlast.Expr {
	switch {
	case p.consume(DEFAULT):
		return &sqlast.Literal{Val: "DEFAULT"}
	case p.at(STRING):
		return p.parseStringLiteral()
	default:
		return &sqlast.Literal{Val: p.readIdent()}
	}
}

// parseSetVariableStatement parses a full SET statement assigning a
// user-defined or session/global system variable. The current token must be
// SET.
func (p *Parser) parseSetVariableStatement() *sqlast.SetVariable {
	p.expect(SET)

	return p.parseSetVariableRest()
}

// parseSetVariableRest parses the remainder of a variable-assigning SET
// statement after SET has been consumed: an optional SESSION/GLOBAL scope
// or a "@"-prefixed user variable, then "= expr".
func (p *Parser) parseSetVariableRest() *sqlast.SetVariable {
	sv := &sqlast.SetVariable{}

	switch {
	case p.consume(SESSION):
		sv.Scope = sqlast.SessionScope
		sv.Name = p.readIdent()
	case p.consume(GLOBAL):
		sv.Scope = sqlast.GlobalScope
		sv.Name = p.readIdent()
	case p.at(AtVariable):
		sv.IsUserVariable = true
		sv.Name = p.tok.Literal
		p.advance()
	default:
		sv.Name = p.readIdent()
	}

	p.expect(EQ)

	sv.Value = p.parseExpr()

	return sv
}
