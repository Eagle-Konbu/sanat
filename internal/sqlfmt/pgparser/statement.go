package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// parseSubqueryStatement parses the SELECT statement inside a parenthesized
// subquery, EXISTS (...), or [NOT] IN (...) predicate.
//
// This is a deliberately minimal SELECT subset — just enough for expr.go's
// subquery contexts to be parseable and testable on their own — because the
// full SELECT grammar (joins, DISTINCT ON, GROUP BY, window clauses, ...) is
// issue #81's scope, not this one. #81 replaces parseSelectStatement below
// with the complete grammar; this file (and this function split) goes away
// once that lands. internal/sqlfmt/parser's expr.go has the same forward
// dependency on its own package's select.go, for the same reason: an
// expression grammar that supports subqueries necessarily needs some form of
// SELECT parsing to exist already.
func (p *Parser) parseSubqueryStatement() pgast.Statement {
	return p.parseSelectStatement()
}

func (p *Parser) parseSelectStatement() *pgast.Select {
	p.expect(SELECT)

	sel := &pgast.Select{SelectExprs: p.parseSelectExprList()}

	if p.consume(FROM) {
		sel.From = p.parseTableExprList()
	}

	if p.consume(WHERE) {
		sel.Where = &pgast.Where{Expr: p.parseExpr()}
	}

	return sel
}

func (p *Parser) parseSelectExprList() []pgast.SelectExpr {
	var exprs []pgast.SelectExpr

	for {
		exprs = append(exprs, p.parseSelectExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

func (p *Parser) parseSelectExpr() pgast.SelectExpr {
	if p.consume(STAR) {
		return &pgast.StarExpr{}
	}

	return &pgast.AliasedExpr{Expr: p.parseExpr(), As: pgast.ColIdent(p.parseOptionalAlias())}
}

// parseOptionalAlias parses an optional trailing "[AS] name" alias, returning
// "" if none is present.
func (p *Parser) parseOptionalAlias() string {
	p.consume(AS)

	if !p.at(IDENT) && !p.at(QuotedIdent) {
		return ""
	}

	name := p.tok.Literal
	p.advance()

	return name
}

func (p *Parser) parseTableExprList() []pgast.TableExpr {
	var exprs []pgast.TableExpr

	for {
		exprs = append(exprs, p.parseTableExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

// parseTableExpr parses a single FROM-item: just a bare (optionally
// aliased) table name, matching the minimal scope described above.
func (p *Parser) parseTableExpr() pgast.TableExpr {
	name := p.readIdent()
	table := pgast.TableName{Name: pgast.TableIdent(name)}

	return &pgast.AliasedTableExpr{Expr: table, As: pgast.TableIdent(p.parseOptionalAlias())}
}

func (p *Parser) readIdent() string {
	if p.tok.Type != IDENT && p.tok.Type != QuotedIdent {
		p.failf("expected identifier, got %s", p.tok.Type)
	}

	lit := p.tok.Literal
	p.advance()

	return lit
}
