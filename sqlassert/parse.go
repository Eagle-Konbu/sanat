package sqlassert

import (
	"fmt"
	"testing"

	"github.com/pingcap/tidb/pkg/parser"
	"github.com/pingcap/tidb/pkg/parser/ast"
	_ "github.com/pingcap/tidb/pkg/parser/test_driver"
)

// ParseOne parses a single SQL statement and returns the AST node.
// Returns an error if the SQL is invalid or contains multiple statements.
func ParseOne(sql string) (ast.StmtNode, error) {
	p := parser.New()
	stmts, _, err := p.Parse(sql, "", "")
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("no statements found")
	}
	if len(stmts) > 1 {
		return nil, fmt.Errorf("expected 1 statement, got %d", len(stmts))
	}
	return stmts[0], nil
}

// RequireParseOne parses a single SQL statement and fails the test if parsing fails.
func RequireParseOne(t *testing.T, sql string) ast.StmtNode {
	t.Helper()
	stmt, err := ParseOne(sql)
	if err != nil {
		t.Fatalf("failed to parse SQL: %v\nSQL: %s", err, sql)
	}
	return stmt
}

// RequireSelect asserts that the statement is a SELECT statement and returns it.
func RequireSelect(t *testing.T, stmt ast.StmtNode) *ast.SelectStmt {
	t.Helper()
	sel, ok := stmt.(*ast.SelectStmt)
	if !ok {
		t.Fatalf("expected *ast.SelectStmt, got %T", stmt)
	}
	return sel
}

// RequireInsert asserts that the statement is an INSERT statement and returns it.
func RequireInsert(t *testing.T, stmt ast.StmtNode) *ast.InsertStmt {
	t.Helper()
	ins, ok := stmt.(*ast.InsertStmt)
	if !ok {
		t.Fatalf("expected *ast.InsertStmt, got %T", stmt)
	}
	return ins
}

// RequireUpdate asserts that the statement is an UPDATE statement and returns it.
func RequireUpdate(t *testing.T, stmt ast.StmtNode) *ast.UpdateStmt {
	t.Helper()
	upd, ok := stmt.(*ast.UpdateStmt)
	if !ok {
		t.Fatalf("expected *ast.UpdateStmt, got %T", stmt)
	}
	return upd
}

// RequireDelete asserts that the statement is a DELETE statement and returns it.
func RequireDelete(t *testing.T, stmt ast.StmtNode) *ast.DeleteStmt {
	t.Helper()
	del, ok := stmt.(*ast.DeleteStmt)
	if !ok {
		t.Fatalf("expected *ast.DeleteStmt, got %T", stmt)
	}
	return del
}
