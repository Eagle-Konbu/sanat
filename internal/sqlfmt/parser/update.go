package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseUpdate parses an UPDATE statement (optionally preceded by a WITH
// clause) from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseUpdate(input string) (upd *sqlast.Update, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	upd = p.parseUpdateStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return upd, nil
}

// parseUpdateStatement parses an UPDATE statement, optionally preceded by a
// WITH clause. The current token must be WITH or UPDATE.
func (p *Parser) parseUpdateStatement() *sqlast.Update {
	upd := &sqlast.Update{With: p.parseOptionalWith()}

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
// UPDATE's target: exactly one table reference with no JOIN.
func isSingleTableUpdate(exprs []sqlast.TableExpr) bool {
	if len(exprs) != 1 {
		return false
	}

	_, isJoin := exprs[0].(*sqlast.JoinTableExpr)

	return !isJoin
}
