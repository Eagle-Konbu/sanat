package sqlassert_test

import (
	"testing"

	"github.com/Eagle-Konbu/sql-assert/sqlassert"
)

// Test case 1: Generic SELECT with WHERE, ORDER BY, and LIMIT
func TestGenericSelect(t *testing.T) {
	sql := `SELECT col1, col2 FROM t WHERE col1 = ? ORDER BY col2 LIMIT 10`

	// Parse and validate it's a SELECT
	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// FROM has table t
	sqlassert.RequireFromHasTable(t, sel, "t")

	// WHERE exists and references col1
	sqlassert.RequireHasWhere(t, sel)
	sqlassert.RequireWhereContainsColumn(t, sel, "", "col1")

	// ORDER BY exists
	sqlassert.RequireHasOrderBy(t, sel)

	// LIMIT exists
	sqlassert.RequireHasLimit(t, sel)
}

// Test case 2: JOIN SELECT with WHERE
func TestJoinSelect(t *testing.T) {
	sql := `SELECT t1.a, t2.b
		FROM t1 JOIN t2 ON t1.id = t2.t1_id
		WHERE t2.b IS NOT NULL`

	// Parse and validate it's a SELECT
	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// FROM has both t1 and t2
	sqlassert.RequireFromHasTable(t, sel, "t1")
	sqlassert.RequireFromHasTable(t, sel, "t2")

	// WHERE references t2.b
	sqlassert.RequireHasWhere(t, sel)
	sqlassert.RequireWhereContainsColumn(t, sel, "t2", "b")
}

// Test case 3: MySQL 8 window function
func TestWindowFunction(t *testing.T) {
	sql := `SELECT id,
		ROW_NUMBER() OVER (PARTITION BY grp ORDER BY created_at) AS rn
		FROM t`

	// Parse and validate it's a SELECT
	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// FROM has table t
	sqlassert.RequireFromHasTable(t, sel, "t")

	// SELECT has alias rn
	sqlassert.RequireSelectHasAlias(t, sel, "rn")

	// Window function ROW_NUMBER exists
	winExpr := sqlassert.RequireHasWindowFunc(t, sel, "ROW_NUMBER")

	// PARTITION BY includes grp
	sqlassert.RequireWindowPartitionByHasColumn(t, winExpr, "grp")

	// ORDER BY includes created_at
	sqlassert.RequireWindowOrderByHasColumn(t, winExpr, "created_at")
}

// Additional test: SELECT with specific columns
func TestSelectColumns(t *testing.T) {
	sql := `SELECT id, name AS user_name, COUNT(*) AS total FROM users`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Check that SELECT has alias user_name
	sqlassert.RequireSelectHasAlias(t, sel, "user_name")

	// Check that SELECT has alias total
	sqlassert.RequireSelectHasAlias(t, sel, "total")

	// Check for specific column with alias
	sqlassert.RequireSelectExpr(t, sel, sqlassert.Selector{
		Alias:  "user_name",
		Column: "name",
	})

	// Check for aggregate function
	sqlassert.RequireSelectExpr(t, sel, sqlassert.Selector{
		Alias: "total",
		Func:  "COUNT",
	})
}

// Additional test: Expression matchers
func TestExpressionMatchers(t *testing.T) {
	sql := `SELECT id, name FROM users WHERE id = 123 AND name = 'test'`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	where := sqlassert.RequireHasWhere(t, sel)

	// Match the AND expression
	andMatcher := sqlassert.Binary("AND",
		sqlassert.Binary("=", sqlassert.Col("", "id"), sqlassert.Any()),
		sqlassert.Binary("=", sqlassert.Col("", "name"), sqlassert.Any()),
	)

	sqlassert.RequireExprMatch(t, where, andMatcher)
}

// Additional test: Function call in SELECT
func TestFunctionInSelect(t *testing.T) {
	sql := `SELECT UPPER(name) AS upper_name FROM users`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Find the UPPER function expression
	expr := sqlassert.RequireSelectExpr(t, sel, sqlassert.Selector{
		Alias: "upper_name",
		Func:  "UPPER",
	})

	// Verify it matches the Func matcher
	sqlassert.RequireExprMatch(t, expr, sqlassert.Func("UPPER", sqlassert.Col("", "name")))
}

