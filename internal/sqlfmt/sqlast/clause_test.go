package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestWhere_String(t *testing.T) {
	tests := []struct {
		name string
		w    *sqlast.Where
		want string
	}{
		{"nil", nil, ""},
		{"nil expr", &sqlast.Where{}, ""},
		{"with expr", &sqlast.Where{Expr: lit("1")}, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}

func TestGroupBy_String(t *testing.T) {
	tests := []struct {
		name string
		g    *sqlast.GroupBy
		want string
	}{
		{"nil", nil, ""},
		{"empty", &sqlast.GroupBy{}, ""},
		{"single", &sqlast.GroupBy{Exprs: exprs("a")}, "GROUP BY a"},
		{"multiple", &sqlast.GroupBy{Exprs: exprs("a", "b")}, "GROUP BY a, b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.g.String())
		})
	}
}

func TestOrder_String(t *testing.T) {
	assertEqual(t, "a", (&sqlast.Order{Expr: lit("a"), Direction: sqlast.AscOrder}).String())
	assertEqual(t, "a DESC", (&sqlast.Order{Expr: lit("a"), Direction: sqlast.DescOrder}).String())
}

func TestOrderBy_String(t *testing.T) {
	assertEqual(t, "", sqlast.OrderBy(nil).String())

	ob := sqlast.OrderBy{
		{Expr: lit("a"), Direction: sqlast.AscOrder},
		{Expr: lit("b"), Direction: sqlast.DescOrder},
	}

	assertEqual(t, "ORDER BY a, b DESC", ob.String())
}

func TestLimit_String(t *testing.T) {
	tests := []struct {
		name string
		l    *sqlast.Limit
		want string
	}{
		{"nil", nil, ""},
		{"without offset", &sqlast.Limit{Rowcount: lit("10")}, "LIMIT 10"},
		{"with offset", &sqlast.Limit{Rowcount: lit("10"), Offset: lit("5")}, "LIMIT 10 OFFSET 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.l.String())
		})
	}
}

func TestLock(t *testing.T) {
	assertEqual(t, "", sqlast.NoLock.ToString())
	assertEqual(t, "for update", sqlast.ForUpdateLock.ToString())
	assertEqual(t, "lock in share mode", sqlast.ShareModeLock.ToString())
	assertEqual(t, "FOR UPDATE", sqlast.ForUpdateLock.String())
}

func TestInsertAction_String(t *testing.T) {
	assertEqual(t, "INSERT", sqlast.InsertAct.String())
	assertEqual(t, "REPLACE", sqlast.ReplaceAct.String())
}

func TestJoinType_ToString(t *testing.T) {
	tests := []struct {
		jt   sqlast.JoinType
		want string
	}{
		{sqlast.NormalJoinType, "join"},
		{sqlast.StraightJoinType, "straight_join"},
		{sqlast.LeftJoinType, "left join"},
		{sqlast.RightJoinType, "right join"},
		{sqlast.NaturalJoinType, "natural join"},
		{sqlast.NaturalLeftJoinType, "natural left join"},
		{sqlast.NaturalRightJoinType, "natural right join"},
		{sqlast.CrossJoinType, "cross join"},
		{sqlast.JoinType(99), "join"},
	}

	for _, tt := range tests {
		assertEqual(t, tt.want, tt.jt.ToString())
	}
}

func TestComparisonOperator_ToString(t *testing.T) {
	tests := []struct {
		op   sqlast.ComparisonOperator
		want string
	}{
		{sqlast.EqualOp, "="},
		{sqlast.LessThanOp, "<"},
		{sqlast.GreaterThanOp, ">"},
		{sqlast.LessEqualOp, "<="},
		{sqlast.GreaterEqualOp, ">="},
		{sqlast.NotEqualOp, "!="},
		{sqlast.NullSafeEqualOp, "<=>"},
		{sqlast.InOp, "in"},
		{sqlast.NotInOp, "not in"},
		{sqlast.LikeOp, "like"},
		{sqlast.NotLikeOp, "not like"},
		{sqlast.RegexpOp, "regexp"},
		{sqlast.NotRegexpOp, "not regexp"},
		{sqlast.ComparisonOperator(99), "="},
	}

	for _, tt := range tests {
		assertEqual(t, tt.want, tt.op.ToString())
	}
}

