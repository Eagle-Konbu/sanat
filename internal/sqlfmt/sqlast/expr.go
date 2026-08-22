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

// String returns ComparisonExpr's SQL text.
func (c *ComparisonExpr) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), strings.ToUpper(c.Operator.ToString()), c.Right.String())
}

// RangeCond represents a [NOT] BETWEEN ... AND ... expression.
type RangeCond struct {
	Not  bool
	Left Expr
	From Expr
	To   Expr
}

// String returns RangeCond's SQL text.
func (r *RangeCond) String() string {
	not := ""
	if r.Not {
		not = "NOT "
	}

	return fmt.Sprintf("%s %sBETWEEN %s AND %s", r.Left.String(), not, r.From.String(), r.To.String())
}

// IsExpr represents an IS [NOT] NULL expression.
type IsExpr struct {
	Not  bool
	Expr Expr
}

// String returns IsExpr's SQL text.
func (i *IsExpr) String() string {
	if i.Not {
		return i.Expr.String() + " IS NOT NULL"
	}

	return i.Expr.String() + " IS NULL"
}

// ValTuple represents a parenthesized list of expressions, e.g. the value
// list on the right-hand side of an IN predicate.
type ValTuple []Expr

// String returns ValTuple's SQL text.
func (v ValTuple) String() string {
	strs := make([]string, len(v))
	for i, e := range v {
		strs[i] = e.String()
	}

	return "(" + strings.Join(strs, ", ") + ")"
}

// ArithmeticOperator represents a binary arithmetic operator.
type ArithmeticOperator int8

const (
	PlusOp ArithmeticOperator = iota
	MinusOp
	MultOp
	DivOp
	ModOp
)

var arithmeticOpStrings = [...]string{
	PlusOp:  "+",
	MinusOp: "-",
	MultOp:  "*",
	DivOp:   "/",
	ModOp:   "%",
}

// ToString returns ArithmeticOperator's SQL operator text.
func (o ArithmeticOperator) ToString() string {
	if o >= 0 && int(o) < len(arithmeticOpStrings) {
		return arithmeticOpStrings[o]
	}

	return "+"
}

// ArithmeticExpr represents a binary arithmetic expression (e.g., a + b).
type ArithmeticExpr struct {
	Operator ArithmeticOperator
	Left     Expr
	Right    Expr
}

// String returns ArithmeticExpr's SQL text.
func (a *ArithmeticExpr) String() string {
	return fmt.Sprintf("%s %s %s", a.Left.String(), a.Operator.ToString(), a.Right.String())
}

// UnaryOperator represents a unary + or - operator.
type UnaryOperator int8

const (
	UPlusOp UnaryOperator = iota
	UMinusOp
)

// ToString returns UnaryOperator's SQL operator text.
func (u UnaryOperator) ToString() string {
	if u == UMinusOp {
		return "-"
	}

	return "+"
}

// UnaryExpr represents a unary +expr or -expr.
type UnaryExpr struct {
	Operator UnaryOperator
	Expr     Expr
}

// String returns UnaryExpr's SQL text.
func (u *UnaryExpr) String() string {
	return u.Operator.ToString() + u.Expr.String()
}

// AndExpr represents an AND expression.
type AndExpr struct {
	Left  Expr
	Right Expr
}

// String returns AndExpr's SQL text.
func (a *AndExpr) String() string {
	return fmt.Sprintf("%s AND %s", a.Left.String(), a.Right.String())
}

// OrExpr represents an OR expression.
type OrExpr struct {
	Left  Expr
	Right Expr
}

// String returns OrExpr's SQL text.
func (o *OrExpr) String() string {
	return fmt.Sprintf("%s OR %s", o.Left.String(), o.Right.String())
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr Expr
}

// String returns NotExpr's SQL text.
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

// String returns CaseExpr's SQL text.
func (c *CaseExpr) String() string {
	var b strings.Builder

	b.WriteString("CASE")

	if c.Expr != nil {
		b.WriteString(" ")
		b.WriteString(c.Expr.String())
	}

	for _, w := range c.Whens {
		fmt.Fprintf(&b, " WHEN %s THEN %s", w.Cond.String(), w.Val.String())
	}

	if c.Else != nil {
		b.WriteString(" ELSE ")
		b.WriteString(c.Else.String())
	}

	b.WriteString(" END")

	return b.String()
}

// ExistsExpr represents an EXISTS expression.
type ExistsExpr struct {
	Subquery *Subquery
}

