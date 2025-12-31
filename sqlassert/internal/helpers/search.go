package helpers

import (
	"strings"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// FindColumnInExpr recursively searches for a column reference in an expression tree.
func FindColumnInExpr(expr ast.ExprNode, tableAlias string, column string) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		colName := e.Name.Name.L
		if !strings.EqualFold(colName, column) {
			return false
		}
		if tableAlias != "" {
			actualAlias := e.Name.Table.L
			return strings.EqualFold(actualAlias, tableAlias)
		}
		return true

	case *ast.BinaryOperationExpr:
		return FindColumnInExpr(e.L, tableAlias, column) || FindColumnInExpr(e.R, tableAlias, column)

	case *ast.UnaryOperationExpr:
		return FindColumnInExpr(e.V, tableAlias, column)

	case *ast.PatternInExpr:
		if FindColumnInExpr(e.Expr, tableAlias, column) {
			return true
		}
		for _, arg := range e.List {
			if FindColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.BetweenExpr:
		return FindColumnInExpr(e.Expr, tableAlias, column) ||
			FindColumnInExpr(e.Left, tableAlias, column) ||
			FindColumnInExpr(e.Right, tableAlias, column)

	case *ast.IsNullExpr:
		return FindColumnInExpr(e.Expr, tableAlias, column)

	case *ast.IsTruthExpr:
		return FindColumnInExpr(e.Expr, tableAlias, column)

	case *ast.PatternLikeOrIlikeExpr:
		return FindColumnInExpr(e.Expr, tableAlias, column) || FindColumnInExpr(e.Pattern, tableAlias, column)

	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			if FindColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.AggregateFuncExpr:
		for _, arg := range e.Args {
			if FindColumnInExpr(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.ParenthesesExpr:
		return FindColumnInExpr(e.Expr, tableAlias, column)

	case *ast.SubqueryExpr:
		return false

	default:
		return false
	}
}

// HasColumn checks if an expression references the given column.
func HasColumn(expr ast.ExprNode, tableAlias string, column string) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		colName := e.Name.Name.L
		if !strings.EqualFold(colName, column) {
			return false
		}
		if tableAlias != "" {
			actualAlias := e.Name.Table.L
			return strings.EqualFold(actualAlias, tableAlias)
		}
		return true

	case *ast.BinaryOperationExpr:
		return HasColumn(e.L, tableAlias, column) || HasColumn(e.R, tableAlias, column)

	case *ast.UnaryOperationExpr:
		return HasColumn(e.V, tableAlias, column)

	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			if HasColumn(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.AggregateFuncExpr:
		for _, arg := range e.Args {
			if HasColumn(arg, tableAlias, column) {
				return true
			}
		}
		return false

	case *ast.ParenthesesExpr:
		return HasColumn(e.Expr, tableAlias, column)

	default:
		return false
	}
}

// FindTableInFrom recursively searches for a table in the FROM clause and JOIN tree.
func FindTableInFrom(from *ast.TableRefsClause, table string, alias string) bool {
	if from == nil || from.TableRefs == nil {
		return false
	}
	return FindTableInResultSetNode(from.TableRefs, table, alias)
}

// FindTableInResultSetNode recursively searches a ResultSetNode for a table.
func FindTableInResultSetNode(node ast.ResultSetNode, table string, alias string) bool {
	switch n := node.(type) {
	case *ast.TableSource:
		return CheckTableSource(n, table, alias)
	case *ast.Join:
		if FindTableInResultSetNode(n.Left, table, alias) {
			return true
		}
		if FindTableInResultSetNode(n.Right, table, alias) {
			return true
		}
	}
	return false
}

// CheckTableSource checks if a TableSource matches the given table and optional alias.
func CheckTableSource(ts *ast.TableSource, table string, alias string) bool {
	tableName, ok := ts.Source.(*ast.TableName)
	if !ok {
		return false
	}

	actualTable := tableName.Name.L
	if !strings.EqualFold(actualTable, table) {
		return false
	}

	if alias != "" {
		actualAlias := ts.AsName.L
		if actualAlias == "" {
			actualAlias = actualTable
		}
		return strings.EqualFold(actualAlias, alias)
	}

	return true
}

// FindWindowFunc recursively searches for a window function in an expression tree.
func FindWindowFunc(expr ast.ExprNode, funcName string) ast.ExprNode {
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
		return nil

	case *ast.FuncCallExpr:
		for _, arg := range e.Args {
			if result := FindWindowFunc(arg, funcName); result != nil {
				return result
			}
		}
		return nil

	case *ast.BinaryOperationExpr:
		if result := FindWindowFunc(e.L, funcName); result != nil {
			return result
		}
		return FindWindowFunc(e.R, funcName)

	case *ast.UnaryOperationExpr:
		return FindWindowFunc(e.V, funcName)

	case *ast.ParenthesesExpr:
		return FindWindowFunc(e.Expr, funcName)

	default:
		return nil
	}
}
