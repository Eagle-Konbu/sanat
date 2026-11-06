package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseUpdate parses an UPDATE statement, optionally led by a WITH clause,
// from input.
func ParseUpdate(input string) (*pgast.Update, error) {
	return parseStatementInput(input, (*Parser).parseUpdateAfterWith)
}

// parseUpdateAfterWith parses an UPDATE statement whose optional WITH clause
// has already been consumed. The current token must be UPDATE.
//
// PostgreSQL's FROM clause joins the target against other tables for use in
// SET/WHERE; unlike MySQL's comma-joined multi-table UPDATE, the target
// itself is always a single table, so the two grammars share no logic.
func (p *Parser) parseUpdateAfterWith() *pgast.Update {
	p.expect(UPDATE)

	upd := &pgast.Update{Table: p.parseRelationExpr()}

	p.expect(SET)

	upd.Set = p.parseSetClauseList()
	upd.From = p.parseOptionalFromClause()
	upd.Where = p.parseUpdateWhereClause()
	upd.Returning = p.parseOptionalReturning()

	return upd
}

// parseUpdateWhereClause parses an optional WHERE clause, rejecting the
// positioned "WHERE CURRENT OF cursor" form explicitly: it needs DECLARE
// CURSOR session state, which is out of scope for a formatter.
func (p *Parser) parseUpdateWhereClause() *pgast.Where {
	if p.at(WHERE) && p.peekAt(CURRENT) && p.peek2At(OF) {
		p.failf("WHERE CURRENT OF is not supported")
	}

	return p.parseOptionalWhereClause(WHERE)
}
