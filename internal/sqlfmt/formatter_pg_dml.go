package sqlfmt

import (
	"fmt"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// formatInsert lays out an INSERT the way the MySQL formatter does: the
// target under INSERT INTO, the column list exploded between parentheses,
// then the row source, with ON CONFLICT and RETURNING after it.
func (f *pgFormatter) formatInsert(b *strings.Builder, s *pgast.Insert, depth int) {
	f.formatWith(b, s.With, depth)

	target := f.tableName(s.Table)
	if !s.Alias.IsEmpty() {
		target += " " + f.keyword("AS") + " " + f.tableIdent(s.Alias)
	}

	f.writeClauseItem(b, "INSERT INTO", target, depth)

	if len(s.Columns) > 0 {
		f.writeClause(b, "(", depth)
		f.writeList(b, f.pad(depth+1), f.exprs(s.Columns, depth+1))
		f.writeClause(b, ")", depth)
	}

	f.formatInsertRows(b, s.Rows, depth)
	f.formatOnConflict(b, s.OnConflict, depth)
	f.formatReturning(b, s.Returning, depth)
}

func (f *pgFormatter) formatInsertRows(b *strings.Builder, rows pgast.InsertRows, depth int) {
	switch r := rows.(type) {
	case pgast.Values:
		f.formatValues(b, r, depth)
	case pgast.DefaultValues:
		f.writeClause(b, "DEFAULT VALUES", depth)
	case *pgast.Select:
		f.formatSelect(b, r, depth)
	case *pgast.SetOp:
		f.formatSetOp(b, r, depth)
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL insert rows type %T", rows))
	}
}

// formatOnConflict writes the conflict target and action on the ON CONFLICT
// line, then a DO UPDATE's assignments one per line below it, like the MySQL
// formatter's ON DUPLICATE KEY UPDATE, and its WHERE condition after them.
func (f *pgFormatter) formatOnConflict(b *strings.Builder, o *pgast.OnConflict, depth int) {
	if o == nil {
		return
	}

	keyword := "ON CONFLICT"

	switch {
	case len(o.Target) > 0:
		elems := make([]string, len(o.Target))
		for i, e := range o.Target {
			elems[i] = f.indexElem(e, depth)
		}

		keyword += " (" + strings.Join(elems, ", ") + ")"

		if o.TargetWhere != nil && o.TargetWhere.Expr != nil {
			keyword += " WHERE " + f.expr(o.TargetWhere.Expr, depth)
		}
	case !o.Constraint.IsEmpty():
		keyword += " ON CONSTRAINT " + f.colIdent(o.Constraint)
	}

	if o.DoNothing {
		f.writeClause(b, keyword+" DO NOTHING", depth)

		return
	}

	f.writeClauseList(b, keyword+" DO UPDATE SET", f.updateExprs(o.Set, depth+1), depth)
	f.formatWhere(b, "WHERE", o.Where, depth)
}

func (f *pgFormatter) indexElem(e *pgast.IndexElem, depth int) string {
	s := f.expr(e.Expr, depth)

	if !e.Collation.IsEmpty() {
		s += " " + f.keyword("COLLATE") + " " + f.tableName(e.Collation)
	}

	if !e.Opclass.IsEmpty() {
		s += " " + f.tableName(e.Opclass)
	}

	return s
}

func (f *pgFormatter) updateExprs(exprs []*pgast.UpdateExpr, depth int) []string {
	items := make([]string, len(exprs))
	for i, e := range exprs {
		items[i] = f.updateExpr(e, depth)
	}

	return items
}

func (f *pgFormatter) updateExpr(u *pgast.UpdateExpr, depth int) string {
	if len(u.Names) == 0 {
		return f.expr(u.Name, depth) + " = " + f.expr(u.Expr, depth)
	}

	return "(" + f.exprList(u.Names, depth) + ") = " + f.expr(u.Expr, depth)
}

// dmlTarget renders an UPDATE, DELETE, or MERGE target. Its alias renders
// without AS, except an alias named set, which PostgreSQL would otherwise
// read as the SET keyword (see pgast's dmlTargetString).
func (f *pgFormatter) dmlTarget(t *pgast.AliasedTableExpr) string {
	if t.As.Name != "set" {
		return f.aliasedTableExprLine(t)
	}

	bare := *t
	bare.As = pgast.TableIdent{}

	return f.aliasedTableExprLine(&bare) + " " + f.keyword("AS") + " " + f.tableIdent(t.As)
}

func (f *pgFormatter) formatUpdate(b *strings.Builder, s *pgast.Update, depth int) {
	f.formatWith(b, s.With, depth)
	f.writeClauseItem(b, "UPDATE", f.dmlTarget(s.Table), depth)
	f.writeClauseList(b, "SET", f.updateExprs(s.Set, depth+1), depth)

	if len(s.From) > 0 {
		f.writeClause(b, "FROM", depth)
		f.formatTableExprs(b, s.From, depth)
	}

	f.formatWhere(b, "WHERE", s.Where, depth)
	f.formatReturning(b, s.Returning, depth)
}

func (f *pgFormatter) formatDelete(b *strings.Builder, s *pgast.Delete, depth int) {
	f.formatWith(b, s.With, depth)
	f.writeClauseItem(b, "DELETE FROM", f.dmlTarget(s.Table), depth)

	if len(s.Using) > 0 {
		f.writeClause(b, "USING", depth)
		f.formatTableExprs(b, s.Using, depth)
	}

	f.formatWhere(b, "WHERE", s.Where, depth)
	f.formatReturning(b, s.Returning, depth)
}

// formatMerge lays out a MERGE with its target, source, and join condition
// each under their own clause keyword, then each WHEN clause with its action
// one level deeper.
func (f *pgFormatter) formatMerge(b *strings.Builder, s *pgast.Merge, depth int) {
	f.formatWith(b, s.With, depth)
	f.writeClauseItem(b, "MERGE INTO", f.dmlTarget(s.Target), depth)
	f.writeClause(b, "USING", depth)
	f.formatTableExprs(b, []pgast.TableExpr{s.Source}, depth)
	f.writeClause(b, "ON", depth)
	f.formatCondition(b, s.On, depth+1, true)

	for _, w := range s.Whens {
		f.formatMergeWhen(b, w, depth)
	}

	f.formatReturning(b, s.Returning, depth)
}

func (f *pgFormatter) formatMergeWhen(b *strings.Builder, w *pgast.MergeWhen, depth int) {
	line := "WHEN " + w.Match.String()
	if w.Condition != nil {
		line += " " + f.keyword("AND") + " " + f.expr(w.Condition, depth)
	}

	f.writeClause(b, line+" THEN", depth)

	pi := f.pad(depth + 1)

	switch a := w.Action.(type) {
	case *pgast.MergeUpdate:
		f.writeClauseList(b, "UPDATE SET", f.updateExprs(a.Set, depth+2), depth+1)
	case *pgast.MergeInsert:
		b.WriteString(pi + f.mergeInsert(a, depth+1) + "\n")
	case pgast.MergeDelete:
		b.WriteString(pi + "DELETE\n")
	case pgast.MergeDoNothing:
		b.WriteString(pi + "DO NOTHING\n")
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL merge action type %T", w.Action))
	}
}

func (f *pgFormatter) mergeInsert(ins *pgast.MergeInsert, depth int) string {
	s := "INSERT"

	if len(ins.Columns) > 0 {
		s += " (" + f.columns(ins.Columns) + ")"
	}

	if o := ins.Overriding.String(); o != "" {
		s += " " + o
	}

	if ins.Values == nil {
		return s + " DEFAULT VALUES"
	}

	return s + " VALUES (" + f.exprList(ins.Values, depth) + ")"
}
