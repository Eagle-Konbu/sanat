package sqlfmt

import (
	"fmt"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// pgAggregateNames are the aggregate and window function names the
// PostgreSQL renderer writes in upper case, the same set the MySQL formatter
// renders upper case. Any other function name keeps its source spelling.
var pgAggregateNames = map[string]bool{
	"count": true, "sum": true, "avg": true, "min": true, "max": true,
	"bit_and": true, "bit_or": true, "bit_xor": true,
	"std": true, "stddev": true, "stddev_pop": true, "stddev_samp": true,
	"var_pop": true, "var_samp": true, "variance": true,
	"row_number": true, "rank": true, "dense_rank": true, "percent_rank": true, "cume_dist": true,
	"first_value": true, "last_value": true, "ntile": true, "nth_value": true, "lag": true, "lead": true,
	"json_arrayagg": true, "json_objectagg": true,
}

// expr renders an expression for a line indented at depth. Subqueries and
// CASE expressions span several lines, indented relative to depth.
//
// The expression kinds are split across several helpers, each reporting
// whether it handled expr, to keep each type switch's complexity down.
func (f *pgFormatter) expr(expr pgast.Expr, depth int) string {
	renderers := []func(pgast.Expr, int) (string, bool){
		f.basicExpr, f.predicateExpr, f.operatorExpr, f.compositeExpr, f.funcFormExpr,
	}

	for _, render := range renderers {
		if s, ok := render(expr, depth); ok {
			return s
		}
	}

	panic(fmt.Sprintf("sqlfmt: unhandled PostgreSQL expr type %T", expr))
}

func (f *pgFormatter) exprs(exprs []pgast.Expr, depth int) []string {
	strs := make([]string, len(exprs))
	for i, e := range exprs {
		strs[i] = f.expr(e, depth)
	}

	return strs
}

func (f *pgFormatter) exprList(exprs []pgast.Expr, depth int) string {
	return strings.Join(f.exprs(exprs, depth), ", ")
}

// basicExpr renders literals, column references, and logical connectives.
func (f *pgFormatter) basicExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.Literal:
		return f.literal(e), true
	case *pgast.ColName:
		return f.colName(e), true
	case *pgast.AndExpr:
		return f.expr(e.Left, depth) + " " + f.keyword("AND") + " " + f.expr(e.Right, depth), true
	case *pgast.OrExpr:
		return f.expr(e.Left, depth) + " " + f.keyword("OR") + " " + f.expr(e.Right, depth), true
	case *pgast.NotExpr:
		return f.keyword("NOT") + " " + f.expr(e.Expr, depth), true
	case *pgast.ParenExpr:
		return "(" + f.expr(e.Expr, depth) + ")", true
	case *pgast.TypedLiteral:
		return f.typedLiteral(e), true
	case *pgast.CastExpr:
		return f.castExpr(e, depth), true
	default:
		return "", false
	}
}

func (f *pgFormatter) colName(c *pgast.ColName) string {
	if c.Qualifier.IsEmpty() {
		return f.colIdent(c.Name)
	}

	return f.tableName(c.Qualifier) + "." + f.colIdent(c.Name)
}

// literal renders a literal as written in the source. NULL, TRUE, and FALSE
// are keywords, so they follow keyword_case instead, and the DEFAULT of a
// VALUES row or SET assignment is a clause keyword, so it is uppercased.
func (f *pgFormatter) literal(l *pgast.Literal) string {
	switch l.Val {
	case "NULL", "TRUE", "FALSE":
		return f.keyword(l.Val)
	case "DEFAULT":
		return l.Val
	}

	if l.Raw != "" {
		return l.Raw
	}

	return l.Val
}

