package sqlast

import (
	"fmt"
	"strings"
)

// Where represents a WHERE or HAVING clause.
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

// GroupBy represents a GROUP BY clause.
type GroupBy struct {
	Exprs []Expr
}

// String returns GroupBy's SQL text.
func (g *GroupBy) String() string {
	if g == nil || len(g.Exprs) == 0 {
		return ""
	}

	strs := make([]string, len(g.Exprs))
	for i, e := range g.Exprs {
		strs[i] = e.String()
	}

	return "GROUP BY " + strings.Join(strs, ", ")
}

// Order represents a single ORDER BY item.
type Order struct {
	Expr      Expr
	Direction OrderDirection
}

// String returns Order's SQL text.
func (o *Order) String() string {
	s := o.Expr.String()
	if o.Direction == DescOrder {
		s += " DESC"
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

// Limit represents a LIMIT clause.
type Limit struct {
	Offset   Expr
	Rowcount Expr
}

// String returns Limit's SQL text.
func (l *Limit) String() string {
	if l == nil {
		return ""
	}

	if l.Offset != nil {
		return fmt.Sprintf("LIMIT %s OFFSET %s", l.Rowcount.String(), l.Offset.String())
	}

	return "LIMIT " + l.Rowcount.String()
}

// OrderDirection represents ASC or DESC.
type OrderDirection int8

const (
	AscOrder OrderDirection = iota
	DescOrder
)

// Lock represents a lock type for SELECT statements.
type Lock int8

const (
	NoLock Lock = iota
	ForUpdateLock
	ShareModeLock
	ForShareLock
)

// ToString returns Lock's lower-case SQL keyword; String upper-cases the result.
func (l Lock) ToString() string {
	switch l {
	case ForUpdateLock:
		return "for update"
	case ShareModeLock:
		return "lock in share mode"
	case ForShareLock:
		return "for share"
	default:
		return ""
	}
}

// String returns Lock's SQL text.
func (l Lock) String() string {
	return strings.ToUpper(l.ToString())
}

// LockWaitType represents the NOWAIT/SKIP LOCKED modifier on a FOR UPDATE
// or FOR SHARE locking clause.
type LockWaitType int8

const (
	NoLockWait LockWaitType = iota
	NowaitType
	SkipLockedType
)

// String returns LockWaitType's SQL text.
func (t LockWaitType) String() string {
	switch t {
	case NowaitType:
		return "NOWAIT"
	case SkipLockedType:
		return "SKIP LOCKED"
	default:
		return ""
	}
}

// InsertAction represents INSERT or REPLACE.
type InsertAction int8

const (
	InsertAct InsertAction = iota
	ReplaceAct
)

// String returns InsertAction's SQL text.
func (a InsertAction) String() string {
	if a == ReplaceAct {
		return "REPLACE"
	}

	return "INSERT"
}

// JoinType represents the type of a JOIN.
type JoinType int8

const (
	NormalJoinType JoinType = iota
	StraightJoinType
	LeftJoinType
	RightJoinType
	NaturalJoinType
	NaturalLeftJoinType
	NaturalRightJoinType
	CrossJoinType
)

const joinStr = "join"

var joinTypeStrings = [...]string{
	NormalJoinType:       joinStr,
	StraightJoinType:     "straight_join",
	LeftJoinType:         "left join",
	RightJoinType:        "right join",
	NaturalJoinType:      "natural join",
	NaturalLeftJoinType:  "natural left join",
	NaturalRightJoinType: "natural right join",
	CrossJoinType:        "cross join",
}

// ToString returns JoinType's lower-case SQL keyword; String upper-cases the result.
func (jt JoinType) ToString() string {
	if int(jt) < len(joinTypeStrings) {
		return joinTypeStrings[jt]
	}

	return joinStr
}

// JoinCondition represents the ON condition of a JOIN.
type JoinCondition struct {
	On Expr
}

// ComparisonOperator represents a comparison operator.
type ComparisonOperator int8

const (
	EqualOp ComparisonOperator = iota
	LessThanOp
	GreaterThanOp
	LessEqualOp
	GreaterEqualOp
	NotEqualOp
	NullSafeEqualOp
	InOp
	NotInOp
	LikeOp
	NotLikeOp
	RegexpOp
	NotRegexpOp
)

var comparisonOpStrings = [...]string{
	EqualOp:         "=",
	LessThanOp:      "<",
	GreaterThanOp:   ">",
	LessEqualOp:     "<=",
	GreaterEqualOp:  ">=",
	NotEqualOp:      "!=",
	NullSafeEqualOp: "<=>",
	InOp:            "in",
	NotInOp:         "not in",
	LikeOp:          "like",
	NotLikeOp:       "not like",
	RegexpOp:        "regexp",
	NotRegexpOp:     "not regexp",
}

// ToString returns ComparisonOperator's lower-case SQL keyword; String upper-cases the result.
func (op ComparisonOperator) ToString() string {
	if int(op) < len(comparisonOpStrings) {
		return comparisonOpStrings[op]
	}

	return "="
}

// IndexHintType represents USE/IGNORE/FORCE index hint types.
type IndexHintType int8

const (
	UseOp IndexHintType = iota
	IgnoreOp
	ForceOp
)

// ToString returns IndexHintType's lower-case SQL keyword; String upper-cases the result.
func (t IndexHintType) ToString() string {
	switch t {
	case IgnoreOp:
		return "ignore index"
	case ForceOp:
		return "force index"
	default:
		return "use index"
	}
}

// IndexHintForType represents what the index hint applies to.
type IndexHintForType int8

const (
	NoForType IndexHintForType = iota
	JoinForType
	GroupByForType
	OrderByForType
)

// ToString returns IndexHintForType's lower-case SQL keyword; String upper-cases the result.
func (t IndexHintForType) ToString() string {
	switch t {
	case JoinForType:
		return joinStr
	case GroupByForType:
		return "group by"
	case OrderByForType:
		return "order by"
	default:
		return ""
	}
}

// IndexHint represents a single index hint.
type IndexHint struct {
	Type    IndexHintType
	ForType IndexHintForType
	Indexes []ColIdent
}

// String returns IndexHint's SQL text.
func (h *IndexHint) String() string {
	idxs := make([]string, len(h.Indexes))
	for i, idx := range h.Indexes {
		idxs[i] = idx.String()
	}

	hintType := strings.ToUpper(h.Type.ToString())
	idxList := "(" + strings.Join(idxs, ", ") + ")"

	forStr := h.ForType.ToString()
	if forStr != "" {
		return hintType + " FOR " + strings.ToUpper(forStr) + " " + idxList
	}

	return hintType + " " + idxList
}

// IndexHints is a slice of IndexHint pointers.
type IndexHints []*IndexHint

// Columns represents a list of column identifiers.
type Columns []ColIdent

// String returns Columns's SQL text.
func (c Columns) String() string {
	strs := make([]string, len(c))
	for i, col := range c {
		strs[i] = col.String()
	}

	return strings.Join(strs, ", ")
}

// Values represents rows of values for INSERT.
type Values [][]Expr

// String returns Values's SQL text.
func (v Values) String() string {
	rows := make([]string, len(v))

	for i, row := range v {
		vals := make([]string, len(row))
		for j, val := range row {
			vals[j] = val.String()
		}

		rows[i] = "(" + strings.Join(vals, ", ") + ")"
	}

	return "VALUES " + strings.Join(rows, ", ")
}

// UpdateExpr represents a SET assignment in an UPDATE statement.
type UpdateExpr struct {
	Name *ColName
	Expr Expr
}

// String returns UpdateExpr's SQL text.
func (u *UpdateExpr) String() string {
	return u.Name.String() + " = " + u.Expr.String()
}

// OnDup represents an ON DUPLICATE KEY UPDATE clause.
type OnDup []*UpdateExpr

// String returns OnDup's SQL text.
func (o OnDup) String() string {
	if len(o) == 0 {
		return ""
	}

	strs := make([]string, len(o))
	for i, expr := range o {
		strs[i] = expr.String()
	}

	return "ON DUPLICATE KEY UPDATE " + strings.Join(strs, ", ")
}

// OverClause represents an OVER clause for window functions.
type OverClause struct {
	WindowName ColIdent
	WindowSpec *WindowSpecification
}

// String returns OverClause's SQL text.
func (o *OverClause) String() string {
	if o == nil {
		return ""
	}

	if !o.WindowName.IsEmpty() && o.WindowSpec == nil {
		return "OVER " + o.WindowName.String()
	}

	if o.WindowSpec == nil {
		return "OVER ()"
	}

	spec := o.WindowSpec.String()
	if spec == "" {
		return "OVER ()"
	}

	return "OVER (" + spec + ")"
}

// WindowSpecification represents the content of an OVER clause.
type WindowSpecification struct {
	PartitionClause []Expr
	OrderClause     OrderBy
	FrameClause     *FrameClause
}

// String returns WindowSpecification's SQL text.
func (w *WindowSpecification) String() string {
	if w == nil {
		return ""
	}

	var parts []string

	if len(w.PartitionClause) > 0 {
		exprs := make([]string, len(w.PartitionClause))
		for i, e := range w.PartitionClause {
			exprs[i] = e.String()
		}

		parts = append(parts, "PARTITION BY "+strings.Join(exprs, ", "))
	}

	if len(w.OrderClause) > 0 {
		strs := make([]string, len(w.OrderClause))
		for i, o := range w.OrderClause {
			strs[i] = o.String()
		}

		parts = append(parts, "ORDER BY "+strings.Join(strs, ", "))
	}

	if w.FrameClause != nil {
		parts = append(parts, w.FrameClause.String())
	}

	return strings.Join(parts, " ")
}

// FrameClause represents a window frame specification.
type FrameClause struct {
	Unit  FrameUnitType
	Start *FramePoint
	End   *FramePoint
}

// String returns FrameClause's SQL text.
func (f *FrameClause) String() string {
	if f == nil {
		return ""
	}

	unit := f.Unit.String()

	if f.End != nil {
		return unit + " BETWEEN " + f.Start.String() + " AND " + f.End.String()
	}

	return unit + " " + f.Start.String()
}

// FrameUnitType represents ROWS or RANGE.
type FrameUnitType int8

const (
	FrameRowsType FrameUnitType = iota
	FrameRangeType
)

// String returns FrameUnitType's SQL text.
func (f FrameUnitType) String() string {
	if f == FrameRangeType {
		return "RANGE"
	}

	return "ROWS"
}

// FramePoint represents a point in a window frame.
type FramePoint struct {
	Type FramePointType
	Expr Expr
}

// String returns FramePoint's SQL text.
func (f *FramePoint) String() string {
	switch f.Type {
	case UnboundedPrecedingType:
		return "UNBOUNDED PRECEDING"
	case UnboundedFollowingType:
		return "UNBOUNDED FOLLOWING"
	case ExprPrecedingType:
		return f.Expr.String() + " PRECEDING"
	case ExprFollowingType:
		return f.Expr.String() + " FOLLOWING"
	default:
		return "CURRENT ROW"
	}
}

// FramePointType represents the type of a frame point.
type FramePointType int8

const (
	CurrentRowType FramePointType = iota
	UnboundedPrecedingType
	UnboundedFollowingType
	ExprPrecedingType
	ExprFollowingType
)

// ArgumentLessWindowExprType represents window functions that take no arguments.
type ArgumentLessWindowExprType int8

const (
	CumeDistExprType ArgumentLessWindowExprType = iota
	DenseRankExprType
	PercentRankExprType
	RankExprType
	RowNumberExprType
)

var arglessWindowExprStrings = [...]string{
	CumeDistExprType:    "CUME_DIST",
	DenseRankExprType:   "DENSE_RANK",
	PercentRankExprType: "PERCENT_RANK",
	RankExprType:        "RANK",
	RowNumberExprType:   "ROW_NUMBER",
}

// String returns ArgumentLessWindowExprType's SQL text.
func (t ArgumentLessWindowExprType) String() string {
	if int(t) < len(arglessWindowExprStrings) {
		return arglessWindowExprStrings[t]
	}

	return "ROW_NUMBER"
}

// FirstOrLastValueExprType represents FIRST_VALUE or LAST_VALUE.
type FirstOrLastValueExprType int8

const (
	FirstValueExprType FirstOrLastValueExprType = iota
	LastValueExprType
)

// String returns FirstOrLastValueExprType's SQL text.
func (t FirstOrLastValueExprType) String() string {
	if t == LastValueExprType {
		return "LAST_VALUE"
	}

	return "FIRST_VALUE"
}

// LagLeadExprType represents LAG or LEAD.
type LagLeadExprType int8

const (
	LagExprType LagLeadExprType = iota
	LeadExprType
)

// String returns LagLeadExprType's SQL text.
func (t LagLeadExprType) String() string {
	if t == LeadExprType {
		return "LEAD"
	}

	return "LAG"
}

// NullTreatmentType represents RESPECT NULLS or IGNORE NULLS.
type NullTreatmentType int8

const (
	RespectNullsType NullTreatmentType = iota
	IgnoreNullsType
)

// String returns NullTreatmentType's SQL text.
func (t NullTreatmentType) String() string {
	if t == IgnoreNullsType {
		return "IGNORE NULLS"
	}

	return "RESPECT NULLS"
}

// NullTreatmentClause wraps NullTreatmentType.
type NullTreatmentClause struct {
	Type NullTreatmentType
}

// String returns NullTreatmentClause's SQL text.
func (n *NullTreatmentClause) String() string {
	if n == nil {
		return ""
	}

	return n.Type.String()
}

// FromFirstLastType represents FROM FIRST or FROM LAST.
type FromFirstLastType int8

const (
	FromFirstType FromFirstLastType = iota
	FromLastType
)

// String returns FromFirstLastType's SQL text.
func (t FromFirstLastType) String() string {
	if t == FromLastType {
		return "FROM LAST"
	}

	return "FROM FIRST"
}

// FromFirstLastClause wraps FromFirstLastType.
type FromFirstLastClause struct {
	Type FromFirstLastType
}

// String returns FromFirstLastClause's SQL text.
func (f *FromFirstLastClause) String() string {
	if f == nil {
		return ""
	}

	return f.Type.String()
}
