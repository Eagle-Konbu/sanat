package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestSelect_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.Select
		want string
	}{
		{
			name: "simple",
			s: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{
					&sqlast.AliasedExpr{Expr: lit("id")},
					&sqlast.AliasedExpr{Expr: lit("name")},
				},
				From: []sqlast.TableExpr{
					&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}},
				},
			},
			want: "SELECT id, name FROM users",
		},
		{
			name: "distinct with where",
			s: &sqlast.Select{
				Distinct:    true,
				SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("id")}},
				From: []sqlast.TableExpr{
					&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}},
				},
				Where: &sqlast.Where{Expr: lit("active = 1")},
			},
			want: "SELECT DISTINCT id FROM users WHERE active = 1",
		},
		{
			name: "with group by and having",
			s: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{
					&sqlast.AliasedExpr{Expr: lit("dept")},
					&sqlast.AliasedExpr{Expr: lit("COUNT(*)")},
				},
				From:    []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "emp"}}},
				GroupBy: &sqlast.GroupBy{Exprs: exprs("dept")},
				Having:  &sqlast.Where{Expr: lit("COUNT(*) > 5")},
			},
			want: "SELECT dept, COUNT(*) FROM emp GROUP BY dept HAVING COUNT(*) > 5",
		},
		{
			name: "with order by and limit",
			s: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{&sqlast.StarExpr{}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				OrderBy:     sqlast.OrderBy{{Expr: lit("id"), Direction: sqlast.DescOrder}},
				Limit:       &sqlast.Limit{Rowcount: lit("10")},
			},
			want: "SELECT * FROM t ORDER BY id DESC LIMIT 10",
		},
		{
			name: "with lock",
			s: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{&sqlast.StarExpr{}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				Lock:        sqlast.ForUpdateLock,
			},
			want: "SELECT * FROM t FOR UPDATE",
		},
		{
			name: "with lock wait",
			s: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{&sqlast.StarExpr{}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				Lock:        sqlast.ForShareLock,
				LockWait:    sqlast.SkipLockedType,
			},
			want: "SELECT * FROM t FOR SHARE SKIP LOCKED",
		},
		{
			name: "with CTE",
			s: &sqlast.Select{
				With: &sqlast.With{
					CTEs: []*sqlast.CommonTableExpr{
						{
							ID:       "cte1",
							Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}},
						},
					},
				},
				SelectExprs: []sqlast.SelectExpr{&sqlast.StarExpr{}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "cte1"}}},
			},
			want: "WITH cte1 AS (SELECT 1) SELECT * FROM cte1",
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
		ins  *sqlast.Insert
		want string
	}{
		{
			name: "simple",
			ins: &sqlast.Insert{
				Action:  sqlast.InsertAct,
				Table:   sqlast.TableName{Name: "users"},
				Columns: sqlast.Columns{"id", "name"},
				Rows:    sqlast.Values{{lit("1"), lit("'a'")}},
			},
			want: "INSERT INTO users (id, name) VALUES (1, 'a')",
		},
		{
			name: "replace ignore",
			ins: &sqlast.Insert{
				Action:  sqlast.ReplaceAct,
				Ignore:  true,
				Table:   sqlast.TableName{Name: "t"},
				Columns: sqlast.Columns{"a"},
				Rows:    sqlast.Values{{lit("1")}},
			},
			want: "REPLACE IGNORE INTO t (a) VALUES (1)",
		},
		{
			name: "with on duplicate key",
			ins: &sqlast.Insert{
				Action:  sqlast.InsertAct,
				Table:   sqlast.TableName{Name: "t"},
				Columns: sqlast.Columns{"a", "b"},
				Rows:    sqlast.Values{{lit("1"), lit("2")}},
				OnDup: sqlast.OnDup{
					{Name: &sqlast.ColName{Name: "b"}, Expr: lit("3")},
				},
			},
			want: "INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE b = 3",
		},
		{
			name: "insert select",
			ins: &sqlast.Insert{
				Action:  sqlast.InsertAct,
				Table:   sqlast.TableName{Name: "t2"},
				Columns: sqlast.Columns{"a"},
				Rows: &sqlast.Select{
					SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("a")}},
					From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t1"}}},
				},
			},
			want: "INSERT INTO t2 (a) SELECT a FROM t1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.ins.String())
		})
	}
}

