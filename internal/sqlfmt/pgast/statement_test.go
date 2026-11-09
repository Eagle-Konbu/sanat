package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestSelect_String(t *testing.T) {
	tests := []struct {
		name string
		s    *pgast.Select
		want string
	}{
		{"star", selectStar(), "SELECT * FROM t"},
		{
			"distinct",
			&pgast.Select{Distinct: true, SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{}}},
			"SELECT DISTINCT *",
		},
		{
			"distinct on",
			&pgast.Select{
				DistinctOn:  &pgast.DistinctOn{Exprs: []pgast.Expr{col("id")}},
				SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{}},
			},
			"SELECT DISTINCT ON (id) *",
		},
		{
			"full clauses",
			&pgast.Select{
				With:        &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "t"}}}},
				Where:       &pgast.Where{Expr: col("active")},
				GroupBy:     &pgast.GroupBy{Elements: []pgast.Expr{col("id")}},
				Having:      &pgast.Where{Expr: col("cond")},
				Window:      []*pgast.NamedWindow{{Name: pgast.ColIdent{Name: "w"}, Spec: &pgast.WindowSpec{}}},
				OrderBy:     pgast.OrderBy{{Expr: col("id")}},
				Limit:       &pgast.Limit{Count: lit("10")},
				Offset:      &pgast.Offset{Start: lit("5")},
				Fetch:       &pgast.Fetch{Count: lit("1")},
				Locks: []*pgast.Lock{
					{Strength: pgast.ForUpdate, Of: []pgast.TableName{{Name: pgast.TableIdent{Name: "t"}}}},
					{Strength: pgast.ForShare},
				},
			},
			"WITH cte AS (SELECT * FROM t) SELECT id FROM t WHERE active GROUP BY id HAVING cond WINDOW w AS () ORDER BY id LIMIT 10 OFFSET 5 FETCH FIRST 1 ROW ONLY FOR UPDATE OF t FOR SHARE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.s.String())
		})
	}
}

func TestInsert_String(t *testing.T) {
	tests := []struct {
		name string
		ins  *pgast.Insert
		want string
	}{
		{
			"values",
			&pgast.Insert{
				Table:   pgast.TableName{Name: pgast.TableIdent{Name: "users"}},
				Columns: []pgast.Expr{col("id"), col("name")},
				Rows:    pgast.Values{{lit("1"), lit("'a'")}},
			},
			"INSERT INTO users (id, name) VALUES (1, 'a')",
		},
		{
			"with alias, on conflict, returning",
			&pgast.Insert{
				With:       &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
				Table:      pgast.TableName{Name: pgast.TableIdent{Name: "users"}},
				Alias:      pgast.TableIdent{Name: "u"},
				Rows:       pgast.DefaultValues{},
				OnConflict: &pgast.OnConflict{DoNothing: true},
				Returning:  &pgast.Returning{Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}}},
			},
			"WITH cte AS (SELECT * FROM t) INSERT INTO users AS u DEFAULT VALUES ON CONFLICT DO NOTHING RETURNING id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.ins.String())
		})
	}
}

func TestDefaultValues_String(t *testing.T) {
	assertEqual(t, "DEFAULT VALUES", pgast.DefaultValues{}.String())
}

func TestValues_String(t *testing.T) {
	v := pgast.Values{{lit("1"), lit("2")}, {lit("3"), lit("4")}}
	assertEqual(t, "VALUES (1, 2), (3, 4)", v.String())
}

func TestUpdate_String(t *testing.T) {
	tests := []struct {
		name string
		u    *pgast.Update
		want string
	}{
		{
			"minimal",
			&pgast.Update{
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
			},
			"UPDATE users SET name = 'a'",
		},
		{
			"with from, where, returning",
			&pgast.Update{
				With:  &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
				From:  []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "other"}}}},
				Where: &pgast.Where{Expr: col("active")},
				Returning: &pgast.Returning{
					Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				},
			},
			"WITH cte AS (SELECT * FROM t) UPDATE users SET name = 'a' FROM other WHERE active RETURNING id",
		},
		{
			"only with alias",
			&pgast.Update{
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}, As: pgast.TableIdent{Name: "u"}, Only: true},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
			},
			"UPDATE ONLY users u SET name = 'a'",
		},
		{
			"descendants without alias",
			&pgast.Update{
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}, IncludeDescendants: true},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
			},
			"UPDATE users * SET name = 'a'",
		},
		{
			"set alias keeps as",
			&pgast.Update{
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}, As: pgast.TableIdent{Name: "set"}, Only: true},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
			},
			"UPDATE ONLY users AS set SET name = 'a'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.u.String())
		})
	}
}

