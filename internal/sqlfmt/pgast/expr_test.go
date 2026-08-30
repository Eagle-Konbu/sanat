package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestComparisonOperator_String(t *testing.T) {
	tests := []struct {
		op   pgast.ComparisonOperator
		want string
	}{
		{pgast.EqualOp, "="},
		{pgast.LessThanOp, "<"},
		{pgast.GreaterThanOp, ">"},
		{pgast.LessEqualOp, "<="},
		{pgast.GreaterEqualOp, ">="},
		{pgast.NotEqualOp, "<>"},
		{pgast.LikeOp, "LIKE"},
		{pgast.NotLikeOp, "NOT LIKE"},
		{pgast.InOp, "IN"},
		{pgast.NotInOp, "NOT IN"},
		{pgast.ComparisonOperator(99), "="},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.op.String())
		})
	}
}

func TestComparisonExpr_String(t *testing.T) {
	c := &pgast.ComparisonExpr{Operator: pgast.EqualOp, Left: col("id"), Right: lit("1")}
	assertEqual(t, "id = 1", c.String())
}

func TestILikeExpr_String(t *testing.T) {
	tests := []struct {
		name string
		i    *pgast.ILikeExpr
		want string
	}{
		{"positive", &pgast.ILikeExpr{Left: col("name"), Right: lit("'a%'")}, "name ILIKE 'a%'"},
		{"negated", &pgast.ILikeExpr{Not: true, Left: col("name"), Right: lit("'a%'")}, "name NOT ILIKE 'a%'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.i.String())
		})
	}
}

func TestRangeCond_String(t *testing.T) {
	tests := []struct {
		name string
		r    *pgast.RangeCond
		want string
	}{
		{"positive", &pgast.RangeCond{Left: col("x"), From: lit("1"), To: lit("10")}, "x BETWEEN 1 AND 10"},
		{
			"negated",
			&pgast.RangeCond{Not: true, Left: col("x"), From: lit("1"), To: lit("10")},
			"x NOT BETWEEN 1 AND 10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.r.String())
		})
	}
}

func TestIsExpr_String(t *testing.T) {
	tests := []struct {
		name string
		i    *pgast.IsExpr
		want string
	}{
		{"is null", &pgast.IsExpr{Expr: col("x")}, "x IS NULL"},
		{"is not null", &pgast.IsExpr{Not: true, Expr: col("x")}, "x IS NOT NULL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.i.String())
		})
	}
}

func TestIsDistinctFromExpr_String(t *testing.T) {
	tests := []struct {
		name string
		i    *pgast.IsDistinctFromExpr
		want string
	}{
		{"positive", &pgast.IsDistinctFromExpr{Left: col("a"), Right: col("b")}, "a IS DISTINCT FROM b"},
		{
			"negated",
			&pgast.IsDistinctFromExpr{Not: true, Left: col("a"), Right: col("b")},
			"a IS NOT DISTINCT FROM b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.i.String())
		})
	}
}

func TestValTuple_String(t *testing.T) {
	v := pgast.ValTuple{lit("1"), lit("2")}
	assertEqual(t, "(1, 2)", v.String())
}

func TestArithmeticOperator_String(t *testing.T) {
	tests := []struct {
		op   pgast.ArithmeticOperator
		want string
	}{
		{pgast.PlusOp, "+"},
		{pgast.MinusOp, "-"},
		{pgast.MultOp, "*"},
		{pgast.DivOp, "/"},
		{pgast.ModOp, "%"},
		{pgast.ArithmeticOperator(99), "+"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.op.String())
		})
	}
}

func TestArithmeticExpr_String(t *testing.T) {
	a := &pgast.ArithmeticExpr{Operator: pgast.PlusOp, Left: col("a"), Right: col("b")}
	assertEqual(t, "a + b", a.String())
}

func TestUnaryOperator_String(t *testing.T) {
	assertEqual(t, "+", pgast.UPlusOp.String())
	assertEqual(t, "-", pgast.UMinusOp.String())
}

func TestUnaryExpr_String(t *testing.T) {
	u := &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: col("a")}
	assertEqual(t, "-a", u.String())
}

func TestAndExpr_String(t *testing.T) {
	a := &pgast.AndExpr{Left: col("a"), Right: col("b")}
	assertEqual(t, "a AND b", a.String())
}

func TestOrExpr_String(t *testing.T) {
	o := &pgast.OrExpr{Left: col("a"), Right: col("b")}
	assertEqual(t, "a OR b", o.String())
}

func TestNotExpr_String(t *testing.T) {
	n := &pgast.NotExpr{Expr: col("a")}
	assertEqual(t, "NOT a", n.String())
}

