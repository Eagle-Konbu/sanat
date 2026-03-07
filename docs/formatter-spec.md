# Formatter Specification

Defines the specification for sanat's SQL formatter.

## Overview

sanat is a CLI tool that automatically formats SQL string literals embedded in Go source files. It detects SQL within raw string literals (backtick strings) and formats them in a consistent style.

## Processing Flow

```mermaid
flowchart TD
    A[Go source file input] --> B[AST parsing]
    B --> C[Extract raw string literals]
    C --> D{MightBeSQL?}
    D -- No --> E[Skip]
    D -- Yes --> F[Placeholder substitution<br/>? → :_sqla_ph_N]
    F --> G[Parse with Vitess SQL parser]
    G --> H{Parse success?}
    H -- No --> I[Keep original string]
    H -- Yes --> J[Format according to SQL statement type]
    J --> K[Restore placeholders<br/>:_sqla_ph_N → ?]
    K --> L[Remove backtick identifiers]
    L --> M[Replace AST node with<br/>formatted string]
    M --> N[Output with go/format]
```

## SQL Detection

See [detect-spec.md](detect-spec.md) for SQL detection rules.

## Format Targets

- Only **raw string literals** (backtick-quoted strings) in Go source files
- Double-quoted strings are excluded

```go
// Format target
db.Exec(`select id from users where id = ?`, 1)

// Not a format target (double-quoted)
db.Exec("select id from users where id = ?", 1)
```

## Placeholder Handling

Since the SQL parser cannot handle `?` correctly, substitution and restoration are performed before and after parsing.

1. **Substitution**: `?` → `:_sqla_ph_0`, `:_sqla_ph_1`, ... (indexed in order of appearance)
2. **Parsing**: Syntax analysis with the Vitess SQL parser
3. **Restoration**: `:_sqla_ph_N` → `?`

## Format Rules

### Common Rules

- SQL keywords are converted to **UPPERCASE**
- Each clause is placed on a **separate line**
- Clause contents are **indented** (default: 2 spaces)
- Backtick identifiers (MySQL style) are removed after formatting
- If parsing fails, the **original string is returned as-is**

### Keyword Uppercasing

The following keywords are uppercased during formatting:

`AS`, `ASC`, `DESC`, `AND`, `OR`, `NOT`, `IN`, `IS`, `LIKE`, `BETWEEN`, `EXISTS`, `NULL`, `TRUE`, `FALSE`, `ON`, `USING`

Clause keywords (`SELECT`, `FROM`, `WHERE`, etc.) are structurally output in uppercase.

### Indentation

```
depth * indent spaces
```

- `depth`: nesting depth (0-based)
- `indent`: indent width (default: 2)

Indentation increases with deeper nesting (e.g., subqueries).

## Statement Type Formatting

### SELECT

```mermaid
flowchart TD
    W0{WITH?} -- Yes --> CTE[WITH / WITH RECURSIVE + CTEs]
    W0 -- No --> S
    CTE --> S[SELECT]
    S --> D{DISTINCT?}
    D -- Yes --> D1["SELECT DISTINCT"]
    D -- No --> D2["SELECT"]
    D1 --> SE[SelectExprs]
    D2 --> SE
    SE --> F{FROM?}
    F -- Yes --> FE[FROM + TableExprs]
    F -- No --> W
    FE --> W{WHERE?}
    W -- Yes --> WE[WHERE + condition]
    W -- No --> G
    WE --> G{GROUP BY?}
    G -- Yes --> GE[GROUP BY + expressions]
    G -- No --> H
    GE --> H{HAVING?}
    H -- Yes --> HE[HAVING + condition]
    H -- No --> O
    HE --> O{ORDER BY?}
    O -- Yes --> OE[ORDER BY + expressions]
    O -- No --> L
    OE --> L{LIMIT?}
    L -- Yes --> LE[LIMIT / OFFSET]
    L -- No --> LK
    LE --> LK{Lock?}
    LK -- Yes --> LKE[FOR UPDATE / FOR SHARE / etc.]
    LK -- No --> END[End]
    LKE --> END
```

**Example output:**

```sql
SELECT
  u.id,
  u.name
FROM
  users u
WHERE
  u.status = ?
  AND u.active = TRUE
GROUP BY
  u.status
HAVING
  count(*) > 1
ORDER BY
  u.id DESC
LIMIT
  10
OFFSET
  20
```

### INSERT

```
INSERT INTO          -- or REPLACE INTO, INSERT IGNORE INTO
  <table>
(                    -- column list (if present)
  <column1>,
  <column2>
)
VALUES               -- or SELECT subquery
  (<value1>, <value2>)
ON DUPLICATE KEY UPDATE  -- if present
  <expr1>,
  <expr2>
```

