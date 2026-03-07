package sqlfmt_test

import (
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

func TestFormatSQL_Select(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{
			name: "simple select",
			in:   "select id, name from users where id = ?",
			want: join(
				"SELECT",
				"  id,",
				"  name",
				"FROM",
				"  users",
				"WHERE",
				"  id = ?",
			),
			ok: true,
		},
		{
			name: "select with order by and limit",
			in:   "select id from users order by id desc limit 10",
			want: join(
				"SELECT",
				"  id",
				"FROM",
				"  users",
				"ORDER BY",
				"  id DESC",
				"LIMIT",
				"  10",
			),
			ok: true,
		},
		{
			name: "select with join",
			in:   "select u.id, o.total from users u join orders o on u.id = o.user_id where u.status = ?",
			want: join(
				"SELECT",
				"  u.id,",
				"  o.total",
				"FROM",
				"  users u",
				"  JOIN",
				"  orders o",
				"    ON u.id = o.user_id",
				"WHERE",
				"  u.status = ?",
			),
			ok: true,
		},
		{
			name: "select with group by and having",
			in:   "select status, count(*) as cnt from users group by status having count(*) > 1",
			want: join(
				"SELECT",
				"  status,",
				"  count(*) AS cnt",
				"FROM",
				"  users",
				"GROUP BY",
				"  status",
				"HAVING",
				"  count(*) > 1",
			),
			ok: true,
		},
		{
			name: "parse failure returns original",
			in:   "this is not sql at all",
			want: "this is not sql at all",
			ok:   false,
		},
		{
			name: "placeholder roundtrip",
			in:   "select * from users where id = ? and status = ?",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"WHERE",
				"  id = ?",
				"  AND status = ?",
			),
			ok: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if ok != tt.ok {
				t.Errorf("FormatSQL ok = %v, want %v", ok, tt.ok)
			}

			got = strings.TrimRight(got, "\n")

			want := strings.TrimRight(tt.want, "\n")

			if got != want {
				t.Errorf("FormatSQL:\ngot:\n%s\n\nwant:\n%s", got, want)
			}
		})
	}
}

