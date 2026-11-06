package pgast

import "strings"

// Select represents a SELECT statement. WITH is modeled on Select (and the
// other DML statements below) rather than as its own wrapper statement,
// since PostgreSQL CTEs prefix any of SELECT/INSERT/UPDATE/DELETE/MERGE.
type Select struct {
	With        *With
	Distinct    bool
	DistinctOn  *DistinctOn
	SelectExprs []SelectExpr
	From        []TableExpr
	Where       *Where
	GroupBy     *GroupBy
	Having      *Where
	Window      []*NamedWindow
	OrderBy     OrderBy
	Limit       *Limit
	Offset      *Offset
	Fetch       *Fetch
	Locks       []*Lock
}

// String returns Select's SQL text.
func (s *Select) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, s.With)

	b.WriteString("SELECT ")

	switch {
	case s.DistinctOn != nil:
		b.WriteString(s.DistinctOn.String())
		b.WriteString(" ")
	case s.Distinct:
		b.WriteString("DISTINCT ")
	}

	writeSelectExprs(&b, s.SelectExprs)
	writeFromClause(&b, s.From)
	writeWhereClause(&b, "WHERE", s.Where)
	writeSuffix(&b, s.GroupBy)
	writeWhereClause(&b, "HAVING", s.Having)
	writeWindowClause(&b, s.Window)
	writeSuffix(&b, s.OrderBy)
	writeSuffix(&b, s.Limit)
	writeSuffix(&b, s.Offset)
	writeSuffix(&b, s.Fetch)

	for _, lock := range s.Locks {
		writeSuffix(&b, lock)
	}

	return b.String()
}

// Insert represents an INSERT statement.
//
// Its target is a TableName plus an Alias rather than the *AliasedTableExpr
// UPDATE, DELETE, and MERGE use, because PostgreSQL's INSERT target is a
// different production: it takes no ONLY or "*", and its alias requires AS,
// so Alias renders with AS where an AliasedTableExpr's would not.
type Insert struct {
	With       *With
	Table      TableName
	Alias      TableIdent
	Columns    Columns
	Rows       InsertRows
	OnConflict *OnConflict
	Returning  *Returning
}

// String returns Insert's SQL text.
func (ins *Insert) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, ins.With)

	b.WriteString("INSERT INTO ")
	b.WriteString(ins.Table.String())

	if !ins.Alias.IsEmpty() {
		b.WriteString(" AS ")
		b.WriteString(ins.Alias.String())
	}

	if len(ins.Columns) > 0 {
		b.WriteString(" (")
		b.WriteString(ins.Columns.String())
		b.WriteString(")")
	}

	b.WriteString(" ")
	b.WriteString(ins.Rows.String())

	if ins.OnConflict != nil {
		b.WriteString(" ")
		b.WriteString(ins.OnConflict.String())
	}

	if s := ins.Returning.String(); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}

	return b.String()
}

// DefaultValues represents INSERT's "DEFAULT VALUES" row source.
type DefaultValues struct{}

// String returns DefaultValues's SQL text.
func (DefaultValues) String() string { return "DEFAULT VALUES" }

// Values represents rows of values for INSERT.
type Values [][]Expr

// String returns Values's SQL text.
func (v Values) String() string {
	rows := make([]string, len(v))
	for i, row := range v {
		rows[i] = "(" + exprListString(row) + ")"
	}

	return "VALUES " + strings.Join(rows, ", ")
}

// Update represents an UPDATE statement.
type Update struct {
	With      *With
	Table     *AliasedTableExpr
	Set       []*UpdateExpr
	From      []TableExpr
	Where     *Where
	Returning *Returning
}

// String returns Update's SQL text.
func (u *Update) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, u.With)

	b.WriteString("UPDATE ")
	b.WriteString(u.Table.String())

	sets := make([]string, len(u.Set))
	for i, e := range u.Set {
		sets[i] = e.String()
	}

	b.WriteString(" SET ")
	b.WriteString(strings.Join(sets, ", "))
	writeFromClause(&b, u.From)
	writeWhereClause(&b, "WHERE", u.Where)

	if s := u.Returning.String(); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}

	return b.String()
}

// Delete represents a DELETE statement.
type Delete struct {
	With      *With
	Table     *AliasedTableExpr
	Using     []TableExpr
	Where     *Where
	Returning *Returning
}

