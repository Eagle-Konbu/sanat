package parser_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// assertStatementRoundTrip checks that parsing input via ParseStatement
// produces a statement of the given concrete type, and that re-stringifying
// it reproduces the canonical SQL text in want.
func assertStatementRoundTrip(t *testing.T, input, want string, wantType sqlast.Statement) {
	t.Helper()

	stmt, err := parser.ParseStatement(input)
	if err != nil {
		t.Fatalf("ParseStatement(%q) error = %v", input, err)
	}

	gotType := fmt.Sprintf("%T", stmt)
	wantTypeStr := fmt.Sprintf("%T", wantType)

	if gotType != wantTypeStr {
		t.Errorf("ParseStatement(%q) type = %s, want %s", input, gotType, wantTypeStr)
	}

	if got := stmt.String(); got != want {
		t.Errorf("ParseStatement(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseStatement_dispatch(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantType       sqlast.Statement
	}{
		{"select", "SELECT id FROM t", "SELECT id FROM t", &sqlast.Select{}},
		{"with select", "WITH c AS (SELECT 1) SELECT * FROM c",
			"WITH c AS (SELECT 1) SELECT * FROM c", &sqlast.Select{}},
		{"union", "SELECT 1 UNION SELECT 2", "SELECT 1 UNION SELECT 2", &sqlast.Union{}},
		{"with union", "WITH c AS (SELECT 1) SELECT * FROM c UNION SELECT * FROM u",
			"WITH c AS (SELECT 1) SELECT * FROM c UNION SELECT * FROM u", &sqlast.Union{}},
		{"insert", "INSERT INTO t (a) VALUES (1)", "INSERT INTO t (a) VALUES (1)", &sqlast.Insert{}},
		{"replace", "REPLACE INTO t (a) VALUES (1)", "REPLACE INTO t (a) VALUES (1)", &sqlast.Insert{}},
		{"update", "UPDATE t SET a = 1", "UPDATE t SET a = 1", &sqlast.Update{}},
		{"with update", "WITH c AS (SELECT 1) UPDATE t SET a = 1 WHERE id IN (SELECT 1 FROM c)",
			"WITH c AS (SELECT 1) UPDATE t SET a = 1 WHERE id IN (SELECT 1 FROM c)", &sqlast.Update{}},
		{"delete", "DELETE FROM t", "DELETE FROM t", &sqlast.Delete{}},
		{"with delete", "WITH c AS (SELECT 1) DELETE FROM t WHERE id IN (SELECT 1 FROM c)",
			"WITH c AS (SELECT 1) DELETE FROM t WHERE id IN (SELECT 1 FROM c)", &sqlast.Delete{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementRoundTrip(t, tt.in, tt.want, tt.wantType)
		})
	}
}

func TestParseStatement_errors(t *testing.T) {
	tests := []string{
		"",
		"WITH c AS (SELECT 1) INSERT INTO t (a) VALUES (1)",
		"CREATE TABLE t (id INT)",
		"SELECT 1 extra tokens",
		"SELECT 1 UNION SELECT 2 extra tokens",
		"INSERT INTO t (a) VALUES (1) extra tokens",
		"UPDATE t SET a = 1 extra tokens",
		"DELETE FROM t extra tokens",
		"SELECT VALUES(a)",
		"UPDATE t SET a = VALUES(name)",
	}

	for _, in := range tests {
		if _, err := parser.ParseStatement(in); err == nil {
			t.Errorf("ParseStatement(%q) expected error, got nil", in)
		}
	}
}

// TestParseStatement_errorType confirms the syntax-error case surfaces as a
// *parser.ParseError specifically, matching the other entry points' error
// model.
func TestParseStatement_errorType(t *testing.T) {
	_, err := parser.ParseStatement("CREATE TABLE t (id INT)")
	if err == nil {
		t.Fatal("ParseStatement(...) expected error, got nil")
	}

	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("ParseStatement(...) error type = %T, want *parser.ParseError", err)
	}
}
