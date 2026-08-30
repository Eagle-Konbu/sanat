package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// parseSubqueryStatement parses the SELECT statement inside a parenthesized
// subquery, EXISTS (...), or [NOT] IN (...) predicate. WITH is deferred to
// #86 (PostgreSQL CTEs can wrap DML bodies, not just SELECT, so CTE support
// is sequenced after INSERT/UPDATE/DELETE land), so the current token must
// be SELECT.
func (p *Parser) parseSubqueryStatement() pgast.Statement {
	return p.parseSelectStatement()
}

// parseSelectStatement parses a SELECT statement. The current token must be
// SELECT.
func (p *Parser) parseSelectStatement() *pgast.Select {
	p.expect(SELECT)

	sel := &pgast.Select{}
	p.parseSelectModifiers(sel)

	sel.SelectExprs = p.parseSelectExprList()
	sel.From = p.parseOptionalFromClause()
	sel.Where = p.parseOptionalWhereClause(WHERE)
	sel.GroupBy = p.parseOptionalGroupBy()
	sel.Having = p.parseOptionalWhereClause(HAVING)
	sel.Window = p.parseOptionalWindowClause()
	sel.OrderBy = p.parseOptionalOrderBy()
	sel.Limit = p.parseOptionalLimit()
	sel.Offset = p.parseOptionalOffset()
	sel.Fetch = p.parseOptionalFetch()
	sel.Lock = p.parseOptionalLock()

	return sel
}

// parseSelectModifiers parses the [ALL | DISTINCT [ON (...)]] modifier
// following SELECT.
func (p *Parser) parseSelectModifiers(sel *pgast.Select) {
	switch {
	case p.consume(ALL):
	case p.consume(DISTINCT):
		if !p.consume(ON) {
			sel.Distinct = true

			return
		}

		p.expect(LPAREN)
		exprs := p.parseExprList()
		p.expect(RPAREN)

		sel.DistinctOn = &pgast.DistinctOn{Exprs: exprs}
	}
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

	if star := p.tryParseQualifiedStar(); star != nil {
		return star
	}

	expr := p.parseExpr()
	alias := p.parseOptionalAlias()

	return &pgast.AliasedExpr{Expr: expr, As: pgast.ColIdent(alias)}
}

// tryParseQualifiedStar consumes and returns a `table.*` select item if the
// next three tokens spell one out; otherwise it consumes nothing and
// returns nil, leaving the caller to parse a general expression.
func (p *Parser) tryParseQualifiedStar() pgast.SelectExpr {
	if (!p.at(IDENT) && !p.at(QuotedIdent)) || !p.peekAt(DOT) || !p.peek2At(STAR) {
		return nil
	}

	name := p.tok.Literal

	for range 3 { // ident, '.', '*'
		p.advance()
	}

	return &pgast.StarExpr{TableName: pgast.TableName{Name: pgast.TableIdent(name)}}
}

// parseOptionalAlias parses an optional trailing "[AS] name" alias (used for
// both select-list and table aliases), returning "" if none is present.
func (p *Parser) parseOptionalAlias() string {
	if p.consume(AS) {
		return p.readIdent()
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		return p.readIdent()
	}

	return ""
}

func (p *Parser) readIdent() string {
	if p.tok.Type != IDENT && p.tok.Type != QuotedIdent {
		p.failf("expected identifier, got %s", p.tok.Type)
	}

	lit := p.tok.Literal
	p.advance()

	return lit
}

func (p *Parser) parseOptionalFromClause() []pgast.TableExpr {
	if !p.consume(FROM) {
		return nil
	}

	return p.parseTableReferenceList()
}

func (p *Parser) parseTableReferenceList() []pgast.TableExpr {
	var tables []pgast.TableExpr

	for {
		tables = append(tables, p.parseTableReference())

		if !p.consume(COMMA) {
			return tables
		}
	}
}

func (p *Parser) parseTableReference() pgast.TableExpr {
	left := p.parseTableFactor()

	for p.isJoinStart() {
		left = p.parseJoin(left)
	}

	return left
}

func (p *Parser) isJoinStart() bool {
	switch p.tok.Type {
	case JOIN, LEFT, RIGHT, FULL, INNER, CROSS, NATURAL:
		return true
	default:
		return false
	}
}

