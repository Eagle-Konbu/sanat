package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestWhere_String(t *testing.T) {
	var nilWhere *pgast.Where

	tests := []struct {
		name string
		w    *pgast.Where
		want string
	}{
		{"nil receiver", nilWhere, ""},
		{"nil expr", &pgast.Where{}, ""},
		{"set", &pgast.Where{Expr: col("active")}, "active"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}

func TestDistinctOn_String(t *testing.T) {
	var nilDistinctOn *pgast.DistinctOn

	tests := []struct {
		name string
		d    *pgast.DistinctOn
		want string
	}{
		{"nil receiver", nilDistinctOn, ""},
		{"empty", &pgast.DistinctOn{}, ""},
		{"exprs", &pgast.DistinctOn{Exprs: []pgast.Expr{col("a"), col("b")}}, "DISTINCT ON (a, b)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.d.String())
		})
	}
}

func TestGroupBy_String(t *testing.T) {
	var nilGroupBy *pgast.GroupBy

	tests := []struct {
		name string
		g    *pgast.GroupBy
		want string
	}{
		{"nil receiver", nilGroupBy, ""},
		{"empty", &pgast.GroupBy{}, ""},
		{"plain exprs", &pgast.GroupBy{Elements: []pgast.Expr{col("a"), col("b")}}, "GROUP BY a, b"},
		{
			"grouping sets",
			&pgast.GroupBy{Elements: []pgast.Expr{
				&pgast.GroupingSets{Sets: [][]pgast.Expr{{col("a")}, {}}},
			}},
			"GROUP BY GROUPING SETS ((a), ())",
		},
		{
			"cube and rollup",
			&pgast.GroupBy{Elements: []pgast.Expr{
				&pgast.Cube{Exprs: []pgast.Expr{col("a"), col("b")}},
				&pgast.Rollup{Exprs: []pgast.Expr{col("c")}},
			}},
			"GROUP BY CUBE (a, b), ROLLUP (c)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.g.String())
		})
	}
}

func TestNullsOrder_String(t *testing.T) {
	tests := []struct {
		n    pgast.NullsOrder
		want string
	}{
		{pgast.NullsDefault, ""},
		{pgast.NullsFirst, "NULLS FIRST"},
		{pgast.NullsLast, "NULLS LAST"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.n.String())
		})
	}
}

func TestOrder_String(t *testing.T) {
	tests := []struct {
		name string
		o    *pgast.Order
		want string
	}{
		{"asc default", &pgast.Order{Expr: col("id")}, "id"},
		{"desc", &pgast.Order{Expr: col("id"), Direction: pgast.DescOrder}, "id DESC"},
		{
			"desc nulls last",
			&pgast.Order{Expr: col("id"), Direction: pgast.DescOrder, Nulls: pgast.NullsLast},
			"id DESC NULLS LAST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.o.String())
		})
	}
}

func TestOrderBy_String(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assertEqual(t, "", pgast.OrderBy{}.String())
	})

	t.Run("multiple", func(t *testing.T) {
		ob := pgast.OrderBy{{Expr: col("a")}, {Expr: col("b"), Direction: pgast.DescOrder}}
		assertEqual(t, "ORDER BY a, b DESC", ob.String())
	})
}

func TestLimit_String(t *testing.T) {
	var nilLimit *pgast.Limit

	tests := []struct {
		name string
		l    *pgast.Limit
		want string
	}{
		{"nil receiver", nilLimit, ""},
		{"ALL", &pgast.Limit{}, "LIMIT ALL"},
		{"count", &pgast.Limit{Count: lit("10")}, "LIMIT 10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.l.String())
		})
	}
}

func TestOffset_String(t *testing.T) {
	var nilOffset *pgast.Offset

	tests := []struct {
		name string
		o    *pgast.Offset
		want string
	}{
		{"nil receiver", nilOffset, ""},
		{"nil start", &pgast.Offset{}, ""},
		{"set", &pgast.Offset{Start: lit("5")}, "OFFSET 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.o.String())
		})
	}
}

