package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestFromItems_String(t *testing.T) {
	call := &pgast.FuncExpr{Name: "f", Args: []pgast.Expr{lit("1")}}

	tests := []struct {
		name string
		n    pgast.SQLNode
		want string
	}{
		{"func table", &pgast.AliasedTableExpr{Expr: &pgast.FuncTable{Funcs: []pgast.Expr{call}}, As: "x"}, "f(1) x"},
		{
			"rows from with ordinality and columns",
			&pgast.AliasedTableExpr{
				Expr:    &pgast.FuncTable{Funcs: []pgast.Expr{call, call}, RowsFrom: true, WithOrdinality: true},
				As:      "r",
				Columns: pgast.Columns{"a", "n"},
				Lateral: true,
			},
			"LATERAL ROWS FROM (f(1), f(1)) WITH ORDINALITY r(a, n)",
		},
		{
			"tablesample",
			&pgast.AliasedTableExpr{
				Expr:   pgast.TableName{Name: "t"},
				Sample: &pgast.TableSample{Method: pgast.TableName{Name: "system"}, Args: []pgast.Expr{lit("1")}, Repeatable: lit("2")},
			},
			"t TABLESAMPLE system (1) REPEATABLE (2)",
		},
		{
			"paren table columns",
			&pgast.ParenTableExpr{Exprs: []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}}, As: "p", Columns: pgast.Columns{"a"}},
			"(t) p(a)",
		},
		{
			"join using alias",
			&pgast.JoinTableExpr{
				Left:      &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "a"}},
				Right:     &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "b"}},
				Condition: &pgast.JoinCondition{Using: pgast.Columns{"id"}, UsingAlias: "j"},
			},
			"a JOIN b USING (id) AS j",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.n.String())
		})
	}
}
