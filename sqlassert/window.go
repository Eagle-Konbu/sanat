package sqlassert

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/Eagle-Konbu/sql-assert/sqlassert/internal/helpers"
)

// RequireHasWindowFunc asserts that the SELECT statement contains a window function
// with the given name and returns the window function expression.
func RequireHasWindowFunc(t *testing.T, sel *ast.SelectStmt, funcName string) ast.ExprNode {
	t.Helper()

	if sel.Fields == nil {
		t.Fatalf("SELECT has no fields")
	}

	for _, field := range sel.Fields.Fields {
		if winExpr := helpers.FindWindowFunc(field.Expr, funcName); winExpr != nil {
			return winExpr
		}
	}

	t.Fatalf("SELECT does not contain window function %q", funcName)
	return nil
}

// RequireWindowPartitionByHasColumn asserts that the window function expression
// has a PARTITION BY clause that includes the given column.
func RequireWindowPartitionByHasColumn(t *testing.T, winExpr ast.ExprNode, column string, tableAliasOpt ...string) {
	t.Helper()

	win, ok := winExpr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("expression is not a WindowFuncExpr, got %T", winExpr)
	}

	if win.Spec.PartitionBy == nil || len(win.Spec.PartitionBy.Items) == 0 {
		t.Fatalf("window function has no PARTITION BY clause")
	}

	var tableAlias string
	if len(tableAliasOpt) > 0 {
		tableAlias = tableAliasOpt[0]
	}

	for _, item := range win.Spec.PartitionBy.Items {
		if helpers.HasColumn(item.Expr, tableAlias, column) {
			return
		}
	}

	if tableAlias != "" {
		t.Fatalf("PARTITION BY does not contain column %s.%s", tableAlias, column)
	} else {
		t.Fatalf("PARTITION BY does not contain column %s", column)
	}
}

// RequireWindowOrderByHasColumn asserts that the window function expression
// has an ORDER BY clause that includes the given column.
func RequireWindowOrderByHasColumn(t *testing.T, winExpr ast.ExprNode, column string, tableAliasOpt ...string) {
	t.Helper()

	win, ok := winExpr.(*ast.WindowFuncExpr)
	if !ok {
		t.Fatalf("expression is not a WindowFuncExpr, got %T", winExpr)
	}

	if win.Spec.OrderBy == nil || len(win.Spec.OrderBy.Items) == 0 {
		t.Fatalf("window function has no ORDER BY clause")
	}

	var tableAlias string
	if len(tableAliasOpt) > 0 {
		tableAlias = tableAliasOpt[0]
	}

	for _, item := range win.Spec.OrderBy.Items {
		if helpers.HasColumn(item.Expr, tableAlias, column) {
			return
		}
	}

	if tableAlias != "" {
		t.Fatalf("ORDER BY does not contain column %s.%s", tableAlias, column)
	} else {
		t.Fatalf("ORDER BY does not contain column %s", column)
	}
}
