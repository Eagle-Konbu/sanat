package pgast

import "strings"

// AliasedTableExpr represents a table expression with an optional alias.
// LATERAL is only meaningful when Expr is a *DerivedTable or a function
// call, but is stored here (not on DerivedTable) since PostgreSQL's grammar
// attaches LATERAL to the FROM item, not to the subquery itself.
type AliasedTableExpr struct {
	Expr    SimpleTableExpr
	As      TableIdent
	Lateral bool
}

// String returns AliasedTableExpr's SQL text.
func (a *AliasedTableExpr) String() string {
	var b strings.Builder

	if a.Lateral {
		b.WriteString("LATERAL ")
	}

	b.WriteString(a.Expr.String())

	if !a.As.IsEmpty() {
		b.WriteString(" ")
		b.WriteString(a.As.String())
	}

	return b.String()
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
	}

	return b.String()
}

// ParenTableExpr represents a parenthesized list of table expressions.
type ParenTableExpr struct {
	Exprs []TableExpr
}

// String returns ParenTableExpr's SQL text.
func (p *ParenTableExpr) String() string {
	strs := make([]string, len(p.Exprs))
	for i, e := range p.Exprs {
		strs[i] = e.String()
	}

	return "(" + strings.Join(strs, ", ") + ")"
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

// StarExpr represents a * or table.* item in a SELECT list.
type StarExpr struct {
	TableName TableName
}

// String returns StarExpr's SQL text.
func (s *StarExpr) String() string {
	if s.TableName.Name.IsEmpty() {
		return "*"
	}

	return s.TableName.Name.String() + ".*"
}
