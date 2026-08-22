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
				"  COUNT(*) AS cnt",
				"FROM",
				"  users",
				"GROUP BY",
				"  status",
				"HAVING",
				"  COUNT(*) > 1",
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
				"  ),",
				"  b AS (",
				"    SELECT",
				"      2",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  a,",
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
				"  SUM(amount) OVER (",
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
				"  ROW_NUMBER() OVER (",
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
				"  RANK() OVER (",
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
				"  SUM(amount) OVER w",
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
	// Each entry: SQL function call -> expected uppercase output in formatted result.
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"COUNT", "SELECT COUNT(id) OVER (ORDER BY id) FROM t", "COUNT(id)"},
		{"COUNT(*)", "SELECT COUNT(*) OVER (ORDER BY id) FROM t", "COUNT(*)"},
		{"AVG", "SELECT AVG(x) OVER (ORDER BY id) FROM t", "AVG(x)"},
		{"MIN", "SELECT MIN(x) OVER (ORDER BY id) FROM t", "MIN(x)"},
		{"MAX", "SELECT MAX(x) OVER (ORDER BY id) FROM t", "MAX(x)"},
		{"BIT_AND", "SELECT BIT_AND(x) OVER (ORDER BY id) FROM t", "BIT_AND(x)"},
		{"BIT_OR", "SELECT BIT_OR(x) OVER (ORDER BY id) FROM t", "BIT_OR(x)"},
		{"BIT_XOR", "SELECT BIT_XOR(x) OVER (ORDER BY id) FROM t", "BIT_XOR(x)"},
		{"STD", "SELECT STD(x) OVER (ORDER BY id) FROM t", "STD(x)"},
		{"STDDEV", "SELECT STDDEV(x) OVER (ORDER BY id) FROM t", "STDDEV(x)"},
		{"STDDEV_POP", "SELECT STDDEV_POP(x) OVER (ORDER BY id) FROM t", "STDDEV_POP(x)"},
		{"STDDEV_SAMP", "SELECT STDDEV_SAMP(x) OVER (ORDER BY id) FROM t", "STDDEV_SAMP(x)"},
		{"VAR_POP", "SELECT VAR_POP(x) OVER (ORDER BY id) FROM t", "VAR_POP(x)"},
		{"VAR_SAMP", "SELECT VAR_SAMP(x) OVER (ORDER BY id) FROM t", "VAR_SAMP(x)"},
		{"VARIANCE", "SELECT VARIANCE(x) OVER (ORDER BY id) FROM t", "VARIANCE(x)"},
		{"DENSE_RANK", "SELECT DENSE_RANK() OVER (ORDER BY id) FROM t", "DENSE_RANK()"},
		{"CUME_DIST", "SELECT CUME_DIST() OVER (ORDER BY id) FROM t", "CUME_DIST()"},
		{"PERCENT_RANK", "SELECT PERCENT_RANK() OVER (ORDER BY id) FROM t", "PERCENT_RANK()"},
		{"FIRST_VALUE", "SELECT FIRST_VALUE(x) OVER (ORDER BY id) FROM t", "FIRST_VALUE(x)"},
		{"LAST_VALUE", "SELECT LAST_VALUE(x) OVER (ORDER BY id) FROM t", "LAST_VALUE(x)"},
		{"NTILE", "SELECT NTILE(4) OVER (ORDER BY id) FROM t", "NTILE(4)"},
		{"NTH_VALUE", "SELECT NTH_VALUE(x, 2) OVER (ORDER BY id) FROM t", "NTH_VALUE(x, 2)"},
		{"LAG", "SELECT LAG(x) OVER (ORDER BY id) FROM t", "LAG(x)"},
		{"LEAD", "SELECT LEAD(x) OVER (ORDER BY id) FROM t", "LEAD(x)"},
		{"JSON_ARRAYAGG", "SELECT JSON_ARRAYAGG(x) OVER (ORDER BY id) FROM t", "JSON_ARRAYAGG(x)"},
		{"JSON_OBJECTAGG", "SELECT JSON_OBJECTAGG(k, v) OVER (ORDER BY id) FROM t", "JSON_OBJECTAGG(k, v)"},
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
		"  id IN (",
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
		"  id IN (",
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
		"  SUM(amount) OVER (",
		"    ORDER BY id",
		"    ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
		"  )",
		"FROM",
		"  t",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_ParenTableExpr(t *testing.T) {
	query := "select id from (users, orders) where users.id = orders.user_id"

	got, ok := sqlfmt.FormatSQL(query, 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  (",
		"    users,",
		"    orders",
		"  )",
		"WHERE",
		"  users.id = orders.user_id",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WhereOr(t *testing.T) {
	query := "select id from users where a = 1 or b = 2"

	got, ok := sqlfmt.FormatSQL(query, 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users",
		"WHERE",
		"  a = 1",
		"  OR b = 2",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_InsertUnionRows(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("INSERT INTO t SELECT a FROM x UNION SELECT b FROM y", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  t",
		"SELECT",
		"  a",
		"FROM",
		"  x",
		"UNION",
		"SELECT",
		"  b",
		"FROM",
		"  y",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_InsertSetRows(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("INSERT INTO t SET name = ?, email = ?", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  t",
		"SET",
		"  name = ?,",
		"  email = ?",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_GroupByWithRollup(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT status, COUNT(*) FROM t GROUP BY status WITH ROLLUP", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  status,",
		"  COUNT(*)",
		"FROM",
		"  t",
		"GROUP BY",
		"  status",
		"WITH ROLLUP",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_SelectDistinct(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT DISTINCT id, name FROM users", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT DISTINCT",
		"  id,",
		"  name",
		"FROM",
		"  users",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_MultiColumnGroupBy(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT status, dept FROM users GROUP BY status, dept", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  status,",
		"  dept",
		"FROM",
		"  users",
		"GROUP BY",
		"  status,",
		"  dept",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_MultiColumnOrderBy(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT id FROM users ORDER BY status, id DESC", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  users",
		"ORDER BY",
		"  status,",
		"  id DESC",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_InsertMultiRowValues(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("INSERT INTO t (a) VALUES (1), (2)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  t",
		"(",
		"  a",
		")",
		"VALUES",
		"  (1),",
		"  (2)",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_UpdateOrderByLimit(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("UPDATE users SET name = ? WHERE id = ? ORDER BY id LIMIT 1", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"UPDATE",
		"  users",
		"SET",
		"  name = ?",
		"WHERE",
		"  id = ?",
		"ORDER BY",
		"  id",
		"LIMIT",
		"  1",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_DeleteOrderByLimit(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("DELETE FROM users WHERE id > ? ORDER BY id LIMIT 1", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"DELETE FROM",
		"  users",
		"WHERE",
		"  id > ?",
		"ORDER BY",
		"  id",
		"LIMIT",
		"  1",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_UnionLimit(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT id FROM a UNION SELECT id FROM b LIMIT 5", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  a",
		"UNION",
		"SELECT",
		"  id",
		"FROM",
		"  b",
		"LIMIT",
		"  5",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_Replace(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("REPLACE INTO users (id, name) VALUES (?, ?)", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"REPLACE INTO",
		"  users",
		"(",
		"  id,",
		"  name",
		")",
		"VALUES",
		"  (?, ?)",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_InsertSelect(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("INSERT INTO t SELECT a, b FROM x", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"INSERT INTO",
		"  t",
		"SELECT",
		"  a,",
		"  b",
		"FROM",
		"  x",
	)

	assertSQL(t, got, want)
}

// TestFormatSQL_MultiTableFrom guards against dropping the comma between
// top-level table references: without it, "FROM a, b" re-formats as
// "FROM\n  a\n  b", which no longer parses as two tables (it reads as
// "a" implicitly aliased "b").
func TestFormatSQL_MultiTableFrom(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT id FROM a, b", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  id",
		"FROM",
		"  a,",
		"  b",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_UpdateMultiTable(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("UPDATE a, b SET a.x = 1 WHERE a.id = b.id", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"UPDATE",
		"  a,",
		"  b",
		"SET",
		"  a.x = 1",
		"WHERE",
		"  a.id = b.id",
	)

	assertSQL(t, got, want)
}

func TestFormatSQL_WindowEmptyOver(t *testing.T) {
	got, ok := sqlfmt.FormatSQL("SELECT SUM(amount) OVER () FROM orders", 2)
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  SUM(amount) OVER ()",
		"FROM",
		"  orders",
	)

	assertSQL(t, got, want)
}

// JSON_TABLE is outside the in-house parser's scope (see docs/parser-spec.md);
// FormatSQL falls back to returning the input unchanged.
func TestFormatSQL_JSONTableExpr(t *testing.T) {
	query := "select id from JSON_TABLE(data, '$[*]' COLUMNS(id INT PATH '$.id')) AS jt"

	got, ok := sqlfmt.FormatSQL(query, 2)
	if ok {
		t.Fatal("expected parse failure")
	}

	if got != query {
		t.Errorf("got:\n%s\n\nwant original input unchanged:\n%s", got, query)
	}
}

func TestFormatSQLWithOptions_KeywordCase(t *testing.T) {
	tests := []struct {
		name        string
		keywordCase string
		want        string
	}{
		{
			name:        "upper (default)",
			keywordCase: "",
			want: join(
				"SELECT",
				"  id",
				"FROM",
				"  users",
				"WHERE",
				"  status = 1",
				"  AND active = TRUE",
				"ORDER BY",
				"  id DESC",
			),
		},
		{
			name:        "upper",
			keywordCase: sqlfmt.KeywordCaseUpper,
			want: join(
				"SELECT",
				"  id",
				"FROM",
				"  users",
				"WHERE",
				"  status = 1",
				"  AND active = TRUE",
				"ORDER BY",
				"  id DESC",
			),
		},
		{
			name:        "lower",
			keywordCase: sqlfmt.KeywordCaseLower,
			want: join(
				"SELECT",
				"  id",
				"FROM",
				"  users",
				"WHERE",
				"  status = 1",
				"  and active = true",
				"ORDER BY",
				"  id desc",
			),
		},
		{
			// The in-house parser, like the Vitess parser before it, doesn't
			// retain the source's original casing for these tokens — it
			// canonicalizes them to uppercase while parsing. So "preserve"
			// falls back to the parser's canonical output, which is upper.
			name:        "preserve",
			keywordCase: sqlfmt.KeywordCasePreserve,
			want: join(
				"SELECT",
				"  id",
				"FROM",
				"  users",
				"WHERE",
				"  status = 1",
				"  AND active = TRUE",
				"ORDER BY",
				"  id DESC",
			),
		},
	}

	in := "select id from users where status = 1 and active = true order by id desc"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, KeywordCase: tt.keywordCase})
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQLWithOptions_KeywordCase_ASAndON(t *testing.T) {
	in := "select u.id as uid from users u join orders o on u.id = o.user_id"

	got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, KeywordCase: sqlfmt.KeywordCaseLower})
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"SELECT",
		"  u.id as uid",
		"FROM",
		"  users u",
		"  JOIN",
		"  orders o",
		"    on u.id = o.user_id",
	)

	assertSQL(t, got, want)
}

func TestFormatSQLWithOptions_CommaStyle(t *testing.T) {
	tests := []struct {
		name       string
		commaStyle string
		want       string
	}{
		{
			name:       "trailing (default)",
			commaStyle: "",
			want: join(
				"SELECT",
				"  id,",
				"  name,",
				"  email",
				"FROM",
				"  users",
			),
		},
		{
			name:       "trailing",
			commaStyle: sqlfmt.CommaStyleTrailing,
			want: join(
				"SELECT",
				"  id,",
				"  name,",
				"  email",
				"FROM",
				"  users",
			),
		},
		{
			name:       "leading",
			commaStyle: sqlfmt.CommaStyleLeading,
			want: join(
				"SELECT",
				"  id",
				", name",
				", email",
				"FROM",
				"  users",
			),
		},
	}

	in := "select id, name, email from users"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, CommaStyle: tt.commaStyle})
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

func TestFormatSQLWithOptions_CommaStyle_With(t *testing.T) {
	in := "with a as (select id from t1), b as (select id from t2) select id from a"

	got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, CommaStyle: sqlfmt.CommaStyleLeading})
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"WITH",
		"  a AS (",
		"    SELECT",
		"      id",
		"    FROM",
		"      t1",
		"  )",
		", b AS (",
		"    SELECT",
		"      id",
		"    FROM",
		"      t2",
		"  )",
		"SELECT",
		"  id",
		"FROM",
		"  a",
	)

	assertSQL(t, got, want)
}

// TestFormatSQLWithOptions_SQLMode covers the two string-literal cases where
// NO_BACKSLASH_ESCAPES changes meaning: a doubled quote (which the default
// mode re-encodes with a backslash, mangling it once NO_BACKSLASH_ESCAPES is
// set) and a literal newline (which the default mode re-encodes as
// backslash-n, changing a multi-line value into a single-line one under
// NO_BACKSLASH_ESCAPES).
func TestFormatSQLWithOptions_SQLMode(t *testing.T) {
	tests := []struct {
		name    string
		sqlMode string
		in      string
		want    string
	}{
		{
			name:    "default (unset) re-encodes doubled quote with backslash",
			sqlMode: "",
			in:      "select 'it''s' from t",
			want:    join("SELECT", `  'it\'s'`, "FROM", "  t"),
		},
		{
			name:    "SQLModeDefault re-encodes doubled quote with backslash",
			sqlMode: sqlfmt.SQLModeDefault,
			in:      "select 'it''s' from t",
			want:    join("SELECT", `  'it\'s'`, "FROM", "  t"),
		},
		{
			name:    "NoBackslashEscapes keeps doubled quote doubled",
			sqlMode: sqlfmt.SQLModeNoBackslashEscapes,
			in:      "select 'it''s' from t",
			want:    join("SELECT", `  'it''s'`, "FROM", "  t"),
		},
		{
			name:    "default re-encodes a literal newline as backslash-n",
			sqlMode: sqlfmt.SQLModeDefault,
			in:      "select 'a\nb' from t",
			want:    join("SELECT", `  'a\nb'`, "FROM", "  t"),
		},
		{
			name:    "NoBackslashEscapes preserves a literal newline",
			sqlMode: sqlfmt.SQLModeNoBackslashEscapes,
			in:      "select 'a\nb' from t",
			want:    join("SELECT", "  'a\nb'", "FROM", "  t"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(tt.in, sqlfmt.Options{Indent: 2, SQLMode: tt.sqlMode})
			if !ok {
				t.Fatal("expected ok")
			}

			assertSQL(t, got, tt.want)
		})
	}
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
