package helpers

import (
	"github.com/pingcap/tidb/pkg/parser/ast"
)

// ExtractColumnName extracts the column name from an expression if it's a ColumnNameExpr.
func ExtractColumnName(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		return e.Name.Name.L
	}
	return ""
}

// ExtractFunctionName extracts the function name from an expression if it's a FuncCallExpr or AggregateFuncExpr.
func ExtractFunctionName(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.FuncCallExpr:
		return e.FnName.L
	case *ast.AggregateFuncExpr:
		return e.F
	case *ast.WindowFuncExpr:
		return e.Name
	}
	return ""
}

// ExtractTableAlias extracts the table alias from a ColumnNameExpr.
func ExtractTableAlias(expr ast.ExprNode) string {
	if expr == nil {
		return ""
	}

	switch e := expr.(type) {
	case *ast.ColumnNameExpr:
		if e.Name.Table.L != "" {
			return e.Name.Table.L
		}
	}
	return ""
}
