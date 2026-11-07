package pgast

import "strings"

// AliasedTableExpr represents a table expression with an optional alias.
// LATERAL is only meaningful when Expr is a *DerivedTable or a function
// call, but is stored here (not on DerivedTable) since PostgreSQL's grammar
// attaches LATERAL to the FROM item, not to the subquery itself.
//
// Only and IncludeDescendants render PostgreSQL's table-inheritance markers
// ("ONLY name" and "name *"). They are only meaningful when Expr is a
// TableName, and the parser never sets both.
//
// Columns is the alias's column list, as in "AS t(a, b)". Sample is a
// table's TABLESAMPLE clause, which follows the alias.
type AliasedTableExpr struct {
	Expr               SimpleTableExpr
	As                 TableIdent
	Columns            Columns
	Lateral            bool
	Only               bool
	IncludeDescendants bool
	Sample             *TableSample
}

// String returns AliasedTableExpr's SQL text.
func (a *AliasedTableExpr) String() string {
	var b strings.Builder

	if a.Lateral {
		b.WriteString("LATERAL ")
	}

	if a.Only {
		b.WriteString("ONLY ")
	}

	b.WriteString(a.Expr.String())

	if a.IncludeDescendants {
		b.WriteString(" *")
	}

	b.WriteString(aliasClauseString(a.As, a.Columns))

	if a.Sample != nil {
		b.WriteString(" ")
		b.WriteString(a.Sample.String())
	}

	return b.String()
}

// aliasClauseString renders an optional " alias[(col, ...)]" suffix.
func aliasClauseString(alias TableIdent, columns Columns) string {
	if alias.IsEmpty() {
		return ""
	}

	if len(columns) == 0 {
		return " " + alias.String()
	}

	return " " + alias.String() + "(" + columns.String() + ")"
}

// TableSample represents a "TABLESAMPLE method (args) [REPEATABLE (seed)]"
// clause.
type TableSample struct {
	Method     TableName
	Args       []Expr
	Repeatable Expr
}

// String returns TableSample's SQL text.
func (t *TableSample) String() string {
	s := "TABLESAMPLE " + t.Method.String() + " (" + exprListString(t.Args) + ")"
	if t.Repeatable != nil {
		s += " REPEATABLE (" + t.Repeatable.String() + ")"
	}

	return s
}

// FuncTable represents a set-returning function call used as a FROM item,
// such as generate_series(1, 3), or several calls zipped side by side by
// ROWS FROM (...), optionally WITH ORDINALITY. It sits inside an
// AliasedTableExpr, which carries its alias and LATERAL.
type FuncTable struct {
	Funcs          []Expr
	RowsFrom       bool
	WithOrdinality bool
}

// String returns FuncTable's SQL text.
func (f *FuncTable) String() string {
	s := exprListString(f.Funcs)
	if f.RowsFrom {
		s = "ROWS FROM (" + s + ")"
	}

	if f.WithOrdinality {
		s += " WITH ORDINALITY"
	}

	return s
}

// JoinType represents the kind of a JOIN.
type JoinType int8

const (
	JoinInner JoinType = iota
	JoinLeft
	JoinRight
	JoinFull
	JoinCross
	JoinNatural
	JoinNaturalLeft
	JoinNaturalRight
	JoinNaturalFull
)

var joinTypeStrings = [...]string{
	JoinInner:        "JOIN",
	JoinLeft:         "LEFT JOIN",
	JoinRight:        "RIGHT JOIN",
	JoinFull:         "FULL JOIN",
	JoinCross:        "CROSS JOIN",
	JoinNatural:      "NATURAL JOIN",
	JoinNaturalLeft:  "NATURAL LEFT JOIN",
	JoinNaturalRight: "NATURAL RIGHT JOIN",
	JoinNaturalFull:  "NATURAL FULL JOIN",
}

// String returns JoinType's SQL text.
func (j JoinType) String() string {
	if j >= 0 && int(j) < len(joinTypeStrings) {
		return joinTypeStrings[j]
	}

	return "JOIN"
}

// JoinTableExpr represents a JOIN expression.
type JoinTableExpr struct {
	Left      TableExpr
	Join      JoinType
	Right     TableExpr
	Condition *JoinCondition
}

// String returns JoinTableExpr's SQL text.
func (j *JoinTableExpr) String() string {
	var b strings.Builder

	b.WriteString(j.Left.String())
	b.WriteString(" ")
	b.WriteString(j.Join.String())
	b.WriteString(" ")
	b.WriteString(j.Right.String())

	switch {
	case j.Condition == nil:
	case j.Condition.On != nil:
		b.WriteString(" ON ")
		b.WriteString(j.Condition.On.String())
	case j.Condition.Using != nil:
		b.WriteString(" USING (")
		b.WriteString(j.Condition.Using.String())
		b.WriteString(")")

		if !j.Condition.UsingAlias.IsEmpty() {
			b.WriteString(" AS ")
			b.WriteString(j.Condition.UsingAlias.String())
		}
	}

	return b.String()
}

// ParenTableExpr represents a parenthesized list of table expressions with
// an optional alias.
type ParenTableExpr struct {
	Exprs   []TableExpr
	As      TableIdent
	Columns Columns
}

// String returns ParenTableExpr's SQL text.
func (p *ParenTableExpr) String() string {
	strs := make([]string, len(p.Exprs))
	for i, e := range p.Exprs {
		strs[i] = e.String()
	}

	return "(" + strings.Join(strs, ", ") + ")" + aliasClauseString(p.As, p.Columns)
}

// DerivedTable represents a subquery used as a table.
type DerivedTable struct {
	Select Statement
}

// String returns DerivedTable's SQL text.
func (d *DerivedTable) String() string {
	return "(" + d.Select.String() + ")"
}

// AliasedExpr represents an expression with an optional alias in a SELECT
// (or RETURNING) list.
type AliasedExpr struct {
	Expr Expr
	As   ColIdent
}

// String returns AliasedExpr's SQL text.
func (a *AliasedExpr) String() string {
	s := a.Expr.String()
	if !a.As.IsEmpty() {
		s += " AS " + a.As.String()
	}

	return s
}

// StarExpr represents a *, table.*, or schema.table.* item in a SELECT list.
type StarExpr struct {
	TableName TableName
}

// String returns StarExpr's SQL text.
func (s *StarExpr) String() string {
	if s.TableName.Name.IsEmpty() {
		return "*"
	}

	return s.TableName.String() + ".*"
}
