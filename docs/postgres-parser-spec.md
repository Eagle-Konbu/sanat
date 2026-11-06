# PostgreSQL Parser Specification

Defines the specification for the PostgreSQL lexer/parser being built in
`internal/sqlfmt/pgast/` and `internal/sqlfmt/pgparser/` (epic
[#76](https://github.com/Eagle-Konbu/sanat/issues/76)). It mirrors
[`parser-spec.md`](parser-spec.md), which covers the MySQL
`sqlast`/`parser` pair, and is filled in incrementally as each statement
grammar lands.

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
and failing with `*ParseError` on any other trailing token.

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