The `IGNORE` modifier is supported: `INSERT IGNORE INTO`.

**Example output:**

```sql
INSERT INTO
  users
(
  name,
  email
)
VALUES
  (?, ?)
ON DUPLICATE KEY UPDATE
  name = values(name),
  email = values(email)
```

```sql
INSERT IGNORE INTO
  users
(
  name
)
VALUES
  (?)
```

### UPDATE

```
UPDATE               -- or UPDATE IGNORE
  <table>
SET
  <expr1>,
  <expr2>
WHERE              -- if present
  <condition>
ORDER BY           -- if present
  <expression>
LIMIT              -- if present
  <value>
```

The `IGNORE` modifier and `WITH` clause (CTE) are supported. Multi-table UPDATE with JOIN is also supported.

**Example output:**

```sql
UPDATE
  users
SET
  name = ?,
  email = ?
WHERE
  id = ?
```

### DELETE

```
DELETE FROM           -- single-table, or DELETE IGNORE FROM
  <table>
WHERE              -- if present
  <condition>
ORDER BY           -- if present
  <expression>
LIMIT              -- if present
  <value>
```

Multi-table DELETE uses a separate target list:

```
DELETE
  <target1>,
  <target2>
FROM
  <table_exprs>
WHERE
  <condition>
```

The `IGNORE` modifier and `WITH` clause (CTE) are supported.

**Example output:**

```sql
DELETE FROM
  users
WHERE
  id = ?
```

```sql
DELETE
  t1,
  t2
FROM
  t1
  JOIN
  t2
    ON t1.id = t2.ref_id
WHERE
  t2.status = ?
```

### UNION / UNION ALL

Formats the left and right SELECT statements independently and joins them with `UNION` or `UNION ALL`. The `WITH` clause (CTE) and locking clauses are supported.

**Example output:**

```sql
SELECT
  id
FROM
  users
UNION ALL
SELECT
  id
FROM
  admins
```

## Expression Formatting

### WHERE Clause Conditions

- Conditions joined with `AND` are expanded to separate lines
- Conditions joined with `OR` are also expanded to separate lines
- The first condition has no prefix; subsequent conditions are prefixed with `AND` / `OR`

```sql
WHERE
  u.status = ?
  AND u.active = TRUE
  OR u.role = 'admin'
```

### Table Expressions

#### Simple Table

```sql
FROM
  users u
```

#### JOIN

Expands the left and right sides of the JOIN, with the `ON` condition at additional indentation.

```sql
FROM
  users u
  JOIN
  orders o
    ON u.id = o.user_id
```

#### Index Hints

Index hints (`USE INDEX`, `FORCE INDEX`, `IGNORE INDEX`) are appended after the table name/alias on the same line.

```sql
FROM
  users USE INDEX (idx_name)
```

```sql
FROM
  users FORCE INDEX (idx_created)
```

The optional `FOR` clause (`FOR JOIN`, `FOR ORDER BY`, `FOR GROUP BY`) is also supported.

#### Derived Table (Subquery)

Wraps the subquery in parentheses and formats the interior with nested indentation.

```sql
FROM
  (
  SELECT
    id
  FROM
    users
  ) t
```

### Subquery Expressions

#### EXISTS

```sql
WHERE
  EXISTS (
    SELECT
      1
    FROM
      orders o
    WHERE
      o.user_id = u.id
  )
```

#### Scalar Subquery

```sql
  (
    SELECT
      count(*)
    FROM
      orders
  )
```

### NOT Expression

The `NOT` prefix is preserved in front of any expression.

```sql
WHERE
  NOT status = 'deleted'
```

```sql
WHERE
  NOT EXISTS (
    SELECT
      1
    FROM
      banned
    WHERE
      banned.user_id = users.id
  )
```

### CASE Expression

CASE expressions are formatted with WHEN/ELSE clauses indented one level deeper than CASE/END.

**Searched CASE (no expression):**

```sql
  CASE
    WHEN status = 1 THEN 'active'
    WHEN status = 2 THEN 'inactive'
    ELSE 'unknown'
  END
```

**Simple CASE (with expression):**

```sql
  CASE status
    WHEN 1 THEN 'active'
    WHEN 2 THEN 'inactive'
  END
```

### Window Functions (OVER Clause)

Aggregate and window functions with OVER clauses are formatted with the window specification on multiple lines.

**Inline window specification:**

```sql
SELECT
  sum(amount) OVER (
    PARTITION BY user_id
    ORDER BY created_at
  )
FROM
  orders
```

**Named window reference:**

