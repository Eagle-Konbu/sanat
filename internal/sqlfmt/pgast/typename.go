package pgast

import "strings"

// TimeZoneSpec represents the optional WITH / WITHOUT TIME ZONE suffix of a
// time or timestamp type.
type TimeZoneSpec int8

const (
	TimeZoneUnspecified TimeZoneSpec = iota
	WithTimeZone
	WithoutTimeZone
)

// String returns TimeZoneSpec's SQL text, or "" when no suffix is given.
func (z TimeZoneSpec) String() string {
	switch z {
	case WithTimeZone:
		return "with time zone"
	case WithoutTimeZone:
		return "without time zone"
	default:
		return ""
	}
}

// TypeName represents a type name, as used by a cast or a typed literal.
//
// It is structured, rather than an opaque string, so the formatter can lay
// out its parts (and so later grammar, such as column definition lists, can
// reuse it). Its words render in lower case, the way PostgreSQL's
// documentation spells built-in type names.
type TypeName struct {
	// Names holds the dotted parts of the name, as SQL text: one part for an
	// unqualified name, more for a schema-qualified one. A built-in
	// multi-word name, such as "double precision" or "character varying", is
	// a single part.
	Names []ColIdent
	// Modifiers holds the parenthesized type modifiers, as in
	// numeric(10, 2) or geometry(point, 4326). PostgreSQL only accepts
	// constants and plain identifiers here, so each is a *Literal or a
	// *ColName.
	Modifiers []Expr
	// TimeZone is the WITH / WITHOUT TIME ZONE suffix of time and timestamp.
	TimeZone TimeZoneSpec
	// IntervalFields is an interval type's field restriction, such as
	// "day to second" or "second(3)", as SQL text.
	IntervalFields string
	// ArrayBounds holds one entry per array dimension: "" for [] or the
	// bound's digits for [n].
	ArrayBounds []string
}

// String returns TypeName's SQL text.
func (t *TypeName) String() string {
	var b strings.Builder

	b.WriteString(t.baseString())

	if t.IntervalFields != "" {
		b.WriteString(" ")
		b.WriteString(t.IntervalFields)
	}

	for _, bound := range t.ArrayBounds {
		b.WriteString("[")
		b.WriteString(bound)
		b.WriteString("]")
	}

	return b.String()
}

// baseString renders the type name without its interval fields or array
// bounds: a typed interval literal puts its fields after the string.
func (t *TypeName) baseString() string {
	names := make([]string, len(t.Names))
	for i, n := range t.Names {
		names[i] = n.String()
	}

	var b strings.Builder

	b.WriteString(strings.Join(names, "."))

	if len(t.Modifiers) > 0 {
		b.WriteString("(")
		b.WriteString(exprListString(t.Modifiers))
		b.WriteString(")")
	}

	if tz := t.TimeZone.String(); tz != "" {
		b.WriteString(" ")
		b.WriteString(tz)
	}

	return b.String()
}

// TypedLiteral represents a string constant preceded by its type, such as
// date '2020-01-01', double precision '1.5', or interval '1' day. Value is
// the constant as SQL text ('...').
type TypedLiteral struct {
	Type  *TypeName
	Value string
}

// String returns TypedLiteral's SQL text. An interval's field restriction
// follows the string, as PostgreSQL's grammar requires.
func (l *TypedLiteral) String() string {
	s := l.Type.baseString() + " " + l.Value
	if l.Type.IntervalFields != "" {
		s += " " + l.Type.IntervalFields
	}

	return s
}
