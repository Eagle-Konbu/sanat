package sqlfmt

import (
	"fmt"
	"regexp"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	parser        = sqlparser.NewTestParser()
	sentinelRe    = regexp.MustCompile(`:_sqla_ph_(\d+)`)
	placeholderRe = regexp.MustCompile(`\?`)
)

// FormatSQL parses and formats a SQL string. Returns formatted SQL and true on success.
// On parse failure, returns the original string and false.
func FormatSQL(sql string, indent int) (string, bool) {
	replaced, count := replacePlaceholders(sql)

	stmt, err := parser.Parse(replaced)
	if err != nil {
		return sql, false
	}

	var b strings.Builder

	formatStatement(&b, stmt, 0, indent)
	result := restorePlaceholders(b.String(), count)
	result = stripIdentifierBackticks(result)

	return result, true
}

func stripIdentifierBackticks(s string) string {
	return strings.ReplaceAll(s, "`", "")
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

func formatStatement(b *strings.Builder, stmt sqlparser.Statement, depth, indent int) {
	switch s := stmt.(type) {
	case *sqlparser.Select:
		formatSelect(b, s, depth, indent)
	case *sqlparser.Insert:
		formatInsert(b, s, depth, indent)
	case *sqlparser.Update:
		formatUpdate(b, s, depth, indent)
	case *sqlparser.Delete:
		formatDelete(b, s, depth, indent)
	case *sqlparser.Union:
		formatUnion(b, s, depth, indent)
	default:
		b.WriteString(sqlparser.String(stmt))
	}
}

func pad(depth, indent int) string {
	return strings.Repeat(" ", depth*indent)
}

func formatSelect(b *strings.Builder, s *sqlparser.Select, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	b.WriteString(p + "SELECT")

	if s.Distinct {
		b.WriteString(" DISTINCT")
	}

	b.WriteString("\n")
	formatSelectExprs(b, s.SelectExprs, pi, indent, depth)

	if len(s.From) > 0 {
		b.WriteString(p + "FROM\n")
		formatTableExprs(b, s.From, pi, indent, depth)
	}

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	if s.GroupBy != nil && len(s.GroupBy.Exprs) > 0 {
		b.WriteString(p + "GROUP BY\n")

		for i, expr := range s.GroupBy.Exprs {
			b.WriteString(pi + formatExpr(expr, indent, depth))

			if i < len(s.GroupBy.Exprs)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}

	if s.Having != nil {
		b.WriteString(p + "HAVING\n")
		formatWhere(b, s.Having.Expr, pi, indent, depth)
	}

	if len(s.OrderBy) > 0 {
		b.WriteString(p + "ORDER BY\n")

		for i, order := range s.OrderBy {
			dir := ""
			if order.Direction == sqlparser.DescOrder {
				dir = " DESC"
			}

			b.WriteString(pi + formatExpr(order.Expr, indent, depth) + dir)

			if i < len(s.OrderBy)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatSelectExprs(b *strings.Builder, exprs *sqlparser.SelectExprs, pi string, indent, depth int) {
	if exprs == nil {
		return
	}

	for i, expr := range exprs.Exprs {
		b.WriteString(pi + formatSelectExpr(expr, indent, depth))

		if i < len(exprs.Exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}
}

func formatSelectExpr(expr sqlparser.SelectExpr, indent, depth int) string {
	switch e := expr.(type) {
	case *sqlparser.AliasedExpr:
		s := formatExpr(e.Expr, indent, depth)
		if !e.As.IsEmpty() {
			s += " AS " + e.As.String()
		}

		return s
	case *sqlparser.StarExpr:
		if e.TableName.Name.IsEmpty() {
			return "*"
		}

		return e.TableName.Name.String() + ".*"
	default:
		return sqlparser.String(expr)
	}
}

func formatTableExprs(b *strings.Builder, exprs []sqlparser.TableExpr, pi string, indent, depth int) {
	for i, expr := range exprs {
		formatTableExpr(b, expr, pi, indent, depth, i > 0)
	}
}

func formatTableExpr(b *strings.Builder, expr sqlparser.TableExpr, pi string, indent, depth int, comma bool) {
	switch e := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		prefix := pi
		if comma {
			prefix = pi
		}

		switch sub := e.Expr.(type) {
		case *sqlparser.DerivedTable:
			b.WriteString(prefix + "(\n")
			formatStatement(b, sub.Select, depth+1, indent)
			b.WriteString(prefix + ")")
		default:
			b.WriteString(prefix + sqlparser.String(e.Expr))
		}

		if !e.As.IsEmpty() {
			b.WriteString(" " + e.As.String())
		}

		b.WriteString("\n")
	case *sqlparser.JoinTableExpr:
		formatTableExpr(b, e.LeftExpr, pi, indent, depth, false)
		joinStr := strings.ToUpper(e.Join.ToString())
		b.WriteString(pi + joinStr + "\n")
		formatTableExpr(b, e.RightExpr, pi, indent, depth, false)

		if e.Condition != nil && e.Condition.On != nil {
			b.WriteString(pad(depth+2, indent) + "ON " + formatExpr(e.Condition.On, indent, depth) + "\n")
		}
	case *sqlparser.ParenTableExpr:
		b.WriteString(pi + "(\n")
		formatTableExprs(b, e.Exprs, pad(depth+2, indent), indent, depth+1)
		b.WriteString(pi + ")\n")
	default:
		b.WriteString(pi + sqlparser.String(expr) + "\n")
	}
}

func formatWhere(b *strings.Builder, expr sqlparser.Expr, pi string, indent, depth int) {
	formatWhereExpr(b, expr, pi, indent, depth, true)
}

func formatWhereExpr(b *strings.Builder, expr sqlparser.Expr, pi string, indent, depth int, first bool) {
	exprDepth := depth + 1

	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		formatWhereExpr(b, e.Left, pi, indent, depth, first)
		formatWhereExpr(b, e.Right, pi, indent, depth, false)
	case *sqlparser.OrExpr:
		formatWhereExpr(b, e.Left, pi, indent, depth, first)
		b.WriteString(pi + "OR " + formatExpr(e.Right, indent, exprDepth) + "\n")
	default:
		if first {
			b.WriteString(pi + formatExpr(expr, indent, exprDepth) + "\n")
		} else {
			b.WriteString(pi + "AND " + formatExpr(expr, indent, exprDepth) + "\n")
		}
	}
}

func formatExpr(expr sqlparser.Expr, indent, depth int) string {
	switch e := expr.(type) {
	case *sqlparser.ExistsExpr:
		var b strings.Builder

		b.WriteString("EXISTS (\n")
		formatStatement(&b, e.Subquery.Select, depth+1, indent)
		b.WriteString(pad(depth, indent) + ")")

		return b.String()
	case *sqlparser.Subquery:
		var b strings.Builder

		b.WriteString("(\n")
		formatStatement(&b, e.Select, depth+1, indent)
		b.WriteString(pad(depth, indent) + ")")

		return b.String()
	case *sqlparser.ComparisonExpr:
		right := formatExpr(e.Right, indent, depth)

		return formatExpr(e.Left, indent, depth) + " " + e.Operator.ToString() + " " + right
	default:
		return upperKeywords(sqlparser.String(expr))
	}
}

func formatInsert(b *strings.Builder, s *sqlparser.Insert, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	action := "INSERT"
	if s.Action == sqlparser.ReplaceAct {
		action = "REPLACE"
	}

	b.WriteString(p + action + " INTO\n")
	b.WriteString(pi + sqlparser.String(s.Table) + "\n")

	if len(s.Columns) > 0 {
		b.WriteString(p + "(\n")

		for i, col := range s.Columns {
			b.WriteString(pi + col.String())

			if i < len(s.Columns)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}

		b.WriteString(p + ")\n")
	}

	switch rows := s.Rows.(type) {
	case sqlparser.Values:
		b.WriteString(p + "VALUES\n")

		for i, row := range rows {
			vals := make([]string, len(row))
			for j, v := range row {
				vals[j] = formatExpr(v, indent, depth)
			}

			b.WriteString(pi + "(" + strings.Join(vals, ", ") + ")")

			if i < len(rows)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	case *sqlparser.Select:
		formatSelect(b, rows, depth, indent)
	default:
		b.WriteString(p + sqlparser.String(s.Rows) + "\n")
	}

	if len(s.OnDup) > 0 {
		b.WriteString(p + "ON DUPLICATE KEY UPDATE\n")

		for i, expr := range s.OnDup {
			b.WriteString(pi + sqlparser.String(expr))

			if i < len(s.OnDup)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}
}

func formatUpdate(b *strings.Builder, s *sqlparser.Update, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	b.WriteString(p + "UPDATE\n")
	formatTableExprs(b, s.TableExprs, pi, indent, depth)

	b.WriteString(p + "SET\n")

	for i, expr := range s.Exprs {
		b.WriteString(pi + upperKeywords(sqlparser.String(expr)))

		if i < len(s.Exprs)-1 {
			b.WriteString(",")
		}

		b.WriteString("\n")
	}

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	if len(s.OrderBy) > 0 {
		b.WriteString(p + "ORDER BY\n")

		for i, order := range s.OrderBy {
			dir := ""
			if order.Direction == sqlparser.DescOrder {
				dir = " DESC"
			}

			b.WriteString(pi + formatExpr(order.Expr, indent, depth) + dir)

			if i < len(s.OrderBy)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatDelete(b *strings.Builder, s *sqlparser.Delete, depth, indent int) {
	p := pad(depth, indent)
	pi := pad(depth+1, indent)

	b.WriteString(p + "DELETE FROM\n")
	formatTableExprs(b, s.TableExprs, pi, indent, depth)

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		formatWhere(b, s.Where.Expr, pi, indent, depth)
	}

	if len(s.OrderBy) > 0 {
		b.WriteString(p + "ORDER BY\n")

		for i, order := range s.OrderBy {
			dir := ""
			if order.Direction == sqlparser.DescOrder {
				dir = " DESC"
			}

			b.WriteString(pi + formatExpr(order.Expr, indent, depth) + dir)

			if i < len(s.OrderBy)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatUnion(b *strings.Builder, s *sqlparser.Union, depth, indent int) {
	p := pad(depth, indent)
	formatStatement(b, s.Left, depth, indent)

	op := "UNION"
	if !s.Distinct {
		op = "UNION ALL"
	}

	b.WriteString(p + op + "\n")
	formatStatement(b, s.Right, depth, indent)

	if len(s.OrderBy) > 0 {
		pi := pad(depth+1, indent)

		b.WriteString(p + "ORDER BY\n")

		for i, order := range s.OrderBy {
			dir := ""
			if order.Direction == sqlparser.DescOrder {
				dir = " DESC"
			}

			b.WriteString(pi + formatExpr(order.Expr, indent, depth) + dir)

			if i < len(s.OrderBy)-1 {
				b.WriteString(",")
			}

			b.WriteString("\n")
		}
	}

	if s.Limit != nil {
		formatLimit(b, s.Limit, p, indent, depth)
	}
}

func formatLimit(b *strings.Builder, limit *sqlparser.Limit, p string, indent, depth int) {
	pi := pad(depth+1, indent)

	if limit.Offset != nil {
		b.WriteString(p + "LIMIT\n")
		b.WriteString(pi + formatExpr(limit.Rowcount, indent, depth) + "\n")
		b.WriteString(p + "OFFSET\n")
		b.WriteString(pi + formatExpr(limit.Offset, indent, depth) + "\n")
	} else {
		b.WriteString(p + "LIMIT\n")
		b.WriteString(pi + formatExpr(limit.Rowcount, indent, depth) + "\n")
	}
}

var keywordReplacer = strings.NewReplacer(
	" as ", " AS ",
	" asc", " ASC",
	" desc", " DESC",
	" and ", " AND ",
	" or ", " OR ",
	" not ", " NOT ",
	" in ", " IN ",
	" is ", " IS ",
	" like ", " LIKE ",
	" between ", " BETWEEN ",
	" exists ", " EXISTS ",
	" null", " NULL",
	" true", " TRUE",
	" false", " FALSE",
	" on ", " ON ",
	" using ", " USING ",
)

func upperKeywords(s string) string {
	return keywordReplacer.Replace(s)
}
