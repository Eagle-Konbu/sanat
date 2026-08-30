package pgast

import "strings"

// Where represents a WHERE, HAVING, or FILTER (WHERE ...) condition.
type Where struct {
	Expr Expr
}

// String returns Where's SQL text.
func (w *Where) String() string {
	if w == nil || w.Expr == nil {
		return ""
	}

	return w.Expr.String()
}

// DistinctOn represents a DISTINCT ON (expr, ...) clause.
type DistinctOn struct {
	Exprs []Expr
}

// String returns DistinctOn's SQL text.
func (d *DistinctOn) String() string {
	if d == nil || len(d.Exprs) == 0 {
		return ""
	}

	strs := make([]string, len(d.Exprs))
	for i, e := range d.Exprs {
		strs[i] = e.String()
	}

	return "DISTINCT ON (" + strings.Join(strs, ", ") + ")"
}

// GroupBy represents a GROUP BY clause. Each element is either a plain Expr
// or one of the grouping_element wrapper types below (*GroupingSets, *Cube,
// *Rollup), which implement Expr so they can sit in the same slice.
type GroupBy struct {
	Elements []Expr
}

// String returns GroupBy's SQL text.
func (g *GroupBy) String() string {
	if g == nil || len(g.Elements) == 0 {
		return ""
	}

	strs := make([]string, len(g.Elements))
	for i, e := range g.Elements {
		strs[i] = e.String()
	}

	return "GROUP BY " + strings.Join(strs, ", ")
}

// GroupingSets represents a GROUPING SETS (...) grouping element. Each entry
// is one set, itself a parenthesized list of expressions (an empty set
// renders as "()", matching grand-total rows).
type GroupingSets struct {
	Sets [][]Expr
}

// String returns GroupingSets's SQL text.
func (g *GroupingSets) String() string {
	sets := make([]string, len(g.Sets))
	for i, set := range g.Sets {
		sets[i] = "(" + exprListString(set) + ")"
	}

	return "GROUPING SETS (" + strings.Join(sets, ", ") + ")"
}

// Cube represents a CUBE (expr, ...) grouping element.
type Cube struct {
	Exprs []Expr
}

// String returns Cube's SQL text.
func (c *Cube) String() string {
	return "CUBE (" + exprListString(c.Exprs) + ")"
}

// Rollup represents a ROLLUP (expr, ...) grouping element.
type Rollup struct {
	Exprs []Expr
}

// String returns Rollup's SQL text.
func (r *Rollup) String() string {
	return "ROLLUP (" + exprListString(r.Exprs) + ")"
}

func exprListString(exprs []Expr) string {
	strs := make([]string, len(exprs))
	for i, e := range exprs {
		strs[i] = e.String()
	}

	return strings.Join(strs, ", ")
}

// NullsOrder represents an explicit NULLS FIRST/LAST modifier on an ORDER BY item.
type NullsOrder int8

const (
	NullsDefault NullsOrder = iota
	NullsFirst
	NullsLast
)

// String returns NullsOrder's SQL text.
func (n NullsOrder) String() string {
	switch n {
	case NullsFirst:
		return "NULLS FIRST"
	case NullsLast:
		return "NULLS LAST"
	default:
		return ""
	}
}

// OrderDirection represents ASC or DESC.
type OrderDirection int8

const (
	AscOrder OrderDirection = iota
	DescOrder
)

// Order represents a single ORDER BY item.
type Order struct {
	Expr      Expr
	Direction OrderDirection
	Nulls     NullsOrder
}

// String returns Order's SQL text.
func (o *Order) String() string {
	s := o.Expr.String()
	if o.Direction == DescOrder {
		s += " DESC"
	}

	if nulls := o.Nulls.String(); nulls != "" {
		s += " " + nulls
	}

	return s
}

// OrderBy represents an ORDER BY clause.
type OrderBy []*Order

// String returns OrderBy's SQL text.
func (o OrderBy) String() string {
	if len(o) == 0 {
		return ""
	}

	strs := make([]string, len(o))
	for i, order := range o {
		strs[i] = order.String()
	}

	return "ORDER BY " + strings.Join(strs, ", ")
}

// Limit represents a LIMIT clause. Count is nil when LIMIT ALL was written.
type Limit struct {
	Count Expr
}

// String returns Limit's SQL text.
func (l *Limit) String() string {
	if l == nil {
		return ""
	}

	if l.Count == nil {
		return "LIMIT ALL"
	}

	return "LIMIT " + l.Count.String()
}

