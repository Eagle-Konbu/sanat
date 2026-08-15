package sqlast

import (
	"fmt"
	"strings"
)

// ComparisonExpr represents a comparison expression (e.g., a = b).
type ComparisonExpr struct {
	Operator ComparisonOperator
	Left     Expr
	Right    Expr
}

func (c *ComparisonExpr) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Operator.ToString(), c.Right.String())
}

// AndExpr represents an AND expression.
type AndExpr struct {
	Left  Expr
	Right Expr
}

func (a *AndExpr) String() string {
	return fmt.Sprintf("%s AND %s", a.Left.String(), a.Right.String())
}

// OrExpr represents an OR expression.
type OrExpr struct {
	Left  Expr
	Right Expr
}

func (o *OrExpr) String() string {
	return fmt.Sprintf("%s OR %s", o.Left.String(), o.Right.String())
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr Expr
}

func (n *NotExpr) String() string {
	return "NOT " + n.Expr.String()
}

// When represents a WHEN clause in a CASE expression.
type When struct {
	Cond Expr
	Val  Expr
}

// CaseExpr represents a CASE expression.
type CaseExpr struct {
	Expr  Expr
	Whens []*When
	Else  Expr
}

func (c *CaseExpr) String() string {
	var b strings.Builder

	b.WriteString("CASE")

	if c.Expr != nil {
		b.WriteString(" " + c.Expr.String())
	}

	for _, w := range c.Whens {
		fmt.Fprintf(&b, " WHEN %s THEN %s", w.Cond.String(), w.Val.String())
	}

	if c.Else != nil {
		b.WriteString(" ELSE " + c.Else.String())
	}

	b.WriteString(" END")

	return b.String()
}

// ExistsExpr represents an EXISTS expression.
type ExistsExpr struct {
	Subquery *Subquery
}

func (e *ExistsExpr) String() string {
	return "EXISTS (" + e.Subquery.Select.String() + ")"
}

// Subquery represents a subquery expression.
type Subquery struct {
	Select Statement
}

func (s *Subquery) String() string {
	return "(" + s.Select.String() + ")"
}

// ColName represents a column name, optionally qualified.
type ColName struct {
	Name      ColIdent
	Qualifier TableName
}

func (c *ColName) String() string {
	if c.Qualifier.IsEmpty() {
		return c.Name.String()
	}

	return c.Qualifier.String() + "." + c.Name.String()
}

// Literal represents a literal value.
type Literal struct {
	Val string
}

func (l *Literal) String() string {
	return l.Val
}

// FuncExpr represents a function call expression.
type FuncExpr struct {
	Qualifier TableIdent
	Name      ColIdent
	Exprs     []Expr
}

func (f *FuncExpr) String() string {
	name := f.Name.String()
	if !f.Qualifier.IsEmpty() {
		name = f.Qualifier.String() + "." + name
	}

	args := make([]string, len(f.Exprs))
	for i, e := range f.Exprs {
		args[i] = e.String()
	}

	return name + "(" + strings.Join(args, ", ") + ")"
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr Expr
}

func (p *ParenExpr) String() string {
	return "(" + p.Expr.String() + ")"
}

// --- Aggregate / Window function types ---
// All implement Expr and have an OverClause field.

func appendOver(b *strings.Builder, oc *OverClause) {
	if oc != nil {
		b.WriteString(" " + oc.String())
	}
}

func formatDistinctAgg(name string, arg Expr, distinct bool, oc *OverClause) string {
	var b strings.Builder

	b.WriteString(name + "(")

	if distinct {
		b.WriteString("DISTINCT ")
	}

	b.WriteString(arg.String() + ")")
	appendOver(&b, oc)

	return b.String()
}

func formatSimpleAgg(name string, arg Expr, oc *OverClause) string {
	var b strings.Builder

	b.WriteString(name + "(" + arg.String() + ")")
	appendOver(&b, oc)

	return b.String()
}

