package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

// assertInsertRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertInsertRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	ins, err := parser.ParseInsert(input)
	if err != nil {
		t.Fatalf("ParseInsert(%q) error = %v", input, err)
	}

	if got := ins.String(); got != want {
		t.Errorf("ParseInsert(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseInsert_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"simple values",
			"INSERT INTO t (a, b) VALUES (1, 2)",
			"INSERT INTO t (a, b) VALUES (1, 2)",
		},
		{
			"no column list",
			"INSERT INTO t VALUES (1, 2)",
			"INSERT INTO t VALUES (1, 2)",
		},
		{
			"multi row values",
			"INSERT INTO t (a, b) VALUES (1, 2), (3, 4)",
			"INSERT INTO t (a, b) VALUES (1, 2), (3, 4)",
		},
		{
			"insert ignore",
			"INSERT IGNORE INTO t (a) VALUES (1)",
			"INSERT IGNORE INTO t (a) VALUES (1)",
		},
		{
			"replace",
			"REPLACE INTO t (a) VALUES (1)",
			"REPLACE INTO t (a) VALUES (1)",
		},
		{
			"qualified table",
			"INSERT INTO mydb.t (a) VALUES (1)",
			"INSERT INTO mydb.t (a) VALUES (1)",
		},
		{
			"expr in values",
			"INSERT INTO t (a) VALUES (1 + 2)",
			"INSERT INTO t (a) VALUES (1 + 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_set(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"single assignment",
			"INSERT INTO t SET a = 1",
			"INSERT INTO t SET a = 1",
		},
		{
			"multiple assignments",
			"INSERT INTO t SET a = 1, b = 2",
			"INSERT INTO t SET a = 1, b = 2",
		},
		{
			"qualified column",
			"INSERT INTO t SET t.a = 1",
			"INSERT INTO t SET t.a = 1",
		},
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
		{
			"basic",
			"INSERT INTO t (a, b) SELECT x, y FROM u",
			"INSERT INTO t (a, b) SELECT x, y FROM u",
		},
		{
			"with where and join",
			"INSERT INTO t (a) SELECT u.x FROM u JOIN v ON u.id = v.id WHERE u.x > 1",
			"INSERT INTO t (a) SELECT u.x FROM u JOIN v ON u.id = v.id WHERE u.x > 1",
		},
		{
			"nested select with cte",
			"INSERT INTO t (a) WITH c AS (SELECT 1) SELECT * FROM c",
			"INSERT INTO t (a) WITH c AS (SELECT 1) SELECT * FROM c",
		},
		{
			"union rows",
			"INSERT INTO t SELECT a FROM x UNION SELECT b FROM y",
			"INSERT INTO t SELECT a FROM x UNION SELECT b FROM y",
		},
		{
			"union rows with leading with",
			"INSERT INTO t WITH c AS (SELECT 1) SELECT a FROM c UNION SELECT b FROM y",
			"INSERT INTO t WITH c AS (SELECT 1) SELECT a FROM c UNION SELECT b FROM y",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_onDuplicateKeyUpdate(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"after values",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE b = 3",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE b = 3",
		},
		{
			"multiple assignments",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE a = a + 1, b = 3",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE a = a + 1, b = 3",
		},
		{
			"after select",
			"INSERT INTO t (a) SELECT x FROM u ON DUPLICATE KEY UPDATE a = 1",
			"INSERT INTO t (a) SELECT x FROM u ON DUPLICATE KEY UPDATE a = 1",
		},
		{
			"deprecated VALUES() function reference",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE a = VALUES(a), b = VALUES(b)",
			"INSERT INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE a = VALUES(a), b = VALUES(b)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_errors(t *testing.T) {
	tests := []string{
		"",
		"INSERT",
		"INSERT INTO",
		"INSERT INTO t",
		"INSERT INTO t (",
		"INSERT INTO t (a",
		"INSERT INTO t (a)",
		"INSERT INTO t (a) VALUES",
		"INSERT INTO t (a) VALUES (",
		"INSERT INTO t (a) VALUES (1",
		"INSERT INTO t (a) VALUES (1)(2)",
		"INSERT INTO t SET",
		"INSERT INTO t SET a",
		"INSERT INTO t SET a =",
		"INSERT INTO t (a) VALUES (1) ON",
		"INSERT INTO t (a) VALUES (1) ON DUPLICATE",
		"INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY",
		"INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY UPDATE",
		"INSERT INTO t (a) VALUES (1) extra tokens",
		"UPDATE t SET a = 1",
		"REPLACE",
		"REPLACE IGNORE INTO t (a) VALUES (1)",
		"REPLACE INTO t (a, b) VALUES (1, 2) ON DUPLICATE KEY UPDATE b = 3",

		// Lexer-level errors positioned deep inside each clause.
		"INSERT INTO 'unterminated",
		"INSERT INTO t ('unterminated",
		"INSERT INTO t (a) VALUES ('unterminated",
		"INSERT INTO t SET a = 'unterminated",
		"INSERT INTO t (a) SELECT 'unterminated",
		"INSERT INTO t (a) VALUES (1) ON DUPLICATE KEY UPDATE a = 'unterminated",
	}

	for _, in := range tests {
		if _, err := parser.ParseInsert(in); err == nil {
			t.Errorf("ParseInsert(%q) expected error, got nil", in)
		}
	}
}
