package sqlfmt

import (
	"fmt"
	"regexp"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	KeywordCaseUpper    = "upper"
	KeywordCaseLower    = "lower"
	KeywordCasePreserve = "preserve"

	CommaStyleTrailing = "trailing"
	CommaStyleLeading  = "leading"
)

var (
	parser        = sqlparser.NewTestParser()
	sentinelRe    = regexp.MustCompile(`:_sqla_ph_(\d+)`)
	placeholderRe = regexp.MustCompile(`\?`)
)

// Options controls how FormatSQLWithOptions renders a SQL statement.
type Options struct {
	Indent int

	// KeywordCase controls the casing of operator/predicate keywords
	// (AS, AND, OR, NOT, IN, IS, LIKE, ...). Defaults to KeywordCaseUpper.
	//
	// The underlying Vitess parser does not retain the source's original
	// keyword casing, so KeywordCasePreserve currently behaves the same
	// as KeywordCaseLower (the parser's natural lowercase output).
	KeywordCase string

	// CommaStyle controls comma placement in rendered lists (SELECT
	// columns, INSERT columns/values, SET assignments, ...). Defaults to
	// CommaStyleTrailing.
	CommaStyle string
}

// formatter holds the resolved rendering options for a single FormatSQL call.
type formatter struct {
	indent      int
	keywordCase string
	commaStyle  string
}

func newFormatter(opts Options) *formatter {
	f := &formatter{
		indent:      opts.Indent,
		keywordCase: opts.KeywordCase,
		commaStyle:  opts.CommaStyle,
	}

	if f.keywordCase == "" {
		f.keywordCase = KeywordCaseUpper
	}

	if f.commaStyle == "" {
		f.commaStyle = CommaStyleTrailing
	}

	return f
}

// FormatSQL formats sql with the given indent width, using the default
// keyword casing and comma style. See FormatSQLWithOptions for full control.
func FormatSQL(sql string, indent int) (string, bool) {
	return FormatSQLWithOptions(sql, Options{Indent: indent})
}