func TestCaseExpr_String(t *testing.T) {
	tests := []struct {
		name string
		c    *pgast.CaseExpr
		want string
	}{
		{
			"searched, no else",
			&pgast.CaseExpr{Whens: []*pgast.When{{Cond: col("a"), Val: lit("1")}}},
			"CASE WHEN a THEN 1 END",
		},
		{
			"simple with else",
			&pgast.CaseExpr{
				Expr:  col("x"),
				Whens: []*pgast.When{{Cond: lit("1"), Val: lit("'one'")}},
				Else:  lit("'other'"),
			},
			"CASE x WHEN 1 THEN 'one' ELSE 'other' END",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.c.String())
		})
	}
}

func TestExistsExpr_String(t *testing.T) {
	tests := []struct {
		name string
		e    *pgast.ExistsExpr
		want string
	}{
		{"basic", &pgast.ExistsExpr{Subquery: &pgast.Subquery{Select: selectStar()}}, "EXISTS (SELECT * FROM t)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.e.String())
		})
	}
}

func TestSubquery_String(t *testing.T) {
	tests := []struct {
		name string
		s    *pgast.Subquery
		want string
	}{
		{"basic", &pgast.Subquery{Select: selectStar()}, "(SELECT * FROM t)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.s.String())
		})
	}
}

func TestColName_String(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		assertEqual(t, "id", col("id").String())
	})

	t.Run("qualified", func(t *testing.T) {
		c := &pgast.ColName{Name: "id", Qualifier: pgast.TableName{Name: "users"}}
		assertEqual(t, "users.id", c.String())
	})
}

func TestLiteral_String(t *testing.T) {
	assertEqual(t, "42", lit("42").String())
}

func TestFuncExpr_String(t *testing.T) {
	tests := []struct {
		name string
		f    *pgast.FuncExpr
		want string
	}{
		{"plain call", &pgast.FuncExpr{Name: "lower", Args: []pgast.Expr{col("name")}}, "lower(name)"},
		{
			"qualified",
			&pgast.FuncExpr{Qualifier: "pg_catalog", Name: "lower", Args: []pgast.Expr{col("name")}},
			"pg_catalog.lower(name)",
		},
		{"count star", &pgast.FuncExpr{Name: "count", Star: true}, "count(*)"},
		{
			"distinct",
			&pgast.FuncExpr{Name: "count", Distinct: true, Args: []pgast.Expr{col("id")}},
			"count(DISTINCT id)",
		},
		{
			"filter",
			&pgast.FuncExpr{
				Name: "count", Star: true,
				Filter: &pgast.Where{Expr: col("active")},
			},
			"count(*) FILTER (WHERE active)",
		},
		{
			"over",
			&pgast.FuncExpr{
				Name: "row_number",
				Over: &pgast.WindowSpec{OrderBy: pgast.OrderBy{{Expr: col("id")}}},
			},
			"row_number() OVER (ORDER BY id)",
		},
		{
			"over bare window name, no parens",
			&pgast.FuncExpr{Name: "row_number", Over: &pgast.WindowSpec{Name: "w"}},
			"row_number() OVER w",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestParenExpr_String(t *testing.T) {
	p := &pgast.ParenExpr{Expr: col("a")}
	assertEqual(t, "(a)", p.String())
}

func TestCastExpr_String(t *testing.T) {
	tests := []struct {
		name string
		c    *pgast.CastExpr
		want string
	}{
		{"shorthand", &pgast.CastExpr{Expr: col("a"), Type: "int"}, "a::int"},
		{"explicit", &pgast.CastExpr{Expr: col("a"), Type: "int", Explicit: true}, "CAST(a AS int)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.c.String())
		})
	}
}

func TestArrayExpr_String(t *testing.T) {
	a := &pgast.ArrayExpr{Elems: []pgast.Expr{lit("1"), lit("2")}}
	assertEqual(t, "ARRAY[1, 2]", a.String())
}

func TestArraySubscriptExpr_String(t *testing.T) {
	tests := []struct {
		name string
		a    *pgast.ArraySubscriptExpr
		want string
	}{
		{"subscript", &pgast.ArraySubscriptExpr{Expr: col("a"), From: lit("1")}, "a[1]"},
		{"slice", &pgast.ArraySubscriptExpr{Expr: col("a"), From: lit("1"), To: lit("3")}, "a[1:3]"},
		{"open slice", &pgast.ArraySubscriptExpr{Expr: col("a"), To: lit("3")}, "a[:3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.a.String())
		})
	}
}

func TestJSONOpExpr_String(t *testing.T) {
	j := &pgast.JSONOpExpr{Left: col("data"), Op: pgast.JSONArrow, Right: lit("'key'")}
	assertEqual(t, "data -> 'key'", j.String())
	assertEqual(t, "->", pgast.JSONArrow.String())
}
