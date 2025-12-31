# sql-assert

[![Test](https://github.com/Eagle-Konbu/sql-assert/actions/workflows/test.yml/badge.svg)](https://github.com/Eagle-Konbu/sql-assert/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/Eagle-Konbu/sql-assert.svg)](https://pkg.go.dev/github.com/Eagle-Konbu/sql-assert)

A Go test helper library for validating SQL queries via AST inspection, without database execution.

## Overview

`sql-assert` provides testify/require-like APIs to validate SQL "contracts" using AST parsing. It enables fast, stable tests for SQL-generating code without needing database connections or fragile string comparisons.

### Why AST-based Contract Validation?

When testing SQL-generating code, you typically want to validate the **semantic intent** of the query rather than exact string equality. String comparison tests are brittle and break on harmless changes like:

- Formatting differences (whitespace, newlines)
- Keyword case changes (`SELECT` vs `select`)
- Identifier quoting (`` `users` `` vs `users`)
- Semantically equivalent refactors

AST-based validation solves this by:

1. **Parsing SQL into a structured tree** - Normalizes formatting and case
2. **Checking semantic properties** - "Does this query join table X?" not "Does it contain 'JOIN X'?"
3. **Being refactor-tolerant** - Validates contracts, not implementation details
4. **Running instantly** - No database setup, connections, or data fixtures
5. **Providing clear failures** - Precise error messages about what's missing

### Design Philosophy

**Contract Validation, Not Equivalence**

This library validates that SQL queries satisfy specific **contracts**:

- ✅ "The query must SELECT from table `users`"
- ✅ "The WHERE clause must reference column `status`"
- ✅ "The query must have a LIMIT clause"
- ✅ "The window function must PARTITION BY `category`"

**Not:**

- ❌ "The query must be exactly equal to this reference query"
- ❌ "The AST must match this exact structure"

**Presence-Based, Not Exhaustive**

Assertions check for the **presence** of required elements, not their absence or exact positioning:

- `RequireFromHasTable(t, sel, "users")` passes if `users` appears **anywhere** in the FROM/JOIN tree
- `RequireWhereContainsColumn(t, sel, "", "id")` passes if `id` appears **anywhere** in the WHERE clause
- `RequireWindowPartitionByHasColumn(t, win, "grp")` passes if `grp` appears in PARTITION BY

This tolerance makes tests robust to refactoring while still catching real contract violations.

## Features

- **No database execution** - Pure AST inspection, no connections required
- **MySQL 8.0 dialect** - Full support including window functions
- **Semantic validation** - Check query structure, not string formatting
- **Clear error messages** - Testify-style assertions with helpful failures
- **Tolerant matching** - Case-insensitive, handles backticks, formatting-agnostic
- **Comprehensive API** - Parse, type, FROM/JOIN, SELECT, WHERE, ORDER BY, LIMIT, window functions, expression matchers

## Installation

```bash
go get github.com/Eagle-Konbu/sql-assert
```

## Usage

### Basic Example: Generic SELECT

```go
package myapp_test

import (
    "testing"
    "github.com/Eagle-Konbu/sql-assert/sqlassert"
)

func TestGenerateUserQuery(t *testing.T) {
    sql := GenerateUserQuery() // Your SQL-generating function
    // Example: "SELECT col1, col2 FROM t WHERE col1 = ? ORDER BY col2 LIMIT 10"

    // Parse and validate it's a SELECT
    stmt := sqlassert.RequireParseOne(t, sql)
    sel := sqlassert.RequireSelect(t, stmt)

    // Validate contracts
    sqlassert.RequireFromHasTable(t, sel, "t")
    sqlassert.RequireHasWhere(t, sel)
    sqlassert.RequireWhereContainsColumn(t, sel, "", "col1")
    sqlassert.RequireHasOrderBy(t, sel)
    sqlassert.RequireHasLimit(t, sel)
}
```

### JOIN Queries

```go
func TestJoinQuery(t *testing.T) {
    sql := `SELECT t1.a, t2.b
        FROM t1 JOIN t2 ON t1.id = t2.t1_id
        WHERE t2.b IS NOT NULL`

    stmt := sqlassert.RequireParseOne(t, sql)
    sel := sqlassert.RequireSelect(t, stmt)

    // Validates both tables appear in FROM/JOIN tree
    sqlassert.RequireFromHasTable(t, sel, "t1")
    sqlassert.RequireFromHasTable(t, sel, "t2")

    // WHERE references t2.b
    sqlassert.RequireWhereContainsColumn(t, sel, "t2", "b")
}
```

### MySQL 8 Window Functions

```go
func TestWindowFunction(t *testing.T) {
    sql := `SELECT id,
        ROW_NUMBER() OVER (PARTITION BY grp ORDER BY created_at) AS rn
        FROM t`

    stmt := sqlassert.RequireParseOne(t, sql)
    sel := sqlassert.RequireSelect(t, stmt)

    // Validate basic structure
    sqlassert.RequireFromHasTable(t, sel, "t")
    sqlassert.RequireSelectHasAlias(t, sel, "rn")

    // Validate window function
    winExpr := sqlassert.RequireHasWindowFunc(t, sel, "ROW_NUMBER")
    sqlassert.RequireWindowPartitionByHasColumn(t, winExpr, "grp")
    sqlassert.RequireWindowOrderByHasColumn(t, winExpr, "created_at")
}
```

### SELECT Clause Validation

```go
func TestSelectColumns(t *testing.T) {
    sql := `SELECT id, name AS user_name, COUNT(*) AS total FROM users`

    stmt := sqlassert.RequireParseOne(t, sql)
    sel := sqlassert.RequireSelect(t, stmt)

    // Check for aliases
    sqlassert.RequireSelectHasAlias(t, sel, "user_name")
    sqlassert.RequireSelectHasAlias(t, sel, "total")

    // Check specific column with alias
    sqlassert.RequireSelectExpr(t, sel, sqlassert.Selector{
        Alias:  "user_name",
        Column: "name",
    })

    // Check aggregate function
    sqlassert.RequireSelectExpr(t, sel, sqlassert.Selector{
        Alias: "total",
        Func:  "COUNT",
    })
}
```

### Expression Matchers

For more complex validations, use composable expression matchers:

```go
func TestComplexWhere(t *testing.T) {
    sql := `SELECT id FROM users WHERE id = 123 AND name = 'test'`

    stmt := sqlassert.RequireParseOne(t, sql)
    sel := sqlassert.RequireSelect(t, stmt)
    where := sqlassert.RequireHasWhere(t, sel)

    // Match the AND expression structure
    andMatcher := sqlassert.Binary("AND",
        sqlassert.Binary("=", sqlassert.Col("", "id"), sqlassert.Any()),
        sqlassert.Binary("=", sqlassert.Col("", "name"), sqlassert.Any()),
    )

    sqlassert.RequireExprMatch(t, where, andMatcher)
}
```

Available matchers:
- `Col(tableAlias, column)` - Match column references
- `Func(name, args...)` - Match function calls
- `Binary(op, left, right)` - Match binary operations
- `Subquery(validator)` - Match subqueries
- `Any()` - Match any expression

## API Reference

### Parsing

```go
func ParseOne(sql string) (ast.StmtNode, error)
func RequireParseOne(t *testing.T, sql string) ast.StmtNode
```

### Statement Type Assertions

```go
func RequireSelect(t *testing.T, stmt ast.StmtNode) *ast.SelectStmt
func RequireInsert(t *testing.T, stmt ast.StmtNode) *ast.InsertStmt
func RequireUpdate(t *testing.T, stmt ast.StmtNode) *ast.UpdateStmt
func RequireDelete(t *testing.T, stmt ast.StmtNode) *ast.DeleteStmt
```

### FROM/JOIN Detection

```go
// Checks if table appears anywhere in FROM/JOIN tree
func RequireFromHasTable(t *testing.T, sel *ast.SelectStmt, table string, aliasOpt ...string)
```

### SELECT Clause

```go
func RequireSelectHasAlias(t *testing.T, sel *ast.SelectStmt, alias string)
func RequireSelectExpr(t *testing.T, sel *ast.SelectStmt, s Selector) ast.ExprNode

type Selector struct {
    Alias  string // The alias (AS name)
    Column string // Column name (for simple column references)
    Func   string // Function name (for function calls)
}
```

### WHERE/ORDER BY/LIMIT

```go
func RequireHasWhere(t *testing.T, node any) ast.ExprNode
func RequireWhereContainsColumn(t *testing.T, node any, tableAliasOpt string, column string)
func RequireHasOrderBy(t *testing.T, sel *ast.SelectStmt)
func RequireHasLimit(t *testing.T, sel *ast.SelectStmt)
```

### Window Functions

```go
func RequireHasWindowFunc(t *testing.T, sel *ast.SelectStmt, funcName string) ast.ExprNode
func RequireWindowPartitionByHasColumn(t *testing.T, winExpr ast.ExprNode, column string, tableAliasOpt ...string)
func RequireWindowOrderByHasColumn(t *testing.T, winExpr ast.ExprNode, column string, tableAliasOpt ...string)
```

### Expression Matchers

```go
type Matcher interface {
    Match(ast.ExprNode) bool
    Describe() string
}

func RequireExprMatch(t *testing.T, expr ast.ExprNode, m Matcher)
func Col(tableAlias, column string) Matcher
func Func(name string, args ...Matcher) Matcher
func Binary(op string, left, right Matcher) Matcher
func Subquery(inner func(*ast.SelectStmt) error) Matcher
func Any() Matcher
```

## MySQL Dialect Support

### LIMIT Formats

All MySQL LIMIT formats are supported:

```sql
LIMIT 10
LIMIT 5, 10
LIMIT 10 OFFSET 5
```

### Identifier Normalization

- **Case-insensitive** - `SELECT`, `select`, `SeLeCt` all match
- **Backtick-tolerant** - `` `users` `` and `users` are equivalent
- **Whitespace-agnostic** - Formatting differences are ignored

### Window Functions

Supported window functions include:
- `ROW_NUMBER()`
- `RANK()`
- `DENSE_RANK()`
- `LAG()`
- `LEAD()`
- And all other MySQL 8.0 window functions

Aggregate functions with `OVER` clause are also supported.

## Non-Goals

This library intentionally does **not**:

- ❌ Execute SQL against a database
- ❌ Validate SQL correctness (syntax errors are caught, but semantic validity is not guaranteed)
- ❌ Check query equivalence (two different queries that produce the same result)
- ❌ Optimize or rewrite queries
- ❌ Support dialects other than MySQL 8.0 (may work with MariaDB, but not guaranteed)
- ❌ Guarantee exhaustive structural matching (uses presence-based validation)

## Implementation

Built on [TiDB SQL Parser](https://github.com/pingcap/parser), a production-grade MySQL-compatible SQL parser extracted from TiDB. The parser is used in read-only mode for AST inspection - no execution or modification occurs.

## Testing

Run the test suite:

```bash
go test ./...
```

Run with coverage:

```bash
go test -coverprofile=coverage.txt ./...
go tool cover -html=coverage.txt
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Related Projects

- [TiDB Parser](https://github.com/pingcap/parser) - The underlying SQL parser
- [testify](https://github.com/stretchr/testify) - Inspiration for the assertion API style
- [sqlparser](https://github.com/xwb1989/sqlparser) - Alternative MySQL parser

## Support

- **Issues**: [GitHub Issues](https://github.com/Eagle-Konbu/sql-assert/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Eagle-Konbu/sql-assert/discussions)
