package sqlast

import "strings"

// AliasedTableExpr represents a table expression with an optional alias.
type AliasedTableExpr struct {
	Expr  SimpleTableExpr
	As    TableIdent
	Hints IndexHints
}

// String returns AliasedTableExpr's SQL text.
func (a *AliasedTableExpr) String() string {
	var b strings.Builder

	b.WriteString(a.Expr.String())

	if !a.As.IsEmpty() {
		b.WriteString(" " + a.As.String())
	}

	for _, hint := range a.Hints {
		b.WriteString(" " + hint.String())
	}

	return b.String()
}

// JoinTableExpr represents a JOIN expression.
type JoinTableExpr struct {
	LeftExpr  TableExpr
	Join      JoinType
	RightExpr TableExpr
	Condition *JoinCondition
}

// String returns JoinTableExpr's SQL text.
func (j *JoinTableExpr) String() string {
	var b strings.Builder

	b.WriteString(j.LeftExpr.String())
	b.WriteString(" " + strings.ToUpper(j.Join.ToString()) + " ")
	b.WriteString(j.RightExpr.String())

	if j.Condition != nil && j.Condition.On != nil {
		b.WriteString(" ON " + j.Condition.On.String())
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

// AliasedExpr represents an expression with an optional alias in a SELECT clause.
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

// StarExpr represents a * or table.* expression in a SELECT clause.
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
