package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestQuantifier_String(t *testing.T) {
	assertEqual(t, "ANY", pgast.QuantifierAny.String())
	assertEqual(t, "SOME", pgast.QuantifierSome.String())
	assertEqual(t, "ALL", pgast.QuantifierAll.String())
}

func TestQuantifiedExpr_String(t *testing.T) {
	array := &pgast.QuantifiedExpr{Left: col("a"), Operator: pgast.Operator{Symbol: "="}, Right: col("b")}
	assertEqual(t, "a = ANY (b)", array.String())

	sub := &pgast.QuantifiedExpr{
		Left:       col("a"),
		Operator:   pgast.Operator{Symbol: "<"},
		Quantifier: pgast.QuantifierAll,
		Right:      &pgast.Subquery{Select: selectID()},
	}
	assertEqual(t, "a < ALL (SELECT id)", sub.String())
}

func TestPredicateForms_String(t *testing.T) {
	tests := []struct {
		name string
		e    pgast.Expr
		want string
	}{
		{"similar to", &pgast.ComparisonExpr{Operator: pgast.SimilarOp, Left: col("a"), Right: lit("'x'")}, "a SIMILAR TO 'x'"},
		{"not similar to", &pgast.ComparisonExpr{Operator: pgast.NotSimilarOp, Left: col("a"), Right: lit("'x'")}, "a NOT SIMILAR TO 'x'"},
		{
			"not between symmetric",
			&pgast.RangeCond{Not: true, Symmetric: true, Left: col("a"), From: lit("1"), To: lit("2")},
			"a NOT BETWEEN SYMMETRIC 1 AND 2",
		},
		{"is normalized", &pgast.IsNormalizedExpr{Expr: col("s")}, "s IS NORMALIZED"},
		{"is not nfc normalized", &pgast.IsNormalizedExpr{Not: true, Expr: col("s"), Form: "NFC"}, "s IS NOT NFC NORMALIZED"},
		{"collate", &pgast.CollateExpr{Expr: col("a"), Collation: pgast.AnyName{pgast.ColIdent{Name: `"C"`}}}, `a COLLATE "C"`},
		{"at time zone", &pgast.AtTimeZoneExpr{Expr: col("t"), Zone: lit("'UTC'")}, "t AT TIME ZONE 'UTC'"},
		{"array subquery", &pgast.ArraySubquery{Select: selectID()}, "ARRAY(SELECT id)"},
		{"nested array", &pgast.ArrayExpr{Elems: []pgast.Expr{&pgast.ArrayExpr{Nested: true, Elems: []pgast.Expr{lit("1")}}}}, "ARRAY[[1]]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.e.String())
		})
	}
}

func selectID() *pgast.Select {
	return &pgast.Select{SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}}}
}