func TestIndexHintType_ToString(t *testing.T) {
	assertEqual(t, "use index", sqlast.UseOp.ToString())
	assertEqual(t, "ignore index", sqlast.IgnoreOp.ToString())
	assertEqual(t, "force index", sqlast.ForceOp.ToString())
}

func TestIndexHintForType_ToString(t *testing.T) {
	assertEqual(t, "", sqlast.NoForType.ToString())
	assertEqual(t, "join", sqlast.JoinForType.ToString())
	assertEqual(t, "group by", sqlast.GroupByForType.ToString())
	assertEqual(t, "order by", sqlast.OrderByForType.ToString())
}

func TestIndexHint_String(t *testing.T) {
	h := &sqlast.IndexHint{
		Type:    sqlast.UseOp,
		ForType: sqlast.NoForType,
		Indexes: []sqlast.ColIdent{"idx1", "idx2"},
	}

	assertEqual(t, "USE INDEX (idx1, idx2)", h.String())

	h2 := &sqlast.IndexHint{
		Type:    sqlast.ForceOp,
		ForType: sqlast.JoinForType,
		Indexes: []sqlast.ColIdent{"idx1"},
	}

	assertEqual(t, "FORCE INDEX FOR JOIN (idx1)", h2.String())
}

func TestColumns_String(t *testing.T) {
	c := sqlast.Columns{"id", "name", "email"}
	assertEqual(t, "id, name, email", c.String())
}

func TestValues_String(t *testing.T) {
	v := sqlast.Values{
		{lit("1"), lit("'a'")},
		{lit("2"), lit("'b'")},
	}

	assertEqual(t, "VALUES (1, 'a'), (2, 'b')", v.String())
}

func TestUpdateExpr_String(t *testing.T) {
	ue := &sqlast.UpdateExpr{
		Name: &sqlast.ColName{Name: "status"},
		Expr: lit("1"),
	}

	assertEqual(t, "status = 1", ue.String())
}

func TestOnDup_String(t *testing.T) {
	assertEqual(t, "", sqlast.OnDup(nil).String())

	od := sqlast.OnDup{
		{Name: &sqlast.ColName{Name: "a"}, Expr: lit("1")},
	}

	assertEqual(t, "ON DUPLICATE KEY UPDATE a = 1", od.String())
}

func TestOverClause_String(t *testing.T) {
	tests := []struct {
		name string
		oc   *sqlast.OverClause
		want string
	}{
		{"nil", nil, ""},
		{"empty spec", &sqlast.OverClause{}, "OVER ()"},
		{"named window", &sqlast.OverClause{WindowName: "w"}, "OVER w"},
		{"with partition", &sqlast.OverClause{
			WindowSpec: &sqlast.WindowSpecification{
				PartitionClause: exprs("dept"),
			},
		}, "OVER (PARTITION BY dept)"},
		{"empty window spec", &sqlast.OverClause{
			WindowSpec: &sqlast.WindowSpecification{},
		}, "OVER ()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.oc.String())
		})
	}
}

func TestWindowSpecification_String(t *testing.T) {
	tests := []struct {
		name string
		ws   *sqlast.WindowSpecification
		want string
	}{
		{"nil", nil, ""},
		{"empty", &sqlast.WindowSpecification{}, ""},
		{"partition only", &sqlast.WindowSpecification{
			PartitionClause: exprs("dept"),
		}, "PARTITION BY dept"},
		{"order only", &sqlast.WindowSpecification{
			OrderClause: sqlast.OrderBy{{Expr: lit("id"), Direction: sqlast.DescOrder}},
		}, "ORDER BY id DESC"},
		{"partition and order", &sqlast.WindowSpecification{
			PartitionClause: exprs("dept"),
			OrderClause:     sqlast.OrderBy{{Expr: lit("id")}},
		}, "PARTITION BY dept ORDER BY id"},
		{"with frame", &sqlast.WindowSpecification{
			OrderClause: sqlast.OrderBy{{Expr: lit("id")}},
			FrameClause: &sqlast.FrameClause{
				Unit:  sqlast.FrameRowsType,
				Start: &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType},
			},
		}, "ORDER BY id ROWS UNBOUNDED PRECEDING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.ws.String())
		})
	}
}

