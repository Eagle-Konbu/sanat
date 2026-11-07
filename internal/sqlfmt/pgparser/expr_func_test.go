package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestParseExpr_aggregateArguments(t *testing.T) {
	t.Run("structure", func(t *testing.T) {
		assertExpr(t, "string_agg(x, ',' ORDER BY y DESC)", &pgast.FuncExpr{
			Name:    "string_agg",
			Args:    []pgast.Expr{col("x"), &pgast.Literal{Val: "','"}},
			OrderBy: pgast.OrderBy{{Expr: col("y"), Direction: pgast.DescOrder}},
		})
	})

	tests := []struct {
		name, in, want string
	}{
		{"aggregate order by", "string_agg(x, ',' ORDER BY y)", "string_agg(x, ',' ORDER BY y)"},
		{"distinct with order by", "array_agg(DISTINCT x ORDER BY x)", "array_agg(DISTINCT x ORDER BY x)"},
		{"within group", "percentile_cont(0.5) WITHIN GROUP (ORDER BY x)", "percentile_cont(0.5) WITHIN GROUP (ORDER BY x)"},
		{
			"within group with filter and over",
			"mode() WITHIN GROUP (ORDER BY x) FILTER (WHERE x > 0) OVER w",
			"mode() WITHIN GROUP (ORDER BY x) FILTER (WHERE x > 0) OVER w",
		},
		{"variadic only argument", "f(VARIADIC arr)", "f(VARIADIC arr)"},
		{"variadic last argument", "concat_ws(',', VARIADIC ARRAY['a'])", "concat_ws(',', VARIADIC ARRAY['a'])"},
		{"named variadic argument", "f(VARIADIC a => arr)", "f(VARIADIC a => arr)"},
		{"all is a no-op", "count(ALL x)", "count(x)"},
		{"whole-row argument", "count(t.*)", "count(t.*)"},
		{"qualified whole-row argument", "row_to_json(s.t.*)", "row_to_json(s.t.*)"},
		{"column named within", "within + 1", "within + 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{
		"f(VARIADIC a, b)", "f(x ORDER BY)", "f(x) WITHIN GROUP (x)", "f(x) WITHIN GROUP", "variadic",
	} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}

func TestParseExpr_indirection(t *testing.T) {
	t.Run("three-part column", func(t *testing.T) {
		assertExpr(t, "s.t.c", &pgast.ColName{
			Qualifier: pgast.TableName{Qualifier: "s", Name: "t"},
			Name:      "c",
		})
	})

	t.Run("field of a parenthesized expression", func(t *testing.T) {
		assertExpr(t, "(x).f", &pgast.FieldSelectExpr{Expr: &pgast.ParenExpr{Expr: col("x")}, Field: "f"})
	})

	tests := []struct {
		name, in, want string
	}{
		{"four-part reference", "d.s.t.c", "d.s.t.c"},
		{"all fields", "(x).*", "(x).*"},
		{"nested fields", "(x).a.b", "(x).a.b"},
		{"field of a subscript", "arr[1].f", "arr[1].f"},
		{"field of a function result", "(f(x)).y", "(f(x)).y"},
		{"keyword field", "(x).from", "(x).from"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{"f(x).y", "(x).", "1 .a"} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}
