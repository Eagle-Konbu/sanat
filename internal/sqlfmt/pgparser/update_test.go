package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertUpdateRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertUpdateRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	upd, err := pgparser.ParseUpdate(input)
	if err != nil {
		t.Fatalf("ParseUpdate(%q) error = %v", input, err)
	}

	if got := upd.String(); got != want {
		t.Errorf("ParseUpdate(%q).String() = %q, want %q", input, got, want)
	}
}

func assertUpdateError(t *testing.T, input string) {
	t.Helper()

	if _, err := pgparser.ParseUpdate(input); err == nil {
		t.Errorf("ParseUpdate(%q) error = nil, want error", input)
	}
}

func TestParseUpdate_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"single table", "UPDATE users SET name = 'bob'", "UPDATE users SET name = 'bob'"},
		{"multiple set", "UPDATE t SET a = 1, b = 2, c = 3", "UPDATE t SET a = 1, b = 2, c = 3"},
		{"composite field target", "UPDATE t SET c.f = 1", "UPDATE t SET c.f = 1"},
		{"quoted identifiers", `UPDATE "Schema"."T" SET "Col" = 1`, `UPDATE "Schema"."T" SET "Col" = 1`},
		{"qualified table", "UPDATE public.users SET a = 1", "UPDATE public.users SET a = 1"},
		{"where", "UPDATE t SET a = 1 WHERE id = 1", "UPDATE t SET a = 1 WHERE id = 1"},
		{"expression value", "UPDATE t SET a = a + 1 WHERE id = $1", "UPDATE t SET a = a + 1 WHERE id = $1"},
		{"subquery value", "UPDATE t SET a = (SELECT max(b) FROM u)", "UPDATE t SET a = (SELECT max(b) FROM u)"},
		{"trailing semicolon", "UPDATE t SET a = 1;", "UPDATE t SET a = 1"},
		{"lowercase keywords", "update t set a = 1 where b = 2", "UPDATE t SET a = 1 WHERE b = 2"},
		{"set as column name", "UPDATE t SET set = set + 1", "UPDATE t SET set = set + 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_target(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"alias", "UPDATE users AS u SET name = 'bob'", "UPDATE users u SET name = 'bob'"},
		{"implicit alias", "UPDATE users u SET name = 'bob'", "UPDATE users u SET name = 'bob'"},
		{"only", "UPDATE ONLY users SET a = 1", "UPDATE ONLY users SET a = 1"},
		{"only with alias", "UPDATE ONLY users AS u SET a = 1", "UPDATE ONLY users u SET a = 1"},
		{"parenthesized only is normalized", "UPDATE ONLY (users) u SET a = 1", "UPDATE ONLY users u SET a = 1"},
		{"descendants star", "UPDATE users * SET a = 1", "UPDATE users * SET a = 1"},
		{"descendants star with alias", "UPDATE public.users * AS u SET a = 1", "UPDATE public.users * u SET a = 1"},
		{"set alias keeps as", "UPDATE t AS set SET a = set.a", "UPDATE t AS set SET a = set.a"},
		{"set alias with only", "UPDATE ONLY t AS set SET a = 1", "UPDATE ONLY t AS set SET a = 1"},
		{"quoted set alias", `UPDATE t "set" SET a = 1`, `UPDATE t "set" SET a = 1`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_setValues(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"default", "UPDATE t SET a = DEFAULT, b = 1", "UPDATE t SET a = DEFAULT, b = 1"},
		{"row value for composite column", "UPDATE t SET c = (1, 'x')", "UPDATE t SET c = (1, 'x')"},
		{"multi-column tuple", "UPDATE t SET (a, b) = (1, 2)", "UPDATE t SET (a, b) = (1, 2)"},
		{"multi-column tuple with default", "UPDATE t SET (a, b) = (DEFAULT, b + 1)", "UPDATE t SET (a, b) = (DEFAULT, b + 1)"},
		{"multi-column single element", "UPDATE t SET (a) = (1)", "UPDATE t SET (a) = (1)"},
		{"multi-column row constructor", "UPDATE t SET (a, b) = ROW(1, DEFAULT)", "UPDATE t SET (a, b) = ROW(1, DEFAULT)"},
		{
			"multi-column sub-select",
			"UPDATE t SET (a, b) = (SELECT x, y FROM u WHERE u.id = t.id)",
			"UPDATE t SET (a, b) = (SELECT x, y FROM u WHERE u.id = t.id)",
		},
		{
			"mixed single and multi-column",
			"UPDATE t SET a = 1, (b, c) = (2, 3), d = DEFAULT",
			"UPDATE t SET a = 1, (b, c) = (2, 3), d = DEFAULT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_from(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"from single table",
			"UPDATE t SET a = u.a FROM u WHERE t.id = u.id",
			"UPDATE t SET a = u.a FROM u WHERE t.id = u.id",
		},
		{
			"from multiple tables",
			"UPDATE t SET a = u.a FROM u, v WHERE t.id = u.id AND u.id = v.id",
			"UPDATE t SET a = u.a FROM u, v WHERE t.id = u.id AND u.id = v.id",
		},
		{
			"from join",
			"UPDATE t SET a = u.a FROM u JOIN v ON u.id = v.id WHERE t.id = u.id",
			"UPDATE t SET a = u.a FROM u JOIN v ON u.id = v.id WHERE t.id = u.id",
		},
		{
			"from derived table",
			"UPDATE t SET a = s.a FROM (SELECT id, a FROM u) AS s WHERE t.id = s.id",
			"UPDATE t SET a = s.a FROM (SELECT id, a FROM u) s WHERE t.id = s.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_returning(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"star", "UPDATE t SET a = 1 RETURNING *", "UPDATE t SET a = 1 RETURNING *"},
		{"columns", "UPDATE t SET a = 1 RETURNING id, a", "UPDATE t SET a = 1 RETURNING id, a"},
		{"aliases", "UPDATE t SET a = 1 RETURNING id AS x, a + 1 y", "UPDATE t SET a = 1 RETURNING id AS x, a + 1 AS y"},
		{"qualified star", "UPDATE t SET a = 1 RETURNING t.*", "UPDATE t SET a = 1 RETURNING t.*"},
		{
			"all clauses",
			"UPDATE ONLY t AS x SET a = u.a FROM u WHERE x.id = u.id RETURNING x.id",
			"UPDATE ONLY t x SET a = u.a FROM u WHERE x.id = u.id RETURNING x.id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_errors(t *testing.T) {
	tests := []string{
		"",
		"UPDATE",
		"UPDATE t",
		"UPDATE t SET",
		"UPDATE t SET a",
		"UPDATE t SET a =",
		"UPDATE t SET a = 1,",
		"UPDATE ONLY t * SET a = 1",
		"UPDATE ONLY (t SET a = 1",
		"UPDATE t SET (a, b) = 1",
		"UPDATE t SET (a, b) 1",
		"UPDATE t SET () = (1)",
		"UPDATE t SET a = 1 FROM",
		"UPDATE t SET a = 1 WHERE",
		"UPDATE t SET a = 1 WHERE CURRENT OF c",
		"UPDATE t SET a = 1 RETURNING",
		"UPDATE t SET a = 1 ORDER BY a",
		"UPDATE t SET a = 1 LIMIT 1",
		"UPDATE t SET a = 1; garbage",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			assertUpdateError(t, tt)
		})
	}
}