```sql
SELECT
  sum(amount) OVER w
FROM
  orders
```

Supported function types: COUNT, COUNT(*), SUM, AVG, MIN, MAX, BIT_AND, BIT_OR, BIT_XOR, STD, STDDEV, STDDEV_POP, STDDEV_SAMP, VAR_POP, VAR_SAMP, VARIANCE, ROW_NUMBER, RANK, DENSE_RANK, PERCENT_RANK, CUME_DIST, FIRST_VALUE, LAST_VALUE, NTILE, NTH_VALUE, LAG, LEAD, JSON_ARRAYAGG, JSON_OBJECTAGG.

### SELECT Expressions

- Each column is placed on a separate line
- Aliases are connected with `AS`
- Wildcard `*` and table-qualified `t.*` are supported

```sql
SELECT
  u.id,
  u.name AS user_name,
  count(*) AS cnt
```

### Locking Clauses

Locking clauses are placed on their own line after LIMIT (or after the last clause if no LIMIT). Supported clauses: `FOR UPDATE`, `FOR SHARE`, `LOCK IN SHARE MODE`, `FOR UPDATE SKIP LOCKED`, `FOR UPDATE NOWAIT`, `FOR SHARE SKIP LOCKED`, `FOR SHARE NOWAIT`.

```sql
SELECT
  *
FROM
  users
FOR UPDATE
```

```sql
SELECT
  *
FROM
  users
FOR UPDATE SKIP LOCKED
```

### WITH Clause (Common Table Expressions)

CTE definitions appear before the main statement. Each CTE subquery is indented. Supported on SELECT, UNION, UPDATE, and DELETE statements.

```sql
WITH
  cte AS (
    SELECT
      id
    FROM
      users
  )
SELECT
  *
FROM
  cte
```

**Multiple CTEs:**

```sql
WITH
  a AS (
    ...
  ),
  b AS (
    ...
  )
SELECT
  *
FROM
  a, b
```

**RECURSIVE CTE:**

```sql
WITH RECURSIVE
  cte AS (
    SELECT
      1 AS id
    UNION ALL
    SELECT
      id + 1
    FROM
      cte
    WHERE
      id < 10
  )
SELECT
  *
FROM
  cte
```

**CTE with column list:**

```sql
WITH
  cte (id, name) AS (
    SELECT
      id,
      name
    FROM
      users
  )
SELECT
  *
FROM
  cte
```

## Configuration

### Configuration File

Configuration files are searched in the following order (first match is used):

1. `.sanat.yml`
2. `.sanat.yaml`
3. `.sanat.toml`

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `write` | bool | `false` | Whether to overwrite files |
| `indent` | int | `2` | SQL indent width (number of spaces) |
| `newline` | bool | `true` | Whether to insert a newline after the opening backtick |

### Configuration Examples

**YAML:**

```yaml
write: true
indent: 4
newline: true
```

**TOML:**

```toml
write = true
indent = 4
newline = true
```

### Precedence

CLI flags > configuration file > default values

When a flag is explicitly specified, it takes precedence over the configuration file value.

## CLI

### Usage

```
sanat [flags] [pattern ...]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--write` | `-w` | `false` | Overwrite files |
| `--indent` | | `2` | SQL indent width |
| `--newline` | | `true` | Newline after opening backtick |
| `--config` | `-c` | | Configuration file path |

### Input Methods

- **File patterns**: `sanat file.go`, `sanat ./...`, `sanat *.go`
- **Standard input**: `cat file.go | sanat`

### Pattern Resolution

- `./...` — recursively traverse directories
- Directory path — traverse `.go` files within the directory
- Glob pattern — target matching `.go` files

### Excluded Directories

The following directories are excluded from traversal:

- `vendor/`
- `.git/`
- `testdata/`

### Output

- Default: output formatted result to stdout
- With `-w`: overwrite files directly (permission 0600)

## Newline Option

When the `newline` option is `true` (default), newlines are inserted before and after the formatted SQL.

**newline: true:**

```go
db.Exec(`
SELECT
  id
FROM
  users
`, 1)
```

**newline: false:**

```go
db.Exec(`SELECT
  id
FROM
  users`, 1)
```

## Parser

The [Vitess](https://vitess.io/) SQL parser (`vitess.io/vitess/go/vt/sqlparser`) is used for SQL syntax analysis. It supports MySQL-compatible SQL syntax.

### Supported SQL Statements

| Statement Type | Supported |
|---------------|-----------|
| SELECT | o |
| INSERT | o |
| REPLACE | o |
| UPDATE | o |
| DELETE | o |
| UNION / UNION ALL | o |
| Other | Falls back to Vitess default output |