func TestDelete_String(t *testing.T) {
	tests := []struct {
		name string
		d    *pgast.Delete
		want string
	}{
		{
			"minimal",
			&pgast.Delete{Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}}},
			"DELETE FROM users",
		},
		{
			"with using, where, returning",
			&pgast.Delete{
				With:  &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}},
				Using: []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "other"}}}},
				Where: &pgast.Where{Expr: col("active")},
				Returning: &pgast.Returning{
					Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				},
			},
			"WITH cte AS (SELECT * FROM t) DELETE FROM users USING other WHERE active RETURNING id",
		},
		{
			"set alias keeps as",
			&pgast.Delete{Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "users"}}, As: pgast.TableIdent{Name: "set"}}},
			"DELETE FROM users AS set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.d.String())
		})
	}
}

func TestSetOpType_String(t *testing.T) {
	tests := []struct {
		op   pgast.SetOpType
		want string
	}{
		{pgast.Union, "UNION"},
		{pgast.Intersect, "INTERSECT"},
		{pgast.Except, "EXCEPT"},
		{pgast.SetOpType(99), "UNION"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assertEqual(t, tt.want, tt.op.String())
		})
	}
}

func TestSetOp_String(t *testing.T) {
	tests := []struct {
		name string
		u    *pgast.SetOp
		want string
	}{
		{
			"basic union",
			&pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Union},
			"SELECT * FROM t UNION SELECT * FROM t",
		},
		{
			"union all",
			&pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Union, All: true},
			"SELECT * FROM t UNION ALL SELECT * FROM t",
		},
		{
			"nested right operand parenthesized",
			&pgast.SetOp{
				Left:  selectStar(),
				Op:    pgast.Union,
				Right: &pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Union},
			},
			"SELECT * FROM t UNION (SELECT * FROM t UNION SELECT * FROM t)",
		},
		{
			"nested left operand parenthesized",
			&pgast.SetOp{
				Left:  &pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Union},
				Op:    pgast.Intersect,
				Right: selectStar(),
			},
			"(SELECT * FROM t UNION SELECT * FROM t) INTERSECT SELECT * FROM t",
		},
		{
			"left-associative nested left operand unparenthesized",
			&pgast.SetOp{
				Left:  &pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Union},
				Op:    pgast.Except,
				Right: selectStar(),
			},
			"SELECT * FROM t UNION SELECT * FROM t EXCEPT SELECT * FROM t",
		},
		{
			"tighter-binding nested right operand unparenthesized",
			&pgast.SetOp{
				Left:  selectStar(),
				Op:    pgast.Union,
				Right: &pgast.SetOp{Left: selectStar(), Right: selectStar(), Op: pgast.Intersect},
			},
			"SELECT * FROM t UNION SELECT * FROM t INTERSECT SELECT * FROM t",
		},
		{
			"nested operand with its own order by parenthesized",
			&pgast.SetOp{
				Left: &pgast.SetOp{
					Left: selectStar(), Right: selectStar(), Op: pgast.Intersect,
					OrderBy: pgast.OrderBy{{Expr: col("id")}},
				},
				Op:    pgast.Union,
				Right: selectStar(),
			},
			"(SELECT * FROM t INTERSECT SELECT * FROM t ORDER BY id) UNION SELECT * FROM t",
		},
		{
			"select operands with locks or with clause parenthesized",
			func() *pgast.SetOp {
				left := selectStar()
				left.Locks = []*pgast.Lock{{Strength: pgast.ForUpdate}}

				right := selectStar()
				right.With = &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}}

				return &pgast.SetOp{Left: left, Right: right, Op: pgast.Union}
			}(),
			"(SELECT * FROM t FOR UPDATE) UNION (WITH cte AS (SELECT * FROM t) SELECT * FROM t)",
		},
		{
			"select operands with local limit parenthesized",
			func() *pgast.SetOp {
				left := selectStar()
				left.OrderBy = pgast.OrderBy{{Expr: col("id")}}
				left.Limit = &pgast.Limit{Count: lit("1")}

				right := selectStar()
				right.Limit = &pgast.Limit{Count: lit("2")}

				return &pgast.SetOp{Left: left, Right: right, Op: pgast.Union}
			}(),
			"(SELECT * FROM t ORDER BY id LIMIT 1) UNION (SELECT * FROM t LIMIT 2)",
		},
		{
			"with clauses",
			&pgast.SetOp{
				With:    &pgast.With{Recursive: true, CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
				Left:    selectStar(),
				Right:   selectStar(),
				Op:      pgast.Except,
				OrderBy: pgast.OrderBy{{Expr: col("id")}},
				Limit:   &pgast.Limit{Count: lit("1")},
				Offset:  &pgast.Offset{Start: lit("2")},
				Fetch:   &pgast.Fetch{Count: lit("3")},
			},
			"WITH RECURSIVE cte AS (SELECT * FROM t) SELECT * FROM t EXCEPT SELECT * FROM t ORDER BY id LIMIT 1 OFFSET 2 FETCH FIRST 3 ROW ONLY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.u.String())
		})
	}
}

