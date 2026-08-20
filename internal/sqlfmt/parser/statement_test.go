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
		{"create table", "CREATE TABLE t (id INT)", "CREATE TABLE t (id INT)", &sqlast.CreateTable{}},
		{"create index", "CREATE INDEX idx ON t (a)", "CREATE INDEX idx ON t (a)", &sqlast.CreateIndex{}},
		{"alter table", "ALTER TABLE t ADD COLUMN a INT", "ALTER TABLE t ADD COLUMN a INT", &sqlast.AlterTable{}},
		{"drop table", "DROP TABLE t", "DROP TABLE t", &sqlast.DropTable{}},
		{"drop index", "DROP INDEX idx ON t", "DROP INDEX idx ON t", &sqlast.DropIndex{}},
		{"truncate table", "TRUNCATE TABLE t", "TRUNCATE TABLE t", &sqlast.TruncateTable{}},
		{"start transaction", "START TRANSACTION", "START TRANSACTION", &sqlast.StartTransaction{}},
		{"begin", "BEGIN", "BEGIN", &sqlast.Begin{}},
		{"commit", "COMMIT", "COMMIT", &sqlast.Commit{}},
		{"rollback", "ROLLBACK", "ROLLBACK", &sqlast.Rollback{}},
		{"savepoint", "SAVEPOINT sp1", "SAVEPOINT sp1", &sqlast.Savepoint{}},
		{"release savepoint", "RELEASE SAVEPOINT sp1", "RELEASE SAVEPOINT sp1", &sqlast.ReleaseSavepoint{}},
		{"set variable", "SET @rank = 0", "SET @rank = 0", &sqlast.SetVariable{}},
		{"set names", "SET NAMES utf8mb4", "SET NAMES utf8mb4", &sqlast.SetNames{}},
		{"show tables", "SHOW TABLES", "SHOW TABLES", &sqlast.ShowTables{}},
		{"show create table", "SHOW CREATE TABLE t", "SHOW CREATE TABLE t", &sqlast.ShowCreateTable{}},
		{"show columns", "SHOW COLUMNS FROM t", "SHOW COLUMNS FROM t", &sqlast.ShowColumns{}},
		{"show index", "SHOW INDEX FROM t", "SHOW INDEX FROM t", &sqlast.ShowIndex{}},
		{"show databases", "SHOW DATABASES", "SHOW DATABASES", &sqlast.ShowDatabases{}},
		{"show variables", "SHOW VARIABLES", "SHOW VARIABLES", &sqlast.ShowVariables{}},
		{"show status", "SHOW STATUS", "SHOW STATUS", &sqlast.ShowStatus{}},
		{"describe", "DESCRIBE t", "DESCRIBE t", &sqlast.Describe{}},
		{"explain", "EXPLAIN SELECT 1", "EXPLAIN SELECT 1", &sqlast.Explain{}},
		{"use", "USE db", "USE db", &sqlast.Use{}},
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
		"WITH c AS (SELECT 1) CREATE TABLE t (id INT)",
		"WITH c AS (SELECT 1) ALTER TABLE t ADD COLUMN a INT",
		"WITH c AS (SELECT 1) DROP TABLE t",
		"WITH c AS (SELECT 1) TRUNCATE TABLE t",
		"WITH c AS (SELECT 1) SHOW TABLES",
		"WITH c AS (SELECT 1) DESCRIBE t",
		"WITH c AS (SELECT 1) USE db",
		"CALL proc()",
		"SELECT 1 extra tokens",
		"SELECT 1 UNION SELECT 2 extra tokens",
		"INSERT INTO t (a) VALUES (1) extra tokens",
		"UPDATE t SET a = 1 extra tokens",
		"DELETE FROM t extra tokens",
		"CREATE TABLE t (id INT) 1",
		"ALTER TABLE t ADD COLUMN a INT extra tokens",
		"DROP TABLE t extra tokens",
		"TRUNCATE TABLE t extra tokens",
		"SELECT VALUES(a)",
		"UPDATE t SET a = VALUES(name)",
		"WITH c AS (SELECT 1) START TRANSACTION",
		"WITH c AS (SELECT 1) SET @rank = 0",
		"START TRANSACTION extra tokens",
		"BEGIN extra tokens",
		"COMMIT extra tokens",
		"ROLLBACK extra tokens",
		"SAVEPOINT sp1 extra tokens",
		"RELEASE SAVEPOINT sp1 extra tokens",
		"SET @rank = 0 extra tokens",
		"SET NAMES utf8mb4 extra tokens",
		"SHOW TABLES extra tokens",
		"SHOW WAT",
		"DESCRIBE t extra tokens extra",
		"EXPLAIN extra tokens",
		"EXPLAIN FORMAT = XML SELECT 1",
		"USE db extra tokens",
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
	_, err := parser.ParseStatement("CALL proc()")
	if err == nil {
		t.Fatal("ParseStatement(...) expected error, got nil")
	}

	var parseErr *parser.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("ParseStatement(...) error type = %T, want *parser.ParseError", err)
	}
}
