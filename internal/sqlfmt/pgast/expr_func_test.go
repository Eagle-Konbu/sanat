package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestFuncExpr_String_aggregateForms(t *testing.T) {
	tests := []struct {
		name string
		f    *pgast.FuncExpr
		want string
	}{
		{"variadic only argument", &pgast.FuncExpr{Name: "f", Args: []pgast.Expr{col("a")}, Variadic: true}, "f(VARIADIC a)"},
		{
			"variadic last argument",
			&pgast.FuncExpr{Name: "f", Args: []pgast.Expr{col("a"), col("b")}, Variadic: true},
			"f(a, VARIADIC b)",
		},
		{
			"order by",
			&pgast.FuncExpr{Name: "string_agg", Args: []pgast.Expr{col("a")}, OrderBy: pgast.OrderBy{{Expr: col("b")}}},
			"string_agg(a ORDER BY b)",
		},
		{
			"within group",
			&pgast.FuncExpr{Name: "mode", WithinGroup: pgast.OrderBy{{Expr: col("b")}}},
			"mode() WITHIN GROUP (ORDER BY b)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.f.String())
		})
	}
}

func TestFieldSelectExpr_String(t *testing.T) {
	assertEqual(t, "(x).f", (&pgast.FieldSelectExpr{Expr: &pgast.ParenExpr{Expr: col("x")}, Field: "f"}).String())
	assertEqual(t, "t.*", (&pgast.FieldSelectExpr{Expr: col("t"), Star: true}).String())
}