// Count represents COUNT(expr) or COUNT(DISTINCT expr).
type Count struct {
	Args       []Expr
	Distinct   bool
	OverClause *OverClause
}

func (c *Count) String() string {
	var b strings.Builder

	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		args[i] = a.String()
	}

	b.WriteString("COUNT(")

	if c.Distinct {
		b.WriteString("DISTINCT ")
	}

	b.WriteString(strings.Join(args, ", ") + ")")
	appendOver(&b, c.OverClause)

	return b.String()
}

// CountStar represents COUNT(*).
type CountStar struct {
	OverClause *OverClause
}

func (c *CountStar) String() string {
	var b strings.Builder

	b.WriteString("COUNT(*)")
	appendOver(&b, c.OverClause)

	return b.String()
}

// Sum represents SUM([DISTINCT] expr).
type Sum struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

func (s *Sum) String() string { return formatDistinctAgg("SUM", s.Arg, s.Distinct, s.OverClause) }

// Avg represents AVG([DISTINCT] expr).
type Avg struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

func (a *Avg) String() string { return formatDistinctAgg("AVG", a.Arg, a.Distinct, a.OverClause) }

// Min represents MIN([DISTINCT] expr).
type Min struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

func (m *Min) String() string { return formatDistinctAgg("MIN", m.Arg, m.Distinct, m.OverClause) }

// Max represents MAX([DISTINCT] expr).
type Max struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

func (m *Max) String() string { return formatDistinctAgg("MAX", m.Arg, m.Distinct, m.OverClause) }

// BitAnd represents BIT_AND(expr).
type BitAnd struct {
	Arg        Expr
	OverClause *OverClause
}

func (ba *BitAnd) String() string { return formatSimpleAgg("BIT_AND", ba.Arg, ba.OverClause) }

// BitOr represents BIT_OR(expr).
type BitOr struct {
	Arg        Expr
	OverClause *OverClause
}

func (bo *BitOr) String() string { return formatSimpleAgg("BIT_OR", bo.Arg, bo.OverClause) }

// BitXor represents BIT_XOR(expr).
type BitXor struct {
	Arg        Expr
	OverClause *OverClause
}

func (bx *BitXor) String() string { return formatSimpleAgg("BIT_XOR", bx.Arg, bx.OverClause) }

// Std represents STD(expr).
type Std struct {
	Arg        Expr
	OverClause *OverClause
}

func (s *Std) String() string { return formatSimpleAgg("STD", s.Arg, s.OverClause) }

// StdDev represents STDDEV(expr).
type StdDev struct {
	Arg        Expr
	OverClause *OverClause
}

func (s *StdDev) String() string { return formatSimpleAgg("STDDEV", s.Arg, s.OverClause) }

// StdPop represents STDDEV_POP(expr).
type StdPop struct {
	Arg        Expr
	OverClause *OverClause
}

func (s *StdPop) String() string { return formatSimpleAgg("STDDEV_POP", s.Arg, s.OverClause) }

// StdSamp represents STDDEV_SAMP(expr).
type StdSamp struct {
	Arg        Expr
	OverClause *OverClause
}

func (s *StdSamp) String() string { return formatSimpleAgg("STDDEV_SAMP", s.Arg, s.OverClause) }

// Variance represents VARIANCE(expr).
type Variance struct {
	Arg        Expr
	OverClause *OverClause
}

func (v *Variance) String() string { return formatSimpleAgg("VARIANCE", v.Arg, v.OverClause) }

// VarPop represents VAR_POP(expr).
type VarPop struct {
	Arg        Expr
	OverClause *OverClause
}

func (v *VarPop) String() string { return formatSimpleAgg("VAR_POP", v.Arg, v.OverClause) }

// VarSamp represents VAR_SAMP(expr).
type VarSamp struct {
	Arg        Expr
	OverClause *OverClause
}

