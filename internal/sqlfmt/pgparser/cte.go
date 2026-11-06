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
	cte := &pgast.CommonTableExpr{Name: pgast.TableIdent(p.readIdent())}

	if p.consume(LPAREN) {
		cte.Columns = p.parseColumnList()
		p.expect(RPAREN)
	}

	p.expect(AS)

	cte.Materialization = p.parseCTEMaterialization()

	p.expect(LPAREN)

	cte.Subquery = p.parseCTEBody()

	p.expect(RPAREN)

	return cte
}

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

// parseCTEBody parses the statement inside a CTE's parentheses: a query or
// an INSERT/UPDATE/DELETE, any of which may carry a WITH clause of its own.
// The statement kind is only known once that WITH clause has been parsed,
// so it is parsed here and attached afterwards.
func (p *Parser) parseCTEBody() pgast.Statement {
	pos := p.tok.Pos
	with := p.parseOptionalWith()

	var stmt pgast.Statement

	switch p.tok.Type {
	case INSERT:
		stmt = p.parseInsertStatement()
	case UPDATE:
		stmt = p.parseUpdateStatement()
	case DELETE:
		stmt = p.parseDeleteStatement()
	default:
		stmt = p.parseQuery()
	}

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
	}

	if *target != nil {
		panic(&ParseError{Pos: pos, Msg: "multiple WITH clauses not allowed"})
	}

	*target = with
}

// isQueryStart reports whether tt can begin a query in a parenthesized
// subquery position. WITH is a reserved keyword, so it never begins an
// expression or table reference instead.
func isQueryStart(tt TokenType) bool {
	return tt == SELECT || tt == WITH
}
