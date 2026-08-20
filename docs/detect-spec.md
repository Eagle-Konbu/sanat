# SQL Detection Specification (MightBeSQL)

Heuristically determines whether a string extracted from a raw string literal is SQL. This is a lightweight pre-filter that runs before the in-house SQL parser (see [parser-spec.md](parser-spec.md)).

## Scope

Detection operates on the **content** of Go raw string literals (`ast.BasicLit` with backtick delimiters). Double-quoted strings are never candidates.

Given the following Go source:

```go
func example(db *sql.DB) {
	db.Exec(`select id from users where id = ?`, 1)
	db.Exec("select name from users", 1)
	msg := "hello world"
	_ = msg
	db.Query(`select * from orders`)
}
```

The Go AST contains these `*ast.BasicLit` (kind = `token.STRING`) nodes:

```
*ast.BasicLit {Value: "`select id from users where id = ?`"}   → raw string → candidate
*ast.BasicLit {Value: "\"select name from users\""}             → double-quoted → skipped
*ast.BasicLit {Value: "\"hello world\""}                        → double-quoted → skipped
*ast.BasicLit {Value: "`select * from orders`"}                 → raw string → candidate
```

Only the two raw string literals are extracted. Their inner content (with backticks stripped) is passed to `MightBeSQL`.

## Detection Flow

```mermaid
flowchart TD
    A[Input string] --> B[Trim whitespace]
    B --> C{Empty string?}
    C -- Yes --> D[Not SQL]
    C -- No --> E{Contains fmt verb?}
    E -- Yes --> D
    E -- No --> F{Starts with SQL keyword?}
    F -- Yes --> G[SQL]
    F -- No --> D
```

## Detection Rules

Evaluated in the following order. The first matching condition determines the result.

1. Trim leading and trailing whitespace
2. Empty string → **Not SQL**
3. Contains `fmt` format verb → **Not SQL**
4. Starts with a SQL keyword → **SQL**
5. Otherwise → **Not SQL**

## fmt Format Verb Detection

Strings matching the following regex pattern are considered Go `fmt` templates and are not treated as SQL.

```
%[+\-# 0]*[*]?[0-9]*[.*]?[0-9]*[vTtbcdoOqxXUeEfFgGsp]
```

This excludes strings containing format verbs such as `%s`, `%d`, `%v`, `%02d`, `%-10s`, etc.

### Example: fmt template excluded

**Go source:**

```go
tpl := `SELECT %s FROM %s WHERE id = %d`
```

**AST node:**

```
*ast.BasicLit {
    Kind:  token.STRING
    Value: "`SELECT %s FROM %s WHERE id = %d`"
}
```

Inner content after backtick stripping: `SELECT %s FROM %s WHERE id = %d`

`MightBeSQL` returns `false` because `%s` matches the fmt verb regex. Although the string starts with `SELECT`, it is a `fmt.Sprintf` template and should not be formatted as SQL.

## SQL Keyword Detection

The following regex pattern is used to detect a leading keyword (case-insensitive).

```
(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|CREATE|ALTER|DROP|TRUNCATE|START|BEGIN|COMMIT|ROLLBACK|SAVEPOINT|RELEASE|SET|SHOW|DESCRIBE|EXPLAIN|USE)\b
```

### Target Keywords

| Keyword | Description |
|---------|-------------|
| `SELECT` | Data retrieval |
| `INSERT` | Data insertion |
| `UPDATE` | Data modification |
| `DELETE` | Data deletion |
| `CREATE` | DDL: create table/index |
| `ALTER` | DDL: alter table |
| `DROP` | DDL: drop table/index |
| `TRUNCATE` | DDL: truncate table |
| `START` | Transaction: start transaction |
| `BEGIN` | Transaction: begin |
| `COMMIT` | Transaction: commit |
| `ROLLBACK` | Transaction: rollback |
| `SAVEPOINT` | Transaction: create savepoint |
| `RELEASE` | Transaction: release savepoint |
| `SET` | Session: set variable |
| `SHOW` | Admin: show tables/columns/etc. |
| `DESCRIBE` | Admin: describe table |
| `EXPLAIN` | Admin: explain statement |
| `USE` | Admin: use database |

Every keyword listed here has a corresponding statement parser in `internal/sqlfmt/parser` (see [parser-spec.md](parser-spec.md)). Statement kinds the parser does not yet support — stored program syntax (`CALL`, `PREPARE`, `EXECUTE`, `DEALLOCATE PREPARE`) and the `DESC` alias for `DESCRIBE` — are deliberately excluded: detecting a statement the formatter cannot format would just send it to `FormatSQLWithOptions`, which would fail and leave the literal untouched, so there is no benefit to detecting it.

Leading whitespace is allowed, but a word boundary (`\b`) is required after the keyword.

## Examples with Go AST Context

### Example 1: Simple SELECT (detected as SQL)

**Go source:**

```go
db.Exec(`select id from users where id = ?`, 1)
```

**AST (simplified):**

```
*ast.CallExpr {
    Fun: *ast.SelectorExpr {X: db, Sel: Exec}
    Args: [
        *ast.BasicLit {Kind: STRING, Value: "`select id from users where id = ?`"},
        *ast.BasicLit {Kind: INT, Value: "1"},
    ]
}
```

Inner content: `select id from users where id = ?`

- Trim → `select id from users where id = ?`
- Empty? → No
- fmt verb? → No
- SQL keyword prefix? → `select` matches `(?i)SELECT\b` → **SQL**

### Example 2: Double-quoted string (never reaches MightBeSQL)