// predicateExpr renders comparisons and the IS, BETWEEN, and LIKE family.
func (f *pgFormatter) predicateExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.ComparisonExpr:
		s := f.expr(e.Left, depth) + " " + f.keyword(e.Operator.String()) + " " + f.expr(e.Right, depth)

		return s + f.escape(e.Escape, depth), true
	case *pgast.ILikeExpr:
		return f.iLikeExpr(e, depth), true
	case *pgast.RangeCond:
		return f.rangeCond(e, depth), true
	case *pgast.IsExpr:
		return f.expr(e.Expr, depth) + " " + f.keyword(isPrefix(e.Not)+"NULL"), true
	case *pgast.IsTruthExpr:
		return f.expr(e.Expr, depth) + " " + f.keyword(isPrefix(e.Not)+e.Value.String()), true
	case *pgast.IsDistinctFromExpr:
		return f.expr(e.Left, depth) + " " + f.keyword(isPrefix(e.Not)+"DISTINCT FROM") + " " + f.expr(e.Right, depth), true
	case *pgast.IsNormalizedExpr:
		return f.isNormalizedExpr(e, depth), true
	case *pgast.ExistsExpr:
		return f.keyword("EXISTS") + " " + f.subquery(e.Subquery.Select, depth), true
	default:
		return "", false
	}
}

func (f *pgFormatter) iLikeExpr(e *pgast.ILikeExpr, depth int) string {
	op := "ILIKE"
	if e.Not {
		op = "NOT ILIKE"
	}

	return f.expr(e.Left, depth) + " " + f.keyword(op) + " " + f.expr(e.Right, depth) + f.escape(e.Escape, depth)
}

func (f *pgFormatter) isNormalizedExpr(e *pgast.IsNormalizedExpr, depth int) string {
	form := ""
	if e.Form != "" {
		form = e.Form + " "
	}

	return f.expr(e.Expr, depth) + " " + f.keyword(isPrefix(e.Not)+form+"NORMALIZED")
}

func isPrefix(not bool) string {
	if not {
		return "IS NOT "
	}

	return "IS "
}

func (f *pgFormatter) escape(escape pgast.Expr, depth int) string {
	if escape == nil {
		return ""
	}

	return " " + f.keyword("ESCAPE") + " " + f.expr(escape, depth)
}

func (f *pgFormatter) rangeCond(r *pgast.RangeCond, depth int) string {
	op := "BETWEEN"
	if r.Symmetric {
		op = "BETWEEN SYMMETRIC"
	}

	if r.Not {
		op = "NOT " + op
	}

	return f.expr(r.Left, depth) + " " + f.keyword(op) + " " + f.expr(r.From, depth) + " " +
		f.keyword("AND") + " " + f.expr(r.To, depth)
}

// operatorExpr renders arithmetic, generic, and quantified operators and
// the postfix COLLATE and AT TIME ZONE forms.
func (f *pgFormatter) operatorExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.ArithmeticExpr:
		return f.expr(e.Left, depth) + " " + e.Operator.String() + " " + f.expr(e.Right, depth), true
	case *pgast.UnaryExpr:
		return f.unaryExpr(e, depth), true
	case *pgast.OperatorExpr:
		return f.operatorApplication(e, depth), true
	case *pgast.QuantifiedExpr:
		return f.quantifiedExpr(e, depth), true
	case *pgast.CollateExpr:
		return f.expr(e.Expr, depth) + " " + f.keyword("COLLATE") + " " + f.anyName(e.Collation), true
	case *pgast.AtTimeZoneExpr:
		return f.expr(e.Expr, depth) + " " + f.keyword("AT TIME ZONE") + " " + f.expr(e.Zone, depth), true
	case *pgast.OverlapsExpr:
		return f.expr(e.Left, depth) + " " + f.keyword("OVERLAPS") + " " + f.expr(e.Right, depth), true
	case *pgast.NamedArgExpr:
		return f.namedArg(e, depth), true
	default:
		return "", false
	}
}

// operatorApplication renders a generic prefix or infix operator applied to
// its operands.
func (f *pgFormatter) operatorApplication(o *pgast.OperatorExpr, depth int) string {
	if o.Left == nil {
		return f.operator(o.Operator) + " " + f.expr(o.Right, depth)
	}

	return f.expr(o.Left, depth) + " " + f.operator(o.Operator) + " " + f.expr(o.Right, depth)
}

func (f *pgFormatter) namedArg(n *pgast.NamedArgExpr, depth int) string {
	op := " => "
	if n.ColonEquals {
		op = " := "
	}

	return f.colIdent(n.Name) + op + f.expr(n.Value, depth)
}

// unaryExpr separates the operator from an operand that starts with an
// operator character, which it would otherwise fuse with, as
// pgast.UnaryExpr.String does.
func (f *pgFormatter) unaryExpr(u *pgast.UnaryExpr, depth int) string {
	operand := f.expr(u.Expr, depth)
	if operand != "" && strings.ContainsRune("~!@#^&|`?+-*/%<>=", rune(operand[0])) {
		return u.Operator.String() + " " + operand
	}

	return u.Operator.String() + operand
}

