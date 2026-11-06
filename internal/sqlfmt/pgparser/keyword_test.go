package pgparser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func assertStatementString(t *testing.T, input, want string) {
	t.Helper()

	stmt, err := pgparser.ParseStatement(input)
	if err != nil {
		t.Fatalf("ParseStatement(%q) error = %v", input, err)
	}

	if got := stmt.String(); got != want {
		t.Errorf("ParseStatement(%q).String() = %q, want %q", input, got, want)
	}
}

// Unreserved and col_name keywords are ColIds: they name columns, tables,
// aliases, CTEs, and windows.
func TestParseStatement_keywordAsColID(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"column key", "SELECT key FROM kv", "SELECT key FROM kv"},
		{"columns first and last", "SELECT first, last FROM t", "SELECT first, last FROM t"},
		{
			"locking and frame words",
			"SELECT next, rows, range, groups, no, of, share, skip, locked, nulls, ties FROM t",
			"SELECT next, rows, range, groups, no, of, share, skip, locked, nulls, ties FROM t",
		},
		{
			"window and grouping words",
			"SELECT filter, over, partition, current, cube, rollup, sets FROM t",
			"SELECT filter, over, partition, current, cube, rollup, sets FROM t",
		},
		{
			"dml words",
			"SELECT conflict, nothing, insert, update, delete, set FROM t",
			"SELECT conflict, nothing, insert, update, delete, set FROM t",
		},
		{"col_name keyword values", "SELECT values FROM t", "SELECT values FROM t"},
		{"upper-case keyword folds to lower case", "SELECT KEY FROM KV", "SELECT key FROM kv"},
		{"qualified column", "SELECT kv.key FROM kv", "SELECT kv.key FROM kv"},
		{"table name", "SELECT * FROM key", "SELECT * FROM key"},
		{"schema-qualified table name", "SELECT * FROM first.last", "SELECT * FROM first.last"},
		{"table alias with as", "SELECT * FROM t AS first", "SELECT * FROM t first"},
		{"implicit table alias", "SELECT * FROM t first WHERE first.a = 1", "SELECT * FROM t first WHERE first.a = 1"},
		{"qualified star", "SELECT key.* FROM t key", "SELECT key.* FROM t key"},
		{"cte name", "WITH key AS (SELECT 1) SELECT * FROM key", "WITH key AS (SELECT 1) SELECT * FROM key"},
		{"cte column list", "WITH x (key, value) AS (SELECT 1, 2) SELECT key FROM x", "WITH x (key, value) AS (SELECT 1, 2) SELECT key FROM x"},
		{"join using", "SELECT * FROM a JOIN b USING (key)", "SELECT * FROM a JOIN b USING (key)"},
		{
			"window name",
			"SELECT sum(a) OVER first FROM t WINDOW first AS (ORDER BY key)",
			"SELECT sum(a) OVER first FROM t WINDOW first AS (ORDER BY key)",
		},
		{
			"base window name",
			"SELECT sum(a) OVER (first ROWS UNBOUNDED PRECEDING) FROM t WINDOW first AS (ORDER BY key)",
			"SELECT sum(a) OVER (first ROWS UNBOUNDED PRECEDING) FROM t WINDOW first AS (ORDER BY key)",
		},
		{"update set target", "UPDATE t SET current = 1", "UPDATE t SET current = 1"},
		{"update composite field", "UPDATE t SET key.first = 1", "UPDATE t SET key.first = 1"},
		{"update target alias", "UPDATE t first SET a = 1", "UPDATE t first SET a = 1"},
		{"insert columns", "INSERT INTO t (key, last) VALUES (1, 2)", "INSERT INTO t (key, last) VALUES (1, 2)"},
		{"insert alias", "INSERT INTO t AS key VALUES (1)", "INSERT INTO t AS key VALUES (1)"},
		{
			"on conflict target",
			"INSERT INTO t (key) VALUES (1) ON CONFLICT (key) DO NOTHING",
			"INSERT INTO t (key) VALUES (1) ON CONFLICT (key) DO NOTHING",
		},
		{"delete target alias", "DELETE FROM t first WHERE first.key = 1", "DELETE FROM t first WHERE first.key = 1"},
		{"for update of", "SELECT * FROM key FOR UPDATE OF key", "SELECT * FROM key FOR UPDATE OF key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}
}

// Unreserved and type_func_name keywords are type_function_names: they name
// functions and types.
func TestParseStatement_keywordAsFunctionName(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"left", "SELECT left(s, 2) FROM t", "SELECT left(s, 2) FROM t"},
		{"right", "SELECT right(s, 2) FROM t", "SELECT right(s, 2) FROM t"},
		{"unreserved", "SELECT first(a) FROM t", "SELECT first(a) FROM t"},
		{"qualified with reserved name", "SELECT s.select(a) FROM t", "SELECT s.select(a) FROM t"},
		{"left function next to left join", "SELECT left(a.s, 1) FROM a LEFT JOIN b ON TRUE", "SELECT left(a.s, 1) FROM a LEFT JOIN b ON TRUE"},
		{"type name", "SELECT a::key FROM t", "SELECT a::key FROM t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}
}

// Any keyword is a ColLabel: a select-list alias after AS, or a name after a
// dot. Without AS, only BARE_LABEL keywords are aliases.
func TestParseStatement_keywordAsLabel(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"reserved alias after as", "SELECT 1 AS from", "SELECT 1 AS from"},
		{"type_func_name alias after as", "SELECT 1 AS left", "SELECT 1 AS left"},
		{"bare unreserved alias", "SELECT 1 first", "SELECT 1 AS first"},
		{"bare reserved alias", "SELECT 1 select", "SELECT 1 AS select"},
		{"bare alias before from", "SELECT a key FROM t", "SELECT a AS key FROM t"},
		{"returning alias", "DELETE FROM t RETURNING id AS end", "DELETE FROM t RETURNING id AS end"},
		{"reserved column after dot", "SELECT t.from FROM t", "SELECT t.from FROM t"},
		{"reserved table after dot", "SELECT * FROM s.where", "SELECT * FROM s.where"},
		{"reserved qualified star", "SELECT s.select.* FROM s.select", "SELECT s.select.* FROM s.select"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}
}

func TestParseStatement_keywordErrors(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"type_func_name as column", "SELECT left FROM t"},
		{"type_func_name as table", "SELECT * FROM left"},
		{"reserved as column", "SELECT from FROM t"},
		{"reserved as table alias after as", "SELECT * FROM t AS where"},
		{"as_label keyword as bare alias", "SELECT 1 over"},
		{"col_name keyword as function", "SELECT values(1)"},
		{"reserved as cte name", "WITH select AS (SELECT 1) SELECT 1"},
		{"reserved as set target", "UPDATE t SET from = 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pgparser.ParseStatement(tt.in)
			if _, ok := errors.AsType[*pgparser.ParseError](err); !ok {
				t.Errorf("ParseStatement(%q) error = %v, want *pgparser.ParseError", tt.in, err)
			}
		})
	}
}
