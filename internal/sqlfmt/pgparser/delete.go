package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseDelete parses a DELETE statement from input.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseDelete(input string) (del *pgast.Delete, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	result := p.parseDeleteStatement()

	p.consume(SEMICOLON)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return result, nil
}

// parseDeleteStatement parses a DELETE statement. The current token must be
// DELETE. A leading WITH is deferred to #86, and WHERE CURRENT OF is out of
// scope (it needs DECLARE CURSOR session state a formatter never has), so
// CURRENT after WHERE fails as an unexpected expression token.
func (p *Parser) parseDeleteStatement() *pgast.Delete {
	p.expect(DELETE)
	p.expect(FROM)

	del := &pgast.Delete{Table: p.parseRelationExpr()}

	if p.consume(USING) {
		del.Using = p.parseTableReferenceList()
	}

	del.Where = p.parseOptionalWhereClause(WHERE)
	del.Returning = p.parseOptionalReturning()

	return del
}

// parseRelationExpr parses a DML target table:
// "[ONLY] name [*] [[AS] alias]", where ONLY may also parenthesize the name.
func (p *Parser) parseRelationExpr() *pgast.AliasedTableExpr {
	target := &pgast.AliasedTableExpr{}

	if p.consume(ONLY) {
		target.Only = true
		target.Expr = p.parseOptionallyParenthesizedTableName()
	} else {
		target.Expr = p.parseTableName()
		target.IncludeDescendants = p.consume(STAR)
	}

	target.As = pgast.TableIdent(p.parseOptionalAlias())

	return target
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
