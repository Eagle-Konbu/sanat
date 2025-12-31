package sqlassert

import (
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/Eagle-Konbu/sql-assert/sqlassert/internal/helpers"
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

	found := helpers.FindTableInFrom(sel.From, table, expectedAlias)
	if !found {
		if expectedAlias != "" {
			t.Fatalf("table %q with alias %q not found in FROM clause", table, expectedAlias)
		} else {
			t.Fatalf("table %q not found in FROM clause", table)
		}
	}
}
