package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// ParseSetOp parses a set operation statement: two or more SELECT operands
// combined by UNION, INTERSECT, or EXCEPT (each with an optional ALL or
// DISTINCT), optionally followed by ORDER BY/LIMIT/OFFSET/FETCH clauses that
// apply to the combined result. It fails with a *ParseError if input is a
// single (possibly parenthesized) SELECT with no set operator, since this
// entry point is specifically for set operations (contrast with ParseSelect,
// the entry point for plain SELECTs).
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func ParseSetOp(input string) (setOp *pgast.SetOp, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	start := p.tok.Pos
	stmt := p.parseQuery()

	p.consume(SEMICOLON)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	result, ok := stmt.(*pgast.SetOp)
	if !ok {
		return nil, &ParseError{Pos: start, Msg: "expected UNION, INTERSECT, or EXCEPT"}
	}

	return result, nil
}

// parseQuery parses a query: an optional WITH clause, then a set-operation
// chain of SELECT operands (or a single one) followed by optional trailing
// ORDER BY/LIMIT/OFFSET/FETCH and locking clauses. The WITH clause scopes the
// whole query, so it is attached to the outermost statement. The current
// token must be WITH, SELECT, or '('.
func (p *Parser) parseQuery() pgast.Statement {
	pos := p.tok.Pos
	with := p.parseOptionalWith()
	stmt := p.parseQueryFrom(p.parseSetOpOperand())
	attachWith(stmt, with, pos)

	return stmt
}

// parseQueryFrom parses the rest of a query whose first operand has already
// been parsed. Expression and FROM contexts use this when a leading
// parenthesized SELECT turns out to be the first operand of a set operation
// (e.g. "IN ((SELECT 1) UNION SELECT 2)"), which they can't tell apart from a
// parenthesized expression or table until after the operand is parsed.
func (p *Parser) parseQueryFrom(first pgast.Statement) pgast.Statement {
	stmt := p.parseUnionChain(first)
	p.attachQueryTail(stmt, p.parseQueryTail())

	return stmt
}

// isQueryContinuation reports whether the current token can continue a query
// after a complete operand: a set operator or a trailing clause keyword.
func (p *Parser) isQueryContinuation() bool {
	switch p.tok.Type {
	case UNION, INTERSECT, EXCEPT, ORDER, LIMIT, OFFSET, FETCH, FOR:
		return true
	default:
		return false
	}
}

// tryContinueQueryFromExpr continues a query from expr if expr is a
// (possibly redundantly parenthesized) scalar subquery and the current
// token continues a query. It returns nil, consuming nothing, otherwise.
func (p *Parser) tryContinueQueryFromExpr(expr pgast.Expr) pgast.Statement {
	if !p.isQueryContinuation() {
		return nil
	}

	sub := unwrapSubquery(expr)
	if sub == nil {
		return nil
	}

	return p.parseQueryFrom(sub.Select)
}

// unwrapSubquery returns expr's subquery if expr is a scalar subquery wrapped
// in any number of redundant parentheses, or nil otherwise.
func unwrapSubquery(expr pgast.Expr) *pgast.Subquery {
	for {
		switch e := expr.(type) {
		case *pgast.ParenExpr:
			expr = e.Expr
		case *pgast.Subquery:
			return e
		default:
			return nil
		}
	}
}

// parseUnionChain parses UNION/EXCEPT operators, which share the lowest
// precedence and associate left. Each side is an INTERSECT chain, so
// INTERSECT binds tighter: "a UNION b INTERSECT c" is "a UNION (b INTERSECT c)".
func (p *Parser) parseUnionChain(first pgast.Statement) pgast.Statement {
	left := p.parseIntersectChain(first)

	for p.at(UNION) || p.at(EXCEPT) {
		op := pgast.Union
		if p.at(EXCEPT) {
			op = pgast.Except
		}

		pos := p.tok.Pos
		p.advance()

		all := p.parseSetQuantifier()
		right := p.parseIntersectChain(p.parseSetOpOperand())

		left = newSetOp(pos, left, op, all, right)
	}

	return left
}

// parseIntersectChain parses left-associative INTERSECT operators following
// first.
func (p *Parser) parseIntersectChain(first pgast.Statement) pgast.Statement {
	left := first

	for p.at(INTERSECT) {
		pos := p.tok.Pos
		p.advance()

		all := p.parseSetQuantifier()
		right := p.parseSetOpOperand()

		left = newSetOp(pos, left, pgast.Intersect, all, right)
	}

	return left
}

// lockingClauseMsg is the error for a locking clause on a set operation or
// on one of its operands, which PostgreSQL rejects alike.
const lockingClauseMsg = "FOR UPDATE/SHARE is not allowed with UNION/INTERSECT/EXCEPT"

// newSetOp combines left and right with the set operator at pos. Like
// PostgreSQL, it rejects an operand that carries its own locking clause,
// as in "(SELECT 1 FOR UPDATE) UNION SELECT 2".
func newSetOp(pos Position, left pgast.Statement, op pgast.SetOpType, all bool, right pgast.Statement) *pgast.SetOp {
	if hasLockingClause(left) || hasLockingClause(right) {
		panic(&ParseError{Pos: pos, Msg: lockingClauseMsg})
	}

	return &pgast.SetOp{Left: left, Op: op, All: all, Right: right}
}