// Offset represents an OFFSET clause.
type Offset struct {
	Start Expr
}

// String returns Offset's SQL text.
func (o *Offset) String() string {
	if o == nil || o.Start == nil {
		return ""
	}

	return "OFFSET " + o.Start.String()
}

// Fetch represents a FETCH { FIRST | NEXT } [count] { ROW | ROWS } { ONLY | WITH TIES } clause.
type Fetch struct {
	Count    Expr // nil means the implicit "1" default
	Next     bool // true renders NEXT, false renders FIRST (purely cosmetic synonyms)
	WithTies bool // true renders WITH TIES, false renders ONLY
}

// String returns Fetch's SQL text.
func (f *Fetch) String() string {
	if f == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("FETCH ")

	if f.Next {
		b.WriteString("NEXT ")
	} else {
		b.WriteString("FIRST ")
	}

	if f.Count != nil {
		b.WriteString(f.Count.String())
		b.WriteString(" ")
	}

	if f.WithTies {
		b.WriteString("ROWS WITH TIES")
	} else {
		b.WriteString("ROW ONLY")
	}

	return b.String()
}

// LockStrength represents the row-locking strength in a SELECT's locking clause.
type LockStrength int8

const (
	NoLock LockStrength = iota
	ForUpdate
	ForNoKeyUpdate
	ForShare
	ForKeyShare
)

// String returns LockStrength's SQL text.
func (l LockStrength) String() string {
	switch l {
	case ForUpdate:
		return "FOR UPDATE"
	case ForNoKeyUpdate:
		return "FOR NO KEY UPDATE"
	case ForShare:
		return "FOR SHARE"
	case ForKeyShare:
		return "FOR KEY SHARE"
	default:
		return ""
	}
}

// LockWait represents the NOWAIT/SKIP LOCKED modifier on a locking clause.
type LockWait int8

const (
	NoLockWait LockWait = iota
	NoWait
	SkipLocked
)

// String returns LockWait's SQL text.
func (w LockWait) String() string {
	switch w {
	case NoWait:
		return "NOWAIT"
	case SkipLocked:
		return "SKIP LOCKED"
	default:
		return ""
	}
}

// Lock represents a SELECT locking clause: FOR UPDATE | NO KEY UPDATE |
// SHARE | KEY SHARE [OF table, ...] [NOWAIT | SKIP LOCKED].
type Lock struct {
	Strength LockStrength
	Of       []TableName
	Wait     LockWait
}

// String returns Lock's SQL text.
func (l *Lock) String() string {
	if l == nil || l.Strength == NoLock {
		return ""
	}

	var b strings.Builder

	b.WriteString(l.Strength.String())

	if len(l.Of) > 0 {
		names := make([]string, len(l.Of))
		for i, t := range l.Of {
			names[i] = t.String()
		}

		b.WriteString(" OF ")
		b.WriteString(strings.Join(names, ", "))
	}

	if wait := l.Wait.String(); wait != "" {
		b.WriteString(" ")
		b.WriteString(wait)
	}

	return b.String()
}

// NamedWindow represents one entry of a top-level WINDOW name AS (...) clause.
type NamedWindow struct {
	Name ColIdent
	Spec *WindowSpec
}

// String returns NamedWindow's SQL text.
func (n *NamedWindow) String() string {
	return n.Name.String() + " AS (" + n.Spec.String() + ")"
}

// WindowSpec represents the content of a window definition, shared between
// the top-level WINDOW clause and a per-call OVER (...) clause.
type WindowSpec struct {
	Name        ColIdent // base window name this spec refines, e.g. OVER (w ORDER BY ...)
	PartitionBy []Expr
	OrderBy     OrderBy
	Frame       *FrameClause
}

// String returns WindowSpec's SQL text.
func (w *WindowSpec) String() string {
	if w == nil {
		return ""
	}

	var parts []string

	if !w.Name.IsEmpty() {
		parts = append(parts, w.Name.String())
	}

	if len(w.PartitionBy) > 0 {
		parts = append(parts, "PARTITION BY "+exprListString(w.PartitionBy))
	}

	if len(w.OrderBy) > 0 {
		parts = append(parts, w.OrderBy.String())
	}

	if w.Frame != nil {
		parts = append(parts, w.Frame.String())
	}

	return strings.Join(parts, " ")
}

