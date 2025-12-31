package sqlassert

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/Eagle-Konbu/sql-assert/sqlassert/internal/helpers"
)

// Selector describes a SELECT list expression to match.
type Selector struct {
	Alias  string // The alias (AS name)
	Column string // Column name (for simple column references)
	Func   string // Function name (for function calls)
}

// RequireSelectHasAlias asserts that the SELECT list contains an expression with the given alias.
func RequireSelectHasAlias(t *testing.T, sel *ast.SelectStmt, alias string) {
	t.Helper()

	if sel.Fields == nil {
		t.Fatalf("SELECT has no fields")
	}

	for _, field := range sel.Fields.Fields {
		if field.AsName.L != "" && strings.EqualFold(field.AsName.L, alias) {
			return
		}
		// If no explicit alias, check if it's a column reference
		if field.AsName.L == "" {
			if colName := helpers.ExtractColumnName(field.Expr); colName != "" {
				if strings.EqualFold(colName, alias) {
					return
				}
			}
		}
	}

	t.Fatalf("SELECT does not have alias %q", alias)
}

// RequireSelectExpr asserts that the SELECT list contains an expression matching the Selector
// and returns the expression node.
func RequireSelectExpr(t *testing.T, sel *ast.SelectStmt, s Selector) ast.ExprNode {
	t.Helper()

	if sel.Fields == nil {
		t.Fatalf("SELECT has no fields")
	}

	for _, field := range sel.Fields.Fields {
		if matchesSelector(field, s) {
			return field.Expr
		}
	}

	t.Fatalf("SELECT does not have expression matching %+v", s)
	return nil
}

// matchesSelector checks if a SelectField matches the given Selector.
func matchesSelector(field *ast.SelectField, s Selector) bool {
	// Check alias if specified
	if s.Alias != "" {
		fieldAlias := field.AsName.L
		if fieldAlias == "" {
			// No explicit alias, try to extract from expression
			if colName := helpers.ExtractColumnName(field.Expr); colName != "" {
				fieldAlias = strings.ToLower(colName)
			}
		}
		if !strings.EqualFold(fieldAlias, s.Alias) {
			return false
		}
	}

	// Check column if specified
	if s.Column != "" {
		colName := helpers.ExtractColumnName(field.Expr)
		if !strings.EqualFold(colName, s.Column) {
			return false
		}
	}

	// Check function if specified
	if s.Func != "" {
		funcName := helpers.ExtractFunctionName(field.Expr)
		if !strings.EqualFold(funcName, s.Func) {
			return false
		}
	}

	return true
}
