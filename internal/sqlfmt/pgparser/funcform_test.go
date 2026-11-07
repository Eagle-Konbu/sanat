package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestParseExpr_specialFunctionForms(t *testing.T) {
	t.Run("extract structure", func(t *testing.T) {
		assertExpr(t, "EXTRACT(year FROM d)", &pgast.ExtractExpr{Field: "year", Source: col("d")})
	})

	tests := []struct {
		name, in, want string
	}{
		{"extract", "extract(YEAR from d)", "EXTRACT(year FROM d)"},
		{"extract string field", "EXTRACT('epoch' FROM d)", "EXTRACT('epoch' FROM d)"},
		{"substring from for", "SUBSTRING(s FROM 1 FOR 2)", "SUBSTRING(s FROM 1 FOR 2)"},
		{"substring for from", "substring(s for 2 from 1)", "SUBSTRING(s FROM 1 FOR 2)"},
		{"substring from", "SUBSTRING(s FROM 'a.c')", "SUBSTRING(s FROM 'a.c')"},
		{"substring for", "SUBSTRING(s FOR 3)", "SUBSTRING(s FOR 3)"},
		{"substring similar", "SUBSTRING(s SIMILAR '%#\"a#\"%' ESCAPE '#')", "SUBSTRING(s SIMILAR '%#\"a#\"%' ESCAPE '#')"},
		{"substring plain arguments", "substring(s, 1, 2)", "substring(s, 1, 2)"},
		{"substring no arguments", "substring()", "substring()"},
		{"position", "POSITION('b' IN s)", "POSITION('b' IN s)"},
		{"position with operators", "position(a || b IN c || d)", "POSITION(a || b IN c || d)"},
		{"trim both characters from", "TRIM(BOTH 'x' FROM s)", "TRIM(BOTH 'x' FROM s)"},
		{"trim leading from", "trim(leading from s)", "TRIM(LEADING FROM s)"},
		{"trim trailing source", "TRIM(TRAILING s)", "TRIM(TRAILING s)"},
		{"trim characters from", "TRIM('x' FROM s)", "TRIM('x' FROM s)"},
		{"trim from", "TRIM(FROM s)", "TRIM(FROM s)"},
		{"trim plain arguments", "trim(s, 'x')", "trim(s, 'x')"},
		{"overlay", "OVERLAY(s PLACING 'x' FROM 2 FOR 3)", "OVERLAY(s PLACING 'x' FROM 2 FOR 3)"},
		{"overlay without for", "overlay(s placing 'x' from 2)", "OVERLAY(s PLACING 'x' FROM 2)"},
		{"overlay plain arguments", "overlay(a, b, 1)", "overlay(a, b, 1)"},
		{"qualified name is an ordinary call", "pg_catalog.substring(s, 1)", "pg_catalog.substring(s, 1)"},
		{"column named trim", "trim + 1", "trim + 1"},
		{"overlaps", "(a, b) OVERLAPS (c, d)", "(a, b) OVERLAPS (c, d)"},
		{"overlaps with row", "ROW(a, b) OVERLAPS ROW(c, d) AND x", "ROW(a, b) OVERLAPS ROW(c, d) AND x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{
		"EXTRACT(year d)", "POSITION(a, b)", "SUBSTRING(s SIMILAR 'x')", "OVERLAY(s PLACING 'x')",
		"TRIM(BOTH)", "(a, b, c) OVERLAPS (d, e)", "a OVERLAPS (b, c)", "(a, b) OVERLAPS c",
	} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}
