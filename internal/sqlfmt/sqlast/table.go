package sqlast

import "strings"

// AliasedTableExpr represents a table expression with an optional alias.
type AliasedTableExpr struct {
	Expr  SimpleTableExpr
	As    TableIdent
	Hints IndexHints
}

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

func (*AliasedTableExpr) iTableExpr() {}

// JoinTableExpr represents a JOIN expression.
type JoinTableExpr struct {
	LeftExpr  TableExpr
	Join      JoinType
	RightExpr TableExpr
	Condition *JoinCondition
}

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

func (*JoinTableExpr) iTableExpr() {}

// ParenTableExpr represents a parenthesized list of table expressions.
type ParenTableExpr struct {
	Exprs []TableExpr
}

func (p *ParenTableExpr) String() string {
	strs := make([]string, len(p.Exprs))
	for i, e := range p.Exprs {
		strs[i] = e.String()
	}

	return "(" + strings.Join(strs, ", ") + ")"
}

func (*ParenTableExpr) iTableExpr() {}

// DerivedTable represents a subquery used as a table.
type DerivedTable struct {
	Select Statement
}

func (d *DerivedTable) String() string {
	return "(" + d.Select.String() + ")"
}

func (*DerivedTable) iSimpleTableExpr() {}

// AliasedExpr represents an expression with an optional alias in a SELECT clause.
type AliasedExpr struct {
	Expr Expr
	As   ColIdent
}

func (a *AliasedExpr) String() string {
	s := a.Expr.String()
	if !a.As.IsEmpty() {
		s += " AS " + a.As.String()
	}

	return s
}

func (*AliasedExpr) iSelectExpr() {}

// StarExpr represents a * or table.* expression in a SELECT clause.
type StarExpr struct {
	TableName TableName
}

func (s *StarExpr) String() string {
	if s.TableName.Name.IsEmpty() {
		return "*"
	}

	return s.TableName.Name.String() + ".*"
}

func (*StarExpr) iSelectExpr() {}