func FormatSQLWithOptions(sql string, opts Options) (string, bool) {
	replaced, count := replacePlaceholders(sql)

	stmt, err := parser.Parse(replaced)
	if err != nil {
		return sql, false
	}

	f := newFormatter(opts)

	var b strings.Builder

	f.formatStatement(&b, stmt, 0)
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

func (f *formatter) formatStatement(b *strings.Builder, stmt sqlparser.Statement, depth int) {
	switch s := stmt.(type) {
	case *sqlparser.Select:
		f.formatSelect(b, s, depth)
	case *sqlparser.Insert:
		f.formatInsert(b, s, depth)
	case *sqlparser.Update:
		f.formatUpdate(b, s, depth)
	case *sqlparser.Delete:
		f.formatDelete(b, s, depth)
	case *sqlparser.Union:
		f.formatUnion(b, s, depth)
	default:
		b.WriteString(sqlparser.String(stmt))
	}
}

func (f *formatter) pad(depth int) string {
	return strings.Repeat(" ", depth*f.indent)
}

// keyword renders a hardcoded clause/operator literal (e.g. "AS", "AND")
// according to the configured keyword case. Unlike applyKeywordCase, this is
// used for keywords the formatter emits itself rather than text produced by
// the underlying parser.
func (f *formatter) keyword(word string) string {
	if f.keywordCase == KeywordCaseUpper {
		return strings.ToUpper(word)
	}

	return strings.ToLower(word)
}

// itemPrefix returns the indentation/comma prefix to write before item i of
// an n-item list, per the configured comma style.
func (f *formatter) itemPrefix(pi string, i int) string {
	if f.commaStyle == CommaStyleLeading && i > 0 {
		return leadingCommaPrefix(pi)
	}

	return pi
}

// itemSuffix returns the trailing comma (if any) to write after item i of an
// n-item list, per the configured comma style.
func (f *formatter) itemSuffix(i, n int) string {
	if f.commaStyle == CommaStyleLeading {
		return ""
	}

	if i < n-1 {
		return ","
	}

	return ""
}

// writeList writes pre-rendered single-line items, applying the configured
// comma style.
func (f *formatter) writeList(b *strings.Builder, pi string, lines []string) {
	for i, line := range lines {
		b.WriteString(f.itemPrefix(pi, i) + line + f.itemSuffix(i, len(lines)) + "\n")
	}
}

func leadingCommaPrefix(pi string) string {
	n := max(len(pi)-2, 0)

	return strings.Repeat(" ", n) + ", "
}

func (f *formatter) formatWith(b *strings.Builder, with *sqlparser.With, depth int) {
	if with == nil {
		return
	}

	p := f.pad(depth)
	pi := f.pad(depth + 1)

	keyword := "WITH"
	if with.Recursive {
		keyword = "WITH RECURSIVE"
	}

	b.WriteString(p + keyword + "\n")

	for i, cte := range with.CTEs {
		name := cte.ID.String()

		if len(cte.Columns) > 0 {
			cols := make([]string, len(cte.Columns))
			for j, col := range cte.Columns {
				cols[j] = col.String()
			}

			name += " (" + strings.Join(cols, ", ") + ")"
		}

		b.WriteString(f.itemPrefix(pi, i) + name + " " + f.keyword("AS") + " (\n")
		f.formatStatement(b, cte.Subquery, depth+2)
		b.WriteString(pi + ")" + f.itemSuffix(i, len(with.CTEs)) + "\n")
	}
}

func (f *formatter) formatSelect(b *strings.Builder, s *sqlparser.Select, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	b.WriteString(p + "SELECT")

	if s.Distinct {
		b.WriteString(" DISTINCT")
	}

	b.WriteString("\n")
	f.formatSelectExprs(b, s.SelectExprs, pi, depth)

	if len(s.From) > 0 {
		b.WriteString(p + "FROM\n")
		f.formatTableExprs(b, s.From, pi, depth)
	}

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatGroupBy(b, s.GroupBy, p, pi, depth)

	if s.Having != nil {
		b.WriteString(p + "HAVING\n")
		f.formatWhere(b, s.Having.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}

	formatLock(b, s.Lock, p)
}

func (f *formatter) formatSelectExprs(b *strings.Builder, exprs *sqlparser.SelectExprs, pi string, depth int) {
	if exprs == nil {
		return
	}

	lines := make([]string, len(exprs.Exprs))
	for i, expr := range exprs.Exprs {
		lines[i] = f.formatSelectExpr(expr, depth)
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatSelectExpr(expr sqlparser.SelectExpr, depth int) string {
	switch e := expr.(type) {
	case *sqlparser.AliasedExpr:
		s := f.formatExpr(e.Expr, depth+1)
		if !e.As.IsEmpty() {
			s += " " + f.keyword("AS") + " " + e.As.String()
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

func (f *formatter) formatTableExprs(b *strings.Builder, exprs []sqlparser.TableExpr, pi string, depth int) {
	for _, expr := range exprs {
		f.formatTableExpr(b, expr, pi, depth)
	}
}

func (f *formatter) formatTableExpr(b *strings.Builder, expr sqlparser.TableExpr, pi string, depth int) {
	switch e := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		f.formatAliasedTableExpr(b, e, pi, depth)
	case *sqlparser.JoinTableExpr:
		f.formatJoinTableExpr(b, e, pi, depth)
	case *sqlparser.ParenTableExpr:
		b.WriteString(pi + "(\n")
		f.formatTableExprs(b, e.Exprs, f.pad(depth+2), depth+1)
		b.WriteString(pi + ")\n")
	default:
		b.WriteString(pi + sqlparser.String(expr) + "\n")
	}
}

func (f *formatter) formatAliasedTableExpr(b *strings.Builder, e *sqlparser.AliasedTableExpr, pi string, depth int) {
	if sub, ok := e.Expr.(*sqlparser.DerivedTable); ok {
		b.WriteString(pi + "(\n")
		f.formatStatement(b, sub.Select, depth+1)
		b.WriteString(pi + ")")
	} else {
		b.WriteString(pi + sqlparser.String(e.Expr))
	}

	if !e.As.IsEmpty() {
		b.WriteString(" " + e.As.String())
	}

	b.WriteString(formatIndexHints(e.Hints))
	b.WriteString("\n")
}

func formatIndexHints(hints sqlparser.IndexHints) string {
	if len(hints) == 0 {
		return ""
	}

	parts := make([]string, len(hints))

	for i, hint := range hints {
		hintType := strings.ToUpper(hint.Type.ToString())

		indexes := make([]string, len(hint.Indexes))
		for j, idx := range hint.Indexes {
			indexes[j] = idx.String()
		}

		s := hintType + " (" + strings.Join(indexes, ", ") + ")"

		forStr := hint.ForType.ToString()
		if forStr != "" {
			s = hintType + " FOR " + strings.ToUpper(forStr) + " (" + strings.Join(indexes, ", ") + ")"
		}

		parts[i] = s
	}

	return " " + strings.Join(parts, " ")
}

func (f *formatter) formatJoinTableExpr(b *strings.Builder, e *sqlparser.JoinTableExpr, pi string, depth int) {
	f.formatTableExpr(b, e.LeftExpr, pi, depth)
	joinStr := strings.ToUpper(e.Join.ToString())
	b.WriteString(pi + joinStr + "\n")
	f.formatTableExpr(b, e.RightExpr, pi, depth)

	if e.Condition != nil && e.Condition.On != nil {
		b.WriteString(f.pad(depth+2) + f.keyword("ON") + " " + f.formatExpr(e.Condition.On, depth) + "\n")
	}
}

func (f *formatter) formatWhere(b *strings.Builder, expr sqlparser.Expr, pi string, depth int) {
	f.formatWhereExpr(b, expr, pi, depth, true)
}

func (f *formatter) formatWhereExpr(b *strings.Builder, expr sqlparser.Expr, pi string, depth int, first bool) {
	exprDepth := depth + 1

	switch e := expr.(type) {
	case *sqlparser.AndExpr:
		f.formatWhereExpr(b, e.Left, pi, depth, first)
		f.formatWhereExpr(b, e.Right, pi, depth, false)
	case *sqlparser.OrExpr:
		f.formatWhereExpr(b, e.Left, pi, depth, first)
		b.WriteString(pi + f.keyword("OR") + " " + f.formatExpr(e.Right, exprDepth) + "\n")
	default:
		if first {
			b.WriteString(pi + f.formatExpr(expr, exprDepth) + "\n")
		} else {
			b.WriteString(pi + f.keyword("AND") + " " + f.formatExpr(expr, exprDepth) + "\n")
		}
	}
}

func (f *formatter) formatExpr(expr sqlparser.Expr, depth int) string {
	switch e := expr.(type) {
	case *sqlparser.ExistsExpr:
		var b strings.Builder

		b.WriteString(f.keyword("EXISTS") + " (\n")
		f.formatStatement(&b, e.Subquery.Select, depth+1)
		b.WriteString(f.pad(depth) + ")")

		return b.String()
	case *sqlparser.Subquery:
		var b strings.Builder

		b.WriteString("(\n")
		f.formatStatement(&b, e.Select, depth+1)
		b.WriteString(f.pad(depth) + ")")

		return b.String()
	case *sqlparser.ComparisonExpr:
		right := f.formatExpr(e.Right, depth)

		return f.formatExpr(e.Left, depth) + " " + e.Operator.ToString() + " " + right
	case *sqlparser.NotExpr:
		return f.keyword("NOT") + " " + f.formatExpr(e.Expr, depth)
	case *sqlparser.CaseExpr:
		return f.formatCaseExpr(e, depth)
	default:
		if accessor := getOverAccessor(expr); accessor != nil {
			if oc := accessor.getOverClause(); oc != nil {
				return f.formatExprWithOver(expr, accessor, depth)
			}
		}

		return f.applyKeywordCase(sqlparser.String(expr))
	}
}

func (f *formatter) formatCaseExpr(e *sqlparser.CaseExpr, depth int) string {
	var b strings.Builder

	pi := f.pad(depth + 1)
	p := f.pad(depth)

	b.WriteString("CASE")

	if e.Expr != nil {
		b.WriteString(" " + f.formatExpr(e.Expr, depth))
	}

	b.WriteString("\n")

	for _, when := range e.Whens {
		cond := f.formatExpr(when.Cond, depth+1)
		val := f.formatExpr(when.Val, depth+1)

		b.WriteString(pi + "WHEN " + cond + " THEN " + val + "\n")
	}

	if e.Else != nil {
		b.WriteString(pi + "ELSE " + f.formatExpr(e.Else, depth+1) + "\n")
	}

	b.WriteString(p + "END")

	return b.String()
}

type overClauseAccessor interface {
	getOverClause() *sqlparser.OverClause
	setOverClause(oc *sqlparser.OverClause)
}

type overClauseField struct {
	field **sqlparser.OverClause
}

func (o overClauseField) getOverClause() *sqlparser.OverClause   { return *o.field }
func (o overClauseField) setOverClause(oc *sqlparser.OverClause) { *o.field = oc }

// Each case is trivially the same — complexity comes from the number of AST types, not logic.
func getOverAccessor(expr sqlparser.Expr) overClauseAccessor { //nolint:cyclop,funlen,ireturn
	switch e := expr.(type) {
	case *sqlparser.Count:
		return overClauseField{&e.OverClause}
	case *sqlparser.CountStar:
		return overClauseField{&e.OverClause}
	case *sqlparser.Sum:
		return overClauseField{&e.OverClause}
	case *sqlparser.Avg:
		return overClauseField{&e.OverClause}
	case *sqlparser.Min:
		return overClauseField{&e.OverClause}
	case *sqlparser.Max:
		return overClauseField{&e.OverClause}
	case *sqlparser.BitAnd:
		return overClauseField{&e.OverClause}
	case *sqlparser.BitOr:
		return overClauseField{&e.OverClause}
	case *sqlparser.BitXor:
		return overClauseField{&e.OverClause}
	case *sqlparser.Std:
		return overClauseField{&e.OverClause}
	case *sqlparser.StdDev:
		return overClauseField{&e.OverClause}
	case *sqlparser.StdPop:
		return overClauseField{&e.OverClause}
	case *sqlparser.StdSamp:
		return overClauseField{&e.OverClause}
	case *sqlparser.VarPop:
		return overClauseField{&e.OverClause}
	case *sqlparser.VarSamp:
		return overClauseField{&e.OverClause}
	case *sqlparser.Variance:
		return overClauseField{&e.OverClause}
	case *sqlparser.ArgumentLessWindowExpr:
		return overClauseField{&e.OverClause}
	case *sqlparser.FirstOrLastValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlparser.NtileExpr:
		return overClauseField{&e.OverClause}
	case *sqlparser.NTHValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlparser.LagLeadExpr:
		return overClauseField{&e.OverClause}
	case *sqlparser.JSONArrayAgg:
		return overClauseField{&e.OverClause}
	case *sqlparser.JSONObjectAgg:
		return overClauseField{&e.OverClause}
	default:
		return nil
	}
}

func (f *formatter) formatExprWithOver(expr sqlparser.Expr, accessor overClauseAccessor, depth int) string {
	oc := accessor.getOverClause()

	// Temporarily remove the OverClause to get the base function string
	accessor.setOverClause(nil)

	base := f.applyKeywordCase(sqlparser.String(expr))

	accessor.setOverClause(oc)

	return base + " " + f.formatOverClause(oc, depth)
}

func (f *formatter) formatOverClause(oc *sqlparser.OverClause, depth int) string {
	if !oc.WindowName.IsEmpty() && oc.WindowSpec == nil {
		return "OVER " + oc.WindowName.String()
	}

	if oc.WindowSpec == nil {
		return "OVER ()"
	}

	parts := f.formatWindowSpecParts(oc.WindowSpec)
	if len(parts) == 0 {
		return "OVER ()"
	}

	pi := f.pad(depth + 1)
	p := f.pad(depth)

	var b strings.Builder

	b.WriteString("OVER (\n")

	for _, part := range parts {
		b.WriteString(pi + part + "\n")
	}

	b.WriteString(p + ")")

	return b.String()
}

func (f *formatter) formatWindowSpecParts(spec *sqlparser.WindowSpecification) []string {
	var parts []string

	if len(spec.PartitionClause) > 0 {
		exprs := make([]string, len(spec.PartitionClause))
		for i, e := range spec.PartitionClause {
			exprs[i] = f.applyKeywordCase(sqlparser.String(e))
		}

		parts = append(parts, "PARTITION BY "+strings.Join(exprs, ", "))
	}

	if len(spec.OrderClause) > 0 {
		parts = append(parts, "ORDER BY "+f.formatOrderExprs(spec.OrderClause))
	}

	if spec.FrameClause != nil {
		parts = append(parts, strings.ToUpper(strings.TrimSpace(sqlparser.String(spec.FrameClause))))
	}

	return parts
}

func (f *formatter) formatOrderExprs(orders sqlparser.OrderBy) string {
	strs := make([]string, len(orders))

	for i, o := range orders {
		dir := ""
		if o.Direction == sqlparser.DescOrder {
			dir = f.descSuffix()
		}

		strs[i] = f.applyKeywordCase(sqlparser.String(o.Expr)) + dir
	}

	return strings.Join(strs, ", ")
}

func (f *formatter) descSuffix() string {
	return " " + f.keyword("DESC")
}

func (f *formatter) formatInsert(b *strings.Builder, s *sqlparser.Insert, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	action := "INSERT"
	if s.Action == sqlparser.ReplaceAct {
		action = "REPLACE"
	}

	if s.Ignore {
		action += " IGNORE"
	}

	b.WriteString(p + action + " INTO\n")
	b.WriteString(pi + sqlparser.String(s.Table) + "\n")

	f.formatInsertColumns(b, s.Columns, p, pi)
	f.formatInsertRows(b, s.Rows, p, pi, depth)
	f.formatOnDupUpdate(b, s.OnDup, p, pi)
}

func (f *formatter) formatInsertColumns(b *strings.Builder, cols sqlparser.Columns, p, pi string) {
	if len(cols) == 0 {
		return
	}

	b.WriteString(p + "(\n")

	lines := make([]string, len(cols))
	for i, col := range cols {
		lines[i] = col.String()
	}

	f.writeList(b, pi, lines)

	b.WriteString(p + ")\n")
}

func (f *formatter) formatInsertRows(b *strings.Builder, rows sqlparser.InsertRows, p, pi string, depth int) {
	switch r := rows.(type) {
	case sqlparser.Values:
		b.WriteString(p + "VALUES\n")

		lines := make([]string, len(r))

		for i, row := range r {
			vals := make([]string, len(row))
			for j, v := range row {
				vals[j] = f.formatExpr(v, depth)
			}

			lines[i] = "(" + strings.Join(vals, ", ") + ")"
		}

		f.writeList(b, pi, lines)
	case *sqlparser.Select:
		f.formatSelect(b, r, depth)
	default:
		b.WriteString(p + sqlparser.String(rows) + "\n")
	}
}

func (f *formatter) formatOnDupUpdate(b *strings.Builder, onDup sqlparser.OnDup, p, pi string) {
	if len(onDup) == 0 {
		return
	}

	b.WriteString(p + "ON DUPLICATE KEY UPDATE\n")

	lines := make([]string, len(onDup))
	for i, expr := range onDup {
		lines[i] = sqlparser.String(expr)
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatUpdate(b *strings.Builder, s *sqlparser.Update, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	action := "UPDATE"
	if s.Ignore {
		action = "UPDATE IGNORE"
	}

	b.WriteString(p + action + "\n")
	f.formatTableExprs(b, s.TableExprs, pi, depth)

	b.WriteString(p + "SET\n")

	lines := make([]string, len(s.Exprs))
	for i, expr := range s.Exprs {
		lines[i] = f.applyKeywordCase(sqlparser.String(expr))
	}

	f.writeList(b, pi, lines)

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}
}

func (f *formatter) formatDelete(b *strings.Builder, s *sqlparser.Delete, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	action := "DELETE"
	if s.Ignore {
		action = "DELETE IGNORE"
	}

	if len(s.Targets) > 0 {
		b.WriteString(p + action + "\n")

		lines := make([]string, len(s.Targets))
		for i, target := range s.Targets {
			lines[i] = sqlparser.String(target)
		}

		f.writeList(b, pi, lines)

		b.WriteString(p + "FROM\n")
	} else {
		b.WriteString(p + action + " FROM\n")
	}

	f.formatTableExprs(b, s.TableExprs, pi, depth)

	if s.Where != nil {
		b.WriteString(p + "WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}
}

func (f *formatter) formatUnion(b *strings.Builder, s *sqlparser.Union, depth int) {
	p := f.pad(depth)

	f.formatWith(b, s.With, depth)

	f.formatStatement(b, s.Left, depth)

	op := "UNION"
	if !s.Distinct {
		op = "UNION ALL"
	}

	b.WriteString(p + op + "\n")
	f.formatStatement(b, s.Right, depth)

	pi := f.pad(depth + 1)
	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}

	formatLock(b, s.Lock, p)
}

func formatLock(b *strings.Builder, lock sqlparser.Lock, p string) {
	if lock == sqlparser.NoLock {
		return
	}

	lockStr := strings.ToUpper(strings.TrimSpace(lock.ToString()))
	b.WriteString(p + lockStr + "\n")
}

func (f *formatter) formatGroupBy(b *strings.Builder, groupBy *sqlparser.GroupBy, p, pi string, depth int) {
	if groupBy == nil || len(groupBy.Exprs) == 0 {
		return
	}

	b.WriteString(p + "GROUP BY\n")

	lines := make([]string, len(groupBy.Exprs))
	for i, expr := range groupBy.Exprs {
		lines[i] = f.formatExpr(expr, depth)
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatOrderBy(b *strings.Builder, orders sqlparser.OrderBy, p, pi string, depth int) {
	if len(orders) == 0 {
		return
	}

	b.WriteString(p + "ORDER BY\n")

	lines := make([]string, len(orders))

	for i, order := range orders {
		dir := ""
		if order.Direction == sqlparser.DescOrder {
			dir = f.descSuffix()
		}

		lines[i] = f.formatExpr(order.Expr, depth) + dir
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatLimit(b *strings.Builder, limit *sqlparser.Limit, p string, depth int) {
	pi := f.pad(depth + 1)

	if limit.Offset != nil {
		b.WriteString(p + "LIMIT\n")
		b.WriteString(pi + f.formatExpr(limit.Rowcount, depth) + "\n")
		b.WriteString(p + "OFFSET\n")
		b.WriteString(pi + f.formatExpr(limit.Offset, depth) + "\n")
	} else {
		b.WriteString(p + "LIMIT\n")
		b.WriteString(pi + f.formatExpr(limit.Rowcount, depth) + "\n")
	}
}

var upperKeywordReplacer = strings.NewReplacer(
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

// applyKeywordCase renders text produced by the underlying parser (which
// always emits these operator/predicate keywords in lowercase) according to
// the configured keyword case.
func (f *formatter) applyKeywordCase(s string) string {
	if f.keywordCase == KeywordCaseUpper {
		return upperKeywordReplacer.Replace(s)
	}

	// The parser's own output is already lowercase, so KeywordCaseLower and
	// KeywordCasePreserve both pass the text through unchanged.
	return s
}
