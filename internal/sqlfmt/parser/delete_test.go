package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

// assertDeleteRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertDeleteRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	del, err := parser.ParseDelete(input)
	if err != nil {
		t.Fatalf("ParseDelete(%q) error = %v", input, err)
	}

	if got := del.String(); got != want {
		t.Errorf("ParseDelete(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseDelete_singleTable(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "DELETE FROM users", "DELETE FROM users"},
		{"qualified table", "DELETE FROM mydb.users", "DELETE FROM mydb.users"},
		{"explicit alias", "DELETE FROM users AS u", "DELETE FROM users u"},
		{"implicit alias", "DELETE FROM users u", "DELETE FROM users u"},
		{"where", "DELETE FROM t WHERE id = 1", "DELETE FROM t WHERE id = 1"},
		{"order by limit", "DELETE FROM t ORDER BY id DESC LIMIT 10", "DELETE FROM t ORDER BY id DESC LIMIT 10"},
		{"ignore", "DELETE IGNORE FROM t", "DELETE IGNORE FROM t"},
		{
			"ignore with alias where order limit",
			"DELETE IGNORE FROM t AS x WHERE x.id = 1 ORDER BY x.id LIMIT 5",
			"DELETE IGNORE FROM t x WHERE x.id = 1 ORDER BY x.id LIMIT 5",
		},
		{"index hint", "DELETE FROM t USE INDEX (idx1) WHERE id = 1", "DELETE FROM t USE INDEX (idx1) WHERE id = 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_multiTableTargetList(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"two targets with join",
			"DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
			"DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
		},
		{
			"single target with where",
			"DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.t1_id WHERE t2.flag = 1",
			"DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.t1_id WHERE t2.flag = 1",
		},
		{
			"ignore",
			"DELETE IGNORE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
			"DELETE IGNORE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_multiTableUsing(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"two targets with join",
			"DELETE FROM t1, t2 USING t1 JOIN t2 ON t1.id = t2.t1_id",
			"DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
		},
		{
			"single target with where",
			"DELETE FROM t1 USING t1 JOIN t2 ON t1.id = t2.t1_id WHERE t2.flag = 1",
			"DELETE t1 FROM t1 JOIN t2 ON t1.id = t2.t1_id WHERE t2.flag = 1",
		},
		{
			"ignore",
			"DELETE IGNORE FROM t1, t2 USING t1 JOIN t2 ON t1.id = t2.t1_id",
			"DELETE IGNORE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_cte(t *testing.T) {
	assertDeleteRoundTrip(t,
		"WITH c AS (SELECT id FROM t) DELETE FROM t WHERE id IN (SELECT id FROM c)",
		"WITH c AS (SELECT id FROM t) DELETE FROM t WHERE id IN (SELECT id FROM c)")
}

func TestParseDelete_errors(t *testing.T) {
	tests := []string{
		"",
		"SELECT * FROM t",
		"DELETE",
		"DELETE FROM",
		"DELETE FROM t1, t2 WHERE t1.id = t2.id",
		"DELETE t1 t2",
		"DELETE FROM t WHERE",
		"DELETE FROM t ORDER",
		"DELETE FROM t LIMIT",
		"DELETE FROM t WHERE id = 1 EXTRA",
		"DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id ORDER BY t1.id",
		"DELETE t1, t2 FROM t1 JOIN t2 ON t1.id = t2.t1_id LIMIT 10",
		"DELETE FROM t1, t2 USING t1 JOIN t2 ON t1.id = t2.t1_id ORDER BY t1.id",
		"DELETE FROM t1, t2 USING t1 JOIN t2 ON t1.id = t2.t1_id LIMIT 10",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			del, err := parser.ParseDelete(in)
			if err == nil {
				t.Fatalf("ParseDelete(%q) expected error, got nil (result: %v)", in, del)
			}

			if _, ok := err.(*parser.ParseError); !ok { //nolint:errorlint // asserting the concrete parse-error type
				t.Errorf("ParseDelete(%q) error type = %T, want *parser.ParseError", in, err)
			}
		})
	}
}

// TestParseDelete_lexError checks that a lexer-level failure deep inside a
// DELETE statement propagates as a *parser.LexError, mirroring the
// advance()/expect() error-propagation coverage in TestParseExpr_errors.
func TestParseDelete_lexError(t *testing.T) {
	_, err := parser.ParseDelete("DELETE FROM t WHERE id = 'unterminated")
	if err == nil {
		t.Fatal("ParseDelete(...) expected error, got nil")
	}

	if _, ok := err.(*parser.LexError); !ok { //nolint:errorlint // asserting the concrete lex-error type
		t.Errorf("ParseDelete(...) error type = %T, want *parser.LexError", err)
	}
}
