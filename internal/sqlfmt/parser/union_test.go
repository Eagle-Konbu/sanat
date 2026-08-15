package parser_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// assertUnionRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertUnionRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	u, err := parser.ParseUnion(input)
	if err != nil {
		t.Fatalf("ParseUnion(%q) error = %v", input, err)
	}

	if got := u.String(); got != want {
		t.Errorf("ParseUnion(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseUnion_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"union distinct implicit", "SELECT 1 UNION SELECT 2", "SELECT 1 UNION SELECT 2"},
		{"union all", "SELECT 1 UNION ALL SELECT 2", "SELECT 1 UNION ALL SELECT 2"},
		{"union distinct explicit renders as bare union",
			"SELECT 1 UNION DISTINCT SELECT 2", "SELECT 1 UNION SELECT 2"},
		{"branches with from/where", "SELECT id FROM a WHERE id > 1 UNION SELECT id FROM b WHERE id < 5",
			"SELECT id FROM a WHERE id > 1 UNION SELECT id FROM b WHERE id < 5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUnionRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUnion_chained(t *testing.T) {
	assertUnionRoundTrip(t,
		"SELECT 1 UNION SELECT 2 UNION ALL SELECT 3 UNION SELECT 4",
		"SELECT 1 UNION SELECT 2 UNION ALL SELECT 3 UNION SELECT 4",
	)
}

func TestParseUnion_chained_structural(t *testing.T) {
	// a UNION b UNION c must build left-associatively: the outer union's
	// Left is itself a Union{Left: a, Right: b}, and its Right is c.
	u, err := parser.ParseUnion("SELECT 1 UNION SELECT 2 UNION ALL SELECT 3")
	if err != nil {
		t.Fatalf("ParseUnion error = %v", err)
	}

	inner, ok := u.Left.(*sqlast.Union)
	if !ok {
		t.Fatalf("Left = %#v, want *sqlast.Union", u.Left)
	}

	if _, ok := u.Right.(*sqlast.Select); !ok {
		t.Fatalf("Right = %#v, want *sqlast.Select", u.Right)
	}

	if _, ok := inner.Left.(*sqlast.Select); !ok {
		t.Fatalf("Left.Left = %#v, want *sqlast.Select", inner.Left)
	}

	if _, ok := inner.Right.(*sqlast.Select); !ok {
		t.Fatalf("Left.Right = %#v, want *sqlast.Select", inner.Right)
	}

	// Only the outermost Union should carry Distinct == false (ALL) for the
	// second UNION; the inner Union (first UNION, implicit DISTINCT) should
	// have Distinct == true.
	if !inner.Distinct {
		t.Errorf("inner.Distinct = false, want true (implicit UNION DISTINCT)")
	}

	if u.Distinct {
		t.Errorf("outer.Distinct = true, want false (UNION ALL)")
	}
}

func TestParseUnion_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "WITH c AS (SELECT 1) SELECT id FROM c UNION SELECT id FROM c",
			"WITH c AS (SELECT 1) SELECT id FROM c UNION SELECT id FROM c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUnionRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUnion_tail(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"order by", "SELECT id FROM a UNION SELECT id FROM b ORDER BY id",
			"SELECT id FROM a UNION SELECT id FROM b ORDER BY id"},
		{"limit", "SELECT id FROM a UNION SELECT id FROM b LIMIT 10",
			"SELECT id FROM a UNION SELECT id FROM b LIMIT 10"},
		{"order by limit", "SELECT id FROM a UNION SELECT id FROM b ORDER BY id DESC LIMIT 10 OFFSET 5",
			"SELECT id FROM a UNION SELECT id FROM b ORDER BY id DESC LIMIT 10 OFFSET 5"},
		{"locking", "SELECT id FROM a UNION SELECT id FROM b FOR UPDATE",
			"SELECT id FROM a UNION SELECT id FROM b FOR UPDATE"},
		{"order limit lock all together",
			"SELECT id FROM a UNION SELECT id FROM b ORDER BY id LIMIT 10 FOR UPDATE",
			"SELECT id FROM a UNION SELECT id FROM b ORDER BY id LIMIT 10 FOR UPDATE"},
		{"with order limit lock",
			"WITH c AS (SELECT 1) SELECT * FROM c UNION SELECT * FROM c ORDER BY 1 LIMIT 10 FOR UPDATE",
			"WITH c AS (SELECT 1) SELECT * FROM c UNION SELECT * FROM c ORDER BY 1 LIMIT 10 FOR UPDATE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUnionRoundTrip(t, tt.in, tt.want)
		})
	}
}

