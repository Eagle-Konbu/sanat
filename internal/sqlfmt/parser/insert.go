package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseInsert parses an INSERT or REPLACE statement from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseInsert(input string) (ins *sqlast.Insert, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	ins = p.parseInsertStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return ins, nil
}

// parseInsertStatement parses an INSERT or REPLACE statement. The current
// token must be INSERT or REPLACE.
func (p *Parser) parseInsertStatement() *sqlast.Insert {
	ins := &sqlast.Insert{Action: p.parseInsertAction()}

	if ins.Action == sqlast.InsertAct {
		ins.Ignore = p.consume(IGNORE)
	}

	p.expect(INTO)

	ins.Table = p.parseTableName()
	ins.Columns = p.parseOptionalColumnList()
	ins.Rows = p.parseInsertRows()

	if ins.Action == sqlast.InsertAct {
		ins.OnDup = p.parseOptionalOnDup()
	}

	return ins
}

// parseInsertAction parses the leading INSERT or REPLACE keyword.
func (p *Parser) parseInsertAction() sqlast.InsertAction {
	if p.consume(REPLACE) {
		return sqlast.ReplaceAct
	}

	p.expect(INSERT)

	return sqlast.InsertAct
}

// parseOptionalColumnList parses an optional parenthesized column list
// following the target table name.
func (p *Parser) parseOptionalColumnList() sqlast.Columns {
	if !p.consume(LPAREN) {
		return nil
	}

	cols := p.parseIdentList()

	p.expect(RPAREN)

	return cols
}

// parseInsertRows parses the VALUES/SET/SELECT form of an INSERT statement's
// row source.
func (p *Parser) parseInsertRows() sqlast.InsertRows {
	switch {
	case p.at(VALUES):
		return p.parseValuesRows()
	case p.at(SET):
		return p.parseSetRows()
	case p.at(SELECT) || p.at(WITH):
		return p.parseSelectStatement()
	default:
		return failReturn[sqlast.InsertRows](p, "expected VALUES, SET, or SELECT, got %s", p.tok.Type)
	}
}

// parseValuesRows parses a VALUES clause: one or more parenthesized,
// comma-separated expression lists. The current token must be VALUES.
func (p *Parser) parseValuesRows() sqlast.Values {
	p.advance()

	var rows sqlast.Values

	for {
		p.expect(LPAREN)

		rows = append(rows, p.parseExprList())

		p.expect(RPAREN)

		if !p.consume(COMMA) {
			return rows
		}
	}
}

// parseSetRows parses a SET clause: a comma-separated list of "col = expr"
// assignments. The current token must be SET.
func (p *Parser) parseSetRows() sqlast.SetExprs {
	p.advance()

	return sqlast.SetExprs(p.parseSetExprList())
}

// parseOptionalOnDup parses an optional trailing ON DUPLICATE KEY UPDATE
// clause.
func (p *Parser) parseOptionalOnDup() sqlast.OnDup {
	if !p.consume(ON) {
		return nil
	}

	p.expect(DUPLICATE)
	p.expect(KEY)
	p.expect(UPDATE)

	return sqlast.OnDup(p.parseSetExprList())
}