// String returns Delete's SQL text.
func (d *Delete) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, d.With)

	b.WriteString("DELETE FROM ")
	b.WriteString(d.Table.String())

	if len(d.Using) > 0 {
		strs := make([]string, len(d.Using))
		for i, u := range d.Using {
			strs[i] = u.String()
		}

		b.WriteString(" USING ")
		b.WriteString(strings.Join(strs, ", "))
	}

	writeWhereClause(&b, "WHERE", d.Where)

	if s := d.Returning.String(); s != "" {
		b.WriteString(" ")
		b.WriteString(s)
	}

	return b.String()
}

// SetOpType represents UNION, INTERSECT, or EXCEPT.
type SetOpType int8

const (
	Union SetOpType = iota
	Intersect
	Except
)

// String returns SetOpType's SQL text.
func (t SetOpType) String() string {
	switch t {
	case Intersect:
		return "INTERSECT"
	case Except:
		return "EXCEPT"
	default:
		return "UNION"
	}
}

func (t SetOpType) precedence() int {
	if t == Intersect {
		return 2
	}

	return 1
}

// SetOp represents a UNION/INTERSECT/EXCEPT statement.
type SetOp struct {
	With    *With
	Left    Statement
	Right   Statement
	Op      SetOpType
	All     bool
	OrderBy OrderBy
	Limit   *Limit
	Offset  *Offset
	Fetch   *Fetch
}

// String returns SetOp's SQL text.
func (u *SetOp) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, u.With)

	writeSetOpOperand(&b, u.Left, needsSetOpParens(u.Left, u.Op, false))
	b.WriteString(" ")
	b.WriteString(u.Op.String())

	if u.All {
		b.WriteString(" ALL")
	}

	b.WriteString(" ")
	writeSetOpOperand(&b, u.Right, needsSetOpParens(u.Right, u.Op, true))
	writeSuffix(&b, u.OrderBy)
	writeSuffix(&b, u.Limit)
	writeSuffix(&b, u.Offset)
	writeSuffix(&b, u.Fetch)

	return b.String()
}

func writeSetOpOperand(b *strings.Builder, stmt Statement, parens bool) {
	if !parens {
		b.WriteString(stmt.String())

		return
	}

	b.WriteByte('(')
	b.WriteString(stmt.String())
	b.WriteByte(')')
}

// needsSetOpParens reports whether stmt must be parenthesized as an operand
// of a parentOp set operation: when it has a WITH or trailing clause of its
// own, which would otherwise render as if scoping the whole set operation,
// or when it is a nested SetOp whose grouping precedence doesn't imply.
func needsSetOpParens(stmt Statement, parentOp SetOpType, isRight bool) bool {
	switch s := stmt.(type) {
	case *SetOp:
		return s.hasOperandScopedClauses() || !s.Op.groupsWithoutParens(parentOp, isRight)
	case *Select:
		return s.hasOperandScopedClauses()
	default:
		return false
	}
}

func (u *SetOp) hasOperandScopedClauses() bool {
	return u.With != nil || len(u.OrderBy) > 0 || u.Limit != nil || u.Offset != nil || u.Fetch != nil
}

func (s *Select) hasOperandScopedClauses() bool {
	return s.With != nil || len(s.OrderBy) > 0 || s.Limit != nil || s.Offset != nil || s.Fetch != nil ||
		len(s.Locks) > 0
}

// groupsWithoutParens reports whether a nested t operation, as the left or
// right operand of a parent operation, already groups as intended without
// parentheses: INTERSECT binds tighter than UNION/EXCEPT, and operators of
// equal precedence associate left.
func (t SetOpType) groupsWithoutParens(parent SetOpType, isRight bool) bool {
	if isRight {
		return t.precedence() > parent.precedence()
	}

	return t.precedence() >= parent.precedence()
}

// With represents a WITH [RECURSIVE] clause.
type With struct {
	CTEs      []*CommonTableExpr
	Recursive bool
}

// String returns With's SQL text.
func (w *With) String() string {
	if w == nil {
		return ""
	}

	keyword := "WITH"
	if w.Recursive {
		keyword = "WITH RECURSIVE"
	}

	ctes := make([]string, len(w.CTEs))
	for i, cte := range w.CTEs {
		ctes[i] = cte.String()
	}

	return keyword + " " + strings.Join(ctes, ", ")
}