func (v *VarSamp) String() string { return formatSimpleAgg("VAR_SAMP", v.Arg, v.OverClause) }

// ArgumentLessWindowExpr represents window functions with no arguments (e.g., ROW_NUMBER()).
type ArgumentLessWindowExpr struct {
	Type       ArgumentLessWindowExprType
	OverClause *OverClause
}

func (a *ArgumentLessWindowExpr) String() string {
	var b strings.Builder

	b.WriteString(a.Type.String() + "()")
	appendOver(&b, a.OverClause)

	return b.String()
}

// FirstOrLastValueExpr represents FIRST_VALUE(expr) or LAST_VALUE(expr).
type FirstOrLastValueExpr struct {
	Type                FirstOrLastValueExprType
	Expr                Expr
	NullTreatmentClause *NullTreatmentClause
	OverClause          *OverClause
}

func (f *FirstOrLastValueExpr) String() string {
	var b strings.Builder

	b.WriteString(f.Type.String() + "(" + f.Expr.String() + ")")

	if f.NullTreatmentClause != nil {
		b.WriteString(" " + f.NullTreatmentClause.String())
	}

	appendOver(&b, f.OverClause)

	return b.String()
}

// NtileExpr represents NTILE(n).
type NtileExpr struct {
	N          Expr
	OverClause *OverClause
}

func (n *NtileExpr) String() string {
	var b strings.Builder

	b.WriteString("NTILE(" + n.N.String() + ")")
	appendOver(&b, n.OverClause)

	return b.String()
}

// NTHValueExpr represents NTH_VALUE(expr, n).
type NTHValueExpr struct {
	Expr                Expr
	N                   Expr
	FromFirstLastClause *FromFirstLastClause
	NullTreatmentClause *NullTreatmentClause
	OverClause          *OverClause
}

func (n *NTHValueExpr) String() string {
	var b strings.Builder

	b.WriteString("NTH_VALUE(" + n.Expr.String() + ", " + n.N.String() + ")")

	if n.FromFirstLastClause != nil {
		b.WriteString(" " + n.FromFirstLastClause.String())
	}

	if n.NullTreatmentClause != nil {
		b.WriteString(" " + n.NullTreatmentClause.String())
	}

	appendOver(&b, n.OverClause)

	return b.String()
}

// LagLeadExpr represents LAG(expr, n, default) or LEAD(expr, n, default).
type LagLeadExpr struct {
	Type                LagLeadExprType
	Expr                Expr
	N                   Expr
	Default             Expr
	NullTreatmentClause *NullTreatmentClause
	OverClause          *OverClause
}

func (l *LagLeadExpr) String() string {
	var b strings.Builder

	args := []string{l.Expr.String()}
	if l.N != nil {
		args = append(args, l.N.String())
	}

	if l.Default != nil {
		args = append(args, l.Default.String())
	}

	b.WriteString(l.Type.String() + "(" + strings.Join(args, ", ") + ")")

	if l.NullTreatmentClause != nil {
		b.WriteString(" " + l.NullTreatmentClause.String())
	}

	appendOver(&b, l.OverClause)

	return b.String()
}

// JSONArrayAgg represents JSON_ARRAYAGG(expr).
type JSONArrayAgg struct {
	Expr       Expr
	OverClause *OverClause
}

func (j *JSONArrayAgg) String() string {
	return formatSimpleAgg("JSON_ARRAYAGG", j.Expr, j.OverClause)
}

// JSONObjectAgg represents JSON_OBJECTAGG(key, value).
type JSONObjectAgg struct {
	Key        Expr
	Value      Expr
	OverClause *OverClause
}

func (j *JSONObjectAgg) String() string {
	var b strings.Builder

	b.WriteString("JSON_OBJECTAGG(" + j.Key.String() + ", " + j.Value.String() + ")")
	appendOver(&b, j.OverClause)

	return b.String()
}
