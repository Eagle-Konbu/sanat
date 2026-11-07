package pgast

import "strings"

// This file holds the SQL-standard function forms whose arguments are
// separated by keywords rather than commas. Their plain comma-separated
// spellings (substring(s, 1, 2), trim(s)) parse as ordinary FuncExprs.

// ExtractExpr represents EXTRACT(field FROM source). Field is SQL text: an
// identifier such as year, or a string constant such as 'epoch'.
type ExtractExpr struct {
	Field  string
	Source Expr
}

// String returns ExtractExpr's SQL text.
func (e *ExtractExpr) String() string {
	return "EXTRACT(" + e.Field + " FROM " + e.Source.String() + ")"
}

// SubstringExpr represents SUBSTRING(s FROM start FOR count), where either
// bound may be omitted, or the regular-expression form
// SUBSTRING(s SIMILAR pattern ESCAPE escape) when Similar is set.
type SubstringExpr struct {
	Expr    Expr
	From    Expr
	For     Expr
	Similar Expr
	Escape  Expr
}

// String returns SubstringExpr's SQL text.
func (s *SubstringExpr) String() string {
	var b strings.Builder

	b.WriteString("SUBSTRING(")
	b.WriteString(s.Expr.String())

	if s.Similar != nil {
		b.WriteString(" SIMILAR " + s.Similar.String() + " ESCAPE " + s.Escape.String() + ")")

		return b.String()
	}

	if s.From != nil {
		b.WriteString(" FROM " + s.From.String())
	}

	if s.For != nil {
		b.WriteString(" FOR " + s.For.String())
	}

	b.WriteString(")")

	return b.String()
}

// PositionExpr represents POSITION(substring IN source).
type PositionExpr struct {
	Substring Expr
	Source    Expr
}

// String returns PositionExpr's SQL text.
func (p *PositionExpr) String() string {
	return "POSITION(" + p.Substring.String() + " IN " + p.Source.String() + ")"
}

// TrimSide is the BOTH, LEADING, or TRAILING of a TrimExpr.
type TrimSide int8

const (
	TrimDefault TrimSide = iota
	TrimBoth
	TrimLeading
	TrimTrailing
)

// String returns TrimSide's SQL text, or "" when no side is given.
func (s TrimSide) String() string {
	switch s {
	case TrimBoth:
		return "BOTH"
	case TrimLeading:
		return "LEADING"
	case TrimTrailing:
		return "TRAILING"
	default:
		return ""
	}
}

// TrimExpr represents TRIM([side] [characters] FROM source, ...) or
// TRIM(side source, ...). From records whether FROM was written, since
// "TRIM(BOTH FROM s)" and "TRIM(BOTH s)" are both valid; it is always set
// when Characters is.
type TrimExpr struct {
	Side       TrimSide
	Characters Expr
	From       bool
	Args       []Expr
}

// String returns TrimExpr's SQL text.
func (t *TrimExpr) String() string {
	var parts []string

	if side := t.Side.String(); side != "" {
		parts = append(parts, side)
	}

	if t.Characters != nil {
		parts = append(parts, t.Characters.String())
	}

	if t.From {
		parts = append(parts, "FROM")
	}

	parts = append(parts, exprListString(t.Args))

	return "TRIM(" + strings.Join(parts, " ") + ")"
}

// OverlayExpr represents OVERLAY(s PLACING replacement FROM start [FOR count]).
type OverlayExpr struct {
	Expr    Expr
	Placing Expr
	From    Expr
	For     Expr
}

// String returns OverlayExpr's SQL text.
func (o *OverlayExpr) String() string {
	s := "OVERLAY(" + o.Expr.String() + " PLACING " + o.Placing.String() + " FROM " + o.From.String()
	if o.For != nil {
		s += " FOR " + o.For.String()
	}

	return s + ")"
}

// OverlapsExpr represents "(start, end) OVERLAPS (start, end)". Each side is
// a two-element ValTuple or RowExpr.
type OverlapsExpr struct {
	Left  Expr
	Right Expr
}

// String returns OverlapsExpr's SQL text.
func (o *OverlapsExpr) String() string {
	return o.Left.String() + " OVERLAPS " + o.Right.String()
}
