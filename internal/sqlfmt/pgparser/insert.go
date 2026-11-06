package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseInsert parses an INSERT statement from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseInsert(input string) (ins *pgast.Insert, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	result := p.parseInsertStatement()

	p.consume(SEMICOLON)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return result, nil
}

// parseInsertStatement parses an INSERT statement with an optional leading
// WITH clause. The current token must be WITH or INSERT. OVERRIDING
// {SYSTEM | USER} VALUE is not supported.
func (p *Parser) parseInsertStatement() *pgast.Insert {
	with := p.parseOptionalWith()

	p.expect(INSERT)
	p.expect(INTO)

	ins := &pgast.Insert{With: with, Table: p.parseTableName()}

	// PostgreSQL's insert target requires AS before an alias, unlike a FROM
	// item, so a bare identifier here is a syntax error rather than an alias.
	if p.consume(AS) {
		ins.Alias = pgast.TableIdent(p.readIdent())
	}

	if p.consume(LPAREN) {
		ins.Columns = p.parseColumnList()
		p.expect(RPAREN)
	}

	ins.Rows = p.parseInsertRows()
	ins.OnConflict = p.parseOptionalOnConflict()
	ins.Returning = p.parseOptionalReturning()

	return ins
}

// parseInsertRows parses the row source of an INSERT statement: DEFAULT
// VALUES, a VALUES list, or a SELECT (which may lead with its own WITH).
func (p *Parser) parseInsertRows() pgast.InsertRows {
	switch p.tok.Type {
	case DEFAULT:
		p.advance()
		p.expect(VALUES)

		return pgast.DefaultValues{}
	case VALUES:
		return p.parseValuesRows()
	case SELECT, WITH:
		return p.parseSelectStatement()
	default:
		return failReturn[pgast.InsertRows](p, "expected VALUES, DEFAULT VALUES, or SELECT, got %s", p.tok.Type)
	}
}

// parseValuesRows parses a VALUES clause: one or more parenthesized,
// comma-separated lists of expressions or DEFAULT. The current token must be
// VALUES.
func (p *Parser) parseValuesRows() pgast.Values {
	p.advance()

	var rows pgast.Values

	for {
		p.expect(LPAREN)

		rows = append(rows, p.parseExprOrDefaultList())

		p.expect(RPAREN)

		if !p.consume(COMMA) {
			return rows
		}
	}
}

func (p *Parser) parseExprOrDefaultList() []pgast.Expr {
	var exprs []pgast.Expr

	for {
		exprs = append(exprs, p.parseExprOrDefault())

		if !p.consume(COMMA) {
			return exprs
		}
	}
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

// parseOptionalOnConflict parses an optional ON CONFLICT [(col, ...)]
// {DO NOTHING | DO UPDATE SET ... [WHERE ...]} clause. The ON CONSTRAINT
// target, expression/COLLATE/operator-class targets, and a WHERE index
// predicate on the target are not supported. The conflict target is optional
// for DO NOTHING but required for DO UPDATE, as in PostgreSQL.
func (p *Parser) parseOptionalOnConflict() *pgast.OnConflict {
	if !p.at(ON) || !p.peekAt(CONFLICT) {
		return nil
	}

	p.advance() // consume ON
	p.advance() // consume CONFLICT

	oc := &pgast.OnConflict{}

	if p.consume(LPAREN) {
		oc.Columns = p.parseColumnList()
		p.expect(RPAREN)
	}

	p.expect(DO)

	if p.consume(NOTHING) {
		oc.DoNothing = true

		return oc
	}

	if oc.Columns == nil {
		p.failf("ON CONFLICT DO UPDATE requires a conflict target")
	}

	p.expect(UPDATE)
	p.expect(SET)

	oc.Set = p.parseConflictAssignments()
	oc.Where = p.parseOptionalWhereClause(WHERE)

	return oc
}

// parseConflictAssignments parses the comma-separated "col = {expr |
// DEFAULT}" list of an ON CONFLICT DO UPDATE SET clause. EXCLUDED.col needs
// no special handling: it parses as an ordinary column reference qualified
// by the EXCLUDED pseudo-table.
func (p *Parser) parseConflictAssignments() []*pgast.UpdateExpr {
	var sets []*pgast.UpdateExpr

	for {
		name := &pgast.ColName{Name: pgast.ColIdent(p.readIdent())}

		p.expect(EQ)

		sets = append(sets, &pgast.UpdateExpr{Name: name, Expr: p.parseExprOrDefault()})

		if !p.consume(COMMA) {
			return sets
		}
	}
}

// parseOptionalReturning parses an optional RETURNING clause, whose item
// grammar is the same as a SELECT list.
func (p *Parser) parseOptionalReturning() *pgast.Returning {
	if !p.consume(RETURNING) {
		return nil
	}

	return &pgast.Returning{Exprs: p.parseSelectExprList()}
}
