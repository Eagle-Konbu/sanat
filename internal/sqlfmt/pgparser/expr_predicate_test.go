package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func assertExprRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	if got := parseExpr(t, input).String(); got != want {
		t.Errorf("ParseExpr(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseExpr_quantified(t *testing.T) {
	t.Run("structure", func(t *testing.T) {
		assertExpr(t, "a = ANY(b)", &pgast.QuantifiedExpr{
			Left:       col("a"),
			Operator:   pgast.Operator{Symbol: "="},
			Quantifier: pgast.QuantifierAny,
			Right:      col("b"),
		})
	})

	tests := []struct {
		name, in, want string
	}{
		{"any array", "x = ANY(arr)", "x = ANY (arr)"},
		{"some", "x <> SOME (arr)", "x <> SOME (arr)"},
		{"all", "x != ALL (ARRAY[1, 2])", "x <> ALL (ARRAY[1, 2])"},
		{"subquery", "x = ANY (SELECT id FROM t)", "x = ANY (SELECT id FROM t)"},
		{"set operation subquery", "x < ALL ((SELECT 1) UNION SELECT 2)", "x < ALL (SELECT 1 UNION SELECT 2)"},
		{"generic operator", "tags @> ANY (ARRAY['a'])", "tags @> ANY (ARRAY['a'])"},
		{"explicit operator", "a OPERATOR(pg_catalog.=) ANY (b)", "a OPERATOR(pg_catalog.=) ANY (b)"},
		{"like", "name LIKE ANY (patterns)", "name LIKE ANY (patterns)"},
		{"not ilike", "name NOT ILIKE ALL (patterns)", "name NOT ILIKE ALL (patterns)"},
		{"in a boolean expression", "a = ANY (b) AND c", "a = ANY (b) AND c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	// ANY and SOME are reserved keywords, so they are not function names.
	for _, in := range []string{"any(a)", "a = ANY", "a = ANY (", "some"} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}

func TestParseExpr_collateAndTimeZone(t *testing.T) {
	t.Run("collate binds looser than unary minus", func(t *testing.T) {
		assertExpr(t, `-a COLLATE "C"`, &pgast.CollateExpr{
			Expr:      &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: col("a")},
			Collation: pgast.TableName{Name: `"C"`},
		})
	})

	t.Run("at time zone binds tighter than ^", func(t *testing.T) {
		assertExpr(t, "a ^ b AT TIME ZONE 'UTC'", &pgast.ArithmeticExpr{
			Left:     col("a"),
			Operator: pgast.ExpOp,
			Right:    &pgast.AtTimeZoneExpr{Expr: col("b"), Zone: &pgast.Literal{Val: "'UTC'"}},
		})
	})

	tests := []struct {
		name, in, want string
	}{
		{"collate", `name COLLATE "de_DE"`, `name COLLATE "de_DE"`},
		{"qualified collation", `a COLLATE pg_catalog."default"`, `a COLLATE pg_catalog."default"`},
		{"collate in comparison", `a < b COLLATE "C"`, `a < b COLLATE "C"`},
		{"at time zone", "ts at time zone 'UTC'", "ts AT TIME ZONE 'UTC'"},
		{"chained at time zone", "ts AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Tokyo'", "ts AT TIME ZONE 'UTC' AT TIME ZONE 'Asia/Tokyo'"},
		{"column named at", "at + 1", "at + 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{"a COLLATE", "a AT TIME ZONE", "collate"} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}

func TestParseExpr_predicateForms(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"similar to", "a SIMILAR TO 'x%'", "a SIMILAR TO 'x%'"},
		{"not similar to with escape", "a NOT SIMILAR TO 'x!%' ESCAPE '!'", "a NOT SIMILAR TO 'x!%' ESCAPE '!'"},
		{"between symmetric", "a BETWEEN SYMMETRIC 10 AND 1", "a BETWEEN SYMMETRIC 10 AND 1"},
		{"not between symmetric", "a NOT BETWEEN SYMMETRIC 1 AND 2", "a NOT BETWEEN SYMMETRIC 1 AND 2"},
		{"between asymmetric is the default", "a BETWEEN ASYMMETRIC 1 AND 2", "a BETWEEN 1 AND 2"},
		{"is normalized", "s IS NORMALIZED", "s IS NORMALIZED"},
		{"is not nfkc normalized", "s IS NOT nfkc NORMALIZED", "s IS NOT NFKC NORMALIZED"},
		{"column named normalized", "normalized IS TRUE", "normalized IS TRUE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	t.Run("similar to structure", func(t *testing.T) {
		assertExpr(t, "a SIMILAR TO b", &pgast.ComparisonExpr{Operator: pgast.SimilarOp, Left: col("a"), Right: col("b")})
	})

	for _, in := range []string{"a SIMILAR 'x'", "a NOT SIMILAR 'x'", "a IS nfc", "a BETWEEN SYMMETRIC 1"} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}

func TestParseExpr_arrayForms(t *testing.T) {
	t.Run("nested structure", func(t *testing.T) {
		assertExpr(t, "ARRAY[[1], []]", &pgast.ArrayExpr{Elems: []pgast.Expr{
			&pgast.ArrayExpr{Nested: true, Elems: []pgast.Expr{num("1")}},
			&pgast.ArrayExpr{Nested: true},
		}})
	})

	tests := []struct {
		name, in, want string
	}{
		{"array subquery", "ARRAY(SELECT id FROM t)", "ARRAY(SELECT id FROM t)"},
		{"array of a set operation", "ARRAY(SELECT 1 UNION SELECT 2)", "ARRAY(SELECT 1 UNION SELECT 2)"},
		{"multidimensional", "ARRAY[[1,2],[3,4]]", "ARRAY[[1, 2], [3, 4]]"},
		{"three dimensions", "ARRAY[[[1]]]", "ARRAY[[[1]]]"},
		{"nested array keyword", "ARRAY[ARRAY[1], ARRAY[2]]", "ARRAY[ARRAY[1], ARRAY[2]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{"ARRAY(1)", "ARRAY[[1], 2]", "ARRAY[[1]", "ARRAY(SELECT 1"} {
		t.Run("error "+in, func(t *testing.T) {
			assertExprError(t, in)
		})
	}
}