// operator renders a generic operator. A word operator, such as the LIKE of
// a quantified "LIKE ANY (...)", follows keyword_case.
func (f *pgFormatter) operator(o pgast.Operator) string {
	if !o.Explicit {
		return f.keyword(o.Symbol)
	}

	if o.Schema.IsEmpty() {
		return "OPERATOR(" + o.Symbol + ")"
	}

	return "OPERATOR(" + f.tableIdent(o.Schema) + "." + o.Symbol + ")"
}

func (f *pgFormatter) quantifiedExpr(q *pgast.QuantifiedExpr, depth int) string {
	s := f.expr(q.Left, depth) + " " + f.operator(q.Operator) + " " + f.keyword(q.Quantifier.String()) + " "

	if sub, ok := q.Right.(*pgast.Subquery); ok {
		return s + f.subquery(sub.Select, depth)
	}

	return s + "(" + f.expr(q.Right, depth) + ")"
}

// compositeExpr renders subqueries, CASE, rows, arrays, and grouping
// elements.
func (f *pgFormatter) compositeExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.Subquery:
		return f.subquery(e.Select, depth), true
	case *pgast.CaseExpr:
		return f.caseExpr(e, depth), true
	case pgast.ValTuple:
		return "(" + f.exprList(e, depth) + ")", true
	case *pgast.RowExpr:
		return "ROW(" + f.exprList(e.Exprs, depth) + ")", true
	case *pgast.ArrayExpr:
		return f.arrayExpr(e, depth), true
	case *pgast.ArraySubquery:
		return "ARRAY" + f.subquery(e.Select, depth), true
	case *pgast.ArraySubscriptExpr:
		return f.arraySubscript(e, depth), true
	case *pgast.FieldSelectExpr:
		return f.fieldSelect(e, depth), true
	default:
		return f.groupingExpr(expr, depth)
	}
}

// arrayExpr renders an ARRAY[...] constructor, or a bare [...] sub-array of
// a multidimensional one.
func (f *pgFormatter) arrayExpr(a *pgast.ArrayExpr, depth int) string {
	if a.Nested {
		return "[" + f.exprList(a.Elems, depth) + "]"
	}

	return "ARRAY[" + f.exprList(a.Elems, depth) + "]"
}

func (f *pgFormatter) fieldSelect(e *pgast.FieldSelectExpr, depth int) string {
	if e.Star {
		return f.expr(e.Expr, depth) + ".*"
	}

	return f.expr(e.Expr, depth) + "." + f.colIdent(e.Field)
}

// subquery renders a parenthesized query with its body laid out one level
// deeper than depth and the closing parenthesis back at depth.
func (f *pgFormatter) subquery(stmt pgast.Statement, depth int) string {
	var b strings.Builder

	b.WriteString("(\n")
	f.formatStatement(&b, stmt, depth+1)
	b.WriteString(f.pad(depth) + ")")

	return b.String()
}

// caseExpr lays out a CASE expression with each WHEN and the ELSE on its own
// line, one level deeper than CASE and END, matching the MySQL formatter.
func (f *pgFormatter) caseExpr(e *pgast.CaseExpr, depth int) string {
	var b strings.Builder

	pi := f.pad(depth + 1)

	b.WriteString("CASE")

	if e.Expr != nil {
		b.WriteString(" " + f.expr(e.Expr, depth))
	}

	b.WriteString("\n")

	for _, when := range e.Whens {
		b.WriteString(pi + "WHEN " + f.expr(when.Cond, depth+1) + " THEN " + f.expr(when.Val, depth+1) + "\n")
	}

	if e.Else != nil {
		b.WriteString(pi + "ELSE " + f.expr(e.Else, depth+1) + "\n")
	}

	b.WriteString(f.pad(depth) + "END")

	return b.String()
}

func (f *pgFormatter) arraySubscript(a *pgast.ArraySubscriptExpr, depth int) string {
	s := f.expr(a.Expr, depth) + "["

	if a.From != nil {
		s += f.expr(a.From, depth)
	}

	if a.Slice {
		s += ":"
	}

	if a.To != nil {
		s += f.expr(a.To, depth)
	}

	return s + "]"
}

