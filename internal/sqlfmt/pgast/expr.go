package pgast

import (
	"fmt"
	"strings"
)

// ComparisonOperator represents a comparison operator.
type ComparisonOperator int8

const (
	EqualOp ComparisonOperator = iota
	LessThanOp
	GreaterThanOp
	LessEqualOp
	GreaterEqualOp
	NotEqualOp
	LikeOp
	NotLikeOp
	InOp
	NotInOp
)

var comparisonOpStrings = [...]string{
	EqualOp:        "=",
	LessThanOp:     "<",
	GreaterThanOp:  ">",
	LessEqualOp:    "<=",
	GreaterEqualOp: ">=",
	NotEqualOp:     "<>",
	LikeOp:         "LIKE",
	NotLikeOp:      "NOT LIKE",
	InOp:           "IN",
	NotInOp:        "NOT IN",
}

// String returns ComparisonOperator's SQL operator text.
func (op ComparisonOperator) String() string {
	if op >= 0 && int(op) < len(comparisonOpStrings) {
		return comparisonOpStrings[op]
	}

	return "="
}

// ComparisonExpr represents a comparison expression (e.g., a = b, a IN (...)).
type ComparisonExpr struct {
	Operator ComparisonOperator
	Left     Expr
	Right    Expr
}

// String returns ComparisonExpr's SQL text.
func (c *ComparisonExpr) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Operator.String(), c.Right.String())
}

// ILikeExpr represents a [NOT] ILIKE expression — PostgreSQL's
// case-insensitive LIKE, with no MySQL equivalent.
type ILikeExpr struct {
	Not   bool
	Left  Expr
	Right Expr
}

