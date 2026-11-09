package pgparser_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertStatementRoundTrip checks that parsing input yields a statement of
// wantType whose re-stringified form is want.
func assertStatementRoundTrip(t *testing.T, input, want string, wantType pgast.Statement) {
	t.Helper()

	stmt, err := pgparser.ParseStatement(input)
	if err != nil {
		t.Fatalf("ParseStatement(%q) error = %v", input, err)
	}

	if got, wantT := reflect.TypeOf(stmt), reflect.TypeOf(wantType); got != wantT {
		t.Errorf("ParseStatement(%q) type = %v, want %v", input, got, wantT)
	}

	if got := stmt.String(); got != want {
		t.Errorf("ParseStatement(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseStatement_dispatch(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantType       pgast.Statement
	}{
		{"select", "SELECT 1", "SELECT 1", &pgast.Select{}},
		{"parenthesized select", "(SELECT 1)", "SELECT 1", &pgast.Select{}},
		{"set operation", "SELECT 1 UNION SELECT 2", "SELECT 1 UNION SELECT 2", &pgast.SetOp{}},
		{"insert", "INSERT INTO t VALUES (1)", "INSERT INTO t VALUES (1)", &pgast.Insert{}},
		{"update", "UPDATE t SET a = 1", "UPDATE t SET a = 1", &pgast.Update{}},
		{"delete", "DELETE FROM t", "DELETE FROM t", &pgast.Delete{}},
		{
			"merge",
			"MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
			"MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
			&pgast.Merge{},
		},
		{"trailing semicolon", "DELETE FROM t;", "DELETE FROM t", &pgast.Delete{}},
		{"lower-case keywords", "select 1 union select 2", "SELECT 1 UNION SELECT 2", &pgast.SetOp{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementRoundTrip(t, tt.in, tt.want, tt.wantType)
		})
	}
}

func TestParseStatement_with(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantType       pgast.Statement
	}{
		{
			"select",
			"WITH x AS (SELECT 1) SELECT * FROM x",
			"WITH x AS (SELECT 1) SELECT * FROM x",
			&pgast.Select{},
		},
		{
			"set operation",
			"WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2",
			"WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2",
			&pgast.SetOp{},
		},
		{
			"insert",
			"WITH x AS (SELECT 1) INSERT INTO t SELECT * FROM x",
			"WITH x AS (SELECT 1) INSERT INTO t SELECT * FROM x",
			&pgast.Insert{},
		},
		{
			"update",
			"WITH RECURSIVE x AS (SELECT 1) UPDATE t SET a = 1",
			"WITH RECURSIVE x AS (SELECT 1) UPDATE t SET a = 1",
			&pgast.Update{},
		},
		{
			"delete",
			"WITH x AS (SELECT 1) DELETE FROM t USING x",
			"WITH x AS (SELECT 1) DELETE FROM t USING x",
			&pgast.Delete{},
		},
		{
			"merge",
			"WITH x AS (SELECT 1 AS id) MERGE INTO t USING x ON t.id = x.id WHEN MATCHED THEN DELETE",
			"WITH x AS (SELECT 1 AS id) MERGE INTO t USING x ON t.id = x.id WHEN MATCHED THEN DELETE",
			&pgast.Merge{},
		},
		{
			"data-modifying cte",
			"WITH d AS (DELETE FROM t RETURNING *) INSERT INTO u SELECT * FROM d",
			"WITH d AS (DELETE FROM t RETURNING *) INSERT INTO u SELECT * FROM d",
			&pgast.Insert{},
		},
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
		";",
		"WITH x AS (SELECT 1)",
		"WITH x AS (SELECT 1) WITH y AS (SELECT 2) SELECT 1",
		"WITH x AS (SELECT 1) (WITH y AS (SELECT 2) SELECT 1)",
		"MERGE INTO t USING s ON TRUE",
		"SELECT 1; SELECT 2",
		"INSERT INTO t VALUES (1) garbage",
	}

	for _, in := range tests {
		t.Run(in, func(t *testing.T) {
			_, err := pgparser.ParseStatement(in)
			if _, ok := errors.AsType[*pgparser.ParseError](err); !ok {
				t.Errorf("ParseStatement(%q) error = %v, want *pgparser.ParseError", in, err)
			}
		})
	}
}