// CTEMaterialization represents a CTE's optional [NOT] MATERIALIZED hint.
type CTEMaterialization int8

const (
	MaterializeDefault CTEMaterialization = iota
	Materialized
	NotMaterialized
)

// String returns CTEMaterialization's SQL text, or "" when no hint is given.
func (m CTEMaterialization) String() string {
	switch m {
	case Materialized:
		return "MATERIALIZED"
	case NotMaterialized:
		return "NOT MATERIALIZED"
	default:
		return ""
	}
}

// CommonTableExpr represents a single CTE definition. Subquery holds
// whichever statement the CTE body parses to — PostgreSQL CTEs can wrap
// INSERT/UPDATE/DELETE bodies, not just SELECT.
type CommonTableExpr struct {
	Name            TableIdent
	Columns         Columns
	Materialization CTEMaterialization
	Subquery        Statement
}

// String returns CommonTableExpr's SQL text.
func (c *CommonTableExpr) String() string {
	var b strings.Builder

	b.WriteString(c.Name.String())

	if len(c.Columns) > 0 {
		b.WriteString(" (")
		b.WriteString(c.Columns.String())
		b.WriteString(")")
	}

	b.WriteString(" AS ")
	writeOptionalPrefix(&b, c.Materialization)
	b.WriteString("(")
	b.WriteString(c.Subquery.String())
	b.WriteString(")")

	return b.String()
}

// Merge represents a MERGE statement (PostgreSQL 15+). Target carries ONLY
// and "*" the same way UPDATE's and DELETE's targets do.
type Merge struct {
	With      *With
	Target    *AliasedTableExpr
	Source    TableExpr
	On        Expr
	Whens     []*MergeWhen
	Returning *Returning // PostgreSQL 17+
}

// String returns Merge's SQL text.
func (m *Merge) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, m.With)

	b.WriteString("MERGE INTO ")
	b.WriteString(m.Target.String())
	b.WriteString(" USING ")
	b.WriteString(m.Source.String())
	b.WriteString(" ON ")
	b.WriteString(m.On.String())

	for _, w := range m.Whens {
		b.WriteString(" ")
		b.WriteString(w.String())
	}

	writeSuffix(&b, m.Returning)

	return b.String()
}

// MergeMatchKind is the condition a MERGE WHEN clause applies to.
type MergeMatchKind int8

const (
	// MergeMatched is "WHEN MATCHED".
	MergeMatched MergeMatchKind = iota
	// MergeNotMatched is "WHEN NOT MATCHED".
	MergeNotMatched
	// MergeNotMatchedByTarget is "WHEN NOT MATCHED BY TARGET" (PostgreSQL
	// 17+). It means the same as MergeNotMatched, and is kept apart only so
	// the clause round-trips as written.
	MergeNotMatchedByTarget
	// MergeNotMatchedBySource is "WHEN NOT MATCHED BY SOURCE" (PostgreSQL
	// 17+).
	MergeNotMatchedBySource
)

// String returns MergeMatchKind's SQL text.
func (k MergeMatchKind) String() string {
	switch k {
	case MergeNotMatched:
		return "NOT MATCHED"
	case MergeNotMatchedByTarget:
		return "NOT MATCHED BY TARGET"
	case MergeNotMatchedBySource:
		return "NOT MATCHED BY SOURCE"
	default:
		return "MATCHED"
	}
}

// MergeWhen represents one "WHEN match [AND condition] THEN action" clause
// of a MERGE statement. Which actions a match kind admits (INSERT only when
// NOT MATCHED [BY TARGET], UPDATE and DELETE otherwise) is enforced by the
// parser, as in PostgreSQL's grammar.
type MergeWhen struct {
	Match     MergeMatchKind
	Condition Expr
	Action    MergeAction
}

// String returns MergeWhen's SQL text.
func (w *MergeWhen) String() string {
	var b strings.Builder

	b.WriteString("WHEN ")
	b.WriteString(w.Match.String())

	if w.Condition != nil {
		b.WriteString(" AND ")
		b.WriteString(w.Condition.String())
	}

	b.WriteString(" THEN ")
	b.WriteString(w.Action.String())

	return b.String()
}

