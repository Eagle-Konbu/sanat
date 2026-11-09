package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestFromItems_String(t *testing.T) {
	call := &pgast.FuncExpr{Name: pgast.ColIdent{Name: "f"}, Args: []pgast.Expr{lit("1")}}

	tests := []struct {
		name string
		n    pgast.SQLNode
		want string
	}{
		{"func table", &pgast.AliasedTableExpr{Expr: &pgast.FuncTable{Funcs: []pgast.Expr{call}}, As: pgast.TableIdent{Name: "x"}}, "f(1) x"},
		{
			"rows from with ordinality and columns",
			&pgast.AliasedTableExpr{
				Expr:    &pgast.FuncTable{Funcs: []pgast.Expr{call, call}, RowsFrom: true, WithOrdinality: true},
				As:      pgast.TableIdent{Name: "r"},
				Columns: pgast.Columns{pgast.ColIdent{Name: "a"}, pgast.ColIdent{Name: "n"}},
				Lateral: true,
			},
			"LATERAL ROWS FROM (f(1), f(1)) WITH ORDINALITY r(a, n)",
		},
		{
			"tablesample",
			&pgast.AliasedTableExpr{
				Expr:   pgast.TableName{Name: pgast.TableIdent{Name: "t"}},
				Sample: &pgast.TableSample{Method: pgast.TableName{Name: pgast.TableIdent{Name: "system"}}, Args: []pgast.Expr{lit("1")}, Repeatable: lit("2")},
			},
			"t TABLESAMPLE system (1) REPEATABLE (2)",
		},
		{
			"paren table columns",
			&pgast.ParenTableExpr{Exprs: []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "t"}}}}, As: pgast.TableIdent{Name: "p"}, Columns: pgast.Columns{pgast.ColIdent{Name: "a"}}},
			"(t) p(a)",
		},
		{
			"join using alias",
			&pgast.JoinTableExpr{
				Left:      &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "a"}}},
				Right:     &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "b"}}},
				Condition: &pgast.JoinCondition{Using: pgast.Columns{pgast.ColIdent{Name: "id"}}, UsingAlias: pgast.TableIdent{Name: "j"}},
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
