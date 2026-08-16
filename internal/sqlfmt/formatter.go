package sqlfmt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

const descDir = " DESC"

var (
	sentinelRe    = regexp.MustCompile(`:_sqla_ph_(\d+)`)
	placeholderRe = regexp.MustCompile(`\?`)
)

func FormatSQL(sql string, indent int) (string, bool) {
	replaced, count := replacePlaceholders(sql)

	stmt, err := parser.ParseStatement(replaced)
	if err != nil {
		return sql, false
	}

	var b strings.Builder

	formatStatement(&b, stmt, 0, indent)
	result := restorePlaceholders(b.String(), count)

	return result, true
}

func replacePlaceholders(sql string) (string, int) {
	count := 0
	result := placeholderRe.ReplaceAllStringFunc(sql, func(_ string) string {
		s := fmt.Sprintf(":_sqla_ph_%d", count)
		count++

		return s
	})

	return result, count
}

func restorePlaceholders(sql string, _ int) string {
	return sentinelRe.ReplaceAllString(sql, "?")
}

func formatStatement(b *strings.Builder, stmt sqlast.Statement, depth, indent int) {
	switch s := stmt.(type) {
	case *sqlast.Select:
		formatSelect(b, s, depth, indent)
	case *sqlast.Insert:
		formatInsert(b, s, depth, indent)
	case *sqlast.Update:
		formatUpdate(b, s, depth, indent)
	case *sqlast.Delete:
		formatDelete(b, s, depth, indent)
	case *sqlast.Union:
		formatUnion(b, s, depth, indent)
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled statement type %T", stmt))
	}
}

func pad(depth, indent int) string {
	return strings.Repeat(" ", depth*indent)
}

