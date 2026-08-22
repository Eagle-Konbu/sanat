package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseUpdate parses an UPDATE statement (optionally preceded by a WITH
// clause) from input, using ModeDefault.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseUpdate(input string) (upd *sqlast.Update, err error) {
	return ParseUpdateWithMode(input, ModeDefault)
}

// ParseUpdateWithMode parses an UPDATE statement (optionally preceded by a
// WITH clause) from input, decoding string literals per mode.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseUpdateWithMode(input string, mode SQLMode) (upd *sqlast.Update, err error) {
	defer recoverParseError(&err)

	p := NewParserWithMode(input, mode)
	result := p.parseUpdateStatement()

	p.consume(SEMICOLON)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return result, nil
}

// parseUpdateStatement parses an UPDATE statement, optionally preceded by a
// WITH clause. The current token must be WITH or UPDATE.
func (p *Parser) parseUpdateStatement() *sqlast.Update {
	return p.parseUpdateStatementAfterWith(p.parseOptionalWith())
}

// parseUpdateStatementAfterWith parses an UPDATE statement given an
// already-parsed (possibly nil) leading WITH clause. The current token must
// be UPDATE.
func (p *Parser) parseUpdateStatementAfterWith(with *sqlast.With) *sqlast.Update {
	upd := &sqlast.Update{With: with}

	p.expect(UPDATE)

	upd.Ignore = p.consume(IGNORE)
	upd.TableExprs = p.parseTableReferenceList()

	p.expect(SET)

	upd.Exprs = p.parseSetExprList()
	upd.Where = p.parseOptionalWhereClause(WHERE)

	// ORDER BY and LIMIT are only valid for the single-table form; MySQL
	// rejects them when TableExprs names more than one table or contains a
	// JOIN, mirroring the same restriction on DELETE (see
	// parseDeleteStatement).
	if isSingleTableUpdate(upd.TableExprs) {
		upd.OrderBy = p.parseOptionalOrderBy()
		upd.Limit = p.parseOptionalLimit()
	}

	return upd
}

// isSingleTableUpdate reports whether exprs is the single-table form of
// UPDATE's target: exactly one plain table name, not a JOIN and not a
// parenthesized table reference (which may itself contain a JOIN).
func isSingleTableUpdate(exprs []sqlast.TableExpr) bool {
	if len(exprs) != 1 {
		return false
	}

	aliased, ok := exprs[0].(*sqlast.AliasedTableExpr)
	if !ok {
		return false
	}

	_, isTableName := aliased.Expr.(sqlast.TableName)

	return isTableName
}