// MergeUpdate represents a MERGE "UPDATE SET ..." action.
type MergeUpdate struct {
	Set []*UpdateExpr
}

// String returns MergeUpdate's SQL text.
func (u *MergeUpdate) String() string {
	sets := make([]string, len(u.Set))
	for i, e := range u.Set {
		sets[i] = e.String()
	}

	return "UPDATE SET " + strings.Join(sets, ", ")
}

// MergeDelete represents a MERGE "DELETE" action.
type MergeDelete struct{}

// String returns MergeDelete's SQL text.
func (MergeDelete) String() string { return "DELETE" }

// MergeDoNothing represents a MERGE "DO NOTHING" action.
type MergeDoNothing struct{}

// String returns MergeDoNothing's SQL text.
func (MergeDoNothing) String() string { return "DO NOTHING" }

// MergeInsert represents a MERGE
// "INSERT [(columns)] [OVERRIDING ...] {VALUES (...) | DEFAULT VALUES}"
// action. A nil Values renders DEFAULT VALUES, which PostgreSQL allows only
// without Columns or Overriding. An empty VALUES row is not valid SQL, so
// nil is unambiguous.
type MergeInsert struct {
	Columns    Columns
	Overriding Overriding
	Values     []Expr
}

// String returns MergeInsert's SQL text.
func (ins *MergeInsert) String() string {
	var b strings.Builder

	b.WriteString("INSERT")

	if len(ins.Columns) > 0 {
		b.WriteString(" (")
		b.WriteString(ins.Columns.String())
		b.WriteString(")")
	}

	writeSuffix(&b, ins.Overriding)

	if ins.Values == nil {
		b.WriteString(" DEFAULT VALUES")

		return b.String()
	}

	b.WriteString(" VALUES (")
	b.WriteString(exprListString(ins.Values))
	b.WriteString(")")

	return b.String()
}

// Overriding is an INSERT's "OVERRIDING {SYSTEM | USER} VALUE" clause, which
// controls how values for identity columns are treated.
type Overriding int8

const (
	OverridingNone Overriding = iota
	OverridingSystemValue
	OverridingUserValue
)

// String returns Overriding's SQL text, or "" when there is no clause.
func (o Overriding) String() string {
	switch o {
	case OverridingSystemValue:
		return "OVERRIDING SYSTEM VALUE"
	case OverridingUserValue:
		return "OVERRIDING USER VALUE"
	default:
		return ""
	}
}

// --- helpers to reduce cyclomatic complexity ---

func writeOptionalPrefix(b *strings.Builder, w SQLNode) {
	s := w.String()
	if s != "" {
		b.WriteString(s)
		b.WriteString(" ")
	}
}

func writeSelectExprs(b *strings.Builder, exprs []SelectExpr) {
	strs := make([]string, len(exprs))
	for i, e := range exprs {
		strs[i] = e.String()
	}

	b.WriteString(strings.Join(strs, ", "))
}

func writeFromClause(b *strings.Builder, from []TableExpr) {
	if len(from) == 0 {
		return
	}

	strs := make([]string, len(from))
	for i, f := range from {
		strs[i] = f.String()
	}

	b.WriteString(" FROM ")
	b.WriteString(strings.Join(strs, ", "))
}

func writeWhereClause(b *strings.Builder, keyword string, w *Where) {
	if w == nil || w.Expr == nil {
		return
	}

	b.WriteString(" ")
	b.WriteString(keyword)
	b.WriteString(" ")
	b.WriteString(w.String())
}

func writeWindowClause(b *strings.Builder, windows []*NamedWindow) {
	if len(windows) == 0 {
		return
	}

	strs := make([]string, len(windows))
	for i, w := range windows {
		strs[i] = w.String()
	}

	b.WriteString(" WINDOW ")
	b.WriteString(strings.Join(strs, ", "))
}

// writeSuffix renders a trailing clause (GROUP BY/ORDER BY/LIMIT/OFFSET/
// FETCH/lock) that already returns "" for its zero value, prefixing it with
// a space when non-empty.
func writeSuffix(b *strings.Builder, c SQLNode) {
	s := c.String()
	if s == "" {
		return
	}

	b.WriteString(" ")
	b.WriteString(s)
}
