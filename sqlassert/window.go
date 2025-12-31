package sqlassert

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// RequireHasWindowFunc asserts that the SELECT statement contains a window function
// with the given name and returns the window function expression.
func RequireHasWindowFunc(t *testing.T, sel *ast.SelectStmt, funcName string) ast.ExprNode {
	t.Helper()

	if sel.Fields == nil {
		t.Fatalf("SELECT has no fields")
	}

	for _, field := range sel.Fields.Fields {
		if winExpr := findWindowFunc(field.Expr, funcName); winExpr != nil {
			return winExpr
		}
	}

	t.Fatalf("SELECT does not contain window function %q", funcName)
	return nil
}

// findWindowFunc recursively searches for a window function in an expression tree.
func findWindowFunc(expr ast.ExprNode, funcName string) ast.ExprNode {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.WindowFuncExpr:
		if strings.EqualFold(e.Name, funcName) {
			return e
		}
		return nil

	case *ast.AggregateFuncExpr:
		// Aggregate functions can also be used as window functions with OVER clause
		// However, in TiDB parser, window functions are represented as WindowFuncExpr
		return nil

	case *ast.FuncCallExpr:
		// Check arguments recursively
		for _, arg := range e.Args {
			if result := findWindowFunc(arg, funcName); result != nil {
				return result
			}
		}
		return nil

	case *ast.BinaryOperationExpr:
		if result := findWindowFunc(e.L, funcName); result != nil {
			return result
		}
		return findWindowFunc(e.R, funcName)

	case *ast.UnaryOperationExpr:
		return findWindowFunc(e.V, funcName)

	case *ast.ParenthesesExpr:
		return findWindowFunc(e.Expr, funcName)

	default:
		return nil
	}
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
		if hasColumn(item.Expr, tableAlias, column) {
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
		if hasColumn(item.Expr, tableAlias, column) {
			return
		}
	}

	if tableAlias != "" {
		t.Fatalf("ORDER BY does not contain column %s.%s", tableAlias, column)
	} else {
		t.Fatalf("ORDER BY does not contain column %s", column)
	}
}

// hasColumn checks if an expression references the given column.
func hasColumn(expr ast.ExprNode, tableAlias string, column string) bool {
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
		return hasColumn(e.L, tableAlias, column) || hasColumn(e.R, tableAlias, column)

	case *ast.UnaryOperationExpr:
		return hasColumn(e.V, tableAlias, column)

	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			if hasColumn(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.AggregateFuncExpr:
		for _, arg := range e.Args {
			if hasColumn(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.ParenthesesExpr:
		return hasColumn(e.Expr, tableAlias, column)

	default:
		return false
	}
}