func TestWith_String(t *testing.T) {
	var nilWith *pgast.With

	tests := []struct {
		name string
		w    *pgast.With
		want string
	}{
		{"nil receiver", nilWith, ""},
		{
			"simple",
			&pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
			"WITH cte AS (SELECT * FROM t)",
		},
		{
			"recursive",
			&pgast.With{Recursive: true, CTEs: []*pgast.CommonTableExpr{{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()}}},
			"WITH RECURSIVE cte AS (SELECT * FROM t)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}

func TestCommonTableExpr_String(t *testing.T) {
	tests := []struct {
		name string
		c    *pgast.CommonTableExpr
		want string
	}{
		{
			"without columns",
			&pgast.CommonTableExpr{Name: pgast.TableIdent{Name: "cte"}, Subquery: selectStar()},
			"cte AS (SELECT * FROM t)",
		},
		{
			"with columns",
			&pgast.CommonTableExpr{Name: pgast.TableIdent{Name: "cte"}, Columns: pgast.Columns{pgast.ColIdent{Name: "a"}, pgast.ColIdent{Name: "b"}}, Subquery: selectStar()},
			"cte (a, b) AS (SELECT * FROM t)",
		},
		{
			"materialized",
			&pgast.CommonTableExpr{Name: pgast.TableIdent{Name: "cte"}, Materialization: pgast.Materialized, Subquery: selectStar()},
			"cte AS MATERIALIZED (SELECT * FROM t)",
		},
		{
			"not materialized with columns",
			&pgast.CommonTableExpr{
				Name: pgast.TableIdent{Name: "cte"}, Columns: pgast.Columns{pgast.ColIdent{Name: "a"}}, Materialization: pgast.NotMaterialized, Subquery: selectStar(),
			},
			"cte (a) AS NOT MATERIALIZED (SELECT * FROM t)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.c.String())
		})
	}
}

func TestMerge_String(t *testing.T) {
	setName := []*pgast.UpdateExpr{{Name: col("name"), Expr: col("s.name")}}

	tests := []struct {
		name string
		m    *pgast.Merge
		want string
	}{
		{
			"update and insert",
			&pgast.Merge{
				Target: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "t"}}},
				Source: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "s"}}},
				On:     &pgast.ComparisonExpr{Operator: pgast.EqualOp, Left: col("t.id"), Right: col("s.id")},
				Whens: []*pgast.MergeWhen{
					{Match: pgast.MergeMatched, Action: &pgast.MergeUpdate{Set: setName}},
					{
						Match:  pgast.MergeNotMatched,
						Action: &pgast.MergeInsert{Columns: pgast.Columns{pgast.ColIdent{Name: "id"}}, Values: []pgast.Expr{col("s.id")}},
					},
				},
			},
			"MERGE INTO t USING s ON t.id = s.id " +
				"WHEN MATCHED THEN UPDATE SET name = s.name " +
				"WHEN NOT MATCHED THEN INSERT (id) VALUES (s.id)",
		},
		{
			"with, only target, and returning with aliases",
			&pgast.Merge{
				With: &pgast.With{CTEs: []*pgast.CommonTableExpr{
					{Name: pgast.TableIdent{Name: "s"}, Subquery: &pgast.Select{SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: lit("1")}}}},
				}},
				Target: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "t"}}, As: pgast.TableIdent{Name: "x"}, Only: true},
				Source: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: pgast.TableIdent{Name: "s"}}},
				On:     lit("TRUE"),
				Whens:  []*pgast.MergeWhen{{Match: pgast.MergeMatched, Action: pgast.MergeDelete{}}},
				Returning: &pgast.Returning{
					Aliases: []*pgast.ReturningAlias{
						{Kind: pgast.ReturningOld, Name: pgast.TableIdent{Name: "o"}},
						{Kind: pgast.ReturningNew, Name: pgast.TableIdent{Name: "n"}},
					},
					Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("o.id")}},
				},
			},
			"WITH s AS (SELECT 1) MERGE INTO ONLY t x USING s ON TRUE " +
				"WHEN MATCHED THEN DELETE RETURNING WITH (OLD AS o, NEW AS n) o.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.m.String())
		})
	}
}

