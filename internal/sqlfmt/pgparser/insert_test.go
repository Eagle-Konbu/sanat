package pgparser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertInsertRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertInsertRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	ins, err := pgparser.ParseInsert(input)
	if err != nil {
		t.Fatalf("ParseInsert(%q) error = %v", input, err)
	}

	if got := ins.String(); got != want {
		t.Errorf("ParseInsert(%q).String() = %q, want %q", input, got, want)
	}
}

func assertInsertError(t *testing.T, input string) {
	t.Helper()

	_, err := pgparser.ParseInsert(input)
	if err == nil {
		t.Fatalf("ParseInsert(%q) error = nil, want error", input)
	}

	if _, ok := errors.AsType[*pgparser.ParseError](err); !ok {
		t.Errorf("ParseInsert(%q) error type = %T, want *pgparser.ParseError", input, err)
	}
}

func TestParseInsert_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple values", "INSERT INTO t (a, b) VALUES (1, 2)", "INSERT INTO t (a, b) VALUES (1, 2)"},
		{"no column list", "INSERT INTO t VALUES (1, 2)", "INSERT INTO t VALUES (1, 2)"},
		{"multi row values", "INSERT INTO t (a, b) VALUES (1, 2), (3, 4)", "INSERT INTO t (a, b) VALUES (1, 2), (3, 4)"},
		{"qualified table", "INSERT INTO public.t (a) VALUES (1)", "INSERT INTO public.t (a) VALUES (1)"},
		{"alias", "INSERT INTO t AS x (a) VALUES (1)", "INSERT INTO t AS x (a) VALUES (1)"},
		{
			"quoted identifiers keep their quotes",
			`INSERT INTO "Schema"."T" AS "X" ("Col") VALUES (1)`,
			`INSERT INTO "Schema"."T" AS "X" ("Col") VALUES (1)`,
		},
		{"expr in values", "INSERT INTO t (a) VALUES (1 + 2)", "INSERT INTO t (a) VALUES (1 + 2)"},
		{
			"placeholders and casts",
			"INSERT INTO t (a, b) VALUES ($1, $2::jsonb)",
			"INSERT INTO t (a, b) VALUES ($1, $2::jsonb)",
		},
		{"default in values", "INSERT INTO t (a, b) VALUES (DEFAULT, 1)", "INSERT INTO t (a, b) VALUES (DEFAULT, 1)"},
		{"default values", "INSERT INTO t DEFAULT VALUES", "INSERT INTO t DEFAULT VALUES"},
		{"trailing semicolon", "INSERT INTO t VALUES (1);", "INSERT INTO t VALUES (1)"},
		{"lowercase keywords", "insert into t (a) values (1)", "INSERT INTO t (a) VALUES (1)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_select(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"select", "INSERT INTO t (a) SELECT a FROM s", "INSERT INTO t (a) SELECT a FROM s"},
		{"select star", "INSERT INTO t SELECT * FROM s", "INSERT INTO t SELECT * FROM s"},
		{
			"select with where and order",
			"INSERT INTO t (a, b) SELECT x, y FROM s WHERE x > 0 ORDER BY x LIMIT 10",
			"INSERT INTO t (a, b) SELECT x, y FROM s WHERE x > 0 ORDER BY x LIMIT 10",
		},
		{
			"select with join",
			"INSERT INTO t SELECT s.a FROM s JOIN u ON s.id = u.id",
			"INSERT INTO t SELECT s.a FROM s JOIN u ON s.id = u.id",
		},
		{
			"select followed by on conflict",
			"INSERT INTO t SELECT * FROM s ON CONFLICT DO NOTHING",
			"INSERT INTO t SELECT * FROM s ON CONFLICT DO NOTHING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_onConflict(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"do nothing", "INSERT INTO t (a) VALUES (1) ON CONFLICT DO NOTHING", "INSERT INTO t (a) VALUES (1) ON CONFLICT DO NOTHING"},
		{
			"do nothing with target",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a, b) DO NOTHING",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a, b) DO NOTHING",
		},
		{
			"do update with excluded",
			"INSERT INTO t (id, n) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET n = EXCLUDED.n",
			"INSERT INTO t (id, n) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET n = excluded.n",
		},
		{
			"do update multiple assignments",
			"INSERT INTO t (id, a, b) VALUES (1, 2, 3) ON CONFLICT (id) DO UPDATE SET a = excluded.a + t.a, b = DEFAULT",
			"INSERT INTO t (id, a, b) VALUES (1, 2, 3) ON CONFLICT (id) DO UPDATE SET a = excluded.a + t.a, b = DEFAULT",
		},
		{
			"do update multi-column assignment",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = (EXCLUDED.a, EXCLUDED.b)",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = (excluded.a, excluded.b)",
		},
		{
			"do update row constructor assignment",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = ROW(excluded.a, DEFAULT)",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = ROW(excluded.a, DEFAULT)",
		},
		{
			"do update sub-select assignment",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = (SELECT x, y FROM u)",
			"INSERT INTO t (a, b) VALUES (1, 2) ON CONFLICT (a) DO UPDATE SET (a, b) = (SELECT x, y FROM u)",
		},
		{
			"do update composite field assignment",
			"INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET c.f = 1",
			"INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET c.f = 1",
		},
		{
			"do update with where",
			"INSERT INTO t AS x (id, n) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET n = excluded.n WHERE x.n < excluded.n",
			"INSERT INTO t AS x (id, n) VALUES (1, 2) ON CONFLICT (id) DO UPDATE SET n = excluded.n WHERE x.n < excluded.n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_returning(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"star", "INSERT INTO t (a) VALUES (1) RETURNING *", "INSERT INTO t (a) VALUES (1) RETURNING *"},
		{"columns", "INSERT INTO t (a) VALUES (1) RETURNING id, a", "INSERT INTO t (a) VALUES (1) RETURNING id, a"},
		{
			"aliased expressions",
			"INSERT INTO t (a) VALUES (1) RETURNING id AS new_id, a + 1 next_a",
			"INSERT INTO t (a) VALUES (1) RETURNING id AS new_id, a + 1 AS next_a",
		},
		{"qualified star", "INSERT INTO t (a) VALUES (1) RETURNING t.*", "INSERT INTO t (a) VALUES (1) RETURNING t.*"},
		{"default values", "INSERT INTO t DEFAULT VALUES RETURNING id", "INSERT INTO t DEFAULT VALUES RETURNING id"},
		{
			"after on conflict",
			"INSERT INTO t (id) VALUES (1) ON CONFLICT (id) DO UPDATE SET id = excluded.id RETURNING id",
			"INSERT INTO t (id) VALUES (1) ON CONFLICT (id) DO UPDATE SET id = excluded.id RETURNING id",
		},
		{
			"after select",
			"INSERT INTO t SELECT * FROM s RETURNING *",
			"INSERT INTO t SELECT * FROM s RETURNING *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_errors(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"empty", ""},
		{"missing into", "INSERT t VALUES (1)"},
		{"missing row source", "INSERT INTO t"},
		{"alias without as", "INSERT INTO t x VALUES (1)"},
		{"empty column list", "INSERT INTO t () VALUES (1)"},
		{"unclosed column list", "INSERT INTO t (a VALUES (1)"},
		{"empty values row", "INSERT INTO t VALUES ()"},
		{"unclosed values row", "INSERT INTO t VALUES (1"},
		{"default without values", "INSERT INTO t DEFAULT"},
		{"default outside values", "INSERT INTO t VALUES (DEFAULT + 1)"},
		{"on without conflict", "INSERT INTO t VALUES (1) ON DO NOTHING"},
		{"conflict without action", "INSERT INTO t VALUES (1) ON CONFLICT (a)"},
		{"do without action", "INSERT INTO t VALUES (1) ON CONFLICT DO"},
		{"do update without target", "INSERT INTO t VALUES (1) ON CONFLICT DO UPDATE SET a = 1"},
		{"do update without set", "INSERT INTO t VALUES (1) ON CONFLICT (a) DO UPDATE a = 1"},
		{"set without equals", "INSERT INTO t VALUES (1) ON CONFLICT (a) DO UPDATE SET a 1"},
		{"returning without items", "INSERT INTO t VALUES (1) RETURNING"},
		{"trailing tokens", "INSERT INTO t VALUES (1) garbage"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertError(t, tt.in)
		})
	}

	t.Run("lex error", func(t *testing.T) {
		_, err := pgparser.ParseInsert("INSERT INTO t VALUES ('unterminated)")
		if _, ok := errors.AsType[*pgparser.LexError](err); !ok {
			t.Errorf("error type = %T, want *pgparser.LexError", err)
		}
	})
}
