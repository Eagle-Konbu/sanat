package sqlassert

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/Eagle-Konbu/sql-assert/sqlassert/internal/helpers"
)

// RequireHasWhere asserts that the statement has a WHERE clause and returns it.
// Supports SelectStmt, UpdateStmt, and DeleteStmt.
func RequireHasWhere(t *testing.T, node any) ast.ExprNode {
	t.Helper()

	var where ast.ExprNode

	switch n := node.(type) {
	case *ast.SelectStmt:
		where = n.Where
	case *ast.UpdateStmt:
		where = n.Where
	case *ast.DeleteStmt:
		where = n.Where
	default:
		t.Fatalf("RequireHasWhere: unsupported node type %T", node)
	}

	if where == nil {
		t.Fatalf("statement has no WHERE clause")
	}

	return where
}

// RequireWhereContainsColumn asserts that the WHERE clause references the given column.
// Optionally checks that the column belongs to a specific table alias.
func RequireWhereContainsColumn(t *testing.T, node any, tableAliasOpt string, column string) {
	t.Helper()

	where := RequireHasWhere(t, node)

	found := helpers.FindColumnInExpr(where, tableAliasOpt, column)
	if !found {
		if tableAliasOpt != "" {
			t.Fatalf("WHERE clause does not reference column %s.%s", tableAliasOpt, column)
		} else {
			t.Fatalf("WHERE clause does not reference column %s", column)
		}
	}
}

// RequireHasOrderBy asserts that the SELECT statement has an ORDER BY clause.
func RequireHasOrderBy(t *testing.T, sel *ast.SelectStmt) {
	t.Helper()

	if sel.OrderBy == nil || len(sel.OrderBy.Items) == 0 {
		t.Fatalf("SELECT has no ORDER BY clause")
	}
}

// RequireHasLimit asserts that the SELECT statement has a LIMIT clause.
func RequireHasLimit(t *testing.T, sel *ast.SelectStmt) {
	t.Helper()

	if sel.Limit == nil {
		t.Fatalf("SELECT has no LIMIT clause")
	}
}
