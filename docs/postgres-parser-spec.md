# PostgreSQL Parser Specification

Defines the specification for the PostgreSQL lexer/parser being built in
`internal/sqlfmt/pgast/` and `internal/sqlfmt/pgparser/` (epic
[#76](https://github.com/Eagle-Konbu/sanat/issues/76)). This document is
filled in incrementally as each grammar issue lands; see
[parser-spec.md](parser-spec.md) for the MySQL counterpart.

## Set Operation Grammar

Implemented in `internal/sqlfmt/pgparser/setop.go`
([#85](https://github.com/Eagle-Konbu/sanat/issues/85)).

### Entry Point

```go
func ParseSetOp(input string) (*pgast.SetOp, error)
```

`ParseSetOp` parses two or more SELECT operands combined by set operators,
plus any trailing clauses on the combined result, and an optional trailing
`;`. It fails with a `*ParseError` when the input is a single (possibly
parenthesized) SELECT with no set operator; use `ParseSelect` for that case.
`ParseSelect` in turn rejects a top-level set operation.

### Grammar

```
query      := union_term trailing
union_term := intersect_term { ( UNION | EXCEPT ) [ ALL | DISTINCT ] intersect_term }
intersect_term := operand { INTERSECT [ ALL | DISTINCT ] operand }
operand    := simple_select | "(" query ")"
trailing   := [ ORDER BY ... ] [ LIMIT ... | OFFSET ... ] [ FETCH ... ] { FOR ... }
```

- `simple_select` is a SELECT from the `SELECT` keyword through its
  `WINDOW` clause. A bare operand cannot carry its own `ORDER BY`/`LIMIT`/
  `OFFSET`/`FETCH`/locking clause; wrap it in parentheses to scope one to it,
  as PostgreSQL requires.
- Precedence follows PostgreSQL: `INTERSECT` binds tighter than `UNION` and
  `EXCEPT`, which share a precedence level. All three associate left, so
  `a UNION b INTERSECT c` is `a UNION (b INTERSECT c)` and
  `a UNION b EXCEPT c` is `(a UNION b) EXCEPT c`. Each operator becomes one
  `pgast.SetOp{Left, Op, All, Right}` node.
- `DISTINCT` is the default quantifier and is not modeled separately: an
  explicit `DISTINCT` renders as the bare operator.
- Trailing clauses after the operator chain apply to the whole set
  operation and are stored on the outermost `SetOp`.
- A trailing clause after a parenthesized query is attached to that query,
  so `(a UNION b) ORDER BY 1` is a single `SetOp` with `OrderBy` set, and
  `(SELECT 1 ORDER BY 1) LIMIT 2` is a single `Select`. Like PostgreSQL,
  giving the same clause at both levels is an error (`multiple ORDER BY
  clauses not allowed`, likewise for `LIMIT` — which `FETCH` counts as — and
  `OFFSET`).
- A locking clause (`FOR UPDATE` and friends) on a set operation is a
  `*ParseError`, matching PostgreSQL's rejection of it; `pgast.SetOp` has no
  field for one. A parenthesized SELECT operand may still carry its own.
- `VALUES` and `TABLE name` operands, and `WITH` (deferred to
  [#86](https://github.com/Eagle-Konbu/sanat/issues/86)), are not supported
  yet.

### Subquery Positions

Every parenthesized subquery position — scalar subqueries, `[NOT] IN (...)`,
`EXISTS (...)`, derived tables (including `LATERAL`), and comparisons
against a subquery — accepts a full `query`, not just a single SELECT. A
leading parenthesized operand such as `IN ((SELECT 1) UNION SELECT 2)` or
`FROM ((SELECT 1) UNION SELECT 2) t` cannot be told apart from a
parenthesized expression or table reference until the token after its `)`,
so those contexts parse the operand first and continue the query when a set
operator or trailing clause follows. `IN ((SELECT 1), 2)` stays a value list
and `FROM ((SELECT 1) a JOIN b ON TRUE)` stays a parenthesized join.

### Rendering

`SetOp.String()` parenthesizes an operand only when needed: a nested
`SetOp` whose grouping precedence and left associativity don't already imply,
or any operand with its own `WITH`, `ORDER BY`, `LIMIT`, `OFFSET`, `FETCH`,
or (for a SELECT) locking clause. Redundant parentheses in the input are
therefore dropped, e.g. `(SELECT 1) UNION (SELECT 2 UNION SELECT 3)` renders
as `SELECT 1 UNION (SELECT 2 UNION SELECT 3)` while
`(SELECT 1 UNION SELECT 2) UNION SELECT 3` renders as
`SELECT 1 UNION SELECT 2 UNION SELECT 3`.

### Testing

`internal/sqlfmt/pgparser/setop_test.go` holds table-driven round-trip tests
through `String()` (mirroring `select_test.go` and
`internal/sqlfmt/parser/union_test.go`), structural tests for precedence,
associativity, and trailing-clause placement, subquery-position tests via
`ParseSelect`, and error cases with positions.