func (f *pgFormatter) groupingExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.GroupingSets:
		sets := make([]string, len(e.Sets))
		for i, set := range e.Sets {
			sets[i] = "(" + f.exprList(set, depth) + ")"
		}

		return "GROUPING SETS (" + strings.Join(sets, ", ") + ")", true
	case *pgast.Cube:
		return "CUBE (" + f.exprList(e.Exprs, depth) + ")", true
	case *pgast.Rollup:
		return "ROLLUP (" + f.exprList(e.Exprs, depth) + ")", true
	default:
		return "", false
	}
}

// funcFormExpr renders function calls, including the SQL-standard forms
// whose arguments are separated by keywords.
func (f *pgFormatter) funcFormExpr(expr pgast.Expr, depth int) (string, bool) {
	switch e := expr.(type) {
	case *pgast.FuncExpr:
		return f.funcExpr(e, depth), true
	case *pgast.ExtractExpr:
		field := e.FieldRaw
		if field == "" {
			field = e.Field
		}

		return "EXTRACT(" + field + " FROM " + f.expr(e.Source, depth) + ")", true
	case *pgast.SubstringExpr:
		return f.substringExpr(e, depth), true
	case *pgast.PositionExpr:
		return "POSITION(" + f.expr(e.Substring, depth) + " IN " + f.expr(e.Source, depth) + ")", true
	case *pgast.TrimExpr:
		return f.trimExpr(e, depth), true
	case *pgast.OverlayExpr:
		s := "OVERLAY(" + f.expr(e.Expr, depth) + " PLACING " + f.expr(e.Placing, depth) + " FROM " + f.expr(e.From, depth)
		if e.For != nil {
			s += " FOR " + f.expr(e.For, depth)
		}

		return s + ")", true
	default:
		return "", false
	}
}

func (f *pgFormatter) substringExpr(s *pgast.SubstringExpr, depth int) string {
	out := "SUBSTRING(" + f.expr(s.Expr, depth)

	if s.Similar != nil {
		return out + " SIMILAR " + f.expr(s.Similar, depth) + " ESCAPE " + f.expr(s.Escape, depth) + ")"
	}

	if s.From != nil {
		out += " FROM " + f.expr(s.From, depth)
	}

	if s.For != nil {
		out += " FOR " + f.expr(s.For, depth)
	}

	return out + ")"
}

func (f *pgFormatter) trimExpr(t *pgast.TrimExpr, depth int) string {
	var parts []string

	if side := t.Side.String(); side != "" {
		parts = append(parts, side)
	}

	if t.Characters != nil {
		parts = append(parts, f.expr(t.Characters, depth))
	}

	if t.From {
		parts = append(parts, "FROM")
	}

	parts = append(parts, f.exprList(t.Args, depth))

	return "TRIM(" + strings.Join(parts, " ") + ")"
}

func (f *pgFormatter) funcExpr(fn *pgast.FuncExpr, depth int) string {
	var b strings.Builder

	if !fn.Qualifier.IsEmpty() {
		b.WriteString(f.tableIdent(fn.Qualifier) + ".")
	}

	b.WriteString(f.funcName(fn))
	b.WriteString("(" + f.funcArgs(fn, depth) + ")")

	if len(fn.WithinGroup) > 0 {
		b.WriteString(" WITHIN GROUP (ORDER BY " + strings.Join(f.orderItems(fn.WithinGroup, depth), ", ") + ")")
	}

	if fn.Filter != nil && fn.Filter.Expr != nil {
		b.WriteString(" FILTER (WHERE " + f.expr(fn.Filter.Expr, depth) + ")")
	}

	switch {
	case fn.Over != nil && fn.OverRef:
		b.WriteString(" OVER " + f.colIdent(fn.Over.Name))
	case fn.Over != nil:
		b.WriteString(" OVER " + f.windowSpec(fn.Over, depth))
	}

	return b.String()
}

// funcName renders a function's name as written, except that an unqualified,
// unquoted aggregate or window function name is uppercased.
func (f *pgFormatter) funcName(fn *pgast.FuncExpr) string {
	name := f.colIdent(fn.Name)

	if fn.Qualifier.IsEmpty() && !strings.HasPrefix(name, `"`) && pgAggregateNames[strings.ToLower(name)] {
		return strings.ToUpper(name)
	}

	return name
}

