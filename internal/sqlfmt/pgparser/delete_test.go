package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertDeleteRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertDeleteRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	del, err := pgparser.ParseDelete(input)
	if err != nil {
		t.Fatalf("ParseDelete(%q) error = %v", input, err)
	}

	if got := del.String(); got != want {
		t.Errorf("ParseDelete(%q).String() = %q, want %q", input, got, want)
	}
}

func assertDeleteError(t *testing.T, input string) {
	t.Helper()

	if _, err := pgparser.ParseDelete(input); err == nil {
		t.Errorf("ParseDelete(%q) error = nil, want error", input)
	}
}

func TestParseDelete_target(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "DELETE FROM users", "DELETE FROM users"},
		{"trailing semicolon", "DELETE FROM users;", "DELETE FROM users"},
		{"qualified table", "DELETE FROM public.users", "DELETE FROM public.users"},
		{"quoted identifiers keep their quotes", `DELETE FROM "Schema"."Users" "U"`, `DELETE FROM "Schema"."Users" "U"`},
		{"explicit alias", "DELETE FROM users AS u", "DELETE FROM users u"},
		{"implicit alias", "DELETE FROM users u", "DELETE FROM users u"},
		{"only", "DELETE FROM ONLY users", "DELETE FROM ONLY users"},
		{"only with alias", "DELETE FROM ONLY public.users AS u", "DELETE FROM ONLY public.users u"},
		{"only parenthesized name", "DELETE FROM ONLY (users) u", "DELETE FROM ONLY users u"},
		{"include descendants", "DELETE FROM users *", "DELETE FROM users *"},
		{"include descendants with alias", "DELETE FROM users * AS u", "DELETE FROM users * u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_using(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"single using table",
			"DELETE FROM films USING producers WHERE producer_id = producers.id AND producers.name = 'foo'",
			"DELETE FROM films USING producers WHERE producer_id = producers.id AND producers.name = 'foo'",
		},
		{
			"multiple using tables with aliases",
			"DELETE FROM t AS x USING a AS y, b z WHERE x.id = y.id AND y.id = z.id",
			"DELETE FROM t x USING a y, b z WHERE x.id = y.id AND y.id = z.id",
		},
		{
			"using join",
			"DELETE FROM t USING a JOIN b ON a.id = b.a_id WHERE t.id = a.id",
			"DELETE FROM t USING a JOIN b ON a.id = b.a_id WHERE t.id = a.id",
		},
		{
			"using derived table",
			"DELETE FROM t USING (SELECT id FROM s) AS sub WHERE t.id = sub.id",
			"DELETE FROM t USING (SELECT id FROM s) sub WHERE t.id = sub.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_where(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"comparison", "DELETE FROM t WHERE id = 1", "DELETE FROM t WHERE id = 1"},
		{"placeholder", "DELETE FROM t WHERE id = $1", "DELETE FROM t WHERE id = $1"},
		{
			"subquery",
			"DELETE FROM t WHERE id IN (SELECT t_id FROM s WHERE s.active)",
			"DELETE FROM t WHERE id IN (SELECT t_id FROM s WHERE s.active)",
		},
		{
			"exists",
			"DELETE FROM t WHERE EXISTS (SELECT 1 FROM s WHERE s.t_id = t.id)",
			"DELETE FROM t WHERE EXISTS (SELECT 1 FROM s WHERE s.t_id = t.id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_returning(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"star", "DELETE FROM t RETURNING *", "DELETE FROM t RETURNING *"},
		{"qualified star", "DELETE FROM t RETURNING t.*", "DELETE FROM t RETURNING t.*"},
		{"columns", "DELETE FROM t RETURNING id, name", "DELETE FROM t RETURNING id, name"},
		{"explicit alias", "DELETE FROM t RETURNING id AS deleted_id", "DELETE FROM t RETURNING id AS deleted_id"},
		{"implicit alias", "DELETE FROM t RETURNING id deleted_id", "DELETE FROM t RETURNING id AS deleted_id"},
		{"expression", "DELETE FROM t RETURNING price * qty AS total", "DELETE FROM t RETURNING price * qty AS total"},
		{
			"all clauses",
			"DELETE FROM ONLY t AS x USING s WHERE x.id = s.id RETURNING x.*, s.name;",
			"DELETE FROM ONLY t x USING s WHERE x.id = s.id RETURNING x.*, s.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_errors(t *testing.T) {
	tests := []string{
		"",
		"DELETE",
		"DELETE users",
		"DELETE FROM",
		"DELETE FROM ONLY",
		"DELETE FROM ONLY (users",
		"DELETE FROM ONLY users *",
		"DELETE FROM users USING",
		"DELETE FROM users WHERE",
		"DELETE FROM users WHERE CURRENT OF c",
		"DELETE FROM users RETURNING",
		"DELETE FROM users ORDER BY id",
		"DELETE FROM users LIMIT 1",
		"DELETE FROM users; garbage",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			assertDeleteError(t, tt)
		})
	}
}
