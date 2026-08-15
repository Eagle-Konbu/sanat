package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// parseSelectStatement parses a SELECT statement, optionally preceded by a
// WITH clause. The current token must be WITH or SELECT.
func (p *Parser) parseSelectStatement() *sqlast.Select {
	sel := &sqlast.Select{With: p.parseOptionalWith()}

	p.expect(SELECT)
	p.parseSelectModifiers(sel)

	sel.SelectExprs = p.parseSelectExprList()
	sel.From = p.parseOptionalFromClause()

	p.parseSelectFilters(sel)
	p.parseSelectTail(sel)

	return sel
}

// parseSelectModifiers parses the [DISTINCT|ALL] modifier following SELECT.
func (p *Parser) parseSelectModifiers(sel *sqlast.Select) {
	sel.Distinct = p.consumeDistinct()

	if !sel.Distinct {
		p.consume(ALL)
	}
}

// parseSelectFilters parses the WHERE/GROUP BY/HAVING clauses into sel.
func (p *Parser) parseSelectFilters(sel *sqlast.Select) {
	sel.Where = p.parseOptionalWhereClause(WHERE)
	sel.GroupBy = p.parseOptionalGroupBy()
	sel.Having = p.parseOptionalWhereClause(HAVING)
}

// parseSelectTail parses the ORDER BY/LIMIT/locking clauses into sel.
func (p *Parser) parseSelectTail(sel *sqlast.Select) {
	sel.OrderBy = p.parseOptionalOrderBy()
	sel.Limit = p.parseOptionalLimit()
	sel.Lock, sel.LockWait = p.parseOptionalLock()
}

func (p *Parser) parseOptionalWith() *sqlast.With {
	if !p.consume(WITH) {
		return nil
	}

	recursive := p.consume(RECURSIVE)

	var ctes []*sqlast.CommonTableExpr

	for {
		ctes = append(ctes, p.parseCTE())

		if !p.consume(COMMA) {
			break
		}
	}

	return &sqlast.With{CTEs: ctes, Recursive: recursive}
}

func (p *Parser) parseCTE() *sqlast.CommonTableExpr {
	name := p.readIdent()

	var columns sqlast.Columns

	if p.consume(LPAREN) {
		columns = p.parseIdentList()
		p.expect(RPAREN)
	}

	p.expect(AS)
	p.expect(LPAREN)

	sel := p.parseSelectStatement()

	p.expect(RPAREN)

	return &sqlast.CommonTableExpr{ID: sqlast.TableIdent(name), Columns: columns, Subquery: sel}
}

func (p *Parser) parseIdentList() sqlast.Columns {
	var cols sqlast.Columns

	for {
		cols = append(cols, sqlast.ColIdent(p.readIdent()))

		if !p.consume(COMMA) {
			return cols
		}
	}
}

