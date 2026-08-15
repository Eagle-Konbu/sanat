package parser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"

// parseSelectStatement parses a SELECT statement, optionally preceded by a
// WITH clause. The current token must be WITH or SELECT.
func (p *Parser) parseSelectStatement() (*sqlast.Select, error) {
	sel := &sqlast.Select{}

	with, err := p.parseOptionalWith()
	if err != nil {
		return nil, err
	}

	sel.With = with

	if err := p.expect(SELECT); err != nil {
		return nil, err
	}

	if err := p.parseSelectModifiers(sel); err != nil {
		return nil, err
	}

	if sel.SelectExprs, err = p.parseSelectExprList(); err != nil {
		return nil, err
	}

	if sel.From, err = p.parseOptionalFromClause(); err != nil {
		return nil, err
	}

	if err := p.parseSelectFilters(sel); err != nil {
		return nil, err
	}

	if err := p.parseSelectTail(sel); err != nil {
		return nil, err
	}

	return sel, nil
}

// parseSelectModifiers parses the [DISTINCT|ALL] modifier following SELECT.
func (p *Parser) parseSelectModifiers(sel *sqlast.Select) error {
	distinct, err := p.consumeDistinct()
	if err != nil {
		return err
	}

	sel.Distinct = distinct

	if !distinct {
		if _, err := p.consume(ALL); err != nil {
			return err
		}
	}

	return nil
}

// parseSelectFilters parses the WHERE/GROUP BY/HAVING clauses into sel.
func (p *Parser) parseSelectFilters(sel *sqlast.Select) error {
	var err error

	if sel.Where, err = p.parseOptionalWhereClause(WHERE); err != nil {
		return err
	}

	if sel.GroupBy, err = p.parseOptionalGroupBy(); err != nil {
		return err
	}

	sel.Having, err = p.parseOptionalWhereClause(HAVING)

	return err
}

// parseSelectTail parses the ORDER BY/LIMIT/locking clauses into sel.
func (p *Parser) parseSelectTail(sel *sqlast.Select) error {
	var err error

	if sel.OrderBy, err = p.parseOptionalOrderBy(); err != nil {
		return err
	}

	if sel.Limit, err = p.parseOptionalLimit(); err != nil {
		return err
	}

	sel.Lock, sel.LockWait, err = p.parseOptionalLock()

	return err
}

func (p *Parser) parseOptionalWith() (*sqlast.With, error) {
	if ok, err := p.consume(WITH); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	recursive, err := p.consume(RECURSIVE)
	if err != nil {
		return nil, err
	}

	var ctes []*sqlast.CommonTableExpr

	for {
		cte, err := p.parseCTE()
		if err != nil {
			return nil, err
		}

		ctes = append(ctes, cte)

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			break
		}
	}

	return &sqlast.With{CTEs: ctes, Recursive: recursive}, nil
}

func (p *Parser) parseCTE() (*sqlast.CommonTableExpr, error) {
	name, err := p.readIdent()
	if err != nil {
		return nil, err
	}

	var columns sqlast.Columns

	if p.at(LPAREN) {
		if err := p.advance(); err != nil {
			return nil, err
		}

		columns, err = p.parseIdentList()
		if err != nil {
			return nil, err
		}

		if err := p.expect(RPAREN); err != nil {
			return nil, err
		}
	}

	if err := p.expect(AS); err != nil {
		return nil, err
	}

	if err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	sel, err := p.parseSelectStatement()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.CommonTableExpr{ID: sqlast.TableIdent(name), Columns: columns, Subquery: sel}, nil
}

func (p *Parser) parseIdentList() (sqlast.Columns, error) {
	var cols sqlast.Columns

	for {
		name, err := p.readIdent()
		if err != nil {
			return nil, err
		}

		cols = append(cols, sqlast.ColIdent(name))

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			return cols, nil
		}
	}
}

func (p *Parser) parseSelectExprList() ([]sqlast.SelectExpr, error) {
	var exprs []sqlast.SelectExpr

	for {
		e, err := p.parseSelectExpr()
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, e)

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			return exprs, nil
		}
	}
}

