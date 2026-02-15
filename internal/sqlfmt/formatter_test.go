package sqlfmt

import (
	"strings"
	"testing"
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
			got, ok := FormatSQL(tt.in, 2)
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
	got, ok := FormatSQL("insert into users (name, email) values (?, ?)", 2)
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
	got, ok := FormatSQL("update users set name = ?, email = ? where id = ?", 2)
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
	got, ok := FormatSQL("delete from users where id = ?", 2)
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
	got, ok := FormatSQL(in, 2)
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

func join(lines ...string) string {
	return strings.Join(lines, "\n")
}
