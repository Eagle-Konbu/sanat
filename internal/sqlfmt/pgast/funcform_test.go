package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestTrimSide_String(t *testing.T) {
	assertEqual(t, "", pgast.TrimDefault.String())
	assertEqual(t, "BOTH", pgast.TrimBoth.String())
	assertEqual(t, "LEADING", pgast.TrimLeading.String())
	assertEqual(t, "TRAILING", pgast.TrimTrailing.String())
}

func TestFuncForms_String(t *testing.T) {
	tests := []struct {
		name string
		e    pgast.Expr
		want string
	}{
		{"extract", &pgast.ExtractExpr{Field: "year", Source: col("d")}, "EXTRACT(year FROM d)"},
		{"substring from for", &pgast.SubstringExpr{Expr: col("s"), From: lit("1"), For: lit("2")}, "SUBSTRING(s FROM 1 FOR 2)"},
		{"substring for", &pgast.SubstringExpr{Expr: col("s"), For: lit("2")}, "SUBSTRING(s FOR 2)"},
		{
			"substring similar",
			&pgast.SubstringExpr{Expr: col("s"), Similar: lit("'p'"), Escape: lit("'#'")},
			"SUBSTRING(s SIMILAR 'p' ESCAPE '#')",
		},
		{"position", &pgast.PositionExpr{Substring: lit("'b'"), Source: col("s")}, "POSITION('b' IN s)"},
		{
			"trim",
			&pgast.TrimExpr{Side: pgast.TrimBoth, Characters: lit("'x'"), From: true, Args: []pgast.Expr{col("s")}},
			"TRIM(BOTH 'x' FROM s)",
		},
		{"trim side only", &pgast.TrimExpr{Side: pgast.TrimLeading, Args: []pgast.Expr{col("s")}}, "TRIM(LEADING s)"},
		{
			"overlay",
			&pgast.OverlayExpr{Expr: col("s"), Placing: lit("'x'"), From: lit("2"), For: lit("3")},
			"OVERLAY(s PLACING 'x' FROM 2 FOR 3)",
		},
		{"overlay without for", &pgast.OverlayExpr{Expr: col("s"), Placing: lit("'x'"), From: lit("2")}, "OVERLAY(s PLACING 'x' FROM 2)"},
		{
			"overlaps",
			&pgast.OverlapsExpr{Left: pgast.ValTuple{col("a"), col("b")}, Right: pgast.ValTuple{col("c"), col("d")}},
			"(a, b) OVERLAPS (c, d)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.e.String())
		})
	}
}