func (p *Parser) parseSelectExpr() (sqlast.SelectExpr, error) {
	if ok, err := p.consume(STAR); err != nil {
		return nil, err
	} else if ok {
		return &sqlast.StarExpr{}, nil
	}

	if star, err := p.tryParseQualifiedStar(); err != nil || star != nil {
		return star, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	alias, err := p.parseOptionalColAlias()
	if err != nil {
		return nil, err
	}

	return &sqlast.AliasedExpr{Expr: expr, As: alias}, nil
}

// tryParseQualifiedStar consumes and returns a `table.*` select item if the
// next three tokens spell one out; otherwise it consumes nothing and
// returns (nil, nil), leaving the caller to parse a general expression.
func (p *Parser) tryParseQualifiedStar() (sqlast.SelectExpr, error) {
	if (!p.at(IDENT) && !p.at(QuotedIdent)) || !p.peekAt(DOT) || !p.peek2At(STAR) {
		return nil, nil
	}

	name := p.tok.Literal

	for range 3 { // ident, '.', '*'
		if err := p.advance(); err != nil {
			return nil, err
		}
	}

	return &sqlast.StarExpr{TableName: sqlast.TableName{Name: sqlast.TableIdent(name)}}, nil
}

func (p *Parser) parseOptionalColAlias() (sqlast.ColIdent, error) {
	if ok, err := p.consume(AS); err != nil {
		return "", err
	} else if ok {
		name, err := p.readIdent()
		if err != nil {
			return "", err
		}

		return sqlast.ColIdent(name), nil
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		name, err := p.readIdent()
		if err != nil {
			return "", err
		}

		return sqlast.ColIdent(name), nil
	}

	return "", nil
}

func (p *Parser) parseOptionalFromClause() ([]sqlast.TableExpr, error) {
	if ok, err := p.consume(FROM); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	var tables []sqlast.TableExpr

	for {
		t, err := p.parseTableReference()
		if err != nil {
			return nil, err
		}

		tables = append(tables, t)

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			return tables, nil
		}
	}
}

func (p *Parser) parseTableReference() (sqlast.TableExpr, error) {
	left, err := p.parseTableFactor()
	if err != nil {
		return nil, err
	}

	for p.isJoinStart() {
		left, err = p.parseJoin(left)
		if err != nil {
			return nil, err
		}
	}

	return left, nil
}

func (p *Parser) isJoinStart() bool {
	switch p.tok.Type {
	case JOIN, LEFT, RIGHT, INNER, OUTER, CROSS, NATURAL, StraightJoin:
		return true
	default:
		return false
	}
}

func (p *Parser) parseJoin(left sqlast.TableExpr) (sqlast.TableExpr, error) {
	joinType, err := p.parseJoinType()
	if err != nil {
		return nil, err
	}

	right, err := p.parseTableFactor()
	if err != nil {
		return nil, err
	}

	var cond *sqlast.JoinCondition

	if ok, err := p.consume(ON); err != nil {
		return nil, err
	} else if ok {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		cond = &sqlast.JoinCondition{On: e}
	}

	return &sqlast.JoinTableExpr{LeftExpr: left, Join: joinType, RightExpr: right, Condition: cond}, nil
}

func (p *Parser) parseJoinType() (sqlast.JoinType, error) {
	switch p.tok.Type {
	case StraightJoin:
		return sqlast.StraightJoinType, p.advance()
	case CROSS:
		return sqlast.CrossJoinType, p.advanceThenExpect(JOIN)
	case NATURAL:
		return p.parseNaturalJoinType()
	case LEFT:
		return sqlast.LeftJoinType, p.advanceOuterJoin()
	case RIGHT:
		return sqlast.RightJoinType, p.advanceOuterJoin()
	case INNER:
		return sqlast.NormalJoinType, p.advanceThenExpect(JOIN)
	case JOIN:
		return sqlast.NormalJoinType, p.advance()
	default:
		return 0, p.errorf("expected JOIN clause, got %s", p.tok.Type)
	}
}

// advanceThenExpect consumes the current token, then requires tt next.
func (p *Parser) advanceThenExpect(tt TokenType) error {
	if err := p.advance(); err != nil {
		return err
	}

	return p.expect(tt)
}

// advanceOuterJoin consumes a LEFT/RIGHT token, an optional OUTER, and the
// required trailing JOIN.
func (p *Parser) advanceOuterJoin() error {
	if err := p.advance(); err != nil {
		return err
	}

	if _, err := p.consume(OUTER); err != nil {
		return err
	}

	return p.expect(JOIN)
}

