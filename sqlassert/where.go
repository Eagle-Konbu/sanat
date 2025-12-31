package sqlassert

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
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

	found := findColumnInExpr(where, tableAliasOpt, column)
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

// findColumnInExpr recursively searches for a column reference in an expression tree.
func findColumnInExpr(expr ast.ExprNode, tableAlias string, column string) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		colName := e.Name.Name.L
		if !strings.EqualFold(colName, column) {
			return false
		}
		// If table alias is specified, check it
		if tableAlias != "" {
			actualAlias := e.Name.Table.L
			return strings.EqualFold(actualAlias, tableAlias)
		}
		return true

	case *ast.BinaryOperationExpr:
		return findColumnInExpr(e.L, tableAlias, column) || findColumnInExpr(e.R, tableAlias, column)

	case *ast.UnaryOperationExpr:
		return findColumnInExpr(e.V, tableAlias, column)

	case *ast.PatternInExpr:
		if findColumnInExpr(e.Expr, tableAlias, column) {
			return true
		}
		for _, arg := range e.List {
			if findColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.BetweenExpr:
		return findColumnInExpr(e.Expr, tableAlias, column) ||
			findColumnInExpr(e.Left, tableAlias, column) ||
			findColumnInExpr(e.Right, tableAlias, column)

	case *ast.IsNullExpr:
		return findColumnInExpr(e.Expr, tableAlias, column)

	case *ast.IsTruthExpr:
		return findColumnInExpr(e.Expr, tableAlias, column)

	case *ast.PatternLikeOrIlikeExpr:
		return findColumnInExpr(e.Expr, tableAlias, column) || findColumnInExpr(e.Pattern, tableAlias, column)

	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			if findColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.AggregateFuncExpr:
		for _, arg := range e.Args {
			if findColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.ParenthesesExpr:
		return findColumnInExpr(e.Expr, tableAlias, column)

	case *ast.SubqueryExpr:
		// For subqueries, we could recursively search but for now we skip
		// This is a reasonable limitation for WHERE column detection
		return false

	default:
		// For unknown expression types, return false
		return false
	}
}
