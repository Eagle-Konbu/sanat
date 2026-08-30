package pgast_test

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

func lit(s string) *pgast.Literal { return &pgast.Literal{Val: s} }

func col(name string) *pgast.ColName { return &pgast.ColName{Name: pgast.ColIdent(name)} }

func selectStar() *pgast.Select {
	return &pgast.Select{
		SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{}},
		From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}},
	}
}