func (f *pgFormatter) funcArgs(fn *pgast.FuncExpr, depth int) string {
	if fn.Star {
		return "*"
	}

	args := f.exprs(fn.Args, depth)

	if fn.Variadic && len(args) > 0 {
		args[len(args)-1] = "VARIADIC " + args[len(args)-1]
	}

	s := strings.Join(args, ", ")

	if fn.Distinct {
		s = "DISTINCT " + s
	}

	if len(fn.OrderBy) > 0 {
		s += " ORDER BY " + strings.Join(f.orderItems(fn.OrderBy, depth), ", ")
	}

	return s
}

// windowSpec renders a window definition's parenthesized body: "()" when
// empty, otherwise each part on its own line one level deeper than depth,
// matching the MySQL formatter's OVER clause.
func (f *pgFormatter) windowSpec(w *pgast.WindowSpec, depth int) string {
	var parts []string

	if !w.Name.IsEmpty() {
		parts = append(parts, f.colIdent(w.Name))
	}

	if len(w.PartitionBy) > 0 {
		parts = append(parts, "PARTITION BY "+f.exprList(w.PartitionBy, depth+1))
	}

	if len(w.OrderBy) > 0 {
		parts = append(parts, "ORDER BY "+strings.Join(f.orderItems(w.OrderBy, depth+1), ", "))
	}

	if w.Frame != nil && w.Frame.Start != nil {
		parts = append(parts, f.frameClause(w.Frame, depth+1))
	}

	if len(parts) == 0 {
		return "()"
	}

	pi := f.pad(depth + 1)

	return "(\n" + pi + strings.Join(parts, "\n"+pi) + "\n" + f.pad(depth) + ")"
}

func (f *pgFormatter) frameClause(c *pgast.FrameClause, depth int) string {
	s := c.Unit.String() + " " + f.framePoint(c.Start, depth)
	if c.End != nil {
		s = c.Unit.String() + " BETWEEN " + f.framePoint(c.Start, depth) + " AND " + f.framePoint(c.End, depth)
	}

	if exclude := c.Exclude.String(); exclude != "" {
		s += " " + exclude
	}

	return s
}

func (f *pgFormatter) framePoint(p *pgast.FramePoint, depth int) string {
	switch p.Type {
	case pgast.ExprPreceding:
		return f.expr(p.Expr, depth) + " PRECEDING"
	case pgast.ExprFollowing:
		return f.expr(p.Expr, depth) + " FOLLOWING"
	default:
		return p.String()
	}
}

func (f *pgFormatter) castExpr(c *pgast.CastExpr, depth int) string {
	if c.Explicit {
		return "CAST(" + f.expr(c.Expr, depth) + " " + f.keyword("AS") + " " + f.typeName(c.Type) + ")"
	}

	return f.expr(c.Expr, depth) + "::" + f.typeName(c.Type)
}

func (f *pgFormatter) typedLiteral(l *pgast.TypedLiteral) string {
	value := l.ValueRaw
	if value == "" {
		value = l.Value
	}

	s := f.typeNameBase(l.Type) + " " + value
	if l.Type.IntervalFields != "" {
		s += " " + l.Type.IntervalFields
	}

	return s
}

// typeName renders a type name with its parts as written in the source. The
// built-in multi-word names (double precision, timestamp with time zone,
// ...) have no single source spelling and render in lower case.
func (f *pgFormatter) typeName(t *pgast.TypeName) string {
	var b strings.Builder

	b.WriteString(f.typeNameBase(t))

	if t.IntervalFields != "" {
		b.WriteString(" " + t.IntervalFields)
	}

	for _, bound := range t.ArrayBounds {
		b.WriteString("[" + bound + "]")
	}

	return b.String()
}

func (f *pgFormatter) typeNameBase(t *pgast.TypeName) string {
	names := make([]string, len(t.Names))
	for i, n := range t.Names {
		names[i] = f.colIdent(n)
	}

	s := strings.Join(names, ".")

	if len(t.Modifiers) > 0 {
		s += "(" + f.exprList(t.Modifiers, 0) + ")"
	}

	if tz := t.TimeZone.String(); tz != "" {
		s += " " + tz
	}

	return s
}