func (p *Parser) parseSelectExprList() []sqlast.SelectExpr {
	var exprs []sqlast.SelectExpr

	for {
		exprs = append(exprs, p.parseSelectExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

func (p *Parser) parseSelectExpr() sqlast.SelectExpr {
	if p.consume(STAR) {
		return &sqlast.StarExpr{}
	}

	if star := p.tryParseQualifiedStar(); star != nil {
		return star
	}

	expr := p.parseExpr()
	alias := p.parseOptionalColAlias()

	return &sqlast.AliasedExpr{Expr: expr, As: alias}
}

// tryParseQualifiedStar consumes and returns a `table.*` select item if the
// next three tokens spell one out; otherwise it consumes nothing and
// returns nil, leaving the caller to parse a general expression.
func (p *Parser) tryParseQualifiedStar() sqlast.SelectExpr {
	if (!p.at(IDENT) && !p.at(QuotedIdent)) || !p.peekAt(DOT) || !p.peek2At(STAR) {
		return nil
	}

	name := p.tok.Literal

	for range 3 { // ident, '.', '*'
		p.advance()
	}

	return &sqlast.StarExpr{TableName: sqlast.TableName{Name: sqlast.TableIdent(name)}}
}

func (p *Parser) parseOptionalColAlias() sqlast.ColIdent {
	if p.consume(AS) {
		return sqlast.ColIdent(p.readIdent())
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		return sqlast.ColIdent(p.readIdent())
	}

	return ""
}

func (p *Parser) parseOptionalFromClause() []sqlast.TableExpr {
	if !p.consume(FROM) {
		return nil
	}

	var tables []sqlast.TableExpr

	for {
		tables = append(tables, p.parseTableReference())

		if !p.consume(COMMA) {
			return tables
		}
	}
}

func (p *Parser) parseTableReference() sqlast.TableExpr {
	left := p.parseTableFactor()

	for p.isJoinStart() {
		left = p.parseJoin(left)
	}

	return left
}

func (p *Parser) isJoinStart() bool {
	switch p.tok.Type {
	case JOIN, LEFT, RIGHT, INNER, OUTER, CROSS, NATURAL, StraightJoin:
		return true
	default:
		return false
	}
}

func (p *Parser) parseJoin(left sqlast.TableExpr) sqlast.TableExpr {
	joinType := p.parseJoinType()
	right := p.parseTableFactor()

	var cond *sqlast.JoinCondition

	if p.consume(ON) {
		cond = &sqlast.JoinCondition{On: p.parseExpr()}
	}

	return &sqlast.JoinTableExpr{LeftExpr: left, Join: joinType, RightExpr: right, Condition: cond}
}

func (p *Parser) parseJoinType() sqlast.JoinType {
	switch p.tok.Type {
	case StraightJoin:
		p.advance()

		return sqlast.StraightJoinType
	case CROSS:
		p.advance()
		p.expect(JOIN)

		return sqlast.CrossJoinType
	case NATURAL:
		return p.parseNaturalJoinType()
	case LEFT:
		p.advanceOuterJoin()

		return sqlast.LeftJoinType
	case RIGHT:
		p.advanceOuterJoin()

		return sqlast.RightJoinType
	case INNER:
		p.advance()
		p.expect(JOIN)

		return sqlast.NormalJoinType
	case JOIN:
		p.advance()

		return sqlast.NormalJoinType
	default:
		return failReturn[sqlast.JoinType](p, "expected JOIN clause, got %s", p.tok.Type)
	}
}

// advanceOuterJoin consumes a LEFT/RIGHT token, an optional OUTER, and the
// required trailing JOIN.
func (p *Parser) advanceOuterJoin() {
	p.advance()
	p.consume(OUTER)
	p.expect(JOIN)
}

func (p *Parser) parseNaturalJoinType() sqlast.JoinType {
	p.advance() // consume NATURAL

	switch p.tok.Type {
	case LEFT:
		p.advanceOuterJoin()

		return sqlast.NaturalLeftJoinType
	case RIGHT:
		p.advanceOuterJoin()

		return sqlast.NaturalRightJoinType
	default:
		p.expect(JOIN)

		return sqlast.NaturalJoinType
	}
}

func (p *Parser) parseTableFactor() sqlast.TableExpr {
	if p.at(LPAREN) {
		return p.parseParenTableExpr()
	}

	name := p.parseTableName()
	alias := p.parseOptionalTableAlias()
	hints := p.parseOptionalIndexHints()

	return &sqlast.AliasedTableExpr{Expr: name, As: alias, Hints: hints}
}

func (p *Parser) parseTableName() sqlast.TableName {
	name := p.readIdent()

	if p.consume(DOT) {
		return sqlast.TableName{Qualifier: sqlast.TableIdent(name), Name: sqlast.TableIdent(p.readIdent())}
	}

	return sqlast.TableName{Name: sqlast.TableIdent(name)}
}

func (p *Parser) parseParenTableExpr() sqlast.TableExpr {
	p.advance() // consume '('

	if p.at(SELECT) || p.at(WITH) {
		return p.parseDerivedTable()
	}

	var tables []sqlast.TableExpr

	for {
		tables = append(tables, p.parseTableReference())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return &sqlast.ParenTableExpr{Exprs: tables}
}

func (p *Parser) parseDerivedTable() sqlast.TableExpr {
	sel := p.parseSelectStatement()

	p.expect(RPAREN)

	alias := p.parseOptionalTableAlias()

	return &sqlast.AliasedTableExpr{Expr: &sqlast.DerivedTable{Select: sel}, As: alias}
}

func (p *Parser) parseOptionalTableAlias() sqlast.TableIdent {
	if p.consume(AS) {
		return sqlast.TableIdent(p.readIdent())
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		return sqlast.TableIdent(p.readIdent())
	}

	return ""
}

func (p *Parser) parseOptionalIndexHints() sqlast.IndexHints {
	var hints sqlast.IndexHints

	for p.at(USE) || p.at(FORCE) || p.at(IGNORE) {
		hints = append(hints, p.parseIndexHint())
	}

	return hints
}

var indexHintTypes = map[TokenType]sqlast.IndexHintType{
	USE:    sqlast.UseOp,
	FORCE:  sqlast.ForceOp,
	IGNORE: sqlast.IgnoreOp,
}

func (p *Parser) parseIndexHint() *sqlast.IndexHint {
	typ := indexHintTypes[p.tok.Type]
	p.advance()
	p.expect(INDEX)

	forType := p.parseOptionalIndexHintFor()

	p.expect(LPAREN)

	indexes := p.parseIdentList()

	p.expect(RPAREN)

	return &sqlast.IndexHint{Type: typ, ForType: forType, Indexes: indexes}
}

func (p *Parser) parseOptionalIndexHintFor() sqlast.IndexHintForType {
	if !p.consume(FOR) {
		return sqlast.NoForType
	}

	switch p.tok.Type {
	case JOIN:
		p.advance()

		return sqlast.JoinForType
	case GROUP:
		p.advance()
		p.expect(BY)

		return sqlast.GroupByForType
	case ORDER:
		p.advance()
		p.expect(BY)

		return sqlast.OrderByForType
	default:
		return failReturn[sqlast.IndexHintForType](p, "expected JOIN, GROUP BY, or ORDER BY after FOR")
	}
}

func (p *Parser) parseOptionalWhereClause(kw TokenType) *sqlast.Where {
	if !p.consume(kw) {
		return nil
	}

	return &sqlast.Where{Expr: p.parseExpr()}
}

func (p *Parser) parseOptionalGroupBy() *sqlast.GroupBy {
	if !p.consume(GROUP) {
		return nil
	}

	p.expect(BY)

	return &sqlast.GroupBy{Exprs: p.parseExprList()}
}

func (p *Parser) parseOptionalOrderBy() sqlast.OrderBy {
	if !p.at(ORDER) {
		return nil
	}

	return p.parseOrderByClause()
}

func (p *Parser) parseOptionalOrderDirection() sqlast.OrderDirection {
	if p.consume(DESC) {
		return sqlast.DescOrder
	}

	p.consume(ASC)

	return sqlast.AscOrder
}

// parseOrderByClause parses an ORDER BY clause; the current token must be
// ORDER. Shared by top-level SELECT/UPDATE/DELETE and window specifications.
func (p *Parser) parseOrderByClause() sqlast.OrderBy {
	p.advance() // consume ORDER
	p.expect(BY)

	var orders sqlast.OrderBy

	for {
		e := p.parseExpr()
		dir := p.parseOptionalOrderDirection()

		orders = append(orders, &sqlast.Order{Expr: e, Direction: dir})

		if !p.consume(COMMA) {
			return orders
		}
	}
}

func (p *Parser) parseOptionalLimit() *sqlast.Limit {
	if !p.consume(LIMIT) {
		return nil
	}

	count := p.parseExpr()

	var offset sqlast.Expr

	if p.consume(OFFSET) {
		offset = p.parseExpr()
	}

	return &sqlast.Limit{Rowcount: count, Offset: offset}
}

func (p *Parser) parseOptionalLock() (sqlast.Lock, sqlast.LockWaitType) {
	switch {
	case p.at(FOR):
		return p.parseForLock()
	case p.at(LOCK):
		p.parseLockInShareMode()

		return sqlast.ShareModeLock, sqlast.NoLockWait
	default:
		return sqlast.NoLock, sqlast.NoLockWait
	}
}

func (p *Parser) parseLockInShareMode() {
	p.advance() // consume LOCK
	p.expect(IN)
	p.expect(SHARE)
	p.expect(MODE)
}

func (p *Parser) parseForLock() (sqlast.Lock, sqlast.LockWaitType) {
	p.advance() // consume FOR

	var lock sqlast.Lock

	switch p.tok.Type {
	case UPDATE:
		lock = sqlast.ForUpdateLock
	case SHARE:
		lock = sqlast.ForShareLock
	default:
		p.failf("expected UPDATE or SHARE after FOR")
	}

	p.advance()

	return lock, p.parseOptionalLockWait()
}

func (p *Parser) parseOptionalLockWait() sqlast.LockWaitType {
	switch {
	case p.consume(NOWAIT):
		return sqlast.NowaitType
	case p.consume(SKIP):
		p.expect(LOCKED)

		return sqlast.SkipLockedType
	default:
		return sqlast.NoLockWait
	}
}
