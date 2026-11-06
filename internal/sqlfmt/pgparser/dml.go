package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// This file holds the grammar pieces that INSERT, UPDATE, and DELETE share
// (and that MERGE's actions reuse): the target relation, SET assignments,
// VALUES rows, and RETURNING.

// parseRelationExpr parses a DML target table:
// "[ONLY] name [*] [[AS] alias]", where ONLY may also parenthesize the name
// ("ONLY (name)" is normalized to "ONLY name").
func (p *Parser) parseRelationExpr() *pgast.AliasedTableExpr {
	target := &pgast.AliasedTableExpr{}

	if p.consume(ONLY) {
		target.Only = true
		target.Expr = p.parseOptionallyParenthesizedTableName()
	} else {
		target.Expr = p.parseTableName()
		target.IncludeDescendants = p.consume(STAR)
	}

	target.As = pgast.TableIdent(p.parseRelationAlias())

	return target
}

// parseRelationAlias parses a DML target's optional alias. As in PostgreSQL,
// a bare SET here is never an alias, since it starts UPDATE's SET clause;
// "AS set" is still an alias.
func (p *Parser) parseRelationAlias() string {
	if p.at(SET) {
		return ""
	}

	return p.parseOptionalTableAlias()
}

// parseOptionallyParenthesizedTableName parses the table name after ONLY,
// accepting PostgreSQL's "ONLY (name)" spelling as well as a bare name.
func (p *Parser) parseOptionallyParenthesizedTableName() pgast.TableName {
	if !p.consume(LPAREN) {
		return p.parseTableName()
	}

	name := p.parseTableName()
	p.expect(RPAREN)

	return name
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
	name := p.readColID()

	if p.consume(DOT) {
		return &pgast.ColName{
			Qualifier: pgast.TableName{Name: pgast.TableIdent(name)},
			Name:      pgast.ColIdent(p.readColLabel()),
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

// parseMultiColumnSource parses the right-hand side of a multi-column SET
// assignment. A '(' starts a sub-SELECT if a query keyword follows it and a
// row otherwise; unlike a parenthesized expression elsewhere, each row item
// may be DEFAULT.
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

// parseValuesRows parses a VALUES clause: one or more parenthesized,
// comma-separated lists of expressions or DEFAULT. The current token must be
// VALUES.
func (p *Parser) parseValuesRows() pgast.Values {
	p.advance()

	var rows pgast.Values

	for {
		rows = append(rows, p.parseParenExprOrDefaultList())

		if !p.consume(COMMA) {
			return rows
		}
	}
}

// parseParenExprOrDefaultList parses "(expr | DEFAULT, ...)": a single VALUES
// row or a ROW(...) constructor's arguments.
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

// parseExprOrDefault parses an expression, or the DEFAULT keyword that
// PostgreSQL accepts in its place only as a VALUES item or a SET assignment's
// right-hand side. DEFAULT is not a value of its own, so it is modeled as a
// Literal rendered verbatim rather than accepted by the general expression
// grammar.
func (p *Parser) parseExprOrDefault() pgast.Expr {
	if p.consume(DEFAULT) {
		return &pgast.Literal{Val: "DEFAULT"}
	}

	return p.parseExpr()
}

// parseOptionalReturning parses an optional RETURNING clause, whose item
// grammar is the same as a SELECT list.
func (p *Parser) parseOptionalReturning() *pgast.Returning {
	if !p.consume(RETURNING) {
		return nil
	}

	return &pgast.Returning{Exprs: p.parseSelectExprList()}
}
