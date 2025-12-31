// Package sqlassert provides test helpers for validating SQL queries via AST inspection,
// without database execution.
//
// This package enables fast, stable tests for SQL-generating code by parsing SQL into an
// Abstract Syntax Tree (AST) and validating semantic properties rather than performing
// fragile string comparisons.
//
// # Design Philosophy
//
// sqlassert validates that SQL queries satisfy specific contracts:
//   - "The query must SELECT from table users"
//   - "The WHERE clause must reference column status"
//   - "The query must have a LIMIT clause"
//   - "The window function must PARTITION BY category"
//
// It does NOT validate:
//   - Exact query equivalence
//   - Complete AST structural equality
//   - SQL execution semantics
//
// # Basic Usage
//
//	func TestGenerateUserQuery(t *testing.T) {
//	    sql := GenerateUserQuery()
//	    // Example: "SELECT col1, col2 FROM t WHERE col1 = ? ORDER BY col2 LIMIT 10"
//
//	    // Parse and validate it's a SELECT
//	    stmt := sqlassert.RequireParseOne(t, sql)
//	    sel := sqlassert.RequireSelect(t, stmt)
//
//	    // Validate contracts
//	    sqlassert.RequireFromHasTable(t, sel, "t")
//	    sqlassert.RequireHasWhere(t, sel)
//	    sqlassert.RequireWhereContainsColumn(t, sel, "", "col1")
//	    sqlassert.RequireHasOrderBy(t, sel)
//	    sqlassert.RequireHasLimit(t, sel)
//	}
//
// # Features
//
//   - No database execution required
//   - MySQL 8.0 dialect support including window functions
//   - Case-insensitive identifier matching
//   - Backtick-tolerant identifier comparison
//   - Formatting-agnostic validation
//   - Composable expression matchers
//   - Clear, testify-style error messages
//
// # Supported SQL Statements
//
//   - SELECT (including JOINs, subqueries, window functions)
//   - INSERT
//   - UPDATE
//   - DELETE
//
// # Implementation
//
// Built on the TiDB SQL Parser (github.com/pingcap/tidb/pkg/parser), a production-grade
// MySQL-compatible parser. The parser is used in read-only mode for AST inspection.
package sqlassert
