package sqlast

import (
	"fmt"
	"strings"
)

// Select represents a SELECT statement.
type Select struct {
	With        *With
	Distinct    bool
	SelectExprs []SelectExpr
	From        []TableExpr
	Where       *Where
	GroupBy     *GroupBy
	Having      *Where
	OrderBy     OrderBy
	Limit       *Limit
	Lock        Lock
}

func (s *Select) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, s.With)

	b.WriteString("SELECT ")

	if s.Distinct {
		b.WriteString("DISTINCT ")
	}

	writeSelectExprs(&b, s.SelectExprs)
	writeFromClause(&b, s.From)
	writeWhereClause(&b, "WHERE", s.Where)
	writeGroupBy(&b, s.GroupBy)
	writeWhereClause(&b, "HAVING", s.Having)
	writeOrderBy(&b, s.OrderBy)
	writeLimit(&b, s.Limit)
	writeLock(&b, s.Lock)

	return b.String()
}

// Insert represents an INSERT or REPLACE statement.
type Insert struct {
	Action  InsertAction
	Ignore  bool
	Table   TableName
	Columns Columns
	Rows    InsertRows
	OnDup   OnDup
}

func (ins *Insert) String() string {
	var b strings.Builder

	b.WriteString(ins.Action.String())

	if ins.Ignore {
		b.WriteString(" IGNORE")
	}

	b.WriteString(" INTO " + ins.Table.String())

	if len(ins.Columns) > 0 {
		b.WriteString(" (" + ins.Columns.String() + ")")
	}

	b.WriteString(" " + ins.Rows.String())

	if len(ins.OnDup) > 0 {
		b.WriteString(" " + ins.OnDup.String())
	}

	return b.String()
}

// Update represents an UPDATE statement.
type Update struct {
	With       *With
	Ignore     bool
	TableExprs []TableExpr
	Exprs      []*UpdateExpr
	Where      *Where
	OrderBy    OrderBy
	Limit      *Limit
}

func (u *Update) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, u.With)

	b.WriteString("UPDATE")

	if u.Ignore {
		b.WriteString(" IGNORE")
	}

	tables := make([]string, len(u.TableExprs))
	for i, t := range u.TableExprs {
		tables[i] = t.String()
	}

	b.WriteString(" " + strings.Join(tables, ", "))

	sets := make([]string, len(u.Exprs))
	for i, e := range u.Exprs {
		sets[i] = e.String()
	}

	b.WriteString(" SET " + strings.Join(sets, ", "))
	writeWhereClause(&b, "WHERE", u.Where)
	writeOrderBy(&b, u.OrderBy)
	writeLimit(&b, u.Limit)

	return b.String()
}

// Delete represents a DELETE statement.
type Delete struct {
	With       *With
	Ignore     bool
	Targets    []TableName
	TableExprs []TableExpr
	Where      *Where
	OrderBy    OrderBy
	Limit      *Limit
}

func (d *Delete) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, d.With)

	b.WriteString("DELETE")

	if d.Ignore {
		b.WriteString(" IGNORE")
	}

	writeTargets(&b, d.Targets)
	writeTableExprs(&b, d.TableExprs)
	writeWhereClause(&b, "WHERE", d.Where)
	writeOrderBy(&b, d.OrderBy)
	writeLimit(&b, d.Limit)

	return b.String()
}

// Union represents a UNION statement.
type Union struct {
	With     *With
	Left     Statement
	Right    Statement
	Distinct bool
	OrderBy  OrderBy
	Limit    *Limit
	Lock     Lock
}

func (u *Union) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, u.With)

	b.WriteString(u.Left.String())

	if u.Distinct {
		b.WriteString(" UNION ")
	} else {
		b.WriteString(" UNION ALL ")
	}

	b.WriteString(u.Right.String())
	writeOrderBy(&b, u.OrderBy)
	writeLimit(&b, u.Limit)
	writeLock(&b, u.Lock)

	return b.String()
}

// With represents a WITH (CTE) clause.
type With struct {
	CTEs      []*CommonTableExpr
	Recursive bool
}

func (w *With) String() string {
	if w == nil {
		return ""
	}

	keyword := "WITH"
	if w.Recursive {
		keyword = "WITH RECURSIVE"
	}

	ctes := make([]string, len(w.CTEs))
	for i, cte := range w.CTEs {
		ctes[i] = cte.String()
	}

	return fmt.Sprintf("%s %s", keyword, strings.Join(ctes, ", "))
}

// CommonTableExpr represents a single CTE definition.
type CommonTableExpr struct {
	ID       TableIdent
	Columns  Columns
	Subquery Statement
}

func (c *CommonTableExpr) String() string {
	name := c.ID.String()
	if len(c.Columns) > 0 {
		name += " (" + c.Columns.String() + ")"
	}

	return name + " AS (" + c.Subquery.String() + ")"
}

// --- helpers to reduce cyclomatic complexity ---

func writeOptionalPrefix(b *strings.Builder, w SQLNode) {
	s := w.String()
	if s != "" {
		b.WriteString(s + " ")
	}
}

func writeSelectExprs(b *strings.Builder, exprs []SelectExpr) {
	strs := make([]string, len(exprs))
	for i, e := range exprs {
		strs[i] = e.String()
	}

	b.WriteString(strings.Join(strs, ", "))
}

func writeFromClause(b *strings.Builder, from []TableExpr) {
	if len(from) == 0 {
		return
	}

	strs := make([]string, len(from))
	for i, f := range from {
		strs[i] = f.String()
	}

	b.WriteString(" FROM " + strings.Join(strs, ", "))
}

func writeWhereClause(b *strings.Builder, keyword string, w *Where) {
	if w == nil || w.Expr == nil {
		return
	}

	b.WriteString(" " + keyword + " " + w.String())
}

func writeGroupBy(b *strings.Builder, g *GroupBy) {
	if g == nil || len(g.Exprs) == 0 {
		return
	}

	b.WriteString(" " + g.String())
}

func writeOrderBy(b *strings.Builder, o OrderBy) {
	if len(o) == 0 {
		return
	}

	b.WriteString(" " + o.String())
}

func writeLimit(b *strings.Builder, l *Limit) {
	if l == nil {
		return
	}

	b.WriteString(" " + l.String())
}

func writeLock(b *strings.Builder, l Lock) {
	if l == NoLock {
		return
	}

	b.WriteString(" " + l.String())
}

func writeTargets(b *strings.Builder, targets []TableName) {
	if len(targets) == 0 {
		return
	}

	strs := make([]string, len(targets))
	for i, t := range targets {
		strs[i] = t.String()
	}

	b.WriteString(" " + strings.Join(strs, ", "))
}

func writeTableExprs(b *strings.Builder, exprs []TableExpr) {
	strs := make([]string, len(exprs))
	for i, e := range exprs {
		strs[i] = e.String()
	}

	b.WriteString(" FROM " + strings.Join(strs, ", "))
}
