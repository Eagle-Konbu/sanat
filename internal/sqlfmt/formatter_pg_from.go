package sqlfmt

import (
	"fmt"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// formatTableExprs writes a comma-separated FROM list (or USING list) one
// item per line at depth+1, splicing the configured comma style onto each
// item's first and last line, as the MySQL formatter's formatTableExprs does.
func (f *pgFormatter) formatTableExprs(b *strings.Builder, exprs []pgast.TableExpr, depth int) {
	pi := f.pad(depth + 1)
	n := len(exprs)

	for i, expr := range exprs {
		var item strings.Builder

		f.formatTableExpr(&item, expr, depth)

		lines := strings.Split(strings.TrimSuffix(item.String(), "\n"), "\n")
		lines[0] = f.itemPrefix(pi, i) + strings.TrimPrefix(lines[0], pi)
		lines[len(lines)-1] += f.itemSuffix(i, n)

		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}
}

// formatTableExpr writes one FROM item at depth+1.
func (f *pgFormatter) formatTableExpr(b *strings.Builder, expr pgast.TableExpr, depth int) {
	pi := f.pad(depth + 1)

	switch e := expr.(type) {
	case *pgast.AliasedTableExpr:
		f.formatAliasedTableExpr(b, e, depth)
	case *pgast.JoinTableExpr:
		f.formatJoinTableExpr(b, e, depth)
	case *pgast.ParenTableExpr:
		b.WriteString(pi + "(\n")
		f.formatTableExprs(b, e.Exprs, depth+1)
		b.WriteString(pi + ")" + f.aliasClause(e.As, e.Columns) + "\n")
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL table expr type %T", expr))
	}
}

// formatAliasedTableExpr writes a table, function, or derived table FROM
// item. A derived table's subquery is laid out between its parentheses,
// which stay at the item's own indentation, matching the MySQL formatter.
func (f *pgFormatter) formatAliasedTableExpr(b *strings.Builder, e *pgast.AliasedTableExpr, depth int) {
	sub, ok := e.Expr.(*pgast.DerivedTable)
	if !ok {
		b.WriteString(f.pad(depth+1) + f.aliasedTableExprLine(e) + "\n")

		return
	}

	b.WriteString(f.pad(depth + 1))

	if e.Lateral {
		b.WriteString("LATERAL ")
	}

	b.WriteString("(\n")
	f.formatStatement(b, sub.Select, depth+1)
	b.WriteString(f.pad(depth+1) + ")" + f.aliasedTableExprSuffix(e, depth) + "\n")
}

// aliasedTableExprLine renders a FROM item that isn't a derived table on a
// single line.
func (f *pgFormatter) aliasedTableExprLine(e *pgast.AliasedTableExpr) string {
	var b strings.Builder

	if e.Lateral {
		b.WriteString("LATERAL ")
	}

	if e.Only {
		b.WriteString("ONLY ")
	}

	switch t := e.Expr.(type) {
	case pgast.TableName:
		b.WriteString(f.tableName(t))
	case *pgast.FuncTable:
		b.WriteString(f.funcTable(t))
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL simple table expr type %T", e.Expr))
	}

	if e.IncludeDescendants {
		b.WriteString(" *")
	}

	b.WriteString(f.aliasedTableExprSuffix(e, 0))

	return b.String()
}

// aliasedTableExprSuffix renders a FROM item's alias and TABLESAMPLE clause.
func (f *pgFormatter) aliasedTableExprSuffix(e *pgast.AliasedTableExpr, depth int) string {
	s := f.aliasClause(e.As, e.Columns)

	if e.Sample != nil {
		s += " TABLESAMPLE " + f.tableName(e.Sample.Method) + " (" + f.exprList(e.Sample.Args, depth) + ")"

		if e.Sample.Repeatable != nil {
			s += " REPEATABLE (" + f.expr(e.Sample.Repeatable, depth) + ")"
		}
	}

	return s
}

// aliasClause renders an optional " alias[(col, ...)]" suffix.
func (f *pgFormatter) aliasClause(alias pgast.TableIdent, columns pgast.Columns) string {
	if alias.IsEmpty() {
		return ""
	}

	if len(columns) == 0 {
		return " " + f.tableIdent(alias)
	}

	return " " + f.tableIdent(alias) + "(" + f.columns(columns) + ")"
}

func (f *pgFormatter) funcTable(t *pgast.FuncTable) string {
	s := f.exprList(t.Funcs, 0)
	if t.RowsFrom {
		s = "ROWS FROM (" + s + ")"
	}

	if t.WithOrdinality {
		s += " WITH ORDINALITY"
	}

	return s
}

// formatJoinTableExpr writes a JOIN's left item, the join keyword, and its
// right item at the same indentation, with an ON or USING condition one
// level deeper, matching the MySQL formatter.
func (f *pgFormatter) formatJoinTableExpr(b *strings.Builder, e *pgast.JoinTableExpr, depth int) {
	f.formatTableExpr(b, e.Left, depth)
	b.WriteString(f.pad(depth+1) + e.Join.String() + "\n")
	f.formatTableExpr(b, e.Right, depth)

	c := e.Condition

	switch {
	case c == nil:
	case c.On != nil:
		b.WriteString(f.pad(depth+2) + f.keyword("ON") + " " + f.expr(c.On, depth+2) + "\n")
	case c.Using != nil:
		b.WriteString(f.pad(depth+2) + f.keyword("USING") + " (" + f.columns(c.Using) + ")")

		if !c.UsingAlias.IsEmpty() {
			b.WriteString(" " + f.keyword("AS") + " " + f.tableIdent(c.UsingAlias))
		}

		b.WriteString("\n")
	}
}
