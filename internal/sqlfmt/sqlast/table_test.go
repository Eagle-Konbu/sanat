package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestAliasedTableExpr_String(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		e := &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}}
		assertEqual(t, "users", e.String())
	})

	t.Run("with alias", func(t *testing.T) {
		e := &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}, As: "u"}
		assertEqual(t, "users u", e.String())
	})

	t.Run("with hints", func(t *testing.T) {
		e := &sqlast.AliasedTableExpr{
			Expr: sqlast.TableName{Name: "users"},
			Hints: sqlast.IndexHints{
				{Type: sqlast.UseOp, Indexes: []sqlast.ColIdent{"idx1"}},
			},
		}

		assertEqual(t, "users USE INDEX (idx1)", e.String())
	})
}

func TestJoinTableExpr_String(t *testing.T) {
	t.Run("with ON", func(t *testing.T) {
		cond := &sqlast.ComparisonExpr{
			Operator: sqlast.EqualOp,
			Left:     &sqlast.ColName{Qualifier: sqlast.TableName{Name: "u"}, Name: "id"},
			Right:    &sqlast.ColName{Qualifier: sqlast.TableName{Name: "o"}, Name: "user_id"},
		}
		e := &sqlast.JoinTableExpr{
			LeftExpr:  &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}, As: "u"},
			Join:      sqlast.LeftJoinType,
			RightExpr: &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "orders"}, As: "o"},
			Condition: &sqlast.JoinCondition{On: cond},
		}

		assertEqual(t, "users u LEFT JOIN orders o ON u.id = o.user_id", e.String())
	})

	t.Run("with USING", func(t *testing.T) {
		e := &sqlast.JoinTableExpr{
			LeftExpr:  &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}, As: "u"},
			Join:      sqlast.NormalJoinType,
			RightExpr: &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "orders"}, As: "o"},
			Condition: &sqlast.JoinCondition{Using: sqlast.Columns{"id", "tenant_id"}},
		}

		assertEqual(t, "users u JOIN orders o USING (id, tenant_id)", e.String())
	})

	t.Run("without condition", func(t *testing.T) {
		e := &sqlast.JoinTableExpr{
			LeftExpr:  &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "a"}},
			Join:      sqlast.CrossJoinType,
			RightExpr: &sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "b"}},
		}

		assertEqual(t, "a CROSS JOIN b", e.String())
	})
}

func TestParenTableExpr_String(t *testing.T) {
	e := &sqlast.ParenTableExpr{
		Exprs: []sqlast.TableExpr{
			&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "a"}},
			&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "b"}},
		},
	}

	assertEqual(t, "(a, b)", e.String())
}

func TestDerivedTable_String(t *testing.T) {
	d := &sqlast.DerivedTable{
		Select: &sqlast.Select{
			SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}},
		},
	}

	assertEqual(t, "(SELECT 1)", d.String())
}

func TestAliasedExpr_String(t *testing.T) {
	t.Run("without alias", func(t *testing.T) {
		e := &sqlast.AliasedExpr{Expr: lit("id")}
		assertEqual(t, "id", e.String())
	})

	t.Run("with alias", func(t *testing.T) {
		e := &sqlast.AliasedExpr{Expr: lit("id"), As: "user_id"}
		assertEqual(t, "id AS user_id", e.String())
	})
}

func TestStarExpr_String(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		e := &sqlast.StarExpr{}
		assertEqual(t, "*", e.String())
	})

	t.Run("qualified", func(t *testing.T) {
		e := &sqlast.StarExpr{TableName: sqlast.TableName{Name: "users"}}
		assertEqual(t, "users.*", e.String())
	})
}
