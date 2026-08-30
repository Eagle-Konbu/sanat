package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestAliasedTableExpr_String(t *testing.T) {
	tests := []struct {
		name string
		a    *pgast.AliasedTableExpr
		want string
	}{
		{"simple", &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}}, "users"},
		{
			"with alias",
			&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}, As: "u"},
			"users u",
		},
		{
			"lateral",
			&pgast.AliasedTableExpr{
				Expr:    &pgast.DerivedTable{Select: selectStar()},
				As:      "sub",
				Lateral: true,
			},
			"LATERAL (SELECT * FROM t) sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.a.String())
		})
	}
}

func TestJoinType_String(t *testing.T) {
	tests := []struct {
		jt   pgast.JoinType
		want string
	}{
		{pgast.JoinInner, "JOIN"},
		{pgast.JoinLeft, "LEFT JOIN"},
		{pgast.JoinRight, "RIGHT JOIN"},
		{pgast.JoinFull, "FULL JOIN"},
		{pgast.JoinCross, "CROSS JOIN"},
		{pgast.JoinNatural, "NATURAL JOIN"},
		{pgast.JoinNaturalLeft, "NATURAL LEFT JOIN"},
		{pgast.JoinNaturalRight, "NATURAL RIGHT JOIN"},
		{pgast.JoinNaturalFull, "NATURAL FULL JOIN"},
		{pgast.JoinType(99), "JOIN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.jt.String())
		})
	}
}

func TestJoinTableExpr_String(t *testing.T) {
	left := pgast.TableName{Name: "a"}
	right := pgast.TableName{Name: "b"}

	tests := []struct {
		name string
		j    *pgast.JoinTableExpr
		want string
	}{
		{
			"with ON",
			&pgast.JoinTableExpr{
				Left: &pgast.AliasedTableExpr{Expr: left}, Join: pgast.JoinInner,
				Right: &pgast.AliasedTableExpr{Expr: right},
				Condition: &pgast.JoinCondition{
					On: &pgast.ComparisonExpr{Operator: pgast.EqualOp, Left: col("a.id"), Right: col("b.id")},
				},
			},
			"a JOIN b ON a.id = b.id",
		},
		{
			"with USING",
			&pgast.JoinTableExpr{
				Left: &pgast.AliasedTableExpr{Expr: left}, Join: pgast.JoinLeft,
				Right:     &pgast.AliasedTableExpr{Expr: right},
				Condition: &pgast.JoinCondition{Using: pgast.Columns{"id"}},
			},
			"a LEFT JOIN b USING (id)",
		},
		{
			"without condition",
			&pgast.JoinTableExpr{
				Left: &pgast.AliasedTableExpr{Expr: left}, Join: pgast.JoinCross,
				Right: &pgast.AliasedTableExpr{Expr: right},
			},
			"a CROSS JOIN b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.j.String())
		})
	}
}

func TestParenTableExpr_String(t *testing.T) {
	p := &pgast.ParenTableExpr{Exprs: []pgast.TableExpr{
		&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "a"}},
		&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "b"}},
	}}
	assertEqual(t, "(a, b)", p.String())
}

func TestDerivedTable_String(t *testing.T) {
	tests := []struct {
		name string
		d    *pgast.DerivedTable
		want string
	}{
		{"basic", &pgast.DerivedTable{Select: selectStar()}, "(SELECT * FROM t)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.d.String())
		})
	}
}

func TestAliasedExpr_String(t *testing.T) {
	t.Run("without alias", func(t *testing.T) {
		a := &pgast.AliasedExpr{Expr: col("id")}
		assertEqual(t, "id", a.String())
	})

	t.Run("with alias", func(t *testing.T) {
		a := &pgast.AliasedExpr{Expr: col("id"), As: "user_id"}
		assertEqual(t, "id AS user_id", a.String())
	})
}

func TestStarExpr_String(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		assertEqual(t, "*", (&pgast.StarExpr{}).String())
	})

	t.Run("qualified", func(t *testing.T) {
		s := &pgast.StarExpr{TableName: pgast.TableName{Name: "users"}}
		assertEqual(t, "users.*", s.String())
	})
}
