package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseInsert parses an INSERT statement, optionally led by a WITH clause,
// from input.
func ParseInsert(input string) (*pgast.Insert, error) {
	return parseStatementInput(input, (*Parser).parseInsertAfterWith)
}

// parseInsertAfterWith parses an INSERT statement whose optional WITH clause
// has already been consumed. The current token must be INSERT. OVERRIDING
// {SYSTEM | USER} VALUE is not supported.
func (p *Parser) parseInsertAfterWith() *pgast.Insert {
	p.expect(INSERT)
	p.expect(INTO)

	ins := &pgast.Insert{Table: p.parseTableName()}

	// PostgreSQL's insert target requires AS before an alias, unlike a FROM
	// item, so a bare identifier here is a syntax error rather than an alias.
	if p.consume(AS) {
		ins.Alias = pgast.TableIdent(p.readColID())
	}

	if !p.atParenthesizedQuery() {
		ins.Columns = p.parseOptionalParenColumnList()
	}

	ins.Rows = p.parseInsertRows()
	ins.OnConflict = p.parseOptionalOnConflict()
	ins.Returning = p.parseOptionalReturning()

	return ins
}

// parseInsertRows parses the row source of an INSERT statement: DEFAULT
// VALUES, a VALUES list, or a query (a SELECT or set operation, which may
// lead with its own WITH).
func (p *Parser) parseInsertRows() pgast.InsertRows {
	switch p.tok.Type {
	case DEFAULT:
		p.advance()
		p.expect(VALUES)

		return pgast.DefaultValues{}
	case VALUES:
		rows := p.parseValuesRows()
		if !p.isQueryContinuation() {
			return rows
		}

		// VALUES followed by a set operator or trailing clause is a query.
		return p.parseQueryFrom(&pgast.Select{Values: rows}).(pgast.InsertRows) //nolint:forcetypeassert // see below
	case SELECT, WITH, TABLE, LPAREN:
		// parseQuery only returns a *pgast.Select or *pgast.SetOp, both of
		// which are InsertRows.
		return p.parseQuery().(pgast.InsertRows) //nolint:forcetypeassert // see above
	default:
		return failReturn[pgast.InsertRows](p, "expected VALUES, DEFAULT VALUES, or SELECT, got %s", p.tok.Type)
	}
}

// atParenthesizedQuery reports whether the current '(' after an INSERT
// target opens a parenthesized query, as in "INSERT INTO t (SELECT ...)",
// rather than a column list, which never starts with a query keyword or '('.
func (p *Parser) atParenthesizedQuery() bool {
	return p.at(LPAREN) && (isQueryStart(p.peekTok.Type, p.peek2Tok.Type) || p.peekAt(LPAREN))
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

	oc := &pgast.OnConflict{Columns: p.parseOptionalParenColumnList()}

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

	oc.Set = p.parseSetClauseList()
	oc.Where = p.parseOptionalWhereClause(WHERE)

	return oc
}