func TestUpdate_String(t *testing.T) {
	tests := []struct {
		name string
		u    *sqlast.Update
		want string
	}{
		{
			name: "simple",
			u: &sqlast.Update{
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}}},
				Exprs:      []*sqlast.UpdateExpr{{Name: &sqlast.ColName{Name: "name"}, Expr: lit("'bob'")}},
				Where:      &sqlast.Where{Expr: lit("id = 1")},
			},
			want: "UPDATE users SET name = 'bob' WHERE id = 1",
		},
		{
			name: "ignore with order and limit",
			u: &sqlast.Update{
				Ignore:     true,
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				Exprs:      []*sqlast.UpdateExpr{{Name: &sqlast.ColName{Name: "a"}, Expr: lit("1")}},
				OrderBy:    sqlast.OrderBy{{Expr: lit("id")}},
				Limit:      &sqlast.Limit{Rowcount: lit("10")},
			},
			want: "UPDATE IGNORE t SET a = 1 ORDER BY id LIMIT 10",
		},
		{
			name: "with CTE",
			u: &sqlast.Update{
				With: &sqlast.With{
					CTEs: []*sqlast.CommonTableExpr{
						{ID: "c", Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}}},
					},
				},
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				Exprs:      []*sqlast.UpdateExpr{{Name: &sqlast.ColName{Name: "a"}, Expr: lit("1")}},
			},
			want: "WITH c AS (SELECT 1) UPDATE t SET a = 1",
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
		d    *sqlast.Delete
		want string
	}{
		{
			name: "simple",
			d: &sqlast.Delete{
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}}},
				Where:      &sqlast.Where{Expr: lit("id = 1")},
			},
			want: "DELETE FROM users WHERE id = 1",
		},
		{
			name: "ignore with order and limit",
			d: &sqlast.Delete{
				Ignore:     true,
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
				OrderBy:    sqlast.OrderBy{{Expr: lit("id")}},
				Limit:      &sqlast.Limit{Rowcount: lit("5")},
			},
			want: "DELETE IGNORE FROM t ORDER BY id LIMIT 5",
		},
		{
			name: "multi-table",
			d: &sqlast.Delete{
				Targets:    []sqlast.TableName{{Name: "t1"}, {Name: "t2"}},
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t1"}}},
			},
			want: "DELETE t1, t2 FROM t1",
		},
		{
			name: "with CTE",
			d: &sqlast.Delete{
				With: &sqlast.With{
					CTEs: []*sqlast.CommonTableExpr{
						{ID: "c", Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}}},
					},
				},
				TableExprs: []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
			},
			want: "WITH c AS (SELECT 1) DELETE FROM t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.d.String())
		})
	}
}

func TestUnion_String(t *testing.T) {
	left := &sqlast.Select{
		SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}},
	}
	right := &sqlast.Select{
		SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("2")}},
	}

	t.Run("distinct", func(t *testing.T) {
		u := &sqlast.Union{Left: left, Right: right, Distinct: true}
		assertEqual(t, "SELECT 1 UNION SELECT 2", u.String())
	})

	t.Run("all", func(t *testing.T) {
		u := &sqlast.Union{Left: left, Right: right, Distinct: false}
		assertEqual(t, "SELECT 1 UNION ALL SELECT 2", u.String())
	})

	t.Run("with order limit lock", func(t *testing.T) {
		u := &sqlast.Union{
			Left: left, Right: right, Distinct: true,
			OrderBy: sqlast.OrderBy{{Expr: lit("1")}},
			Limit:   &sqlast.Limit{Rowcount: lit("10")},
			Lock:    sqlast.ForUpdateLock,
		}

		assertEqual(t, "SELECT 1 UNION SELECT 2 ORDER BY 1 LIMIT 10 FOR UPDATE", u.String())
	})
}

func TestWith_String(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assertEqual(t, "", (*sqlast.With)(nil).String())
	})

	t.Run("simple", func(t *testing.T) {
		w := &sqlast.With{
			CTEs: []*sqlast.CommonTableExpr{
				{ID: "cte1", Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}}},
			},
		}

		assertEqual(t, "WITH cte1 AS (SELECT 1)", w.String())
	})

	t.Run("recursive", func(t *testing.T) {
		w := &sqlast.With{
			Recursive: true,
			CTEs: []*sqlast.CommonTableExpr{
				{ID: "cte1", Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}}},
			},
		}

		assertEqual(t, "WITH RECURSIVE cte1 AS (SELECT 1)", w.String())
	})
}

func TestCommonTableExpr_String(t *testing.T) {
	t.Run("without columns", func(t *testing.T) {
		cte := &sqlast.CommonTableExpr{
			ID:       "cte1",
			Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}}},
		}

		assertEqual(t, "cte1 AS (SELECT 1)", cte.String())
	})

	t.Run("with columns", func(t *testing.T) {
		cte := &sqlast.CommonTableExpr{
			ID:       "cte1",
			Columns:  sqlast.Columns{"a", "b"},
			Subquery: &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}, &sqlast.AliasedExpr{Expr: lit("2")}}},
		}

		assertEqual(t, "cte1 (a, b) AS (SELECT 1, 2)", cte.String())
	})
}
