package sqlfmt

import (
	"fmt"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// pgFormatter renders a PostgreSQL pgast statement with the same layout as
// the MySQL formatter: one clause keyword per line, its content indented
// below it. It walks pgast directly instead of post-processing String(), so
// keyword casing only touches keywords and literals and identifiers keep
// their source spelling (their Raw text).
type pgFormatter struct {
	renderer
}

// formatPostgreSQL formats sql with the PostgreSQL parser. Unlike the MySQL
// path, '?' is not rewritten: pgparser itself tells a '?' placeholder from
// the jsonb '?' operator by position. SQL with a comment is left unchanged,
// since the AST has nowhere to keep it.
func formatPostgreSQL(sql string, opts Options) (string, bool) {
	if pgparser.ContainsComment(sql) {
		return sql, false
	}

	stmt, err := pgparser.ParseStatement(sql)
	if err != nil {
		return sql, false
	}

	result, ok := formatParsedPGStatement(opts, stmt)
	if !ok {
		return sql, false
	}

	return result, true
}

// formatParsedPGStatement renders stmt, recovering a panic from a node type
// the renderer doesn't handle into an ok=false failure, like
// formatParsedStatement does for MySQL.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func formatParsedPGStatement(opts Options, stmt pgast.Statement) (result string, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = "", false
		}
	}()

	f := &pgFormatter{renderer: newRenderer(opts)}

	var b strings.Builder

	f.formatStatement(&b, stmt, 0)

	return b.String(), true
}

func (f *pgFormatter) formatStatement(b *strings.Builder, stmt pgast.Statement, depth int) {
	switch s := stmt.(type) {
	case *pgast.Select:
		f.formatSelect(b, s, depth)
	case *pgast.SetOp:
		f.formatSetOp(b, s, depth)
	case *pgast.Insert:
		f.formatInsert(b, s, depth)
	case *pgast.Update:
		f.formatUpdate(b, s, depth)
	case *pgast.Delete:
		f.formatDelete(b, s, depth)
	case *pgast.Merge:
		f.formatMerge(b, s, depth)
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL statement type %T", stmt))
	}
}

// writeClause writes a clause keyword on its own line at depth.
func (f *pgFormatter) writeClause(b *strings.Builder, keyword string, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(keyword)
	b.WriteString("\n")
}

// writeClauseList writes a clause keyword on its own line, followed by its
// pre-rendered items one per line, per the configured comma style.
func (f *pgFormatter) writeClauseList(b *strings.Builder, keyword string, items []string, depth int) {
	f.writeClause(b, keyword, depth)
	f.writeList(b, f.pad(depth+1), items)
}

// writeClauseItem writes a clause keyword on its own line, followed by a
// single pre-rendered item.
func (f *pgFormatter) writeClauseItem(b *strings.Builder, keyword, item string, depth int) {
	f.writeClause(b, keyword, depth)
	b.WriteString(f.pad(depth + 1))
	b.WriteString(item)
	b.WriteString("\n")
}

func (f *pgFormatter) formatWith(b *strings.Builder, with *pgast.With, depth int) {
	if with == nil {
		return
	}

	keyword := "WITH"
	if with.Recursive {
		keyword = "WITH RECURSIVE"
	}

	f.writeClause(b, keyword, depth)

	pi := f.pad(depth + 1)
	n := len(with.CTEs)

	for i, cte := range with.CTEs {
		b.WriteString(f.itemPrefix(pi, i))
		b.WriteString(f.tableIdent(cte.Name))

		if len(cte.Columns) > 0 {
			b.WriteString(" (" + f.columns(cte.Columns) + ")")
		}

		b.WriteString(" " + f.keyword("AS") + " ")

		if m := cte.Materialization.String(); m != "" {
			b.WriteString(m + " ")
		}

		b.WriteString("(\n")
		f.formatStatement(b, cte.Subquery, depth+2)
		b.WriteString(pi + ")" + f.itemSuffix(i, n) + "\n")
	}
}

func (f *pgFormatter) formatSelect(b *strings.Builder, s *pgast.Select, depth int) {
	f.formatWith(b, s.With, depth)

	switch {
	case s.Values != nil:
		f.formatValues(b, s.Values, depth)
	case s.Table != nil:
		b.WriteString(f.pad(depth) + "TABLE " + f.aliasedTableExprLine(s.Table) + "\n")
	default:
		f.formatSelectCore(b, s, depth)
	}

	f.formatQueryTail(b, s.OrderBy, s.Limit, s.Offset, s.Fetch, depth)

	for _, lock := range s.Locks {
		b.WriteString(f.pad(depth) + f.lock(lock) + "\n")
	}
}

