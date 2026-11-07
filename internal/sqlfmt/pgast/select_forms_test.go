package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestSelect_String_queryForms(t *testing.T) {
	tests := []struct {
		name string
		s    *pgast.Select
		want string
	}{
		{
			"values with trailing clauses",
			&pgast.Select{Values: [][]pgast.Expr{{lit("1")}, {lit("2")}}, OrderBy: pgast.OrderBy{{Expr: lit("1")}}},
			"VALUES (1), (2) ORDER BY 1",
		},
		{"table", &pgast.Select{Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}, Only: true}}, "TABLE ONLY t"},
		{"empty select list", &pgast.Select{From: []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}}}, "SELECT FROM t"},
		{"bare select", &pgast.Select{}, "SELECT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.s.String())
		})
	}
}