func (p *Parser) parseJoin(left pgast.TableExpr) pgast.TableExpr {
	joinType := p.parseJoinType()
	right := p.parseTableFactor()

	// NATURAL joins determine their join columns implicitly and PostgreSQL
	// rejects an explicit ON/USING alongside NATURAL, so this doesn't even
	// look for one: any ON/USING left in the token stream is then unconsumed
	// input, which fails elsewhere with a parse error instead of being
	// silently accepted (and possibly dropped) here.
	if isNaturalJoinType(joinType) {
		return &pgast.JoinTableExpr{Left: left, Join: joinType, Right: right}
	}

	cond := p.parseOptionalJoinCondition()
	if cond == nil && joinRequiresCondition(joinType) {
		p.failf("expected ON or USING after %s", joinType)
	}

	return &pgast.JoinTableExpr{Left: left, Join: joinType, Right: right, Condition: cond}
}

// joinRequiresCondition reports whether PostgreSQL requires an ON or USING
// clause for joinType. INNER/CROSS may omit one; LEFT, RIGHT, and FULL
// (OUTER) may not. NATURAL joins are handled separately in parseJoin, since
// they may not have a condition at all.
func joinRequiresCondition(joinType pgast.JoinType) bool {
	switch joinType {
	case pgast.JoinLeft, pgast.JoinRight, pgast.JoinFull:
		return true
	default:
		return false
	}
}

func isNaturalJoinType(joinType pgast.JoinType) bool {
	switch joinType {
	case pgast.JoinNatural, pgast.JoinNaturalLeft, pgast.JoinNaturalRight, pgast.JoinNaturalFull:
		return true
	default:
		return false
	}
}

func (p *Parser) parseJoinType() pgast.JoinType {
	switch p.tok.Type {
	case CROSS:
		p.advance()
		p.expect(JOIN)

		return pgast.JoinCross
	case NATURAL:
		return p.parseNaturalJoinType()
	case LEFT:
		p.advanceOuterJoin()

		return pgast.JoinLeft
	case RIGHT:
		p.advanceOuterJoin()

		return pgast.JoinRight
	case FULL:
		p.advanceOuterJoin()

		return pgast.JoinFull
	case INNER:
		p.advance()
		p.expect(JOIN)

		return pgast.JoinInner
	case JOIN:
		p.advance()

		return pgast.JoinInner
	default:
		return failReturn[pgast.JoinType](p, "expected JOIN clause, got %s", p.tok.Type)
	}
}

// advanceOuterJoin consumes a LEFT/RIGHT/FULL token, an optional OUTER, and
// the required trailing JOIN.
func (p *Parser) advanceOuterJoin() {
	p.advance()
	p.consume(OUTER)
	p.expect(JOIN)
}

func (p *Parser) parseNaturalJoinType() pgast.JoinType {
	p.advance() // consume NATURAL

	switch p.tok.Type {
	case LEFT:
		p.advanceOuterJoin()

		return pgast.JoinNaturalLeft
	case RIGHT:
		p.advanceOuterJoin()

		return pgast.JoinNaturalRight
	case FULL:
		p.advanceOuterJoin()

		return pgast.JoinNaturalFull
	default:
		p.expect(JOIN)

		return pgast.JoinNatural
	}
}

func (p *Parser) parseOptionalJoinCondition() *pgast.JoinCondition {
	if p.consume(ON) {
		return &pgast.JoinCondition{On: p.parseExpr()}
	}

	if p.consume(USING) {
		p.expect(LPAREN)
		cols := p.parseColumnList()
		p.expect(RPAREN)

		return &pgast.JoinCondition{Using: cols}
	}

	return nil
}

func (p *Parser) parseColumnList() pgast.Columns {
	var cols pgast.Columns

	for {
		cols = append(cols, pgast.ColIdent(p.readIdent()))

		if !p.consume(COMMA) {
			return cols
		}
	}
}

// parseTableFactor parses a single FROM-item: a table name, a parenthesized
// table reference list, a derived table (parenthesized subquery), or a
// LATERAL derived table. A bare set-returning function call as a FROM-item
// (LATERAL or not) is deferred, alongside the issue's other deferred
// table-function machinery (WITH ORDINALITY, ROWS FROM(...)) — pgast has no
// function-valued TableExpr/SimpleTableExpr shape yet, and adding one is a
// bigger, separate design than this issue's join/subquery/DISTINCT ON scope.
func (p *Parser) parseTableFactor() pgast.TableExpr {
	if p.consume(LATERAL) {
		return p.parseLateralDerivedTable()
	}

	if p.at(LPAREN) {
		return p.parseParenTableExpr()
	}

	name := p.parseTableName()
	alias := p.parseOptionalAlias()

	return &pgast.AliasedTableExpr{Expr: name, As: pgast.TableIdent(alias)}
}

