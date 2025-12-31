package sqlassert

import (
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
)

// RequireFromHasTable asserts that the SELECT statement references the given table
// in its FROM clause or JOIN tree. Optionally checks for a specific alias.
func RequireFromHasTable(t *testing.T, sel *ast.SelectStmt, table string, aliasOpt ...string) {
	t.Helper()

	if sel.From == nil {
		t.Fatalf("SELECT has no FROM clause")
	}

	var expectedAlias string
	if len(aliasOpt) > 0 {
		expectedAlias = aliasOpt[0]
	}

	found := findTableInFrom(sel.From, table, expectedAlias)
	if !found {
		if expectedAlias != "" {
			t.Fatalf("table %q with alias %q not found in FROM clause", table, expectedAlias)
		} else {
			t.Fatalf("table %q not found in FROM clause", table)
		}
	}
}

// findTableInFrom recursively searches for a table in the FROM clause and JOIN tree.
func findTableInFrom(from *ast.TableRefsClause, table string, alias string) bool {
	if from == nil || from.TableRefs == nil {
		return false
	}
	return findTableInResultSetNode(from.TableRefs, table, alias)
}

// findTableInResultSetNode recursively searches a ResultSetNode for a table.
func findTableInResultSetNode(node ast.ResultSetNode, table string, alias string) bool {
	switch n := node.(type) {
	case *ast.TableSource:
		return checkTableSource(n, table, alias)
	case *ast.Join:
		// Check both left and right sides of the join
		if findTableInResultSetNode(n.Left, table, alias) {
			return true
		}
		if findTableInResultSetNode(n.Right, table, alias) {
			return true
		}
	}
	return false
}

// checkTableSource checks if a TableSource matches the given table and optional alias.
func checkTableSource(ts *ast.TableSource, table string, alias string) bool {
	// Get the actual table name
	tableName, ok := ts.Source.(*ast.TableName)
	if !ok {
		return false
	}

	actualTable := tableName.Name.L
	if !strings.EqualFold(actualTable, table) {
		return false
	}

	// If alias is specified, check it
	if alias != "" {
		actualAlias := ts.AsName.L
		if actualAlias == "" {
			actualAlias = actualTable
		}
		return strings.EqualFold(actualAlias, alias)
	}

	return true
}

// normalizeIdentifier normalizes an identifier for case-insensitive comparison.
// Removes backticks and converts to lowercase.
func normalizeIdentifier(s string) string {
	s = strings.Trim(s, "`")
	return strings.ToLower(s)
}
