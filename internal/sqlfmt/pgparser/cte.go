package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// parseOptionalWith parses an optional leading
// "WITH [RECURSIVE] cte [, ...]" clause, returning nil if none is present.
// The SEARCH and CYCLE clauses of recursive CTEs are not supported.
func (p *Parser) parseOptionalWith() *pgast.With {
	if !p.consume(WITH) {
		return nil
	}

	with := &pgast.With{Recursive: p.consume(RECURSIVE)}

	for {
		with.CTEs = append(with.CTEs, p.parseCommonTableExpr())

		if !p.consume(COMMA) {
			return with
		}
	}
}

// parseCommonTableExpr parses
// "name [(column, ...)] AS [[NOT] MATERIALIZED] (statement)".
func (p *Parser) parseCommonTableExpr() *pgast.CommonTableExpr {
	cte := &pgast.CommonTableExpr{Name: pgast.TableIdent(p.readColID())}
	cte.Columns = p.parseOptionalParenColumnList()

	p.expect(AS)

	cte.Materialization = p.parseCTEMaterialization()

	p.expect(LPAREN)

	cte.Subquery = p.parseStatement()

	p.expect(RPAREN)

	return cte
}

// parseCTEMaterialization parses the optional [NOT] MATERIALIZED hint
// between a CTE's AS and its parenthesized body.
func (p *Parser) parseCTEMaterialization() pgast.CTEMaterialization {
	switch {
	case p.consume(MATERIALIZED):
		return pgast.Materialized
	case p.consume(NOT):
		p.expect(MATERIALIZED)

		return pgast.NotMaterialized
	default:
		return pgast.MaterializeDefault
	}
}

// isQueryStart reports whether tt can begin a query in a parenthesized
// subquery position. WITH is a reserved keyword, so it never begins an
// expression or table reference instead.
func isQueryStart(tt TokenType) bool {
	return tt == SELECT || tt == WITH
}
