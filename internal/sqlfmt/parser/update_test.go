package parser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

// assertUpdateRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertUpdateRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	upd, err := parser.ParseUpdate(input)
	if err != nil {
		t.Fatalf("ParseUpdate(%q) error = %v", input, err)
	}

	if got := upd.String(); got != want {
		t.Errorf("ParseUpdate(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseUpdate_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"single table", "UPDATE users SET name = 'bob'", "UPDATE users SET name = 'bob'"},
		{"ignore", "UPDATE IGNORE users SET name = 'bob'", "UPDATE IGNORE users SET name = 'bob'"},
		{"multiple set", "UPDATE t SET a = 1, b = 2, c = 3", "UPDATE t SET a = 1, b = 2, c = 3"},
		{"qualified column", "UPDATE t SET t.a = 1", "UPDATE t SET t.a = 1"},
		{"where", "UPDATE t SET a = 1 WHERE id = 1", "UPDATE t SET a = 1 WHERE id = 1"},
		{"order by limit", "UPDATE t SET a = 1 ORDER BY id DESC LIMIT 10",
			"UPDATE t SET a = 1 ORDER BY id DESC LIMIT 10"},
		{"order by asc default", "UPDATE t SET a = 1 ORDER BY id ASC", "UPDATE t SET a = 1 ORDER BY id"},
		{"where order by limit", "UPDATE t SET a = 1 WHERE a > 0 ORDER BY id LIMIT 5",
			"UPDATE t SET a = 1 WHERE a > 0 ORDER BY id LIMIT 5"},
		{"table alias", "UPDATE users AS u SET u.name = 'bob'", "UPDATE users u SET u.name = 'bob'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_multiTableJoin(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"comma tables", "UPDATE a, b SET a.x = 1, b.y = 2", "UPDATE a, b SET a.x = 1, b.y = 2"},
		{
			"join",
			"UPDATE a JOIN b ON a.id = b.a_id SET a.x = 1, b.y = 2",
			"UPDATE a JOIN b ON a.id = b.a_id SET a.x = 1, b.y = 2",
		},
		{
			"left join ignore",
			"UPDATE IGNORE a LEFT JOIN b ON a.id = b.a_id SET b.y = 2 WHERE a.x = 1",
			"UPDATE IGNORE a LEFT JOIN b ON a.id = b.a_id SET b.y = 2 WHERE a.x = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

// TestParseUpdate_multiTableOrderByLimitRejected verifies that ORDER BY and
// LIMIT, which MySQL only allows on the single-table form of UPDATE, are
// rejected (as trailing unconsumed tokens) when the target is a comma list
// or a JOIN -- mirroring the same restriction already enforced for DELETE's
// multi-table forms (see TestParseDelete_errors).
func TestParseUpdate_multiTableOrderByLimitRejected(t *testing.T) {
	tests := []string{
		"UPDATE a, b SET a.x = 1 ORDER BY a.id",
		"UPDATE a, b SET a.x = 1 LIMIT 10",
		"UPDATE a JOIN b ON a.id = b.id SET a.x = 1 ORDER BY a.id",
		"UPDATE a JOIN b ON a.id = b.id SET a.x = 1 LIMIT 10",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			upd, err := parser.ParseUpdate(in)
			if err == nil {
				t.Fatalf("ParseUpdate(%q) expected error, got nil (result: %v)", in, upd)
			}
		})
	}
}

func TestParseUpdate_cte(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "WITH c AS (SELECT 1) UPDATE t SET a = c.a", "WITH c AS (SELECT 1) UPDATE t SET a = c.a"},
		{
			"recursive",
			"WITH RECURSIVE c AS (SELECT 1) UPDATE t SET a = 1",
			"WITH RECURSIVE c AS (SELECT 1) UPDATE t SET a = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_structural(t *testing.T) {
	upd, err := parser.ParseUpdate("UPDATE t SET a = 1 WHERE id = 1")
	if err != nil {
		t.Fatalf("ParseUpdate error = %v", err)
	}

	if len(upd.Exprs) != 1 {
		t.Fatalf("len(Exprs) = %d, want 1", len(upd.Exprs))
	}

	setExpr := upd.Exprs[0]
	if setExpr.Name.String() != "a" {
		t.Errorf("Exprs[0].Name = %q, want %q", setExpr.Name.String(), "a")
	}

	if setExpr.Expr.String() != "1" {
		t.Errorf("Exprs[0].Expr = %q, want %q", setExpr.Expr.String(), "1")
	}

	if upd.Where == nil || upd.Where.Expr.String() != "id = 1" {
		t.Errorf("Where = %#v, want expr %q", upd.Where, "id = 1")
	}
}

func TestParseUpdate_errors(t *testing.T) {
	tests := []string{
		"",
		"UPDATE",
		"UPDATE IGNORE",
		"UPDATE t",
		"UPDATE t SET",
		"UPDATE t SET a",
		"UPDATE t SET a =",
		"UPDATE t SET a = 1 WHERE",
		"UPDATE t SET a = 1 ORDER",
		"UPDATE t SET a = 1 ORDER BY",
		"UPDATE t SET a = 1 LIMIT",
		"UPDATE t JOIN",
		"WITH",
		"WITH c",
		"UPDATE t SET a = 1 extra tokens",
		"SELECT 1",

		// Lexer-level errors positioned deep inside each clause, mirroring
		// the SELECT parser's error-propagation coverage.
		"UPDATE 'unterminated",
		"UPDATE t SET 'unterminated",
		"UPDATE t SET a = 'unterminated",
		"UPDATE t SET a = 1 WHERE 'unterminated",
		"UPDATE t SET a = 1 ORDER BY 'unterminated",
		"UPDATE t SET a = 1 LIMIT 'unterminated",
		"UPDATE t JOIN 'unterminated",
	}

	for _, in := range tests {
		if _, err := parser.ParseUpdate(in); err == nil {
			t.Errorf("ParseUpdate(%q) expected error, got nil", in)
		}
	}
}

func TestParseUpdate_errorType(t *testing.T) {
	_, err := parser.ParseUpdate("UPDATE t SET")
	if err == nil {
		t.Fatal("ParseUpdate expected error, got nil")
	}

	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("ParseUpdate error type = %T, want *parser.ParseError", err)
	}
}