func TestMergeWhen_String(t *testing.T) {
	tests := []struct {
		name string
		w    *pgast.MergeWhen
		want string
	}{
		{
			"matched with condition, do nothing",
			&pgast.MergeWhen{Match: pgast.MergeMatched, Condition: col("cond"), Action: pgast.MergeDoNothing{}},
			"WHEN MATCHED AND cond THEN DO NOTHING",
		},
		{
			"matched delete",
			&pgast.MergeWhen{Match: pgast.MergeMatched, Action: pgast.MergeDelete{}},
			"WHEN MATCHED THEN DELETE",
		},
		{
			"not matched by source do nothing",
			&pgast.MergeWhen{Match: pgast.MergeNotMatchedBySource, Action: pgast.MergeDoNothing{}},
			"WHEN NOT MATCHED BY SOURCE THEN DO NOTHING",
		},
		{
			"not matched by source update",
			&pgast.MergeWhen{
				Match:  pgast.MergeNotMatchedBySource,
				Action: &pgast.MergeUpdate{Set: []*pgast.UpdateExpr{{Name: col("x"), Expr: lit("1")}}},
			},
			"WHEN NOT MATCHED BY SOURCE THEN UPDATE SET x = 1",
		},
		{
			"not matched insert without columns",
			&pgast.MergeWhen{Match: pgast.MergeNotMatched, Action: &pgast.MergeInsert{Values: []pgast.Expr{lit("1")}}},
			"WHEN NOT MATCHED THEN INSERT VALUES (1)",
		},
		{
			"explicit by target",
			&pgast.MergeWhen{Match: pgast.MergeNotMatchedByTarget, Action: &pgast.MergeInsert{Values: []pgast.Expr{lit("1")}}},
			"WHEN NOT MATCHED BY TARGET THEN INSERT VALUES (1)",
		},
		{
			"insert default values",
			&pgast.MergeWhen{Match: pgast.MergeNotMatched, Action: &pgast.MergeInsert{}},
			"WHEN NOT MATCHED THEN INSERT DEFAULT VALUES",
		},
		{
			"insert overriding system value",
			&pgast.MergeWhen{Match: pgast.MergeNotMatched, Action: &pgast.MergeInsert{
				Columns:    pgast.Columns{pgast.ColIdent{Name: "id"}},
				Overriding: pgast.OverridingSystemValue,
				Values:     []pgast.Expr{lit("1")},
			}},
			"WHEN NOT MATCHED THEN INSERT (id) OVERRIDING SYSTEM VALUE VALUES (1)",
		},
		{
			"insert overriding user value",
			&pgast.MergeWhen{Match: pgast.MergeNotMatched, Action: &pgast.MergeInsert{
				Overriding: pgast.OverridingUserValue,
				Values:     []pgast.Expr{lit("DEFAULT")},
			}},
			"WHEN NOT MATCHED THEN INSERT OVERRIDING USER VALUE VALUES (DEFAULT)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}