func (p *Parser) parseNaturalJoinType() (sqlast.JoinType, error) {
	if err := p.advance(); err != nil { // consume NATURAL
		return 0, err
	}

	switch p.tok.Type {
	case LEFT:
		return sqlast.NaturalLeftJoinType, p.advanceOuterJoin()
	case RIGHT:
		return sqlast.NaturalRightJoinType, p.advanceOuterJoin()
	default:
		return sqlast.NaturalJoinType, p.expect(JOIN)
	}
}

func (p *Parser) parseTableFactor() (sqlast.TableExpr, error) {
	if p.at(LPAREN) {
		return p.parseParenTableExpr()
	}

	name, err := p.parseTableName()
	if err != nil {
		return nil, err
	}

	alias, err := p.parseOptionalTableAlias()
	if err != nil {
		return nil, err
	}

	hints, err := p.parseOptionalIndexHints()
	if err != nil {
		return nil, err
	}

	return &sqlast.AliasedTableExpr{Expr: name, As: alias, Hints: hints}, nil
}

func (p *Parser) parseTableName() (sqlast.TableName, error) {
	name, err := p.readIdent()
	if err != nil {
		return sqlast.TableName{}, err
	}

	if ok, err := p.consume(DOT); err != nil {
		return sqlast.TableName{}, err
	} else if ok {
		table, err := p.readIdent()
		if err != nil {
			return sqlast.TableName{}, err
		}

		return sqlast.TableName{Qualifier: sqlast.TableIdent(name), Name: sqlast.TableIdent(table)}, nil
	}

	return sqlast.TableName{Name: sqlast.TableIdent(name)}, nil
}

func (p *Parser) parseParenTableExpr() (sqlast.TableExpr, error) {
	if err := p.advance(); err != nil { // consume '('
		return nil, err
	}

	if p.at(SELECT) || p.at(WITH) {
		return p.parseDerivedTable()
	}

	var tables []sqlast.TableExpr

	for {
		t, err := p.parseTableReference()
		if err != nil {
			return nil, err
		}

		tables = append(tables, t)

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			break
		}
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.ParenTableExpr{Exprs: tables}, nil
}

func (p *Parser) parseDerivedTable() (sqlast.TableExpr, error) {
	sel, err := p.parseSelectStatement()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	alias, err := p.parseOptionalTableAlias()
	if err != nil {
		return nil, err
	}

	return &sqlast.AliasedTableExpr{Expr: &sqlast.DerivedTable{Select: sel}, As: alias}, nil
}

func (p *Parser) parseOptionalTableAlias() (sqlast.TableIdent, error) {
	if ok, err := p.consume(AS); err != nil {
		return "", err
	} else if ok {
		name, err := p.readIdent()
		if err != nil {
			return "", err
		}

		return sqlast.TableIdent(name), nil
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		name, err := p.readIdent()
		if err != nil {
			return "", err
		}

		return sqlast.TableIdent(name), nil
	}

	return "", nil
}

func (p *Parser) parseOptionalIndexHints() (sqlast.IndexHints, error) {
	var hints sqlast.IndexHints

	for p.at(USE) || p.at(FORCE) || p.at(IGNORE) {
		h, err := p.parseIndexHint()
		if err != nil {
			return nil, err
		}

		hints = append(hints, h)
	}

	return hints, nil
}

var indexHintTypes = map[TokenType]sqlast.IndexHintType{
	USE:    sqlast.UseOp,
	FORCE:  sqlast.ForceOp,
	IGNORE: sqlast.IgnoreOp,
}

func (p *Parser) parseIndexHint() (*sqlast.IndexHint, error) {
	typ := indexHintTypes[p.tok.Type]

	if err := p.advance(); err != nil {
		return nil, err
	}

	if err := p.expect(INDEX); err != nil {
		return nil, err
	}

	forType, err := p.parseOptionalIndexHintFor()
	if err != nil {
		return nil, err
	}

	if err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	indexes, err := p.parseIdentList()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.IndexHint{Type: typ, ForType: forType, Indexes: indexes}, nil
}