// String returns ExistsExpr's SQL text.
func (e *ExistsExpr) String() string {
	return "EXISTS (" + e.Subquery.Select.String() + ")"
}

// Subquery represents a subquery expression.
type Subquery struct {
	Select Statement
}

// String returns Subquery's SQL text.
func (s *Subquery) String() string {
	return "(" + s.Select.String() + ")"
}

// ColName represents a column name, optionally qualified.
type ColName struct {
	Name      ColIdent
	Qualifier TableName
}

// String returns ColName's SQL text.
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

// String returns Literal's SQL text.
func (l *Literal) String() string {
	return l.Val
}

// FuncExpr represents a function call expression.
type FuncExpr struct {
	Qualifier TableIdent
	Name      ColIdent
	Exprs     []Expr
}

// String returns FuncExpr's SQL text.
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

// String returns ParenExpr's SQL text.
func (p *ParenExpr) String() string {
	return "(" + p.Expr.String() + ")"
}

// --- Aggregate / Window function types ---
// All implement Expr and have an OverClause field.

func appendOver(b *strings.Builder, oc *OverClause) {
	if oc != nil {
		b.WriteString(" ")
		b.WriteString(oc.String())
	}
}

func formatDistinctAgg(name string, arg Expr, distinct bool, oc *OverClause) string {
	var b strings.Builder

	b.WriteString(name)
	b.WriteString("(")

	if distinct {
		b.WriteString("DISTINCT ")
	}

	b.WriteString(arg.String())
	b.WriteString(")")
	appendOver(&b, oc)

	return b.String()
}

func formatSimpleAgg(name string, arg Expr, oc *OverClause) string {
	var b strings.Builder

	b.WriteString(name)
	b.WriteString("(")
	b.WriteString(arg.String())
	b.WriteString(")")
	appendOver(&b, oc)

	return b.String()
}

// Count represents COUNT(expr) or COUNT(DISTINCT expr).
type Count struct {
	Args       []Expr
	Distinct   bool
	OverClause *OverClause
}

// String returns Count's SQL text.
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

	b.WriteString(strings.Join(args, ", "))
	b.WriteString(")")
	appendOver(&b, c.OverClause)

	return b.String()
}

// CountStar represents COUNT(*).
type CountStar struct {
	OverClause *OverClause
}

// String returns CountStar's SQL text.
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

// String returns Sum's SQL text.
func (s *Sum) String() string { return formatDistinctAgg("SUM", s.Arg, s.Distinct, s.OverClause) }

// Avg represents AVG([DISTINCT] expr).
type Avg struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

// String returns Avg's SQL text.
func (a *Avg) String() string { return formatDistinctAgg("AVG", a.Arg, a.Distinct, a.OverClause) }

// Min represents MIN([DISTINCT] expr).
type Min struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

// String returns Min's SQL text.
func (m *Min) String() string { return formatDistinctAgg("MIN", m.Arg, m.Distinct, m.OverClause) }

// Max represents MAX([DISTINCT] expr).
type Max struct {
	Arg        Expr
	Distinct   bool
	OverClause *OverClause
}

// String returns Max's SQL text.
func (m *Max) String() string { return formatDistinctAgg("MAX", m.Arg, m.Distinct, m.OverClause) }

// BitAnd represents BIT_AND(expr).
type BitAnd struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns BitAnd's SQL text.
func (ba *BitAnd) String() string { return formatSimpleAgg("BIT_AND", ba.Arg, ba.OverClause) }

// BitOr represents BIT_OR(expr).
type BitOr struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns BitOr's SQL text.
func (bo *BitOr) String() string { return formatSimpleAgg("BIT_OR", bo.Arg, bo.OverClause) }

// BitXor represents BIT_XOR(expr).
type BitXor struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns BitXor's SQL text.
func (bx *BitXor) String() string { return formatSimpleAgg("BIT_XOR", bx.Arg, bx.OverClause) }

// Std represents STD(expr).
type Std struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns Std's SQL text.
func (s *Std) String() string { return formatSimpleAgg("STD", s.Arg, s.OverClause) }

// StdDev represents STDDEV(expr).
type StdDev struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns StdDev's SQL text.
func (s *StdDev) String() string { return formatSimpleAgg("STDDEV", s.Arg, s.OverClause) }

// StdPop represents STDDEV_POP(expr).
type StdPop struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns StdPop's SQL text.
func (s *StdPop) String() string { return formatSimpleAgg("STDDEV_POP", s.Arg, s.OverClause) }

