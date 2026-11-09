package pgast

import (
	"fmt"
	"strings"
)

// ComparisonOperator represents a comparison operator.
type ComparisonOperator int8

// Comparison operators.
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
	SimilarOp
	NotSimilarOp
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
	SimilarOp:      "SIMILAR TO",
	NotSimilarOp:   "NOT SIMILAR TO",
}

// String returns ComparisonOperator's SQL operator text.
func (op ComparisonOperator) String() string {
	if op >= 0 && int(op) < len(comparisonOpStrings) {
		return comparisonOpStrings[op]
	}

	return "="
}

// ComparisonExpr represents a comparison expression (e.g., a = b, a IN (...),
// a SIMILAR TO b).
type ComparisonExpr struct {
	Operator ComparisonOperator
	Left     Expr
	Right    Expr
	Escape   Expr // [NOT] LIKE / SIMILAR TO ... ESCAPE expr; nil for every other operator
}

// String returns ComparisonExpr's SQL text.
func (c *ComparisonExpr) String() string {
	return fmt.Sprintf("%s %s %s", c.Left.String(), c.Operator.String(), c.Right.String()) + escapeString(c.Escape)
}

func escapeString(escape Expr) string {
	if escape == nil {
		return ""
	}

	return " ESCAPE " + escape.String()
}

// ILikeExpr represents a [NOT] ILIKE expression — PostgreSQL's
// case-insensitive LIKE, with no MySQL equivalent.
type ILikeExpr struct {
	Not    bool
	Left   Expr
	Right  Expr
	Escape Expr
}

// String returns ILikeExpr's SQL text.
func (i *ILikeExpr) String() string {
	op := "ILIKE"
	if i.Not {
		op = "NOT ILIKE"
	}

	return fmt.Sprintf("%s %s %s", i.Left.String(), op, i.Right.String()) + escapeString(i.Escape)
}

// RangeCond represents a [NOT] BETWEEN [SYMMETRIC] ... AND ... expression.
// An explicit ASYMMETRIC is the default and is not modeled.
type RangeCond struct {
	Not       bool
	Symmetric bool
	Left      Expr
	From      Expr
	To        Expr
}