// formatSelectCore writes the SELECT keyword through the WINDOW clause.
func (f *pgFormatter) formatSelectCore(b *strings.Builder, s *pgast.Select, depth int) {
	keyword := "SELECT"

	switch {
	case s.DistinctOn != nil:
		keyword += " DISTINCT ON (" + f.exprList(s.DistinctOn.Exprs, depth) + ")"
	case s.Distinct:
		keyword += " DISTINCT"
	}

	f.writeClauseList(b, keyword, f.selectExprs(s.SelectExprs, depth+1), depth)

	if len(s.From) > 0 {
		f.writeClause(b, "FROM", depth)
		f.formatTableExprs(b, s.From, depth)
	}

	f.formatWhere(b, "WHERE", s.Where, depth)

	if s.GroupBy != nil && len(s.GroupBy.Elements) > 0 {
		f.writeClauseList(b, "GROUP BY", f.exprs(s.GroupBy.Elements, depth+1), depth)
	}

	f.formatWhere(b, "HAVING", s.Having, depth)

	if len(s.Window) > 0 {
		items := make([]string, len(s.Window))
		for i, w := range s.Window {
			items[i] = f.colIdent(w.Name) + " " + f.keyword("AS") + " " + f.windowSpec(w.Spec, depth+1)
		}

		f.writeClauseList(b, "WINDOW", items, depth)
	}
}

func (f *pgFormatter) formatValues(b *strings.Builder, rows [][]pgast.Expr, depth int) {
	items := make([]string, len(rows))
	for i, row := range rows {
		items[i] = "(" + f.exprList(row, depth+1) + ")"
	}

	f.writeClauseList(b, "VALUES", items, depth)
}

// formatQueryTail writes the ORDER BY, LIMIT, OFFSET, and FETCH clauses a
// SELECT and a set operation share.
func (f *pgFormatter) formatQueryTail(
	b *strings.Builder, orderBy pgast.OrderBy, limit *pgast.Limit, offset *pgast.Offset, fetch *pgast.Fetch, depth int,
) {
	if len(orderBy) > 0 {
		f.writeClauseList(b, "ORDER BY", f.orderItems(orderBy, depth+1), depth)
	}

	if limit != nil {
		count := "ALL"
		if limit.Count != nil {
			count = f.expr(limit.Count, depth+1)
		}

		f.writeClauseItem(b, "LIMIT", count, depth)
	}

	if offset != nil && offset.Start != nil {
		f.writeClauseItem(b, "OFFSET", f.expr(offset.Start, depth+1), depth)
	}

	if fetch != nil {
		b.WriteString(f.pad(depth) + f.fetch(fetch, depth) + "\n")
	}
}

// fetch renders a FETCH clause on one line. The AST doesn't keep whether
// ROW or ROWS was written (they are synonyms), so ROWS follows a count and
// ROW stands alone, as in "FETCH FIRST 10 ROWS ONLY" and "FETCH FIRST ROW
// ONLY".
func (f *pgFormatter) fetch(fetch *pgast.Fetch, depth int) string {
	s := "FETCH FIRST "
	if fetch.Next {
		s = "FETCH NEXT "
	}

	rows := "ROW"

	if fetch.Count != nil {
		s += f.expr(fetch.Count, depth) + " "
		rows = "ROWS"
	}

	if fetch.WithTies {
		return s + rows + " WITH TIES"
	}

	return s + rows + " ONLY"
}

func (f *pgFormatter) lock(lock *pgast.Lock) string {
	s := lock.Strength.String()

	if len(lock.Of) > 0 {
		names := make([]string, len(lock.Of))
		for i, t := range lock.Of {
			names[i] = f.tableName(t)
		}

		s += " OF " + strings.Join(names, ", ")
	}

	if wait := lock.Wait.String(); wait != "" {
		s += " " + wait
	}

	return s
}

// formatSetOp writes each operand at the same depth with the operator on its
// own line between them, parenthesizing an operand where its own clauses or
// the operators' precedence require it.
func (f *pgFormatter) formatSetOp(b *strings.Builder, s *pgast.SetOp, depth int) {
	f.formatWith(b, s.With, depth)
	f.formatSetOpOperand(b, s.Left, pgast.NeedsSetOpParens(s.Left, s.Op, false), depth)

	op := s.Op.String()
	if s.All {
		op += " ALL"
	}

	f.writeClause(b, op, depth)
	f.formatSetOpOperand(b, s.Right, pgast.NeedsSetOpParens(s.Right, s.Op, true), depth)
	f.formatQueryTail(b, s.OrderBy, s.Limit, s.Offset, s.Fetch, depth)
}

func (f *pgFormatter) formatSetOpOperand(b *strings.Builder, stmt pgast.Statement, parens bool, depth int) {
	if !parens {
		f.formatStatement(b, stmt, depth)

		return
	}

	f.writeClause(b, "(", depth)
	f.formatStatement(b, stmt, depth+1)
	f.writeClause(b, ")", depth)
}