// String returns ILikeExpr's SQL text.
func (i *ILikeExpr) String() string {
	op := "ILIKE"
	if i.Not {
		op = "NOT ILIKE"
	}

	return fmt.Sprintf("%s %s %s", i.Left.String(), op, i.Right.String())
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

// IsExpr represents an IS [NOT] NULL expression. PostgreSQL's postfix
// ISNULL/NOTNULL synonyms parse into this same node — pgparser canonicalizes
// them, the same way sqlast canonicalizes keyword casing elsewhere.
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

// IsDistinctFromExpr represents an IS [NOT] DISTINCT FROM expression.
type IsDistinctFromExpr struct {
	Not   bool
	Left  Expr
	Right Expr
}

// String returns IsDistinctFromExpr's SQL text.
func (i *IsDistinctFromExpr) String() string {
	op := "IS DISTINCT FROM"
	if i.Not {
		op = "IS NOT DISTINCT FROM"
	}

	return fmt.Sprintf("%s %s %s", i.Left.String(), op, i.Right.String())
}

// ValTuple represents a parenthesized list of expressions, e.g. the value
// list on the right-hand side of an IN predicate.
type ValTuple []Expr

// String returns ValTuple's SQL text.
func (v ValTuple) String() string {
	return "(" + exprListString(v) + ")"
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

// String returns ArithmeticOperator's SQL operator text.
func (o ArithmeticOperator) String() string {
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
	return fmt.Sprintf("%s %s %s", a.Left.String(), a.Operator.String(), a.Right.String())
}

// UnaryOperator represents a unary + or - operator.
type UnaryOperator int8

const (
	UPlusOp UnaryOperator = iota
	UMinusOp
)

// String returns UnaryOperator's SQL operator text.
func (u UnaryOperator) String() string {
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
	return u.Operator.String() + u.Expr.String()
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

// ExistsExpr represents an EXISTS (subquery) expression.
type ExistsExpr struct {
	Subquery *Subquery
}

// String returns ExistsExpr's SQL text.
func (e *ExistsExpr) String() string {
	return "EXISTS (" + e.Subquery.Select.String() + ")"
}

// Subquery represents a parenthesized subquery expression.
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

// Literal represents a literal value (number, string, boolean, NULL, ...)
// rendered as-is.
type Literal struct {
	Val string
}

// String returns Literal's SQL text.
func (l *Literal) String() string {
	return l.Val
}

// FuncExpr represents a function call expression, covering plain calls,
// aggregates (including COUNT(*) and DISTINCT), and window functions
// (via Over). PostgreSQL has no per-builtin-aggregate grammar the way
// sqlast models one type per MySQL aggregate function — a single generic
// shape covers every case the formatter needs to round-trip.
type FuncExpr struct {
	Qualifier TableIdent
	Name      ColIdent
	Star      bool // COUNT(*)
	Distinct  bool
	Args      []Expr
	Filter    *Where      // FILTER (WHERE ...)
	Over      *WindowSpec // OVER (...) / OVER window_name
}

// String returns FuncExpr's SQL text.
func (f *FuncExpr) String() string {
	var b strings.Builder

	if !f.Qualifier.IsEmpty() {
		b.WriteString(f.Qualifier.String())
		b.WriteString(".")
	}

	b.WriteString(f.Name.String())
	b.WriteString("(")

	switch {
	case f.Star:
		b.WriteString("*")
	default:
		if f.Distinct {
			b.WriteString("DISTINCT ")
		}

		b.WriteString(exprListString(f.Args))
	}

	b.WriteString(")")

	if f.Filter != nil && f.Filter.Expr != nil {
		b.WriteString(" FILTER (WHERE ")
		b.WriteString(f.Filter.String())
		b.WriteString(")")
	}

	if f.Over != nil {
		b.WriteString(" OVER ")
		b.WriteString(f.Over.overString())
	}

	return b.String()
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr Expr
}

// String returns ParenExpr's SQL text.
func (p *ParenExpr) String() string {
	return "(" + p.Expr.String() + ")"
}

// CastExpr represents a "::" cast or an equivalent CAST(expr AS type).
// Explicit records which source syntax was used, since sanat formats
// existing SQL rather than normalizing it to one canonical spelling.
type CastExpr struct {
	Expr     Expr
	Type     ColIdent
	Explicit bool // true for CAST(expr AS type), false for expr::type
}

// String returns CastExpr's SQL text.
func (c *CastExpr) String() string {
	if c.Explicit {
		return "CAST(" + c.Expr.String() + " AS " + c.Type.String() + ")"
	}

	return c.Expr.String() + "::" + c.Type.String()
}

// ArrayExpr represents an ARRAY[expr, ...] literal. The '{1,2,3}'
// array-string-literal form is not modeled separately — syntactically it is
// just a string literal (see Literal) until PostgreSQL interprets it against
// an array type, so there is nothing structural for the formatter to print.
type ArrayExpr struct {
	Elems []Expr
}

// String returns ArrayExpr's SQL text.
func (a *ArrayExpr) String() string {
	return "ARRAY[" + exprListString(a.Elems) + "]"
}

// ArraySubscriptExpr represents array subscripting (a[1]) or slicing (a[1:3]).
// To is nil for a plain subscript.
type ArraySubscriptExpr struct {
	Expr Expr
	From Expr
	To   Expr
}

// String returns ArraySubscriptExpr's SQL text.
func (a *ArraySubscriptExpr) String() string {
	var b strings.Builder

	b.WriteString(a.Expr.String())
	b.WriteString("[")

	if a.From != nil {
		b.WriteString(a.From.String())
	}

	if a.To != nil {
		b.WriteString(":")
		b.WriteString(a.To.String())
	}

	b.WriteString("]")

	return b.String()
}

// JSONOp represents a JSON/JSONB operator.
type JSONOp string

const (
	JSONArrow         JSONOp = "->"
	JSONArrowText     JSONOp = "->>"
	JSONHashArrow     JSONOp = "#>"
	JSONHashArrowText JSONOp = "#>>"
	JSONContains      JSONOp = "@>"
	JSONContainedBy   JSONOp = "<@"
	JSONExists        JSONOp = "?"
	JSONExistsAny     JSONOp = "?|"
	JSONExistsAll     JSONOp = "?&"
)

// String returns JSONOp's SQL operator text.
func (j JSONOp) String() string { return string(j) }

// JSONOpExpr represents a binary JSON/JSONB operator expression
// (->, ->>, #>, #>>, @>, <@, ?, ?|, ?&).
type JSONOpExpr struct {
	Left  Expr
	Op    JSONOp
	Right Expr
}

// String returns JSONOpExpr's SQL text.
func (j *JSONOpExpr) String() string {
	return fmt.Sprintf("%s %s %s", j.Left.String(), j.Op.String(), j.Right.String())
}
