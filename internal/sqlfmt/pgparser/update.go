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

	upd := &pgast.Update{}
	p.parseUpdateTarget(upd)

	p.expect(SET)

	upd.Set = p.parseSetClauseList()
	upd.From = p.parseOptionalFromClause()
	upd.Where = p.parseUpdateWhereClause()
	upd.Returning = p.parseOptionalReturning()

	return upd
}

// parseUpdateTarget parses "[ONLY] table_name [*] [[AS] alias]". PostgreSQL
// also accepts "ONLY (table_name)", which is normalized to "ONLY table_name".
func (p *Parser) parseUpdateTarget(upd *pgast.Update) {
	upd.Only = p.consume(ONLY)

	var name pgast.TableName

	if upd.Only && p.consume(LPAREN) {
		name = p.parseTableName()
		p.expect(RPAREN)
	} else {
		name = p.parseTableName()
	}

	if !upd.Only {
		upd.Descendants = p.consume(STAR)
	}

	// As in PostgreSQL, a bare SET here always starts the SET clause. "AS set"
	// is rejected too: aliases render without AS, so it could not round-trip.
	if p.at(AS) && p.peekAt(SET) {
		p.failf("SET cannot be used as an UPDATE target alias")
	}

	var alias string
	if !p.at(SET) {
		alias = p.parseOptionalAlias()
	}

	upd.Table = &pgast.AliasedTableExpr{Expr: name, As: pgast.TableIdent(alias)}
}

func (p *Parser) parseSetClauseList() []*pgast.UpdateExpr {
	var exprs []*pgast.UpdateExpr

	for {
		exprs = append(exprs, p.parseSetClause())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

func (p *Parser) parseSetClause() *pgast.UpdateExpr {
	if p.at(LPAREN) {
		return p.parseMultiColumnSetClause()
	}

	name := p.parseSetTarget()
	p.expect(EQ)

	return &pgast.UpdateExpr{Name: name, Expr: p.parseExprOrDefault()}
}

// parseSetTarget parses a SET target column. PostgreSQL rejects a
// table-qualified target, but "col.field" names a composite-type field, so
// one dotted level is accepted (stored as ColName's qualifier, which renders
// identically). Array-subscript targets ("col[1] = ...") are not supported.
func (p *Parser) parseSetTarget() *pgast.ColName {
	name := p.readIdent()

	if p.consume(DOT) {
		return &pgast.ColName{
			Qualifier: pgast.TableName{Name: pgast.TableIdent(name)},
			Name:      pgast.ColIdent(p.readIdent()),
		}
	}

	return &pgast.ColName{Name: pgast.ColIdent(name)}
}

// parseMultiColumnSetClause parses "(col, ...) = source", where source is a
// parenthesized row, a ROW(...) constructor, or a sub-SELECT.
func (p *Parser) parseMultiColumnSetClause() *pgast.UpdateExpr {
	p.advance() // consume '('

	var names []*pgast.ColName

	for {
		names = append(names, p.parseSetTarget())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)
	p.expect(EQ)

	return &pgast.UpdateExpr{Names: names, Expr: p.parseMultiColumnSource()}
}

func (p *Parser) parseMultiColumnSource() pgast.Expr {
	switch {
	case p.consume(ROW):
		return &pgast.RowExpr{Exprs: p.parseParenExprOrDefaultList()}
	case p.at(LPAREN) && isQueryStart(p.peekTok.Type):
		p.advance() // consume '('

		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &pgast.Subquery{Select: sel}
	case p.at(LPAREN):
		return pgast.ValTuple(p.parseParenExprOrDefaultList())
	default:
		return failReturn[pgast.Expr](p, "expected row constructor or sub-SELECT, got %s", p.tok.Type)
	}
}

// parseParenExprOrDefaultList parses "(expr | DEFAULT, ...)".
func (p *Parser) parseParenExprOrDefaultList() []pgast.Expr {
	p.expect(LPAREN)

	var exprs []pgast.Expr

	for {
		exprs = append(exprs, p.parseExprOrDefault())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return exprs
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