func TestFetch_String(t *testing.T) {
	var nilFetch *pgast.Fetch

	tests := []struct {
		name string
		f    *pgast.Fetch
		want string
	}{
		{"nil receiver", nilFetch, ""},
		{"first only, no count", &pgast.Fetch{}, "FETCH FIRST ROW ONLY"},
		{"next with ties", &pgast.Fetch{Count: lit("5"), Next: true, WithTies: true}, "FETCH NEXT 5 ROWS WITH TIES"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestLockStrength_String(t *testing.T) {
	tests := []struct {
		l    pgast.LockStrength
		want string
	}{
		{pgast.NoLock, ""},
		{pgast.ForUpdate, "FOR UPDATE"},
		{pgast.ForNoKeyUpdate, "FOR NO KEY UPDATE"},
		{pgast.ForShare, "FOR SHARE"},
		{pgast.ForKeyShare, "FOR KEY SHARE"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.l.String())
		})
	}
}

func TestLockWait_String(t *testing.T) {
	tests := []struct {
		w    pgast.LockWait
		want string
	}{
		{pgast.NoLockWait, ""},
		{pgast.NoWait, "NOWAIT"},
		{pgast.SkipLocked, "SKIP LOCKED"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}

func TestLock_String(t *testing.T) {
	var nilLock *pgast.Lock

	tests := []struct {
		name string
		l    *pgast.Lock
		want string
	}{
		{"nil receiver", nilLock, ""},
		{"no lock", &pgast.Lock{}, ""},
		{"update", &pgast.Lock{Strength: pgast.ForUpdate}, "FOR UPDATE"},
		{
			"update of with nowait",
			&pgast.Lock{
				Strength: pgast.ForUpdate,
				Of:       []pgast.TableName{{Name: "a"}, {Name: "b"}},
				Wait:     pgast.NoWait,
			},
			"FOR UPDATE OF a, b NOWAIT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.l.String())
		})
	}
}

func TestNamedWindow_String(t *testing.T) {
	n := &pgast.NamedWindow{Name: "w", Spec: &pgast.WindowSpec{PartitionBy: []pgast.Expr{col("a")}}}
	assertEqual(t, "w AS (PARTITION BY a)", n.String())
}

func TestWindowSpec_String(t *testing.T) {
	var nilSpec *pgast.WindowSpec

	tests := []struct {
		name string
		w    *pgast.WindowSpec
		want string
	}{
		{"nil receiver", nilSpec, ""},
		{"empty", &pgast.WindowSpec{}, ""},
		{"name only", &pgast.WindowSpec{Name: "w"}, "w"},
		{
			"full",
			&pgast.WindowSpec{
				Name:        "w",
				PartitionBy: []pgast.Expr{col("dept")},
				OrderBy:     pgast.OrderBy{{Expr: col("salary")}},
				Frame: &pgast.FrameClause{
					Unit:  pgast.FrameRows,
					Start: &pgast.FramePoint{Type: pgast.UnboundedPreceding},
				},
			},
			"w PARTITION BY dept ORDER BY salary ROWS UNBOUNDED PRECEDING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}

func TestFrameUnit_String(t *testing.T) {
	tests := []struct {
		f    pgast.FrameUnit
		want string
	}{
		{pgast.FrameRows, "ROWS"},
		{pgast.FrameRange, "RANGE"},
		{pgast.FrameGroups, "GROUPS"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestFramePoint_String(t *testing.T) {
	tests := []struct {
		name string
		f    *pgast.FramePoint
		want string
	}{
		{"current row", &pgast.FramePoint{Type: pgast.CurrentRow}, "CURRENT ROW"},
		{"unbounded preceding", &pgast.FramePoint{Type: pgast.UnboundedPreceding}, "UNBOUNDED PRECEDING"},
		{"unbounded following", &pgast.FramePoint{Type: pgast.UnboundedFollowing}, "UNBOUNDED FOLLOWING"},
		{"expr preceding", &pgast.FramePoint{Type: pgast.ExprPreceding, Expr: lit("1")}, "1 PRECEDING"},
		{"expr following", &pgast.FramePoint{Type: pgast.ExprFollowing, Expr: lit("1")}, "1 FOLLOWING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestFrameClause_String(t *testing.T) {
	var nilFrame *pgast.FrameClause

	tests := []struct {
		name string
		f    *pgast.FrameClause
		want string
	}{
		{"nil receiver", nilFrame, ""},
		{"nil start", &pgast.FrameClause{}, ""},
		{
			"start only",
			&pgast.FrameClause{Unit: pgast.FrameRows, Start: &pgast.FramePoint{Type: pgast.CurrentRow}},
			"ROWS CURRENT ROW",
		},
		{
			"start and end",
			&pgast.FrameClause{
				Unit:  pgast.FrameRange,
				Start: &pgast.FramePoint{Type: pgast.UnboundedPreceding},
				End:   &pgast.FramePoint{Type: pgast.CurrentRow},
			},
			"RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestReturning_String(t *testing.T) {
	var nilReturning *pgast.Returning

	tests := []struct {
		name string
		r    *pgast.Returning
		want string
	}{
		{"nil receiver", nilReturning, ""},
		{"empty", &pgast.Returning{}, ""},
		{
			"exprs",
			&pgast.Returning{Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}}},
			"RETURNING id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.r.String())
		})
	}
}

func TestUpdateExpr_String(t *testing.T) {
	u := &pgast.UpdateExpr{Name: col("name"), Expr: lit("'bob'")}
	assertEqual(t, "name = 'bob'", u.String())
}

func TestOnConflict_String(t *testing.T) {
	var nilOnConflict *pgast.OnConflict

	tests := []struct {
		name string
		o    *pgast.OnConflict
		want string
	}{
		{"nil receiver", nilOnConflict, ""},
		{"do nothing", &pgast.OnConflict{DoNothing: true}, "ON CONFLICT DO NOTHING"},
		{
			"columns do nothing",
			&pgast.OnConflict{Columns: pgast.Columns{"id"}, DoNothing: true},
			"ON CONFLICT (id) DO NOTHING",
		},
		{
			"constraint do update",
			&pgast.OnConflict{
				Constraint: "users_pkey",
				Set:        []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'bob'")}},
			},
			"ON CONFLICT ON CONSTRAINT users_pkey DO UPDATE SET name = 'bob'",
		},
		{
			"do update with where",
			&pgast.OnConflict{
				Columns: pgast.Columns{"id"},
				Set:     []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'bob'")}},
				Where:   &pgast.Where{Expr: col("active")},
			},
			"ON CONFLICT (id) DO UPDATE SET name = 'bob' WHERE active",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.o.String())
		})
	}
}