func TestFormatSQL_Insert(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("insert into users (name, email) values (?, ?)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  users",
		"(",
		"  name,",
		"  email",
		")",
		"VALUES",
		"  (?, ?)",
	)
	got = strings.TrimRight(got, "\n")

	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_Update(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("update users set name = ?, email = ? where id = ?", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"UPDATE",
		"  users",
		"SET",
		"  name = ?,",
		"  email = ?",
		"WHERE",
		"  id = ?",
	)
	got = strings.TrimRight(got, "\n")

	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_Delete(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("delete from users where id = ?", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"DELETE FROM",
		"  users",
		"WHERE",
		"  id = ?",
	)
	got = strings.TrimRight(got, "\n")

	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_Subquery(t *testing.T) {
	in := "select u.id, u.name from users u where exists (select 1 from orders o where o.user_id = u.id and o.created_at >= ?) and u.status = ?"

	got, ok := sqlfmt.FormatSQL(in, 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  u.id,",
		"  u.name",
		"FROM",
		"  users u",
		"WHERE",
		"  EXISTS (",
		"    SELECT",
		"      1",
		"    FROM",
		"      orders o",
		"    WHERE",
		"      o.user_id = u.id",
		"      AND o.created_at >= ?",
		"  )",
		"  AND u.status = ?",
	)
	got = strings.TrimRight(got, "\n")

	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_Union(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("select id from users union all select id from admins", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users",
		"UNION ALL",
		"SELECT",
		"  id",
		"FROM",
		"  admins",
	)
	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_InsertOnDuplicateKey(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("insert into users (name, email) values (?, ?) on duplicate key update name = values(name), email = values(email)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  users",
		"(",
		"  name,",
		"  email",
		")",
		"VALUES",
		"  (?, ?)",
		"ON DUPLICATE KEY UPDATE",
		"  name = values(name),",
		"  email = values(email)",
	)
	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_DerivedTable(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("select t.id from (select id from users) t", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  t.id",
		"FROM",
		"  (",
		"  SELECT",
		"    id",
		"  FROM",
		"    users",
		"  ) t",
	)
	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_LimitOffset(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("select id from users limit 10 offset 20", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users",
		"LIMIT",
		"  10",
		"OFFSET",
		"  20",
	)
	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQL_NotExpr(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "NOT condition",
			in:   "SELECT * FROM users WHERE NOT status = 'deleted'",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"WHERE",
				"  NOT status = 'deleted'",
			),
		},
		{
			name: "NOT EXISTS subquery",
			in:   "SELECT * FROM users WHERE NOT EXISTS (SELECT 1 FROM banned WHERE banned.user_id = users.id)",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"WHERE",
				"  NOT EXISTS (",
				"    SELECT",
				"      1",
				"    FROM",
				"      banned",
				"    WHERE",
				"      banned.user_id = users.id",
				"  )",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_InsertIgnore(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("INSERT IGNORE INTO users (name) VALUES (?)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT IGNORE INTO",
		"  users",
		"(",
		"  name",
		")",
		"VALUES",
		"  (?)",
	)
	assertSQL(t, got, want)
}

func TestFormatSQL_Lock(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "FOR UPDATE",
			in:   "SELECT * FROM users FOR UPDATE",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"FOR UPDATE",
			),
		},
		{
			name: "FOR SHARE",
			in:   "SELECT * FROM users FOR SHARE",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"FOR SHARE",
			),
		},
		{
			name: "LOCK IN SHARE MODE",
			in:   "SELECT * FROM users LOCK IN SHARE MODE",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"LOCK IN SHARE MODE",
			),
		},
		{
			name: "FOR UPDATE SKIP LOCKED",
			in:   "SELECT * FROM users FOR UPDATE SKIP LOCKED",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"FOR UPDATE SKIP LOCKED",
			),
		},
		{
			name: "FOR UPDATE NOWAIT",
			in:   "SELECT * FROM users FOR UPDATE NOWAIT",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users",
				"FOR UPDATE NOWAIT",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_IndexHints(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "USE INDEX",
			in:   "SELECT * FROM users USE INDEX (idx_name) WHERE id = ?",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users USE INDEX (idx_name)",
				"WHERE",
				"  id = ?",
			),
		},
		{
			name: "FORCE INDEX",
			in:   "SELECT * FROM users FORCE INDEX (idx_created) WHERE id = ?",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users FORCE INDEX (idx_created)",
				"WHERE",
				"  id = ?",
			),
		},
		{
			name: "IGNORE INDEX",
			in:   "SELECT * FROM users IGNORE INDEX (idx_old) WHERE id = ?",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  users IGNORE INDEX (idx_old)",
				"WHERE",
				"  id = ?",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_CaseExpr(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "searched CASE with ELSE",
			in:   "SELECT CASE WHEN status = 1 THEN 'active' WHEN status = 2 THEN 'inactive' ELSE 'unknown' END AS label FROM users",
			want: join(
				"SELECT",
				"  CASE",
				"    WHEN status = 1 THEN 'active'",
				"    WHEN status = 2 THEN 'inactive'",
				"    ELSE 'unknown'",
				"  END AS label",
				"FROM",
				"  users",
			),
		},
		{
			name: "simple CASE",
			in:   "SELECT CASE status WHEN 1 THEN 'active' WHEN 2 THEN 'inactive' END FROM users",
			want: join(
				"SELECT",
				"  CASE status",
				"    WHEN 1 THEN 'active'",
				"    WHEN 2 THEN 'inactive'",
				"  END",
				"FROM",
				"  users",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_DeleteMultiTable(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "multi-table DELETE with JOIN",
			in:   "DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.id WHERE t2.status = ?",
			want: join(
				"DELETE",
				"  t1",
				"FROM",
				"  t1",
				"  JOIN",
				"  t2",
				"    ON t1.id = t2.id",
				"WHERE",
				"  t2.status = ?",
			),
		},
		{
			name: "multi-table DELETE multiple targets",
			in:   "DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.ref_id WHERE t2.status = ?",
			want: join(
				"DELETE",
				"  t1,",
				"  t2",
				"FROM",
				"  t1",
				"  JOIN",
				"  t2",
				"    ON t1.id = t2.ref_id",
				"WHERE",
				"  t2.status = ?",
			),
		},
		{
			name: "DELETE IGNORE",
			in:   "DELETE IGNORE FROM users WHERE id = ?",
			want: join(
				"DELETE IGNORE FROM",
				"  users",
				"WHERE",
				"  id = ?",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_UpdateIgnore(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("UPDATE IGNORE users SET name = ? WHERE id = ?", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"UPDATE IGNORE",
		"  users",
		"SET",
		"  name = ?",
		"WHERE",
		"  id = ?",
	)
	assertSQL(t, got, want)
}

func TestFormatSQL_WithCTE(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "single CTE",
			in:   "WITH cte AS (SELECT id FROM users) SELECT * FROM cte",
			want: join(
				"WITH",
				"  cte AS (",
				"    SELECT",
				"      id",
				"    FROM",
				"      users",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  cte",
			),
		},
		{
			name: "multiple CTEs",
			in:   "WITH a AS (SELECT 1), b AS (SELECT 2) SELECT * FROM a, b",
			want: join(
				"WITH",
				"  a AS (",
				"    SELECT",
				"      1",
				"    FROM",
				"      dual",
				"  ),",
				"  b AS (",
				"    SELECT",
				"      2",
				"    FROM",
				"      dual",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  a",
				"  b",
			),
		},
		{
			name: "RECURSIVE CTE",
			in:   "WITH RECURSIVE cte AS (SELECT 1 AS id UNION ALL SELECT id + 1 FROM cte WHERE id < 10) SELECT * FROM cte",
			want: join(
				"WITH RECURSIVE",
				"  cte AS (",
				"    SELECT",
				"      1 AS id",
				"    FROM",
				"      dual",
				"    UNION ALL",
				"    SELECT",
				"      id + 1",
				"    FROM",
				"      cte",
				"    WHERE",
				"      id < 10",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  cte",
			),
		},
		{
			name: "CTE with column list",
			in:   "WITH cte (id, name) AS (SELECT id, name FROM users) SELECT * FROM cte",
			want: join(
				"WITH",
				"  cte (id, name) AS (",
				"    SELECT",
				"      id,",
				"      name",
				"    FROM",
				"      users",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  cte",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_WindowFunction(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "SUM with OVER",
			in:   "SELECT SUM(amount) OVER (PARTITION BY user_id ORDER BY created_at) FROM orders",
			want: join(
				"SELECT",
				"  sum(amount) OVER (",
				"    PARTITION BY user_id",
				"    ORDER BY created_at",
				"  )",
				"FROM",
				"  orders",
			),
		},
		{
			name: "ROW_NUMBER",
			in:   "SELECT ROW_NUMBER() OVER (ORDER BY id) FROM users",
			want: join(
				"SELECT",
				"  row_number() OVER (",
				"    ORDER BY id",
				"  )",
				"FROM",
				"  users",
			),
		},
		{
			name: "RANK with alias",
			in:   "SELECT id, RANK() OVER (PARTITION BY department ORDER BY salary DESC) AS rnk FROM employees",
			want: join(
				"SELECT",
				"  id,",
				"  rank() OVER (",
				"    PARTITION BY department",
				"    ORDER BY salary DESC",
				"  ) AS rnk",
				"FROM",
				"  employees",
			),
		},
		{
			name: "named window reference",
			in:   "SELECT SUM(amount) OVER w FROM orders",
			want: join(
				"SELECT",
				"  sum(amount) OVER w",
				"FROM",
				"  orders",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQL_WindowFunction_AllTypes(t *testing.T) {
	// Tests that all aggregate/window function types with OVER clause are formatted correctly.
	// Each entry: SQL function call -> expected lowercase output in formatted result.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"COUNT", "SELECT COUNT(id) OVER (ORDER BY id) FROM t", "count(id)"},
		{"COUNT(*)", "SELECT COUNT(*) OVER (ORDER BY id) FROM t", "count(*)"},
		{"AVG", "SELECT AVG(x) OVER (ORDER BY id) FROM t", "avg(x)"},
		{"MIN", "SELECT MIN(x) OVER (ORDER BY id) FROM t", "min(x)"},
		{"MAX", "SELECT MAX(x) OVER (ORDER BY id) FROM t", "max(x)"},
		{"BIT_AND", "SELECT BIT_AND(x) OVER (ORDER BY id) FROM t", "bit_and(x)"},
		{"BIT_OR", "SELECT BIT_OR(x) OVER (ORDER BY id) FROM t", "bit_or(x)"},
		{"BIT_XOR", "SELECT BIT_XOR(x) OVER (ORDER BY id) FROM t", "bit_xor(x)"},
		{"STD", "SELECT STD(x) OVER (ORDER BY id) FROM t", "std(x)"},
		{"STDDEV", "SELECT STDDEV(x) OVER (ORDER BY id) FROM t", "stddev(x)"},
		{"STDDEV_POP", "SELECT STDDEV_POP(x) OVER (ORDER BY id) FROM t", "stddev_pop(x)"},
		{"STDDEV_SAMP", "SELECT STDDEV_SAMP(x) OVER (ORDER BY id) FROM t", "stddev_samp(x)"},
		{"VAR_POP", "SELECT VAR_POP(x) OVER (ORDER BY id) FROM t", "var_pop(x)"},
		{"VAR_SAMP", "SELECT VAR_SAMP(x) OVER (ORDER BY id) FROM t", "var_samp(x)"},
		{"VARIANCE", "SELECT VARIANCE(x) OVER (ORDER BY id) FROM t", "variance(x)"},
		{"DENSE_RANK", "SELECT DENSE_RANK() OVER (ORDER BY id) FROM t", "dense_rank()"},
		{"CUME_DIST", "SELECT CUME_DIST() OVER (ORDER BY id) FROM t", "cume_dist()"},
		{"PERCENT_RANK", "SELECT PERCENT_RANK() OVER (ORDER BY id) FROM t", "percent_rank()"},
		{"FIRST_VALUE", "SELECT FIRST_VALUE(x) OVER (ORDER BY id) FROM t", "first_value(x)"},
		{"LAST_VALUE", "SELECT LAST_VALUE(x) OVER (ORDER BY id) FROM t", "last_value(x)"},
		{"NTILE", "SELECT NTILE(4) OVER (ORDER BY id) FROM t", "ntile(4)"},
		{"NTH_VALUE", "SELECT NTH_VALUE(x, 2) OVER (ORDER BY id) FROM t", "nth_value(x, 2)"},
		{"LAG", "SELECT LAG(x) OVER (ORDER BY id) FROM t", "lag(x)"},
		{"LEAD", "SELECT LEAD(x) OVER (ORDER BY id) FROM t", "lead(x)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQL(tt.in, 2)
			if !ok {
				t.Fatal("expected ok")
			}

			want := join(
				"SELECT",
				"  "+tt.want+" OVER (",
				"    ORDER BY id",
				"  )",
				"FROM",
				"  t",
			)

			assertSQL(t, got, want)
		})
	}
}

func TestFormatSQL_IndexHintForType(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT id FROM users USE INDEX FOR ORDER BY (idx_name) WHERE id = ?", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users USE INDEX FOR ORDER BY (idx_name)",
		"WHERE",
		"  id = ?",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_UnionLock(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT id FROM users UNION ALL SELECT id FROM admins FOR UPDATE", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users",
		"UNION ALL",
		"SELECT",
		"  id",
		"FROM",
		"  admins",
		"FOR UPDATE",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WithCTE_Update(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("WITH cte AS (SELECT id FROM users WHERE active = 1) UPDATE users SET status = 0 WHERE id IN (SELECT id FROM cte)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"WITH",
		"  cte AS (",
		"    SELECT",
		"      id",
		"    FROM",
		"      users",
		"    WHERE",
		"      active = 1",
		"  )",
		"UPDATE",
		"  users",
		"SET",
		"  status = 0",
		"WHERE",
		"  id in (",
		"    SELECT",
		"      id",
		"    FROM",
		"      cte",
		"  )",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WithCTE_Delete(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("WITH cte AS (SELECT id FROM users WHERE active = 0) DELETE FROM users WHERE id IN (SELECT id FROM cte)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"WITH",
		"  cte AS (",
		"    SELECT",
		"      id",
		"    FROM",
		"      users",
		"    WHERE",
		"      active = 0",
		"  )",
		"DELETE FROM",
		"  users",
		"WHERE",
		"  id in (",
		"    SELECT",
		"      id",
		"    FROM",
		"      cte",
		"  )",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WithCTE_Union(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("WITH cte AS (SELECT id FROM users) SELECT id FROM cte UNION ALL SELECT id FROM admins", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"WITH",
		"  cte AS (",
		"    SELECT",
		"      id",
		"    FROM",
		"      users",
		"  )",
		"SELECT",
		"  id",
		"FROM",
		"  cte",
		"UNION ALL",
		"SELECT",
		"  id",
		"FROM",
		"  admins",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WindowFrameClause(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT SUM(amount) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM t", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  sum(amount) OVER (",
		"    ORDER BY id",
		"    ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
		"  )",
		"FROM",
		"  t",
	)

	assertSQL(t, got, want)
}

func assertSQL(t *testing.T, got, want string) {
	t.Helper()

	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("FormatSQL:\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func join(lines ...string) string {
	return strings.Join(lines, "\n")
}