func formatWith(b *strings.Builder, with *sqlast.With, depth, indent int) {
	if with == nil {
		return
	}

	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	keyword := "WITH"
	if with.Recursive {
		keyword = "WITH RECURSIVE"
	}

	b.WriteString(p)
	b.WriteString(keyword)
	b.WriteString("\n")

	for i, cte := range with.CTEs {
		name := cte.ID.String()

		if len(cte.Columns) > 0 {
			name += " (" + cte.Columns.String() + ")"
		}

		b.WriteString(pi)
		b.WriteString(name)
		b.WriteString(" AS (\n")
		formatStatement(b, cte.Subquery, depth+2, indent)
		b.WriteString(pi)
		b.WriteString(")")

		if i < len(with.CTEs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatSelect(b *strings.Builder, s *sqlast.Select, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	formatWith(b, s.With, depth, indent)

	b.WriteString(p)
	b.WriteString("SELECT")

	if s.Distinct {
		b.WriteString(" DISTINCT")
	}

	b.WriteString("\n")
	formatSelectExprs(b, s.SelectExprs, pi, indent, depth)

	if len(s.From) > 0 {
		b.WriteString(p)
		b.WriteString("FROM\n")
		formatTableExprs(b, s.From, pi, indent, depth)
	}

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	formatGroupBy(b, s.GroupBy, p, pi, indent, depth)

	if s.Having != nil {
		b.WriteString(p)
		b.WriteString("HAVING\n")
		formatWhere(b, s.Having.Expr, pi, indent, depth)
	}

	formatOrderBy(b, s.OrderBy, p, pi, indent, depth)

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}

	formatLock(b, s.Lock, s.LockWait, p)
}

func formatSelectExprs(b *strings.Builder, exprs []sqlast.SelectExpr, pi string, indent, depth int) {
	for i, expr := range exprs {
		b.WriteString(pi)
		b.WriteString(formatSelectExpr(expr, indent, depth))

		if i < len(exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatSelectExpr(expr sqlast.SelectExpr, indent, depth int) string {
	switch e := expr.(type) {
	case *sqlast.AliasedExpr:
		s := formatExpr(e.Expr, indent, depth+1)
		if !e.As.IsEmpty() {
			s += " AS " + e.As.String()
		}

		return s
	case *sqlast.StarExpr:
		return e.String()
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled select expr type %T", expr))
	}
}

// formatTableExprs renders a comma-separated table reference list (a FROM
// clause, an UPDATE/DELETE target list, or a parenthesized table list). Each
// element is rendered into its own builder first so a trailing comma can be
// spliced in before the element's final newline, without threading "is this
// the last element" through formatTableExpr's JOIN/paren recursion.
func formatTableExprs(b *strings.Builder, exprs []sqlast.TableExpr, pi string, indent, depth int) {
	for i, expr := range exprs {
		var elem strings.Builder

		formatTableExpr(&elem, expr, pi, indent, depth)

		s := elem.String()
		if i < len(exprs)-1 {
			s = strings.TrimSuffix(s, "\n") + ",\n"
		}

		b.WriteString(s)
	}
}

func formatTableExpr(b *strings.Builder, expr sqlast.TableExpr, pi string, indent, depth int) {
	switch e := expr.(type) {
	case *sqlast.AliasedTableExpr:
		formatAliasedTableExpr(b, e, pi, indent, depth)
	case *sqlast.JoinTableExpr:
		formatJoinTableExpr(b, e, pi, indent, depth)
	case *sqlast.ParenTableExpr:
		b.WriteString(pi)
		b.WriteString("(\n")
		formatTableExprs(b, e.Exprs, pad(depth+2, indent), indent, depth+1)
		b.WriteString(pi)
		b.WriteString(")\n")
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled table expr type %T", expr))
	}
}

func formatAliasedTableExpr(b *strings.Builder, e *sqlast.AliasedTableExpr, pi string, indent, depth int) {
	if sub, ok := e.Expr.(*sqlast.DerivedTable); ok {
		b.WriteString(pi)
		b.WriteString("(\n")
		formatStatement(b, sub.Select, depth+1, indent)
		b.WriteString(pi)
		b.WriteString(")")
	} else {
		b.WriteString(pi)
		b.WriteString(e.Expr.String())
	}

	if !e.As.IsEmpty() {
		b.WriteString(" ")
		b.WriteString(e.As.String())
	}

	b.WriteString(formatIndexHints(e.Hints))
	b.WriteString("\n")
}

func formatIndexHints(hints sqlast.IndexHints) string {
	if len(hints) == 0 {
		return ""
	}

	parts := make([]string, len(hints))
	for i, hint := range hints {
		parts[i] = hint.String()
	}

	return " " + strings.Join(parts, " ")
}

func formatJoinTableExpr(b *strings.Builder, e *sqlast.JoinTableExpr, pi string, indent, depth int) {
	formatTableExpr(b, e.LeftExpr, pi, indent, depth)
	joinStr := strings.ToUpper(e.Join.ToString())

	b.WriteString(pi)
	b.WriteString(joinStr)
	b.WriteString("\n")
	formatTableExpr(b, e.RightExpr, pi, indent, depth)

	if e.Condition != nil && e.Condition.On != nil {
		b.WriteString(pad(depth+2, indent))
		b.WriteString("ON ")
		b.WriteString(formatExpr(e.Condition.On, indent, depth))
		b.WriteString("\n")
	}
}

func formatWhere(b *strings.Builder, expr sqlast.Expr, pi string, indent, depth int) {
	formatWhereExpr(b, expr, pi, indent, depth, true)
}

func formatWhereExpr(b *strings.Builder, expr sqlast.Expr, pi string, indent, depth int, first bool) {
	exprDepth := depth + 1

	switch e := expr.(type) {
	case *sqlast.AndExpr:
		formatWhereExpr(b, e.Left, pi, indent, depth, first)
		formatWhereExpr(b, e.Right, pi, indent, depth, false)
	case *sqlast.OrExpr:
		formatWhereExpr(b, e.Left, pi, indent, depth, first)
		b.WriteString(pi)
		b.WriteString("OR ")
		b.WriteString(formatExpr(e.Right, indent, exprDepth))
		b.WriteString("\n")
	default:
		if first {
			b.WriteString(pi)
			b.WriteString(formatExpr(expr, indent, exprDepth))
			b.WriteString("\n")
		} else {
			b.WriteString(pi)
			b.WriteString("AND ")
			b.WriteString(formatExpr(expr, indent, exprDepth))
			b.WriteString("\n")
		}
	}
}

func formatExpr(expr sqlast.Expr, indent, depth int) string {
	switch e := expr.(type) {
	case *sqlast.ExistsExpr:
		var b strings.Builder

		b.WriteString("EXISTS (\n")
		formatStatement(&b, e.Subquery.Select, depth+1, indent)
		b.WriteString(pad(depth, indent))
		b.WriteString(")")

		return b.String()
	case *sqlast.Subquery:
		var b strings.Builder

		b.WriteString("(\n")
		formatStatement(&b, e.Select, depth+1, indent)
		b.WriteString(pad(depth, indent))
		b.WriteString(")")

		return b.String()
	case *sqlast.ComparisonExpr:
		right := formatExpr(e.Right, indent, depth)

		return formatExpr(e.Left, indent, depth) + " " + e.Operator.ToString() + " " + right
	case *sqlast.NotExpr:
		return "NOT " + formatExpr(e.Expr, indent, depth)
	case *sqlast.CaseExpr:
		return formatCaseExpr(e, indent, depth)
	default:
		if accessor := getOverAccessor(expr); accessor != nil {
			if oc := accessor.getOverClause(); oc != nil {
				return formatExprWithOver(expr, accessor, indent, depth)
			}
		}

		return expr.String()
	}
}

func formatCaseExpr(e *sqlast.CaseExpr, indent, depth int) string {
	var b strings.Builder

	pi := pad(depth+1, indent)
	p := pad(depth, indent)

	b.WriteString("CASE")

	if e.Expr != nil {
		b.WriteString(" ")
		b.WriteString(formatExpr(e.Expr, indent, depth))
	}

	b.WriteString("\n")

	for _, when := range e.Whens {
		cond := formatExpr(when.Cond, indent, depth+1)
		val := formatExpr(when.Val, indent, depth+1)

		b.WriteString(pi)
		b.WriteString("WHEN ")
		b.WriteString(cond)
		b.WriteString(" THEN ")
		b.WriteString(val)
		b.WriteString("\n")
	}

	if e.Else != nil {
		b.WriteString(pi)
		b.WriteString("ELSE ")
		b.WriteString(formatExpr(e.Else, indent, depth+1))
		b.WriteString("\n")
	}

	b.WriteString(p)
	b.WriteString("END")

	return b.String()
}

type overClauseAccessor interface {
	getOverClause() *sqlast.OverClause
	setOverClause(oc *sqlast.OverClause)
}

type overClauseField struct {
	field **sqlast.OverClause
}

func (o overClauseField) getOverClause() *sqlast.OverClause   { return *o.field }
func (o overClauseField) setOverClause(oc *sqlast.OverClause) { *o.field = oc }

// Each case is trivially the same — complexity comes from the number of AST types, not logic.
func getOverAccessor(expr sqlast.Expr) overClauseAccessor { //nolint:cyclop,funlen,ireturn
	switch e := expr.(type) {
	case *sqlast.Count:
		return overClauseField{&e.OverClause}
	case *sqlast.CountStar:
		return overClauseField{&e.OverClause}
	case *sqlast.Sum:
		return overClauseField{&e.OverClause}
	case *sqlast.Avg:
		return overClauseField{&e.OverClause}
	case *sqlast.Min:
		return overClauseField{&e.OverClause}
	case *sqlast.Max:
		return overClauseField{&e.OverClause}
	case *sqlast.BitAnd:
		return overClauseField{&e.OverClause}
	case *sqlast.BitOr:
		return overClauseField{&e.OverClause}
	case *sqlast.BitXor:
		return overClauseField{&e.OverClause}
	case *sqlast.Std:
		return overClauseField{&e.OverClause}
	case *sqlast.StdDev:
		return overClauseField{&e.OverClause}
	case *sqlast.StdPop:
		return overClauseField{&e.OverClause}
	case *sqlast.StdSamp:
		return overClauseField{&e.OverClause}
	case *sqlast.VarPop:
		return overClauseField{&e.OverClause}
	case *sqlast.VarSamp:
		return overClauseField{&e.OverClause}
	case *sqlast.Variance:
		return overClauseField{&e.OverClause}
	case *sqlast.ArgumentLessWindowExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.FirstOrLastValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.NtileExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.NTHValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.LagLeadExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.JSONArrayAgg:
		return overClauseField{&e.OverClause}
	case *sqlast.JSONObjectAgg:
		return overClauseField{&e.OverClause}
	default:
		return nil
	}
}

func formatExprWithOver(expr sqlast.Expr, accessor overClauseAccessor, indent, depth int) string {
	oc := accessor.getOverClause()

	// Temporarily remove the OverClause to get the base function string
	accessor.setOverClause(nil)

	base := expr.String()

	accessor.setOverClause(oc)

	return base + " " + formatOverClause(oc, indent, depth)
}

func formatOverClause(oc *sqlast.OverClause, indent, depth int) string {
	if !oc.WindowName.IsEmpty() {
		return "OVER " + oc.WindowName.String()
	}

	parts := formatWindowSpecParts(oc.WindowSpec)
	if len(parts) == 0 {
		return "OVER ()"
	}

	pi := pad(depth+1, indent)
	p := pad(depth, indent)

	var b strings.Builder

	b.WriteString("OVER (\n")

	for _, part := range parts {
		b.WriteString(pi)
		b.WriteString(part)
		b.WriteString("\n")
	}

	b.WriteString(p)
	b.WriteString(")")

	return b.String()
}

func formatWindowSpecParts(spec *sqlast.WindowSpecification) []string {
	var parts []string

	if len(spec.PartitionClause) > 0 {
		exprs := make([]string, len(spec.PartitionClause))
		for i, e := range spec.PartitionClause {
			exprs[i] = e.String()
		}

		parts = append(parts, "PARTITION BY "+strings.Join(exprs, ", "))
	}

	if len(spec.OrderClause) > 0 {
		parts = append(parts, spec.OrderClause.String())
	}

	if spec.FrameClause != nil {
		parts = append(parts, spec.FrameClause.String())
	}

	return parts
}

func formatInsert(b *strings.Builder, s *sqlast.Insert, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	action := "INSERT"
	if s.Action == sqlast.ReplaceAct {
		action = "REPLACE"
	}

	if s.Ignore {
		action += " IGNORE"
	}

	b.WriteString(p)
	b.WriteString(action)
	b.WriteString(" INTO\n")
	b.WriteString(pi)
	b.WriteString(s.Table.String())
	b.WriteString("\n")

	formatInsertColumns(b, s.Columns, p, pi)
	formatInsertRows(b, s.Rows, p, pi, indent, depth)
	formatOnDupUpdate(b, s.OnDup, p, pi)
}

func formatInsertColumns(b *strings.Builder, cols sqlast.Columns, p, pi string) {
	if len(cols) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("(\n")

	for i, col := range cols {
		b.WriteString(pi)
		b.WriteString(col.String())

		if i < len(cols)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}

	b.WriteString(p)
	b.WriteString(")\n")
}

func formatInsertRows(b *strings.Builder, rows sqlast.InsertRows, p, pi string, indent, depth int) {
	switch r := rows.(type) {
	case sqlast.Values:
		formatValuesRows(b, r, p, pi, indent, depth)
	case *sqlast.Select:
		formatSelect(b, r, depth, indent)
	case *sqlast.Union:
		formatUnion(b, r, depth, indent)
	case sqlast.SetExprs:
		formatSetExprs(b, r, p, pi)
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled insert rows type %T", rows))
	}
}

func formatValuesRows(b *strings.Builder, rows sqlast.Values, p, pi string, indent, depth int) {
	b.WriteString(p)
	b.WriteString("VALUES\n")

	for i, row := range rows {
		vals := make([]string, len(row))
		for j, v := range row {
			vals[j] = formatExpr(v, indent, depth)
		}

		b.WriteString(pi)
		b.WriteString("(")
		b.WriteString(strings.Join(vals, ", "))
		b.WriteString(")")

		if i < len(rows)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatSetExprs(b *strings.Builder, exprs sqlast.SetExprs, p, pi string) {
	b.WriteString(p)
	b.WriteString("SET\n")

	for i, expr := range exprs {
		b.WriteString(pi)
		b.WriteString(expr.String())

		if i < len(exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatOnDupUpdate(b *strings.Builder, onDup sqlast.OnDup, p, pi string) {
	if len(onDup) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("ON DUPLICATE KEY UPDATE\n")

	for i, expr := range onDup {
		b.WriteString(pi)
		b.WriteString(expr.String())

		if i < len(onDup)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatUpdate(b *strings.Builder, s *sqlast.Update, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	formatWith(b, s.With, depth, indent)

	action := "UPDATE"
	if s.Ignore {
		action = "UPDATE IGNORE"
	}

	b.WriteString(p)
	b.WriteString(action)
	b.WriteString("\n")
	formatTableExprs(b, s.TableExprs, pi, indent, depth)

	b.WriteString(p)
	b.WriteString("SET\n")

	for i, expr := range s.Exprs {
		b.WriteString(pi)
		b.WriteString(expr.String())

		if i < len(s.Exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	formatOrderBy(b, s.OrderBy, p, pi, indent, depth)

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatDelete(b *strings.Builder, s *sqlast.Delete, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	formatWith(b, s.With, depth, indent)

	action := "DELETE"
	if s.Ignore {
		action = "DELETE IGNORE"
	}

	if len(s.Targets) > 0 {
		b.WriteString(p)
		b.WriteString(action)
		b.WriteString("\n")

		for i, target := range s.Targets {
			b.WriteString(pi)
			b.WriteString(target.String())

			if i < len(s.Targets)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}

		b.WriteString(p)
		b.WriteString("FROM\n")
	} else {
		b.WriteString(p)
		b.WriteString(action)
		b.WriteString(" FROM\n")
	}

	formatTableExprs(b, s.TableExprs, pi, indent, depth)

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	formatOrderBy(b, s.OrderBy, p, pi, indent, depth)

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatUnion(b *strings.Builder, s *sqlast.Union, depth, indent int) {
	p := pad(depth, indent)

	formatWith(b, s.With, depth, indent)

	formatStatement(b, s.Left, depth, indent)

	op := "UNION"
	if !s.Distinct {
		op = "UNION ALL"
	}

	b.WriteString(p)
	b.WriteString(op)
	b.WriteString("\n")
	formatStatement(b, s.Right, depth, indent)

	pi := pad(depth+1, indent)
	formatOrderBy(b, s.OrderBy, p, pi, indent, depth)

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}

	formatLock(b, s.Lock, s.LockWait, p)
}

func formatLock(b *strings.Builder, lock sqlast.Lock, wait sqlast.LockWaitType, p string) {
	if lock == sqlast.NoLock {
		return
	}

	lockStr := lock.String()
	if w := wait.String(); w != "" {
		lockStr += " " + w
	}

	b.WriteString(p)
	b.WriteString(lockStr)
	b.WriteString("\n")
}

func formatGroupBy(b *strings.Builder, groupBy *sqlast.GroupBy, p, pi string, indent, depth int) {
	if groupBy == nil || len(groupBy.Exprs) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("GROUP BY\n")

	for i, expr := range groupBy.Exprs {
		b.WriteString(pi)
		b.WriteString(formatExpr(expr, indent, depth))

		if i < len(groupBy.Exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}

	if groupBy.WithRollup {
		b.WriteString(p)
		b.WriteString("WITH ROLLUP\n")
	}
}

func formatOrderBy(b *strings.Builder, orders sqlast.OrderBy, p, pi string, indent, depth int) {
	if len(orders) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("ORDER BY\n")

	for i, order := range orders {
		dir := ""
		if order.Direction == sqlast.DescOrder {
			dir = descDir
		}

		b.WriteString(pi)
		b.WriteString(formatExpr(order.Expr, indent, depth))
		b.WriteString(dir)

		if i < len(orders)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatLimit(b *strings.Builder, limit *sqlast.Limit, p string, indent, depth int) {
	pi := pad(depth+1, indent)

	if limit.Offset != nil {
		b.WriteString(p)
		b.WriteString("LIMIT\n")
		b.WriteString(pi)
		b.WriteString(formatExpr(limit.Rowcount, indent, depth))
		b.WriteString("\n")
		b.WriteString(p)
		b.WriteString("OFFSET\n")
		b.WriteString(pi)
		b.WriteString(formatExpr(limit.Offset, indent, depth))
		b.WriteString("\n")
	} else {
		b.WriteString(p)
		b.WriteString("LIMIT\n")
		b.WriteString(pi)
		b.WriteString(formatExpr(limit.Rowcount, indent, depth))
		b.WriteString("\n")
	}
}