// TestParseUnion_tailNotOnBranches verifies structurally that the trailing
// ORDER BY/LIMIT/locking clause attaches only to the outermost *sqlast.Union
// and not to the individual branch *sqlast.Select nodes (the
// tail-attachment problem: a bare branch parsed via parseSelectCore must
// never itself greedily consume a trailing clause meant for the union).
func TestParseUnion_tailNotOnBranches(t *testing.T) {
	u, err := parser.ParseUnion("SELECT id FROM a UNION SELECT id FROM b ORDER BY id LIMIT 10 FOR UPDATE")
	if err != nil {
		t.Fatalf("ParseUnion error = %v", err)
	}

	// Full structural comparison against the expected shape, mirroring
	// TestUnion_String's "with order limit lock" case in sqlast: the branch
	// *sqlast.Select nodes carry none of OrderBy/Limit/Lock/With, which
	// belong solely to the outermost *sqlast.Union.
	want := &sqlast.Union{
		Left: &sqlast.Select{
			SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: col("id")}},
			From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "a"}}},
		},
		Right: &sqlast.Select{
			SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: col("id")}},
			From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "b"}}},
		},
		Distinct: true,
		OrderBy:  sqlast.OrderBy{{Expr: col("id")}},
		Limit:    &sqlast.Limit{Rowcount: num("10")},
		Lock:     sqlast.ForUpdateLock,
	}

	if !reflect.DeepEqual(u, want) {
		t.Errorf("ParseUnion =\n  %#v\nwant\n  %#v", u, want)
	}
}

func TestParseUnion_errors(t *testing.T) {
	tests := []string{
		"",
		// A plain SELECT with no UNION keyword at all: ParseUnion is
		// specifically the entry point for union statements.
		"SELECT 1",
		"SELECT * FROM t WHERE a = 1",
		"WITH c AS (SELECT 1) SELECT * FROM c",

		"SELECT 1 UNION",
		"SELECT 1 UNION ALL",
		"SELECT 1 UNION DISTINCT",
		"UNION SELECT 1",
		"SELECT 1 UNION SELECT 2 UNION",
		"SELECT 1 UNION SELECT 2 ORDER",
		"SELECT 1 UNION SELECT 2 ORDER BY",
		"SELECT 1 UNION SELECT 2 LIMIT",
		"SELECT 1 UNION SELECT 2 FOR",
		"SELECT 1 UNION SELECT 2 LOCK",
		"WITH",
		"SELECT 1 UNION SELECT 2 extra tokens FROM u",

		// Lexer-level errors positioned inside a union-specific position.
		"SELECT 1 UNION 'unterminated",
		"SELECT 1 UNION SELECT 2 ORDER BY 'unterminated",
		"SELECT 1 UNION SELECT 2 LIMIT 'unterminated",
	}

	for _, in := range tests {
		if _, err := parser.ParseUnion(in); err == nil {
			t.Errorf("ParseUnion(%q) expected error, got nil", in)
		}
	}
}

// TestParseUnion_errorType confirms the syntax-error case surfaces as a
// *parser.ParseError specifically, matching ParseSelect's error model.
func TestParseUnion_errorType(t *testing.T) {
	_, err := parser.ParseUnion("SELECT 1")
	if err == nil {
		t.Fatal("ParseUnion(\"SELECT 1\") expected error, got nil")
	}

	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("ParseUnion(\"SELECT 1\") error type = %T, want *parser.ParseError", err)
	}
}