// String returns RangeCond's SQL text.
func (r *RangeCond) String() string {
	op := "BETWEEN"
	if r.Symmetric {
		op = "BETWEEN SYMMETRIC"
	}

	if r.Not {
		op = "NOT " + op
	}

	return fmt.Sprintf("%s %s %s AND %s", r.Left.String(), op, r.From.String(), r.To.String())
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

// IsNormalizedExpr represents an IS [NOT] [form] NORMALIZED test of a
// string's Unicode normalization. Form is NFC, NFD, NFKC, or NFKD, or ""
// when omitted.
type IsNormalizedExpr struct {
	Not  bool
	Expr Expr
	Form string
}

// String returns IsNormalizedExpr's SQL text.
func (i *IsNormalizedExpr) String() string {
	var b strings.Builder

	b.WriteString(i.Expr.String())
	b.WriteString(" IS ")

	if i.Not {
		b.WriteString("NOT ")
	}

	if i.Form != "" {
		b.WriteString(i.Form)
		b.WriteString(" ")
	}

	b.WriteString("NORMALIZED")

	return b.String()
}

// TruthValue represents the operand of an IS [NOT] TRUE / FALSE / UNKNOWN test.
type TruthValue int8

// Truth values tested by IS [NOT] TRUE / FALSE / UNKNOWN.
const (
	TruthTrue TruthValue = iota
	TruthFalse
	TruthUnknown
)

// String returns TruthValue's SQL text.
func (v TruthValue) String() string {
	switch v {
	case TruthFalse:
		return "FALSE"
	case TruthUnknown:
		return "UNKNOWN"
	default:
		return "TRUE"
	}
}

// IsTruthExpr represents an IS [NOT] TRUE / FALSE / UNKNOWN expression.
type IsTruthExpr struct {
	Not   bool
	Expr  Expr
	Value TruthValue
}

// String returns IsTruthExpr's SQL text.
func (i *IsTruthExpr) String() string {
	if i.Not {
		return i.Expr.String() + " IS NOT " + i.Value.String()
	}

	return i.Expr.String() + " IS " + i.Value.String()
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

// RowExpr represents an explicit ROW(expr, ...) constructor. The implicit
// (a, b) form is a ValTuple.
type RowExpr struct {
	Exprs []Expr
}

// String returns RowExpr's SQL text.
func (r *RowExpr) String() string {
	return "ROW(" + exprListString(r.Exprs) + ")"
}

// ArithmeticOperator represents a binary arithmetic operator.
type ArithmeticOperator int8

// Binary arithmetic operators.
const (
	PlusOp ArithmeticOperator = iota
	MinusOp
	MultOp
	DivOp
	ModOp
	ExpOp
)

var arithmeticOpStrings = [...]string{
	PlusOp:  "+",
	MinusOp: "-",
	MultOp:  "*",
	DivOp:   "/",
	ModOp:   "%",
	ExpOp:   "^",
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

// Unary operators.
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

// String returns UnaryExpr's SQL text. The operator is separated from an
// operand that starts with an operator character, which it would otherwise
// fuse with: "- -a" would read as a "--" comment, and "- ~ a" as a "-~"
// operator.
func (u *UnaryExpr) String() string {
	operand := u.Expr.String()
	if operand != "" && strings.ContainsRune("~!@#^&|`?+-*/%<>=", rune(operand[0])) {
		return u.Operator.String() + " " + operand
	}

	return u.Operator.String() + operand
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
//
// Val is the normalized SQL text String renders: pgparser re-encodes every
// string constant ('...', E'...', $tag$...$tag$, and adjacent constants
// joined across a newline) as a plain '...'. Raw is the literal exactly as
// written in the source, spanning every part of a joined constant along with
// the whitespace between them, for a renderer that must reproduce the user's
// spelling. Raw is display-only and empty for a literal that was not parsed
// from source text.
type Literal struct {
	Val string
	Raw string
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
	Qualifier   TableIdent
	Name        ColIdent
	Star        bool // COUNT(*)
	Distinct    bool
	Args        []Expr
	Variadic    bool        // the last argument is passed with VARIADIC
	OrderBy     OrderBy     // an aggregate's ORDER BY inside the parentheses
	WithinGroup OrderBy     // WITHIN GROUP (ORDER BY ...)
	Filter      *Where      // FILTER (WHERE ...)
	Over        *WindowSpec // OVER (...) / OVER window_name
	OverRef     bool        // OVER window_name, unparenthesized; the name is Over.Name
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
	b.WriteString(f.argsString())
	b.WriteString(")")

	if len(f.WithinGroup) > 0 {
		b.WriteString(" WITHIN GROUP (")
		b.WriteString(f.WithinGroup.String())
		b.WriteString(")")
	}

	if f.Filter != nil && f.Filter.Expr != nil {
		b.WriteString(" FILTER (WHERE ")
		b.WriteString(f.Filter.String())
		b.WriteString(")")
	}

	switch {
	case f.Over != nil && f.OverRef:
		b.WriteString(" OVER ")
		b.WriteString(f.Over.Name.String())
	case f.Over != nil:
		b.WriteString(" OVER (")
		b.WriteString(f.Over.String())
		b.WriteString(")")
	}

	return b.String()
}

// argsString renders the content of a call's parentheses: "*", or the
// arguments with their DISTINCT, VARIADIC, and ORDER BY modifiers.
func (f *FuncExpr) argsString() string {
	if f.Star {
		return "*"
	}

	s := exprListString(f.Args)

	if f.Variadic && len(f.Args) > 0 {
		last := len(f.Args) - 1

		s = "VARIADIC " + f.Args[last].String()
		if last > 0 {
			s = exprListString(f.Args[:last]) + ", " + s
		}
	}

	if f.Distinct {
		s = "DISTINCT " + s
	}

	if len(f.OrderBy) > 0 {
		s += " " + f.OrderBy.String()
	}

	return s
}

// FieldSelectExpr represents selecting a field of a composite value,
// "(expr).field", or all of its fields, "(expr).*" or "t.*" when Star is
// set. It also holds the parts of a dotted column reference beyond the three
// a ColName holds.
type FieldSelectExpr struct {
	Expr  Expr
	Field ColIdent
	Star  bool
}

// String returns FieldSelectExpr's SQL text.
func (f *FieldSelectExpr) String() string {
	if f.Star {
		return f.Expr.String() + ".*"
	}

	return f.Expr.String() + "." + f.Field.String()
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
	Type     *TypeName
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
//
// Nested marks a sub-array written as bare brackets inside a
// multidimensional array, such as each "[1, 2]" in ARRAY[[1, 2], [3, 4]];
// it renders without the ARRAY keyword.
type ArrayExpr struct {
	Elems  []Expr
	Nested bool
}

// String returns ArrayExpr's SQL text.
func (a *ArrayExpr) String() string {
	if a.Nested {
		return "[" + exprListString(a.Elems) + "]"
	}

	return "ARRAY[" + exprListString(a.Elems) + "]"
}

// ArraySubquery represents an ARRAY(subquery) constructor, which collects
// the subquery's single-column result into an array.
type ArraySubquery struct {
	Select Statement
}

// String returns ArraySubquery's SQL text.
func (a *ArraySubquery) String() string {
	return "ARRAY(" + a.Select.String() + ")"
}

// ArraySubscriptExpr represents array subscripting (a[1]) or slicing (a[1:3]).
// Slice is set for the slice form, where either bound may be omitted (a[1:],
// a[:3], a[:]); From and To are nil for omitted bounds.
type ArraySubscriptExpr struct {
	Expr  Expr
	From  Expr
	To    Expr
	Slice bool
}

// String returns ArraySubscriptExpr's SQL text.
func (a *ArraySubscriptExpr) String() string {
	var b strings.Builder

	b.WriteString(a.Expr.String())
	b.WriteString("[")

	if a.From != nil {
		b.WriteString(a.From.String())
	}

	if a.Slice {
		b.WriteString(":")
	}

	if a.To != nil {
		b.WriteString(a.To.String())
	}

	b.WriteString("]")

	return b.String()
}

// Operator names an operator applied by an OperatorExpr. Symbol is its text
// (such as "||", "@>", or "<->"). Explicit marks the OPERATOR(schema.op)
// spelling, which may also apply one of SQL's fixed operators, such as +, at
// the generic operator precedence level.
type Operator struct {
	Schema   TableIdent
	Symbol   string
	Explicit bool
}

// String returns Operator's SQL text.
func (o Operator) String() string {
	if !o.Explicit {
		return o.Symbol
	}

	if o.Schema.IsEmpty() {
		return "OPERATOR(" + o.Symbol + ")"
	}

	return "OPERATOR(" + o.Schema.String() + "." + o.Symbol + ")"
}

// OperatorExpr represents an operator with no dedicated node: any operator
// outside SQL's fixed comparison and arithmetic set, such as string
// concatenation (||), pattern matching (~), JSON (->, @>), geometric (<->),
// or a user-defined operator. Left is nil for a prefix operator. All of them
// share one precedence level, between the predicates and + and -.
type OperatorExpr struct {
	Left     Expr
	Operator Operator
	Right    Expr
}

// String returns OperatorExpr's SQL text. A prefix operator is separated from
// its operand by a space, since "@ -1" and "@-1" lex differently.
func (o *OperatorExpr) String() string {
	if o.Left == nil {
		return o.Operator.String() + " " + o.Right.String()
	}

	return o.Left.String() + " " + o.Operator.String() + " " + o.Right.String()
}

// NamedArgExpr represents a named function argument, "name => value", or the
// older "name := value" spelling when ColonEquals is set.
type NamedArgExpr struct {
	Name        ColIdent
	Value       Expr
	ColonEquals bool
}

// String returns NamedArgExpr's SQL text.
func (n *NamedArgExpr) String() string {
	op := " => "
	if n.ColonEquals {
		op = " := "
	}

	return n.Name.String() + op + n.Value.String()
}

// Quantifier is the ANY, SOME, or ALL of a QuantifiedExpr. SOME is a synonym
// for ANY, kept apart only so the expression round-trips as written.
type Quantifier int8

// Quantifiers of a quantified comparison.
const (
	QuantifierAny Quantifier = iota
	QuantifierSome
	QuantifierAll
)

// String returns Quantifier's SQL text.
func (q Quantifier) String() string {
	switch q {
	case QuantifierSome:
		return "SOME"
	case QuantifierAll:
		return "ALL"
	default:
		return "ANY"
	}
}

// QuantifiedExpr represents an operator applied to each element of an array
// or each row of a subquery: "left op ANY | SOME | ALL (right)". Operator
// is a comparison or generic operator, or [NOT] LIKE / [NOT] ILIKE. Right is
// a *Subquery or an array-valued expression.
type QuantifiedExpr struct {
	Left       Expr
	Operator   Operator
	Quantifier Quantifier
	Right      Expr
}

// String returns QuantifiedExpr's SQL text.
func (q *QuantifiedExpr) String() string {
	right := q.Right.String()
	if _, ok := q.Right.(*Subquery); !ok {
		right = "(" + right + ")"
	}

	return q.Left.String() + " " + q.Operator.String() + " " + q.Quantifier.String() + " " + right
}

// CollateExpr represents "expr COLLATE collation".
type CollateExpr struct {
	Expr      Expr
	Collation AnyName
}

// String returns CollateExpr's SQL text.
func (c *CollateExpr) String() string {
	return c.Expr.String() + " COLLATE " + c.Collation.String()
}

// AtTimeZoneExpr represents "expr AT TIME ZONE zone".
type AtTimeZoneExpr struct {
	Expr Expr
	Zone Expr
}

// String returns AtTimeZoneExpr's SQL text.
func (a *AtTimeZoneExpr) String() string {
	return a.Expr.String() + " AT TIME ZONE " + a.Zone.String()
}