// formatWhere writes a WHERE or HAVING clause, breaking its top-level AND/OR
// chain one condition per line, the same way the MySQL formatter does.
func (f *pgFormatter) formatWhere(b *strings.Builder, keyword string, where *pgast.Where, depth int) {
	if where == nil || where.Expr == nil {
		return
	}

	f.writeClause(b, keyword, depth)
	f.formatCondition(b, where.Expr, depth+1, true)
}

func (f *pgFormatter) formatCondition(b *strings.Builder, expr pgast.Expr, depth int, first bool) {
	switch e := expr.(type) {
	case *pgast.AndExpr:
		f.formatCondition(b, e.Left, depth, first)
		f.formatCondition(b, e.Right, depth, false)
	case *pgast.OrExpr:
		f.formatCondition(b, e.Left, depth, first)
		b.WriteString(f.pad(depth) + f.keyword("OR") + " " + f.expr(e.Right, depth) + "\n")
	default:
		prefix := ""
		if !first {
			prefix = f.keyword("AND") + " "
		}

		b.WriteString(f.pad(depth) + prefix + f.expr(expr, depth) + "\n")
	}
}

func (f *pgFormatter) selectExprs(exprs []pgast.SelectExpr, depth int) []string {
	items := make([]string, len(exprs))
	for i, e := range exprs {
		items[i] = f.selectExpr(e, depth)
	}

	return items
}

func (f *pgFormatter) selectExpr(expr pgast.SelectExpr, depth int) string {
	switch e := expr.(type) {
	case *pgast.AliasedExpr:
		s := f.expr(e.Expr, depth)
		if !e.As.IsEmpty() {
			s += " " + f.keyword("AS") + " " + f.colIdent(e.As)
		}

		return s
	case *pgast.StarExpr:
		if e.TableName.IsEmpty() {
			return "*"
		}

		return f.tableName(e.TableName) + ".*"
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL select expr type %T", expr))
	}
}

// formatReturning writes a RETURNING clause, its PostgreSQL 18 output
// aliases on the keyword's line and its items one per line below.
func (f *pgFormatter) formatReturning(b *strings.Builder, r *pgast.Returning, depth int) {
	if r == nil || len(r.Exprs) == 0 {
		return
	}

	keyword := "RETURNING"

	if len(r.Aliases) > 0 {
		aliases := make([]string, len(r.Aliases))

		for i, a := range r.Aliases {
			kind := "OLD"
			if a.Kind == pgast.ReturningNew {
				kind = "NEW"
			}

			aliases[i] = kind + " " + f.keyword("AS") + " " + f.tableIdent(a.Name)
		}

		keyword += " WITH (" + strings.Join(aliases, ", ") + ")"
	}

	f.writeClauseList(b, keyword, f.selectExprs(r.Exprs, depth+1), depth)
}

func (f *pgFormatter) orderItems(orders pgast.OrderBy, depth int) []string {
	items := make([]string, len(orders))
	for i, o := range orders {
		items[i] = f.orderItem(o, depth)
	}

	return items
}

func (f *pgFormatter) orderItem(o *pgast.Order, depth int) string {
	s := f.expr(o.Expr, depth)

	switch {
	case o.Using != nil:
		s += " " + f.keyword("USING") + " " + f.operator(*o.Using)
	case o.Direction == pgast.DescOrder:
		s += " " + f.keyword("DESC")
	}

	if nulls := o.Nulls.String(); nulls != "" {
		s += " " + f.keyword(nulls)
	}

	return s
}

// --- identifiers ---

// colIdent renders an identifier as written in the source, falling back to
// its normalized SQL text for one that wasn't parsed from source.
func (f *pgFormatter) colIdent(c pgast.ColIdent) string {
	if c.Raw != "" {
		return c.Raw
	}

	return c.Name
}

func (f *pgFormatter) tableIdent(t pgast.TableIdent) string {
	if t.Raw != "" {
		return t.Raw
	}

	return t.Name
}

func (f *pgFormatter) tableName(t pgast.TableName) string {
	if t.Qualifier.IsEmpty() {
		return f.tableIdent(t.Name)
	}

	return f.tableIdent(t.Qualifier) + "." + f.tableIdent(t.Name)
}

func (f *pgFormatter) columns(cols pgast.Columns) string {
	strs := make([]string, len(cols))
	for i, c := range cols {
		strs[i] = f.colIdent(c)
	}

	return strings.Join(strs, ", ")
}

func (f *pgFormatter) anyName(n pgast.AnyName) string {
	strs := make([]string, len(n))
	for i, part := range n {
		strs[i] = f.colIdent(part)
	}

	return strings.Join(strs, ".")
}