func TestFrameClause_String(t *testing.T) {
	tests := []struct {
		name string
		fc   *sqlast.FrameClause
		want string
	}{
		{"nil", nil, ""},
		{"rows unbounded preceding", &sqlast.FrameClause{
			Unit:  sqlast.FrameRowsType,
			Start: &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType},
		}, "ROWS UNBOUNDED PRECEDING"},
		{"range between", &sqlast.FrameClause{
			Unit:  sqlast.FrameRangeType,
			Start: &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType},
			End:   &sqlast.FramePoint{Type: sqlast.CurrentRowType},
		}, "RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.fc.String())
		})
	}
}

func TestFrameUnitType_String(t *testing.T) {
	assertEqual(t, "ROWS", sqlast.FrameRowsType.String())
	assertEqual(t, "RANGE", sqlast.FrameRangeType.String())
}

func TestFramePoint_String(t *testing.T) {
	tests := []struct {
		fp   *sqlast.FramePoint
		want string
	}{
		{&sqlast.FramePoint{Type: sqlast.CurrentRowType}, "CURRENT ROW"},
		{&sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType}, "UNBOUNDED PRECEDING"},
		{&sqlast.FramePoint{Type: sqlast.UnboundedFollowingType}, "UNBOUNDED FOLLOWING"},
		{&sqlast.FramePoint{Type: sqlast.ExprPrecedingType, Expr: lit("3")}, "3 PRECEDING"},
		{&sqlast.FramePoint{Type: sqlast.ExprFollowingType, Expr: lit("5")}, "5 FOLLOWING"},
	}

	for _, tt := range tests {
		assertEqual(t, tt.want, tt.fp.String())
	}
}

func TestArgumentLessWindowExprType_String(t *testing.T) {
	tests := []struct {
		t    sqlast.ArgumentLessWindowExprType
		want string
	}{
		{sqlast.CumeDistExprType, "CUME_DIST"},
		{sqlast.DenseRankExprType, "DENSE_RANK"},
		{sqlast.PercentRankExprType, "PERCENT_RANK"},
		{sqlast.RankExprType, "RANK"},
		{sqlast.RowNumberExprType, "ROW_NUMBER"},
		{sqlast.ArgumentLessWindowExprType(99), "ROW_NUMBER"},
	}

	for _, tt := range tests {
		assertEqual(t, tt.want, tt.t.String())
	}
}

func TestFirstOrLastValueExprType_String(t *testing.T) {
	assertEqual(t, "FIRST_VALUE", sqlast.FirstValueExprType.String())
	assertEqual(t, "LAST_VALUE", sqlast.LastValueExprType.String())
}

func TestLagLeadExprType_String(t *testing.T) {
	assertEqual(t, "LAG", sqlast.LagExprType.String())
	assertEqual(t, "LEAD", sqlast.LeadExprType.String())
}

func TestNullTreatmentType_String(t *testing.T) {
	assertEqual(t, "RESPECT NULLS", sqlast.RespectNullsType.String())
	assertEqual(t, "IGNORE NULLS", sqlast.IgnoreNullsType.String())
}

func TestNullTreatmentClause_String(t *testing.T) {
	assertEqual(t, "", (*sqlast.NullTreatmentClause)(nil).String())
	assertEqual(t, "IGNORE NULLS", (&sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType}).String())
}

func TestFromFirstLastType_String(t *testing.T) {
	assertEqual(t, "FROM FIRST", sqlast.FromFirstType.String())
	assertEqual(t, "FROM LAST", sqlast.FromLastType.String())
}

func TestFromFirstLastClause_String(t *testing.T) {
	assertEqual(t, "", (*sqlast.FromFirstLastClause)(nil).String())
	assertEqual(t, "FROM LAST", (&sqlast.FromFirstLastClause{Type: sqlast.FromLastType}).String())
}

// helpers

func lit(s string) *sqlast.Literal {
	return &sqlast.Literal{Val: s}
}

func exprs(vals ...string) []sqlast.Expr {
	result := make([]sqlast.Expr, len(vals))
	for i, v := range vals {
		result[i] = lit(v)
	}

	return result
}