// hasLockingClause reports whether stmt is a SELECT with a locking clause.
func hasLockingClause(stmt pgast.Statement) bool {
	sel, ok := stmt.(*pgast.Select)

	return ok && len(sel.Locks) > 0
}

// parseSetQuantifier parses the optional ALL or DISTINCT following a set
// operator, reporting whether it was ALL. DISTINCT is the default, so it
// isn't modeled separately.
func (p *Parser) parseSetQuantifier() bool {
	if p.consume(ALL) {
		return true
	}

	p.consume(DISTINCT)

	return false
}

// parseSetOpOperand parses one set-operation operand: a bare SELECT core
// (SELECT through WINDOW, leaving any trailing clause to the enclosing query)
// or a parenthesized query, which may carry its own trailing clauses.
func (p *Parser) parseSetOpOperand() pgast.Statement {
	if !p.consume(LPAREN) {
		return p.parseSimpleSelect()
	}

	stmt := p.parseQuery()
	p.expect(RPAREN)

	return stmt
}

// queryTail holds the trailing clauses that follow a query's set-operation
// chain, before they are attached to the statement they apply to.
type queryTail struct {
	pos     Position
	orderBy pgast.OrderBy
	limit   *pgast.Limit
	offset  *pgast.Offset
	fetch   *pgast.Fetch
	locks   []*pgast.Lock
}

// parseQueryTail parses the optional ORDER BY, LIMIT/OFFSET/FETCH, and
// locking clauses that can follow a query's set-operation chain.
func (p *Parser) parseQueryTail() *queryTail {
	tail := &queryTail{pos: p.tok.Pos}
	tail.orderBy = p.parseOptionalOrderBy()
	p.parseRowLimitClauses(tail)

	for lock := p.parseOptionalLock(); lock != nil; lock = p.parseOptionalLock() {
		tail.locks = append(tail.locks, lock)
	}

	return tail
}

// attachQueryTail attaches tail's clauses to stmt. stmt may already carry
// clauses of its own when it is a parenthesized query, as in
// "(SELECT 1 ORDER BY 1) LIMIT 2"; like PostgreSQL, this rejects a clause
// given at both levels rather than silently picking one.
func (p *Parser) attachQueryTail(stmt pgast.Statement, tail *queryTail) {
	switch s := stmt.(type) {
	case *pgast.Select:
		checkTailConflicts(tail, len(s.OrderBy) > 0, s.Limit != nil || s.Fetch != nil, s.Offset != nil)
		s.OrderBy = mergeOrderBy(s.OrderBy, tail.orderBy)
		s.Limit = mergeClause(s.Limit, tail.limit)
		s.Offset = mergeClause(s.Offset, tail.offset)
		s.Fetch = mergeClause(s.Fetch, tail.fetch)
		s.Locks = append(s.Locks, tail.locks...)
	case *pgast.SetOp:
		checkTailConflicts(tail, len(s.OrderBy) > 0, s.Limit != nil || s.Fetch != nil, s.Offset != nil)

		// PostgreSQL rejects locking a set operation's result outright, so
		// pgast.SetOp has nowhere to hold one.
		if len(tail.locks) > 0 {
			panic(&ParseError{Pos: tail.pos, Msg: lockingClauseMsg})
		}

		s.OrderBy = mergeOrderBy(s.OrderBy, tail.orderBy)
		s.Limit = mergeClause(s.Limit, tail.limit)
		s.Offset = mergeClause(s.Offset, tail.offset)
		s.Fetch = mergeClause(s.Fetch, tail.fetch)
	}
}

// checkTailConflicts fails if tail repeats a clause the statement it is
// attached to already has. LIMIT and FETCH count as the same clause, since
// FETCH is the SQL-standard spelling of LIMIT.
func checkTailConflicts(tail *queryTail, hasOrderBy, hasLimit, hasOffset bool) {
	switch {
	case hasOrderBy && len(tail.orderBy) > 0:
		panic(&ParseError{Pos: tail.pos, Msg: "multiple ORDER BY clauses not allowed"})
	case hasLimit && (tail.limit != nil || tail.fetch != nil):
		panic(&ParseError{Pos: tail.pos, Msg: "multiple LIMIT clauses not allowed"})
	case hasOffset && tail.offset != nil:
		panic(&ParseError{Pos: tail.pos, Msg: "multiple OFFSET clauses not allowed"})
	}
}

// mergeOrderBy returns added if it is non-empty, or existing otherwise.
func mergeOrderBy(existing, added pgast.OrderBy) pgast.OrderBy {
	if len(added) > 0 {
		return added
	}

	return existing
}

// mergeClause returns added if it is non-nil, or existing otherwise.
func mergeClause[T any](existing, added *T) *T {
	if added != nil {
		return added
	}

	return existing
}