// StdSamp represents STDDEV_SAMP(expr).
type StdSamp struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns StdSamp's SQL text.
func (s *StdSamp) String() string { return formatSimpleAgg("STDDEV_SAMP", s.Arg, s.OverClause) }

// Variance represents VARIANCE(expr).
type Variance struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns Variance's SQL text.
func (v *Variance) String() string { return formatSimpleAgg("VARIANCE", v.Arg, v.OverClause) }

// VarPop represents VAR_POP(expr).
type VarPop struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns VarPop's SQL text.
func (v *VarPop) String() string { return formatSimpleAgg("VAR_POP", v.Arg, v.OverClause) }

// VarSamp represents VAR_SAMP(expr).
type VarSamp struct {
	Arg        Expr
	OverClause *OverClause
}

// String returns VarSamp's SQL text.
func (v *VarSamp) String() string { return formatSimpleAgg("VAR_SAMP", v.Arg, v.OverClause) }

// ArgumentLessWindowExpr represents window functions with no arguments (e.g., ROW_NUMBER()).
type ArgumentLessWindowExpr struct {
	Type       ArgumentLessWindowExprType
	OverClause *OverClause
}

// String returns ArgumentLessWindowExpr's SQL text.
func (a *ArgumentLessWindowExpr) String() string {
	var b strings.Builder

	b.WriteString(a.Type.String())
	b.WriteString("()")
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

// String returns FirstOrLastValueExpr's SQL text.
func (f *FirstOrLastValueExpr) String() string {
	var b strings.Builder

	b.WriteString(f.Type.String())
	b.WriteString("(")
	b.WriteString(f.Expr.String())
	b.WriteString(")")

	if f.NullTreatmentClause != nil {
		b.WriteString(" ")
		b.WriteString(f.NullTreatmentClause.String())
	}

	appendOver(&b, f.OverClause)

	return b.String()
}

// NtileExpr represents NTILE(n).
type NtileExpr struct {
	N          Expr
	OverClause *OverClause
}

// String returns NtileExpr's SQL text.
func (n *NtileExpr) String() string {
	var b strings.Builder

	b.WriteString("NTILE(")
	b.WriteString(n.N.String())
	b.WriteString(")")
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

// String returns NTHValueExpr's SQL text.
func (n *NTHValueExpr) String() string {
	var b strings.Builder

	b.WriteString("NTH_VALUE(")
	b.WriteString(n.Expr.String())
	b.WriteString(", ")
	b.WriteString(n.N.String())
	b.WriteString(")")

	if n.FromFirstLastClause != nil {
		b.WriteString(" ")
		b.WriteString(n.FromFirstLastClause.String())
	}

	if n.NullTreatmentClause != nil {
		b.WriteString(" ")
		b.WriteString(n.NullTreatmentClause.String())
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

// String returns LagLeadExpr's SQL text.
func (l *LagLeadExpr) String() string {
	var b strings.Builder

	args := []string{l.Expr.String()}
	if l.N != nil {
		args = append(args, l.N.String())
	}

	if l.Default != nil {
		args = append(args, l.Default.String())
	}

	b.WriteString(l.Type.String())
	b.WriteString("(")
	b.WriteString(strings.Join(args, ", "))
	b.WriteString(")")

	if l.NullTreatmentClause != nil {
		b.WriteString(" ")
		b.WriteString(l.NullTreatmentClause.String())
	}

	appendOver(&b, l.OverClause)

	return b.String()
}

// JSONArrayAgg represents JSON_ARRAYAGG(expr).
type JSONArrayAgg struct {
	Expr       Expr
	OverClause *OverClause
}

// String returns JSONArrayAgg's SQL text.
func (j *JSONArrayAgg) String() string {
	return formatSimpleAgg("JSON_ARRAYAGG", j.Expr, j.OverClause)
}

// JSONObjectAgg represents JSON_OBJECTAGG(key, value).
type JSONObjectAgg struct {
	Key        Expr
	Value      Expr
	OverClause *OverClause
}

// String returns JSONObjectAgg's SQL text.
func (j *JSONObjectAgg) String() string {
	var b strings.Builder

	b.WriteString("JSON_OBJECTAGG(")
	b.WriteString(j.Key.String())
	b.WriteString(", ")
	b.WriteString(j.Value.String())
	b.WriteString(")")
	appendOver(&b, j.OverClause)

	return b.String()
}
