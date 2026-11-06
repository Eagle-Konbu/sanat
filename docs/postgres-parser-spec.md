# PostgreSQL Parser Specification

Defines the specification for the PostgreSQL 13+ DML lexer/parser being built
in `internal/sqlfmt/pgast/` and `internal/sqlfmt/pgparser/`, tracked by
[epic #76](https://github.com/Eagle-Konbu/sanat/issues/76). It mirrors the
MySQL parser described in [parser-spec.md](parser-spec.md); this document is
filled in incrementally as each statement parser lands.

## Package Layout

```text
internal/sqlfmt/
├── pgast/     AST node types + String() fallback serialization (no parsing logic)
└── pgparser/  Lexer + recursive-descent parser producing pgast nodes
```

## INSERT Statement Grammar

Implemented in `internal/sqlfmt/pgparser/insert.go`
([#82](https://github.com/Eagle-Konbu/sanat/issues/82)):

```go
func ParseInsert(input string) (*pgast.Insert, error)
```

Like `ParseSelect`, it fully consumes `input`, accepting one optional trailing
`;` and failing with a `*ParseError` (or `*LexError`) otherwise.

```text
INSERT INTO table_name [ AS alias ] [ ( column [, ...] ) ]
    { DEFAULT VALUES | VALUES ( { expr | DEFAULT } [, ...] ) [, ...] | select }
    [ ON CONFLICT { [ ( column [, ...] ) ] DO NOTHING
                  | ( column [, ...] ) DO UPDATE SET
                        column = { expr | DEFAULT } [, ...] [ WHERE condition ] } ]
    [ RETURNING { * | output_expr [ [ AS ] output_name ] } [, ...] ]
```

- **Target alias**: PostgreSQL requires `AS` before an `INSERT` target alias
  (unlike a `FROM` item), so `INSERT INTO t x ...` is rejected.
- **`DEFAULT`**: accepted only as a `VALUES` item or as the right-hand side of
  a `DO UPDATE SET` assignment, not as a general expression. It is modeled as
  a `pgast.Literal` rendered verbatim.
- **`select`** row source: a plain `SELECT` statement (`parseSelectStatement`).
- **Conflict target**: optional for `DO NOTHING`, required for `DO UPDATE`.
  PostgreSQL rejects a `DO UPDATE` without a target during analysis, not in
  its grammar, but no input with one is valid, so it fails at parse time here.
- **`EXCLUDED.col`**: needs no special handling — it parses as an ordinary
  column reference qualified by the `EXCLUDED` pseudo-table, inside the
  `DO UPDATE SET` and `WHERE` expressions.
- **`RETURNING`** reuses the `SELECT` list grammar (`*`, `table.*`, and
  `expr [[AS] name]`); `String()` always renders an alias with `AS`.

Not yet implemented: a leading `WITH` clause
([#86](https://github.com/Eagle-Konbu/sanat/issues/86)),
`OVERRIDING { SYSTEM | USER } VALUE`, `ON CONFLICT ON CONSTRAINT name`,
expression/`COLLATE`/operator-class conflict targets, a
`WHERE index_predicate` on the conflict target, and PostgreSQL 18's
`RETURNING WITH (...)`.

## DELETE Statement Grammar

Implemented in `internal/sqlfmt/pgparser/delete.go`
([#84](https://github.com/Eagle-Konbu/sanat/issues/84)).

```go
func ParseDelete(input string) (*pgast.Delete, error)
```

```
DELETE FROM [ ONLY ] table_name [ * ] [ [ AS ] alias ]
    [ USING from_item [, ...] ]
    [ WHERE condition ]
    [ RETURNING { * | output_expr [ [ AS ] output_name ] } [, ...] ]
```

`ParseDelete` fully consumes `input`, accepting one optional trailing `;`
and failing with a `*ParseError` on any other trailing token. Malformed input
reached during lookahead may fail with a `*LexError` instead.

**Target table**: an optionally schema-qualified name, parsed into
`pgast.Delete.Table` (an `*pgast.AliasedTableExpr`). PostgreSQL's
table-inheritance markers are preserved: `ONLY name` sets
`AliasedTableExpr.Only` (the `ONLY (name)` spelling is also accepted and
renders as `ONLY name`), and a trailing `name *` sets
`AliasedTableExpr.IncludeDescendants`. The two are mutually exclusive —
`ONLY name *` is rejected, matching PostgreSQL. The alias may be written with
or without `AS`; `String()` always renders it without `AS`, the same as
`SELECT` table aliases.

**`USING`** is PostgreSQL's mechanism for joining other tables into a
`DELETE`. It takes the same `from_item` list as `SELECT`'s `FROM` clause
(comma lists, `JOIN` variants, parenthesized and derived tables), sharing
`parseTableReferenceList`. This is new grammar, not a port of the MySQL
parser's multi-table `DELETE t1 FROM t1, t2 ...` form.

**`RETURNING`** reuses the `SELECT` list grammar (`*`, `table.*`, and
`expr [[AS] name]`) and is stored as `pgast.Returning`.

PostgreSQL has no `ORDER BY` or `LIMIT` on `DELETE`, so both are rejected as
trailing tokens.

Deferred:

- A leading `WITH` clause ([#86](https://github.com/Eagle-Konbu/sanat/issues/86)).
- `WHERE CURRENT OF cursor_name` — it needs `DECLARE CURSOR` session state a
  formatter never has; it currently fails to parse.
- `RETURNING WITH (OLD/NEW AS ...)` — PostgreSQL 18 only.