// Additional test: INSERT statement
func TestInsertStatement(t *testing.T) {
	sql := `INSERT INTO users (name, email) VALUES ('John', 'john@example.com')`

	stmt := sqlassert.RequireParseOne(t, sql)
	sqlassert.RequireInsert(t, stmt)
}

// Additional test: UPDATE statement
func TestUpdateStatement(t *testing.T) {
	sql := `UPDATE users SET name = 'Jane' WHERE id = 123`

	stmt := sqlassert.RequireParseOne(t, sql)
	upd := sqlassert.RequireUpdate(t, stmt)

	// WHERE exists and references id
	sqlassert.RequireHasWhere(t, upd)
	sqlassert.RequireWhereContainsColumn(t, upd, "", "id")
}

// Additional test: DELETE statement
func TestDeleteStatement(t *testing.T) {
	sql := `DELETE FROM users WHERE created_at < NOW()`

	stmt := sqlassert.RequireParseOne(t, sql)
	del := sqlassert.RequireDelete(t, stmt)

	// WHERE exists and references created_at
	sqlassert.RequireHasWhere(t, del)
	sqlassert.RequireWhereContainsColumn(t, del, "", "created_at")
}

// Additional test: Multiple window functions
func TestMultipleWindowFunctions(t *testing.T) {
	sql := `SELECT
		id,
		ROW_NUMBER() OVER (PARTITION BY category ORDER BY created_at) AS rn,
		RANK() OVER (PARTITION BY category ORDER BY score DESC) AS rank_score
		FROM products`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Check ROW_NUMBER window function
	rnExpr := sqlassert.RequireHasWindowFunc(t, sel, "ROW_NUMBER")
	sqlassert.RequireWindowPartitionByHasColumn(t, rnExpr, "category")
	sqlassert.RequireWindowOrderByHasColumn(t, rnExpr, "created_at")

	// Check RANK window function
	rankExpr := sqlassert.RequireHasWindowFunc(t, sel, "RANK")
	sqlassert.RequireWindowPartitionByHasColumn(t, rankExpr, "category")
	sqlassert.RequireWindowOrderByHasColumn(t, rankExpr, "score")
}

// Additional test: Complex JOIN with aliases
func TestComplexJoinWithAliases(t *testing.T) {
	sql := `SELECT u.id, p.title
		FROM users u
		INNER JOIN posts p ON u.id = p.user_id
		LEFT JOIN comments c ON p.id = c.post_id
		WHERE u.active = 1`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Check all tables with aliases
	sqlassert.RequireFromHasTable(t, sel, "users", "u")
	sqlassert.RequireFromHasTable(t, sel, "posts", "p")
	sqlassert.RequireFromHasTable(t, sel, "comments", "c")

	// WHERE references u.active
	sqlassert.RequireWhereContainsColumn(t, sel, "u", "active")
}

// Additional test: LIMIT with different formats
func TestLimitFormats(t *testing.T) {
	testCases := []string{
		`SELECT * FROM users LIMIT 10`,
		`SELECT * FROM users LIMIT 5, 10`,
		`SELECT * FROM users LIMIT 10 OFFSET 5`,
	}

	for _, sql := range testCases {
		stmt := sqlassert.RequireParseOne(t, sql)
		sel := sqlassert.RequireSelect(t, stmt)
		sqlassert.RequireHasLimit(t, sel)
	}
}

// Additional test: Case insensitive matching
func TestCaseInsensitiveMatching(t *testing.T) {
	sql := `SELECT ID, NAME FROM USERS WHERE STATUS = 'active'`

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Should match regardless of case
	sqlassert.RequireFromHasTable(t, sel, "users")
	sqlassert.RequireWhereContainsColumn(t, sel, "", "status")
}

// Additional test: Backtick quoted identifiers
func TestBacktickIdentifiers(t *testing.T) {
	sql := "SELECT `user_id`, `full_name` FROM `users` WHERE `status` = 'active'"

	stmt := sqlassert.RequireParseOne(t, sql)
	sel := sqlassert.RequireSelect(t, stmt)

	// Should match with or without backticks
	sqlassert.RequireFromHasTable(t, sel, "users")
	sqlassert.RequireWhereContainsColumn(t, sel, "", "status")
}