**Go source:**

```go
db.Exec("select name from users", 1)
```

**AST:**

```
*ast.BasicLit {Kind: STRING, Value: "\"select name from users\""}
```

The scanner checks `isRawStringLit` first. Since the value starts with `"` (not `` ` ``), this literal is **skipped entirely** and never passed to `MightBeSQL`.

### Example 3: fmt template (detected as Not SQL)

**Go source:**

```go
query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = %d`, cols, table, id)
```

**AST:**

```
*ast.CallExpr {
    Fun: *ast.SelectorExpr {X: fmt, Sel: Sprintf}
    Args: [
        *ast.BasicLit {Kind: STRING, Value: "`SELECT %s FROM %s WHERE id = %d`"},
        ...
    ]
}
```

Inner content: `SELECT %s FROM %s WHERE id = %d`

- Trim → `SELECT %s FROM %s WHERE id = %d`
- Empty? → No
- fmt verb? → `%s` matches → **Not SQL**

### Example 4: URL containing SQL keyword (detected as Not SQL)

**Go source:**

```go
url := `https://example.com/select/users`
```

**AST:**

```
*ast.BasicLit {Kind: STRING, Value: "`https://example.com/select/users`"}
```

Inner content: `https://example.com/select/users`

- Trim → `https://example.com/select/users`
- Empty? → No
- fmt verb? → No
- SQL keyword prefix? → starts with `https`, not a SQL keyword → **Not SQL**

### Example 5: Non-SQL plain text (detected as Not SQL)

**Go source:**

```go
msg := `failed to execute query`
```

Inner content: `failed to execute query`

- Trim → `failed to execute query`
- Empty? → No
- fmt verb? → No
- SQL keyword prefix? → starts with `failed`, not a SQL keyword → **Not SQL**

### Example 6: Leading whitespace (detected as SQL)

**Go source:**

```go
db.Query(`  SELECT id FROM users`)
```

Inner content: `  SELECT id FROM users`

- Trim → `SELECT id FROM users`
- Empty? → No
- fmt verb? → No
- SQL keyword prefix? → `SELECT` matches → **SQL**

## Detection Result Summary

| Input | Result | Reason |
|-------|--------|--------|
| `select id from users` | SQL | Starts with `SELECT` |
| `INSERT INTO users (name) VALUES (?)` | SQL | Starts with `INSERT` |
| `update users set name = ?` | SQL | Starts with `UPDATE` (lowercase) |
| `delete from users where id = ?` | SQL | Starts with `DELETE` (lowercase) |
| `  SELECT id FROM users` | SQL | `SELECT` after leading whitespace |
| `SELECT %s FROM %s` | Not SQL | Contains fmt verb `%s` |
| `SELECT * FROM users LIMIT %d` | Not SQL | Contains fmt verb `%d` |
| `hello world` | Not SQL | Does not start with a SQL keyword |
| `https://example.com/select/users` | Not SQL | Does not start with a SQL keyword |
| `the SELECT statement` | Not SQL | `SELECT` is not at the start |
| _(empty string)_ | Not SQL | Empty string |
| `CREATE TABLE users (...)` | SQL | Starts with `CREATE` |
| `ALTER TABLE users ADD COLUMN name VARCHAR(255)` | SQL | Starts with `ALTER` |
| `DROP TABLE users` | SQL | Starts with `DROP` |
| `TRUNCATE TABLE users` | SQL | Starts with `TRUNCATE` |
| `START TRANSACTION` | SQL | Starts with `START` |
| `BEGIN` | SQL | Starts with `BEGIN` |
| `COMMIT` | SQL | Starts with `COMMIT` |
| `ROLLBACK` | SQL | Starts with `ROLLBACK` |
| `SAVEPOINT sp1` | SQL | Starts with `SAVEPOINT` |
| `RELEASE SAVEPOINT sp1` | SQL | Starts with `RELEASE` |
| `SET @x = 1` | SQL | Starts with `SET` |
| `SHOW TABLES` | SQL | Starts with `SHOW` |
| `DESCRIBE users` | SQL | Starts with `DESCRIBE` |
| `EXPLAIN SELECT * FROM users` | SQL | Starts with `EXPLAIN` |
| `USE mydb` | SQL | Starts with `USE` |
| `CALL my_proc()` | Not SQL | `CALL` is not a target keyword — parser does not support it |
| `PREPARE stmt FROM 'SELECT 1'` | Not SQL | `PREPARE` is not a target keyword — parser does not support it |
| `EXECUTE stmt` | Not SQL | `EXECUTE` is not a target keyword — parser does not support it |
| `DEALLOCATE PREPARE stmt` | Not SQL | `DEALLOCATE` is not a target keyword — parser does not support it |
| `DESC users` | Not SQL | `DESC` is not a target keyword — MySQL accepts it as a synonym for `DESCRIBE`, but the parser does not support it as a statement prefix (it only recognizes `DESC` as an `ORDER BY` direction) |

## Design Rationale

- **Minimize false positives**: Avoid misdetecting strings that resemble SQL, such as fmt templates and URLs
- **Lightweight pre-filter**: Reduce unnecessary input to the in-house parser to maintain performance
- **Conservative detection**: Limit target keywords to statement kinds the in-house parser can actually format (DML, DDL, transaction/session, and admin/utility statements); excludes statement kinds outside the parser's grammar, such as stored program syntax (`CALL`/`PREPARE`/`EXECUTE`/`DEALLOCATE`), since detecting a statement the formatter can't format only wastes a parse attempt
