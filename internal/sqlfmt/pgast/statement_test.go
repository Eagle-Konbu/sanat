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
				With:        &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}},
				Where:       &pgast.Where{Expr: col("active")},
				GroupBy:     &pgast.GroupBy{Elements: []pgast.Expr{col("id")}},
				Having:      &pgast.Where{Expr: col("cond")},
				Window:      []*pgast.NamedWindow{{Name: "w", Spec: &pgast.WindowSpec{}}},
				OrderBy:     pgast.OrderBy{{Expr: col("id")}},
				Limit:       &pgast.Limit{Count: lit("10")},
				Offset:      &pgast.Offset{Start: lit("5")},
				Fetch:       &pgast.Fetch{Count: lit("1")},
				Lock:        &pgast.Lock{Strength: pgast.ForUpdate},
			},
			"WITH cte AS (SELECT * FROM t) SELECT id FROM t WHERE active GROUP BY id HAVING cond WINDOW w AS () ORDER BY id LIMIT 10 OFFSET 5 FETCH FIRST 1 ROW ONLY FOR UPDATE",
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
				Table:   pgast.TableName{Name: "users"},
				Columns: pgast.Columns{"id", "name"},
				Rows:    pgast.Values{{lit("1"), lit("'a'")}},
			},
			"INSERT INTO users (id, name) VALUES (1, 'a')",
		},
		{
			"with alias, on conflict, returning",
			&pgast.Insert{
				With:       &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
				Table:      pgast.TableName{Name: "users"},
				Alias:      "u",
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
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
			},
			"UPDATE users SET name = 'a'",
		},
		{
			"with from, where, returning",
			&pgast.Update{
				With:  &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}},
				Set:   []*pgast.UpdateExpr{{Name: col("name"), Expr: lit("'a'")}},
				From:  []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "other"}}},
				Where: &pgast.Where{Expr: col("active")},
				Returning: &pgast.Returning{
					Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				},
			},
			"WITH cte AS (SELECT * FROM t) UPDATE users SET name = 'a' FROM other WHERE active RETURNING id",
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
			&pgast.Delete{Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}}},
			"DELETE FROM users",
		},
		{
			"with using, where, returning",
			&pgast.Delete{
				With:  &pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
				Table: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "users"}},
				Using: []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "other"}}},
				Where: &pgast.Where{Expr: col("active")},
				Returning: &pgast.Returning{
					Exprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				},
			},
			"WITH cte AS (SELECT * FROM t) DELETE FROM users USING other WHERE active RETURNING id",
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
			"with clauses",
			&pgast.SetOp{
				With:    &pgast.With{Recursive: true, CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
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
			&pgast.With{CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
			"WITH cte AS (SELECT * FROM t)",
		},
		{
			"recursive",
			&pgast.With{Recursive: true, CTEs: []*pgast.CommonTableExpr{{Name: "cte", Subquery: selectStar()}}},
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
			&pgast.CommonTableExpr{Name: "cte", Subquery: selectStar()},
			"cte AS (SELECT * FROM t)",
		},
		{
			"with columns",
			&pgast.CommonTableExpr{Name: "cte", Columns: pgast.Columns{"a", "b"}, Subquery: selectStar()},
			"cte (a, b) AS (SELECT * FROM t)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.c.String())
		})
	}
}

func TestMerge_String(t *testing.T) {
	m := &pgast.Merge{
		Target: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}},
		Source: &pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "s"}},
		On:     &pgast.ComparisonExpr{Operator: pgast.EqualOp, Left: col("t.id"), Right: col("s.id")},
		Whens: []*pgast.MergeWhen{
			{Matched: true, Set: []*pgast.UpdateExpr{{Name: col("name"), Expr: col("s.name")}}},
			{Columns: pgast.Columns{"id"}, Values: []pgast.Expr{col("s.id")}},
		},
	}

	want := "MERGE INTO t USING s ON t.id = s.id " +
		"WHEN MATCHED THEN UPDATE SET name = s.name " +
		"WHEN NOT MATCHED THEN INSERT (id) VALUES (s.id)"
	assertEqual(t, want, m.String())
}

func TestMergeWhen_String(t *testing.T) {
	tests := []struct {
		name string
		w    *pgast.MergeWhen
		want string
	}{
		{
			"matched with condition, do nothing",
			&pgast.MergeWhen{Matched: true, Condition: col("cond"), DoNothing: true},
			"WHEN MATCHED AND cond THEN DO NOTHING",
		},
		{
			"matched delete",
			&pgast.MergeWhen{Matched: true, Delete: true},
			"WHEN MATCHED THEN DELETE",
		},
		{
			"not matched by source",
			&pgast.MergeWhen{BySource: true, DoNothing: true},
			"WHEN NOT MATCHED BY SOURCE THEN DO NOTHING",
		},
		{
			"not matched insert no columns",
			&pgast.MergeWhen{Values: []pgast.Expr{lit("1")}},
			"WHEN NOT MATCHED THEN INSERT VALUES (1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.w.String())
		})
	}
}
