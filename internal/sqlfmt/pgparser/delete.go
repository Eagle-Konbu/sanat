package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseDelete parses a DELETE statement, optionally led by a WITH clause,
// from input.
func ParseDelete(input string) (*pgast.Delete, error) {
	return parseStatementInput(input, (*Parser).parseDeleteAfterWith)
}

// parseDeleteAfterWith parses a DELETE statement whose optional WITH clause
// has already been consumed. The current token must be DELETE. WHERE CURRENT
// OF is out of scope (it needs DECLARE CURSOR session state a formatter
// never has), so CURRENT after WHERE fails as an unexpected expression token.
func (p *Parser) parseDeleteAfterWith() *pgast.Delete {
	p.expect(DELETE)
	p.expect(FROM)

	del := &pgast.Delete{Table: p.parseRelationExpr()}

	if p.consume(USING) {
		del.Using = p.parseTableReferenceList()
	}

	del.Where = p.parseOptionalWhereClause(WHERE)
	del.Returning = p.parseOptionalReturning()

	return del
}
