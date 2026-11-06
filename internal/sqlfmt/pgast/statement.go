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
	With  *With
	Only  bool // UPDATE ONLY t: exclude inheritance descendants
	Table *AliasedTableExpr
	// Descendants records an explicit "t *" (include descendants, which is
	// already PostgreSQL's default) so the target round-trips as written.
	Descendants bool
	Set         []*UpdateExpr
	From        []TableExpr
	Where       *Where
	Returning   *Returning
}

// String returns Update's SQL text.
func (u *Update) String() string {
	var b strings.Builder

	writeOptionalPrefix(&b, u.With)

	b.WriteString("UPDATE ")
	b.WriteString(u.targetString())

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

// targetString renders the target table, placing ONLY before the table name
// and an explicit "*" between the table name and its alias.
func (u *Update) targetString() string {
	if !u.Only && !u.Descendants {
		return u.Table.String()
	}

	var b strings.Builder

	if u.Only {
		b.WriteString("ONLY ")
	}

	b.WriteString(u.Table.Expr.String())

	if u.Descendants {
		b.WriteString(" *")
	}

	if !u.Table.As.IsEmpty() {
		b.WriteString(" ")
		b.WriteString(u.Table.As.String())
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

// CommonTableExpr represents a single CTE definition. Subquery holds
// whichever statement the CTE body parses to — PostgreSQL CTEs can wrap
// INSERT/UPDATE/DELETE bodies, not just SELECT.
type CommonTableExpr struct {
	Name     TableIdent
	Columns  Columns
	Subquery Statement
}

// String returns CommonTableExpr's SQL text.
func (c *CommonTableExpr) String() string {
	name := c.Name.String()
	if len(c.Columns) > 0 {
		name += " (" + c.Columns.String() + ")"
	}

	return name + " AS (" + c.Subquery.String() + ")"
}

// Merge represents a MERGE statement.
type Merge struct {
	With   *With
	Target *AliasedTableExpr
	Source TableExpr
	On     Expr
	Whens  []*MergeWhen
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

	return b.String()
}

// MergeWhen represents one WHEN [NOT] MATCHED [BY SOURCE] THEN ... clause of
// a MERGE statement. Exactly one of DoNothing/Delete/Set/(Columns+Values) is
// set, depending on the clause's action.
type MergeWhen struct {
	Matched   bool
	BySource  bool
	Condition Expr
	DoNothing bool
	Delete    bool
	Set       []*UpdateExpr // WHEN MATCHED / NOT MATCHED BY SOURCE ... THEN UPDATE SET ...
	Columns   Columns       // WHEN NOT MATCHED ... THEN INSERT (columns...)
	Values    []Expr        // ... VALUES (...)
}

// String returns MergeWhen's SQL text.
func (w *MergeWhen) String() string {
	var b strings.Builder

	if w.Matched {
		b.WriteString("WHEN MATCHED")
	} else {
		b.WriteString("WHEN NOT MATCHED")

		if w.BySource {
			b.WriteString(" BY SOURCE")
		}
	}

	if w.Condition != nil {
		b.WriteString(" AND ")
		b.WriteString(w.Condition.String())
	}

	b.WriteString(" THEN ")
	w.writeAction(&b)

	return b.String()
}

func (w *MergeWhen) writeAction(b *strings.Builder) {
	switch {
	case w.DoNothing:
		b.WriteString("DO NOTHING")
	case w.Delete:
		b.WriteString("DELETE")
	case w.Matched || w.BySource || len(w.Set) > 0:
		sets := make([]string, len(w.Set))
		for i, s := range w.Set {
			sets[i] = s.String()
		}

		b.WriteString("UPDATE SET ")
		b.WriteString(strings.Join(sets, ", "))
	default:
		b.WriteString("INSERT")

		if len(w.Columns) > 0 {
			b.WriteString(" (")
			b.WriteString(w.Columns.String())
			b.WriteString(")")
		}

		b.WriteString(" VALUES (")
		b.WriteString(exprListString(w.Values))
		b.WriteString(")")
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