func (p *Parser) parseLateralDerivedTable() pgast.TableExpr {
	p.expect(LPAREN)

	sel := p.parseSubqueryStatement()

	p.expect(RPAREN)

	alias := p.parseOptionalAlias()

	return &pgast.AliasedTableExpr{Expr: &pgast.DerivedTable{Select: sel}, As: pgast.TableIdent(alias), Lateral: true}
}

func (p *Parser) parseTableName() pgast.TableName {
	name := p.readIdent()

	if p.consume(DOT) {
		return pgast.TableName{Qualifier: pgast.TableIdent(name), Name: pgast.TableIdent(p.readIdent())}
	}

	return pgast.TableName{Name: pgast.TableIdent(name)}
}

func (p *Parser) parseParenTableExpr() pgast.TableExpr {
	p.advance() // consume '('

	if p.at(SELECT) {
		return p.parseDerivedTable()
	}

	var tables []pgast.TableExpr

	for {
		tables = append(tables, p.parseTableReference())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return &pgast.ParenTableExpr{Exprs: tables}
}

func (p *Parser) parseDerivedTable() pgast.TableExpr {
	sel := p.parseSubqueryStatement()

	p.expect(RPAREN)

	alias := p.parseOptionalAlias()

	return &pgast.AliasedTableExpr{Expr: &pgast.DerivedTable{Select: sel}, As: pgast.TableIdent(alias)}
}

func (p *Parser) parseOptionalWhereClause(kw TokenType) *pgast.Where {
	if !p.consume(kw) {
		return nil
	}

	return &pgast.Where{Expr: p.parseExpr()}
}

func (p *Parser) parseOptionalGroupBy() *pgast.GroupBy {
	if !p.consume(GROUP) {
		return nil
	}

	p.expect(BY)

	var elems []pgast.Expr

	for {
		elems = append(elems, p.parseGroupingElement())

		if !p.consume(COMMA) {
			return &pgast.GroupBy{Elements: elems}
		}
	}
}

// parseGroupingElement parses a single GROUP BY item: a plain expression, or
// a GROUPING SETS(...)/CUBE(...)/ROLLUP(...) construct.
func (p *Parser) parseGroupingElement() pgast.Expr {
	switch {
	case p.consume(GROUPING):
		p.expect(SETS)

		return p.parseGroupingSets()
	case p.consume(CUBE):
		p.expect(LPAREN)
		exprs := p.parseExprList()
		p.expect(RPAREN)

		return &pgast.Cube{Exprs: exprs}
	case p.consume(ROLLUP):
		p.expect(LPAREN)
		exprs := p.parseExprList()
		p.expect(RPAREN)

		return &pgast.Rollup{Exprs: exprs}
	default:
		return p.parseExpr()
	}
}

func (p *Parser) parseGroupingSets() pgast.Expr {
	p.expect(LPAREN)

	var sets [][]pgast.Expr

	for {
		sets = append(sets, p.parseParenExprListOrEmpty())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return &pgast.GroupingSets{Sets: sets}
}

// parseParenExprListOrEmpty parses a "(expr, ...)" list, or "()" for an
// empty set (the grand-total row in a GROUPING SETS list).
func (p *Parser) parseParenExprListOrEmpty() []pgast.Expr {
	p.expect(LPAREN)

	if p.consume(RPAREN) {
		return nil
	}

	exprs := p.parseExprList()
	p.expect(RPAREN)

	return exprs
}

func (p *Parser) parseOptionalWindowClause() []*pgast.NamedWindow {
	if !p.consume(WINDOW) {
		return nil
	}

	var windows []*pgast.NamedWindow

	for {
		name := p.readIdent()
		p.expect(AS)
		p.expect(LPAREN)

		spec := p.parseWindowSpecBody()

		p.expect(RPAREN)

		windows = append(windows, &pgast.NamedWindow{Name: pgast.ColIdent(name), Spec: spec})

		if !p.consume(COMMA) {
			return windows
		}
	}
}

// parseOptionalOver parses a function call's trailing "OVER (...)" or
// "OVER window_name" clause.
func (p *Parser) parseOptionalOver() *pgast.WindowSpec {
	if !p.consume(OVER) {
		return nil
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		name := p.tok.Literal
		p.advance()

		return &pgast.WindowSpec{Name: pgast.ColIdent(name)}
	}

	p.expect(LPAREN)

	spec := p.parseWindowSpecBody()

	p.expect(RPAREN)

	return spec
}

// parseWindowSpecBody parses the content inside a window definition's
// parens (or, equivalently, a top-level WINDOW clause entry's body): an
// optional base window name to refine, followed by PARTITION BY, ORDER BY,
// and a frame clause, each optional. A leading bare identifier here can only
// be the base window name reference — every other production starts with a
// dedicated keyword (PARTITION/ORDER/ROWS/RANGE/GROUPS) or ')'.
func (p *Parser) parseWindowSpecBody() *pgast.WindowSpec {
	spec := &pgast.WindowSpec{}

	if p.at(IDENT) || p.at(QuotedIdent) {
		spec.Name = pgast.ColIdent(p.tok.Literal)
		p.advance()
	}

	if p.consume(PARTITION) {
		p.expect(BY)

		spec.PartitionBy = p.parseExprList()
	}

	if p.at(ORDER) {
		spec.OrderBy = p.parseOrderByClause()
	}

	if p.at(ROWS) || p.at(RANGE) || p.at(GROUPS) {
		spec.Frame = p.parseFrameClause()
	}

	return spec
}

func (p *Parser) parseFrameClause() *pgast.FrameClause {
	unit := pgast.FrameRows

	switch {
	case p.at(RANGE):
		unit = pgast.FrameRange
	case p.at(GROUPS):
		unit = pgast.FrameGroups
	}

	p.advance() // consume ROWS/RANGE/GROUPS

	if p.consume(BETWEEN) {
		start := p.parseFramePoint()
		p.expect(AND)

		end := p.parseFramePoint()

		return &pgast.FrameClause{Unit: unit, Start: start, End: end}
	}

	return &pgast.FrameClause{Unit: unit, Start: p.parseFramePoint()}
}

func (p *Parser) parseFramePoint() *pgast.FramePoint {
	switch {
	case p.consume(CURRENT):
		p.expect(ROW)

		return &pgast.FramePoint{Type: pgast.CurrentRow}
	case p.consume(UNBOUNDED):
		return p.parseUnboundedFramePoint()
	default:
		return p.parseExprFramePoint()
	}
}

func (p *Parser) parseUnboundedFramePoint() *pgast.FramePoint {
	switch {
	case p.consume(PRECEDING):
		return &pgast.FramePoint{Type: pgast.UnboundedPreceding}
	case p.consume(FOLLOWING):
		return &pgast.FramePoint{Type: pgast.UnboundedFollowing}
	default:
		return failReturn[*pgast.FramePoint](p, "expected PRECEDING or FOLLOWING after UNBOUNDED")
	}
}

func (p *Parser) parseExprFramePoint() *pgast.FramePoint {
	expr := p.parseAdditiveExpr()

	switch {
	case p.consume(PRECEDING):
		return &pgast.FramePoint{Type: pgast.ExprPreceding, Expr: expr}
	case p.consume(FOLLOWING):
		return &pgast.FramePoint{Type: pgast.ExprFollowing, Expr: expr}
	default:
		return failReturn[*pgast.FramePoint](p, "expected PRECEDING or FOLLOWING")
	}
}

func (p *Parser) parseOptionalOrderBy() pgast.OrderBy {
	if !p.at(ORDER) {
		return nil
	}

	return p.parseOrderByClause()
}

// parseOrderByClause parses an ORDER BY clause; the current token must be
// ORDER. Shared by top-level SELECT and window specifications.
func (p *Parser) parseOrderByClause() pgast.OrderBy {
	p.advance() // consume ORDER
	p.expect(BY)

	var orders pgast.OrderBy

	for {
		e := p.parseExpr()
		dir := p.parseOptionalOrderDirection()
		nulls := p.parseOptionalNullsOrder()

		orders = append(orders, &pgast.Order{Expr: e, Direction: dir, Nulls: nulls})

		if !p.consume(COMMA) {
			return orders
		}
	}
}

func (p *Parser) parseOptionalOrderDirection() pgast.OrderDirection {
	if p.consume(DESC) {
		return pgast.DescOrder
	}

	p.consume(ASC)

	return pgast.AscOrder
}

func (p *Parser) parseOptionalNullsOrder() pgast.NullsOrder {
	if !p.consume(NULLS) {
		return pgast.NullsDefault
	}

	if p.consume(FIRST) {
		return pgast.NullsFirst
	}

	p.expect(LAST)

	return pgast.NullsLast
}

// parseOptionalLimit parses an optional "LIMIT { count | ALL }" clause.
func (p *Parser) parseOptionalLimit() *pgast.Limit {
	if !p.consume(LIMIT) {
		return nil
	}

	if p.consume(ALL) {
		return &pgast.Limit{}
	}

	return &pgast.Limit{Count: p.parseExpr()}
}

// parseOptionalOffset parses an optional "OFFSET start [ROW | ROWS]" clause;
// the ROW/ROWS noise word (if present) is discarded, not modeled.
func (p *Parser) parseOptionalOffset() *pgast.Offset {
	if !p.consume(OFFSET) {
		return nil
	}

	start := p.parseExpr()

	if !p.consume(ROW) {
		p.consume(ROWS)
	}

	return &pgast.Offset{Start: start}
}

// parseOptionalFetch parses an optional
// "FETCH {FIRST|NEXT} [count] {ROW|ROWS} {ONLY|WITH TIES}" clause.
func (p *Parser) parseOptionalFetch() *pgast.Fetch {
	if !p.consume(FETCH) {
		return nil
	}

	next := p.parseFetchFirstOrNext()

	var count pgast.Expr
	if !p.at(ROW) && !p.at(ROWS) {
		count = p.parseExpr()
	}

	if !p.consume(ROW) {
		p.expect(ROWS)
	}

	withTies := p.parseFetchOnlyOrWithTies()

	return &pgast.Fetch{Count: count, Next: next, WithTies: withTies}
}

func (p *Parser) parseFetchFirstOrNext() bool {
	switch {
	case p.consume(FIRST):
		return false
	case p.consume(NEXT):
		return true
	default:
		return failReturn[bool](p, "expected FIRST or NEXT after FETCH")
	}
}

func (p *Parser) parseFetchOnlyOrWithTies() bool {
	switch {
	case p.consume(ONLY):
		return false
	case p.consume(WITH):
		p.expect(TIES)

		return true
	default:
		return failReturn[bool](p, "expected ONLY or WITH TIES")
	}
}

// parseOptionalLock parses an optional
// "FOR {UPDATE|NO KEY UPDATE|SHARE|KEY SHARE} [OF table, ...] [NOWAIT|SKIP LOCKED]"
// locking clause.
func (p *Parser) parseOptionalLock() *pgast.Lock {
	if !p.consume(FOR) {
		return nil
	}

	strength := p.parseLockStrength()

	var of []pgast.TableName
	if p.consume(OF) {
		of = p.parseTableNameList()
	}

	return &pgast.Lock{Strength: strength, Of: of, Wait: p.parseOptionalLockWait()}
}

func (p *Parser) parseLockStrength() pgast.LockStrength {
	switch {
	case p.consume(UPDATE):
		return pgast.ForUpdate
	case p.consume(NO):
		p.expect(KEY)
		p.expect(UPDATE)

		return pgast.ForNoKeyUpdate
	case p.consume(KEY):
		p.expect(SHARE)

		return pgast.ForKeyShare
	case p.consume(SHARE):
		return pgast.ForShare
	default:
		return failReturn[pgast.LockStrength](p, "expected UPDATE, NO KEY UPDATE, SHARE, or KEY SHARE after FOR")
	}
}

func (p *Parser) parseTableNameList() []pgast.TableName {
	var names []pgast.TableName

	for {
		names = append(names, p.parseTableName())

		if !p.consume(COMMA) {
			return names
		}
	}
}

func (p *Parser) parseOptionalLockWait() pgast.LockWait {
	switch {
	case p.consume(NOWAIT):
		return pgast.NoWait
	case p.consume(SKIP):
		p.expect(LOCKED)

		return pgast.SkipLocked
	default:
		return pgast.NoLockWait
	}
}