func (p *Parser) parseOptionalIndexHintFor() (sqlast.IndexHintForType, error) {
	if ok, err := p.consume(FOR); err != nil {
		return sqlast.NoForType, err
	} else if !ok {
		return sqlast.NoForType, nil
	}

	switch p.tok.Type {
	case JOIN:
		return sqlast.JoinForType, p.advance()
	case GROUP:
		return sqlast.GroupByForType, p.advanceThenExpect(BY)
	case ORDER:
		return sqlast.OrderByForType, p.advanceThenExpect(BY)
	default:
		return sqlast.NoForType, p.errorf("expected JOIN, GROUP BY, or ORDER BY after FOR")
	}
}

func (p *Parser) parseOptionalWhereClause(kw TokenType) (*sqlast.Where, error) {
	if ok, err := p.consume(kw); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	e, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.Where{Expr: e}, nil
}

func (p *Parser) parseOptionalGroupBy() (*sqlast.GroupBy, error) {
	if ok, err := p.consume(GROUP); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	if err := p.expect(BY); err != nil {
		return nil, err
	}

	exprs, err := p.parseExprList()
	if err != nil {
		return nil, err
	}

	return &sqlast.GroupBy{Exprs: exprs}, nil
}

func (p *Parser) parseOptionalOrderBy() (sqlast.OrderBy, error) {
	if !p.at(ORDER) {
		return nil, nil
	}

	return p.parseOrderByClause()
}

func (p *Parser) parseOptionalOrderDirection() (sqlast.OrderDirection, error) {
	if ok, err := p.consume(DESC); err != nil {
		return sqlast.AscOrder, err
	} else if ok {
		return sqlast.DescOrder, nil
	}

	_, err := p.consume(ASC)

	return sqlast.AscOrder, err
}

// parseOrderByClause parses an ORDER BY clause; the current token must be
// ORDER. Shared by top-level SELECT/UPDATE/DELETE and window specifications.
func (p *Parser) parseOrderByClause() (sqlast.OrderBy, error) {
	if err := p.advanceThenExpect(BY); err != nil {
		return nil, err
	}

	var orders sqlast.OrderBy

	for {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		dir, err := p.parseOptionalOrderDirection()
		if err != nil {
			return nil, err
		}

		orders = append(orders, &sqlast.Order{Expr: e, Direction: dir})

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			return orders, nil
		}
	}
}

func (p *Parser) parseOptionalLimit() (*sqlast.Limit, error) {
	if ok, err := p.consume(LIMIT); err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	count, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	var offset sqlast.Expr

	if ok, err := p.consume(OFFSET); err != nil {
		return nil, err
	} else if ok {
		offset, err = p.parseExpr()
		if err != nil {
			return nil, err
		}
	}

	return &sqlast.Limit{Rowcount: count, Offset: offset}, nil
}

func (p *Parser) parseOptionalLock() (sqlast.Lock, sqlast.LockWaitType, error) {
	switch {
	case p.at(FOR):
		return p.parseForLock()
	case p.at(LOCK):
		return sqlast.ShareModeLock, sqlast.NoLockWait, p.parseLockInShareMode()
	default:
		return sqlast.NoLock, sqlast.NoLockWait, nil
	}
}

func (p *Parser) parseLockInShareMode() error {
	if err := p.advance(); err != nil { // consume LOCK
		return err
	}

	if err := p.expect(IN); err != nil {
		return err
	}

	if err := p.expect(SHARE); err != nil {
		return err
	}

	return p.expect(MODE)
}

func (p *Parser) parseForLock() (sqlast.Lock, sqlast.LockWaitType, error) {
	if err := p.advance(); err != nil { // consume FOR
		return 0, 0, err
	}

	var lock sqlast.Lock

	switch p.tok.Type {
	case UPDATE:
		lock = sqlast.ForUpdateLock
	case SHARE:
		lock = sqlast.ForShareLock
	default:
		return 0, 0, p.errorf("expected UPDATE or SHARE after FOR")
	}

	if err := p.advance(); err != nil {
		return 0, 0, err
	}

	wait, err := p.parseOptionalLockWait()
	if err != nil {
		return 0, 0, err
	}

	return lock, wait, nil
}

func (p *Parser) parseOptionalLockWait() (sqlast.LockWaitType, error) {
	switch {
	case p.at(NOWAIT):
		return sqlast.NowaitType, p.advance()
	case p.at(SKIP):
		return sqlast.SkipLockedType, p.advanceThenExpect(LOCKED)
	default:
		return sqlast.NoLockWait, nil
	}
}