// overString renders w for a function call's OVER clause: a bare window
// name with no parentheses when w is nothing but a name reference (matching
// the common "OVER w" form, as opposed to a top-level WINDOW clause entry,
// which always parenthesizes its body), parenthesized otherwise.
func (w *WindowSpec) overString() string {
	if w != nil && !w.Name.IsEmpty() && len(w.PartitionBy) == 0 && len(w.OrderBy) == 0 && w.Frame == nil {
		return w.Name.String()
	}

	return "(" + w.String() + ")"
}

// FrameUnit represents ROWS, RANGE, or GROUPS.
type FrameUnit int8

const (
	FrameRows FrameUnit = iota
	FrameRange
	FrameGroups
)

// String returns FrameUnit's SQL text.
func (f FrameUnit) String() string {
	switch f {
	case FrameRange:
		return "RANGE"
	case FrameGroups:
		return "GROUPS"
	default:
		return "ROWS"
	}
}

// FramePointType represents the kind of a window frame boundary.
type FramePointType int8

const (
	CurrentRow FramePointType = iota
	UnboundedPreceding
	UnboundedFollowing
	ExprPreceding
	ExprFollowing
)

// FramePoint represents one boundary of a window frame.
type FramePoint struct {
	Type FramePointType
	Expr Expr
}

// String returns FramePoint's SQL text.
func (f *FramePoint) String() string {
	switch f.Type {
	case UnboundedPreceding:
		return "UNBOUNDED PRECEDING"
	case UnboundedFollowing:
		return "UNBOUNDED FOLLOWING"
	case ExprPreceding:
		return f.Expr.String() + " PRECEDING"
	case ExprFollowing:
		return f.Expr.String() + " FOLLOWING"
	default:
		return "CURRENT ROW"
	}
}

// FrameClause represents a window frame specification.
type FrameClause struct {
	Unit  FrameUnit
	Start *FramePoint
	End   *FramePoint
}

// String returns FrameClause's SQL text.
func (f *FrameClause) String() string {
	if f == nil || f.Start == nil {
		return ""
	}

	unit := f.Unit.String()

	if f.End != nil {
		return unit + " BETWEEN " + f.Start.String() + " AND " + f.End.String()
	}

	return unit + " " + f.Start.String()
}

// Returning represents a RETURNING clause, shared by INSERT/UPDATE/DELETE.
// Its item grammar is identical to a SELECT list, so it reuses SelectExpr.
type Returning struct {
	Exprs []SelectExpr
}

// String returns Returning's SQL text.
func (r *Returning) String() string {
	if r == nil || len(r.Exprs) == 0 {
		return ""
	}

	strs := make([]string, len(r.Exprs))
	for i, e := range r.Exprs {
		strs[i] = e.String()
	}

	return "RETURNING " + strings.Join(strs, ", ")
}

// UpdateExpr represents a "col = expr" SET assignment, shared by UPDATE and
// the DO UPDATE SET clause of ON CONFLICT.
type UpdateExpr struct {
	Name *ColName
	Expr Expr
}

// String returns UpdateExpr's SQL text.
func (u *UpdateExpr) String() string {
	return u.Name.String() + " = " + u.Expr.String()
}

// OnConflict represents an INSERT ... ON CONFLICT clause.
type OnConflict struct {
	Columns    Columns  // ON CONFLICT (col, ...)
	Constraint ColIdent // ON CONFLICT ON CONSTRAINT name
	DoNothing  bool
	Set        []*UpdateExpr // DO UPDATE SET ...
	Where      *Where        // WHERE condition on the DO UPDATE
}

// String returns OnConflict's SQL text.
func (o *OnConflict) String() string {
	if o == nil {
		return ""
	}

	var b strings.Builder

	b.WriteString("ON CONFLICT")

	switch {
	case len(o.Columns) > 0:
		b.WriteString(" (")
		b.WriteString(o.Columns.String())
		b.WriteString(")")
	case !o.Constraint.IsEmpty():
		b.WriteString(" ON CONSTRAINT ")
		b.WriteString(o.Constraint.String())
	}

	if o.DoNothing {
		b.WriteString(" DO NOTHING")

		return b.String()
	}

	sets := make([]string, len(o.Set))
	for i, s := range o.Set {
		sets[i] = s.String()
	}

	b.WriteString(" DO UPDATE SET ")
	b.WriteString(strings.Join(sets, ", "))

	if o.Where != nil && o.Where.Expr != nil {
		b.WriteString(" WHERE ")
		b.WriteString(o.Where.String())
	}

	return b.String()
}

// JoinCondition represents the ON or USING condition of a JOIN. Exactly one
// of On/Using is set.
type JoinCondition struct {
	On    Expr
	Using Columns
}
