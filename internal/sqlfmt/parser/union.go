package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// ParseUnion parses a UNION statement (two or more SELECT branches combined
// by UNION [ALL|DISTINCT]), optionally preceded by a WITH clause and
// followed by a trailing ORDER BY/LIMIT/locking clause on the combined
// result. It fails with a *ParseError if input is a single SELECT with no
// UNION keyword, since this entry point is specifically for union
// statements (contrast with ParseSelect, the entry point for plain SELECTs).
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseUnion(input string) (u *sqlast.Union, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	u = p.parseUnionStatement()

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return u, nil
}

// parseUnionStatement parses a UNION statement, optionally preceded by a
// WITH clause. The current token must be WITH or SELECT. It fails if no
// UNION keyword is present at all, i.e. the input is just a single SELECT.
func (p *Parser) parseUnionStatement() *sqlast.Union {
	with := p.parseOptionalWith()

	left := p.parseUnionBranch()

	if !p.at(UNION) {
		p.failf("expected UNION, got %s", p.tok.Type)
	}

	u := p.parseUnionTail(left)

	u.With = with
	u.OrderBy = p.parseOptionalOrderBy()
	u.Limit = p.parseOptionalLimit()
	u.Lock, u.LockWait = p.parseOptionalLock()

	return u
}

// parseUnionTail parses one or more `UNION [ALL|DISTINCT] select_core`
// suffixes following left, left-associatively folding them into a chain of
// nested *sqlast.Union nodes (e.g. `a UNION b UNION c` becomes a Union node
// whose Left is itself a Union{Left: a, Right: b} and whose Right is c). The
// current token must be UNION, guaranteeing the loop runs at least once.
func (p *Parser) parseUnionTail(left sqlast.Statement) *sqlast.Union {
	var u *sqlast.Union

	for p.consume(UNION) {
		distinct := p.parseOptionalUnionDistinct()
		right := p.parseUnionBranch()

		u = &sqlast.Union{Left: left, Right: right, Distinct: distinct}
		left = u
	}

	return u
}

// parseOptionalUnionDistinct parses the optional [ALL|DISTINCT] modifier
// following UNION, reporting whether the union should render as DISTINCT
// (the default, and also what an explicit DISTINCT renders as).
func (p *Parser) parseOptionalUnionDistinct() bool {
	if p.consume(ALL) {
		return false
	}

	p.consume(DISTINCT)

	return true
}

// parseUnionBranch parses a single UNION branch: just the SELECT keyword
// through the WHERE/GROUP BY/HAVING filters, via the shared parseSelectCore.
// A bare branch must not itself consume a trailing ORDER BY/LIMIT/locking
// clause, since when one follows the last branch it belongs to the union as
// a whole.
func (p *Parser) parseUnionBranch() sqlast.Statement {
	sel := &sqlast.Select{}
	p.parseSelectCore(sel)

	return sel
}
