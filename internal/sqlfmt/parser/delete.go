package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseDelete parses a DELETE statement (optionally preceded by a WITH
// clause) from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseDelete(input string) (del *sqlast.Delete, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	del = p.parseDeleteStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return del, nil
}

// parseDeleteStatement parses a DELETE statement, optionally preceded by a
// WITH clause. The current token must be WITH or DELETE.
//
// MySQL's DELETE grammar has three shapes, disambiguated after DELETE
// [IGNORE]:
//
//   - Form A (single-table): FROM tbl_name [[AS] alias] [WHERE ...]
//     [ORDER BY ...] [LIMIT ...]
//   - Form B (multi-table, target list first): tbl_name [, tbl_name ...]
//     FROM table_references [WHERE ...]
//   - Form C (multi-table, USING): FROM tbl_name [, tbl_name ...]
//     USING table_references [WHERE ...]
//
// If the token after DELETE [IGNORE] isn't FROM, it must be Form B. Otherwise
// FROM is consumed and a comma-separated list of plain table names is parsed;
// a following USING makes it Form C (those names become Targets), and
// anything else makes it Form A (the list must hold exactly one name).
func (p *Parser) parseDeleteStatement() *sqlast.Delete {
	return p.parseDeleteStatementAfterWith(p.parseOptionalWith())
}

// parseDeleteStatementAfterWith parses a DELETE statement given an
// already-parsed (possibly nil) leading WITH clause. The current token must
// be DELETE.
func (p *Parser) parseDeleteStatementAfterWith(with *sqlast.With) *sqlast.Delete {
	del := &sqlast.Delete{With: with}

	p.expect(DELETE)

	del.Ignore = p.consume(IGNORE)

	var singleTable bool

	if p.at(FROM) {
		singleTable = p.parseDeleteFromClause(del)
	} else {
		del.Targets = p.parseTableNameList()
		p.expect(FROM)
		del.TableExprs = p.parseTableReferenceList()
	}

	del.Where = p.parseOptionalWhereClause(WHERE)

	// ORDER BY and LIMIT are only valid for the single-table form (Form A);
	// MySQL rejects them on the multi-table forms (Form B and Form C).
	if singleTable {
		del.OrderBy = p.parseOptionalOrderBy()
		del.Limit = p.parseOptionalLimit()
	}

	return del
}

// parseDeleteFromClause parses the FROM clause of a DELETE statement,
// disambiguating between the single-table form (Form A) and the multi-table
// USING form (Form C). The current token must be FROM. It reports whether
// the statement is the single-table form.
func (p *Parser) parseDeleteFromClause(del *sqlast.Delete) bool {
	p.advance() // consume FROM

	names := p.parseTableNameList()

	if p.consume(USING) {
		del.Targets = names
		del.TableExprs = p.parseTableReferenceList()

		return false
	}

	if len(names) != 1 {
		p.failf("expected USING after multiple table names in DELETE FROM")
	}

	alias := p.parseOptionalTableAlias()
	hints := p.parseOptionalIndexHints()

	del.TableExprs = []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: names[0], As: alias, Hints: hints}}

	return true
}

// parseTableNameList parses a comma-separated list of plain (possibly
// qualified) table names, without aliases or index hints.
func (p *Parser) parseTableNameList() []sqlast.TableName {
	var names []sqlast.TableName

	for {
		names = append(names, p.parseTableName())

		if !p.consume(COMMA) {
			return names
		}
	}
}
