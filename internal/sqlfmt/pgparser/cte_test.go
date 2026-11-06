package pgparser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestParseSelect_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "WITH x AS (SELECT 1) SELECT * FROM x", "WITH x AS (SELECT 1) SELECT * FROM x"},
		{
			"recursive",
			"WITH RECURSIVE t (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM t WHERE n < 10) SELECT n FROM t",
			"WITH RECURSIVE t (n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM t WHERE n < 10) SELECT n FROM t",
		},
		{
			"multiple ctes",
			"WITH a AS (SELECT 1), b AS (SELECT * FROM a) SELECT * FROM b",
			"WITH a AS (SELECT 1), b AS (SELECT * FROM a) SELECT * FROM b",
		},
		{
			"column list",
			"WITH x (a, b) AS (SELECT 1, 2) SELECT a, b FROM x",
			"WITH x (a, b) AS (SELECT 1, 2) SELECT a, b FROM x",
		},
		{
			"materialized",
			"WITH x AS MATERIALIZED (SELECT 1) SELECT * FROM x",
			"WITH x AS MATERIALIZED (SELECT 1) SELECT * FROM x",
		},
		{
			"not materialized",
			"WITH x AS NOT MATERIALIZED (SELECT 1) SELECT * FROM x",
			"WITH x AS NOT MATERIALIZED (SELECT 1) SELECT * FROM x",
		},
		{
			"lower-case keywords",
			"with recursive x as not materialized (select 1) select * from x",
			"WITH RECURSIVE x AS NOT MATERIALIZED (SELECT 1) SELECT * FROM x",
		},
		{"quoted cte name", `WITH "X" AS (SELECT 1) SELECT * FROM "X"`, `WITH "X" AS (SELECT 1) SELECT * FROM "X"`},
		{
			"trailing clauses after with",
			"WITH x AS (SELECT 1 AS a) SELECT a FROM x ORDER BY a LIMIT 1 FOR UPDATE",
			"WITH x AS (SELECT 1 AS a) SELECT a FROM x ORDER BY a LIMIT 1 FOR UPDATE",
		},
		{
			"cte body with trailing clauses",
			"WITH x AS (SELECT a FROM t ORDER BY a LIMIT 5) SELECT * FROM x",
			"WITH x AS (SELECT a FROM t ORDER BY a LIMIT 5) SELECT * FROM x",
		},
		{
			"parenthesized cte body",
			"WITH x AS ((SELECT 1)) SELECT * FROM x",
			"WITH x AS (SELECT 1) SELECT * FROM x",
		},
		{
			"nested with in cte body",
			"WITH x AS (WITH y AS (SELECT 1) SELECT * FROM y) SELECT * FROM x",
			"WITH x AS (WITH y AS (SELECT 1) SELECT * FROM y) SELECT * FROM x",
		},
		{
			"with inside parenthesized cte body",
			"WITH x AS ((WITH y AS (SELECT 1) SELECT * FROM y)) SELECT * FROM x",
			"WITH x AS (WITH y AS (SELECT 1) SELECT * FROM y) SELECT * FROM x",
		},
		{
			"insert body",
			"WITH ins AS (INSERT INTO t (a) VALUES (1) RETURNING *) SELECT * FROM ins",
			"WITH ins AS (INSERT INTO t (a) VALUES (1) RETURNING *) SELECT * FROM ins",
		},
		{
			"update body",
			"WITH upd AS (UPDATE t SET a = 1 WHERE b = 2 RETURNING a) SELECT * FROM upd",
			"WITH upd AS (UPDATE t SET a = 1 WHERE b = 2 RETURNING a) SELECT * FROM upd",
		},
		{
			"delete body",
			"WITH del AS (DELETE FROM t WHERE a = 1 RETURNING *) SELECT count(*) FROM del",
			"WITH del AS (DELETE FROM t WHERE a = 1 RETURNING *) SELECT count(*) FROM del",
		},
		{
			"insert body with its own with",
			"WITH i AS (WITH v AS (SELECT 1) INSERT INTO t SELECT * FROM v RETURNING *) SELECT * FROM i",
			"WITH i AS (WITH v AS (SELECT 1) INSERT INTO t SELECT * FROM v RETURNING *) SELECT * FROM i",
		},
		{
			"update body with its own with",
			"WITH u AS (WITH v AS (SELECT 1 AS a) UPDATE t SET a = v.a FROM v RETURNING t.a) SELECT * FROM u",
			"WITH u AS (WITH v AS (SELECT 1 AS a) UPDATE t SET a = v.a FROM v RETURNING t.a) SELECT * FROM u",
		},
		{
			"delete body with its own with",
			"WITH d AS (WITH old AS (SELECT id FROM t) DELETE FROM t USING old WHERE t.id = old.id RETURNING t.id) SELECT * FROM d",
			"WITH d AS (WITH old AS (SELECT id FROM t) DELETE FROM t USING old WHERE t.id = old.id RETURNING t.id) SELECT * FROM d",
		},
		{"trailing semicolon", "WITH x AS (SELECT 1) SELECT * FROM x;", "WITH x AS (SELECT 1) SELECT * FROM x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_withInSubqueries(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"derived table",
			"SELECT * FROM (WITH x AS (SELECT 1) SELECT * FROM x) t",
			"SELECT * FROM (WITH x AS (SELECT 1) SELECT * FROM x) t",
		},
		{
			"lateral derived table",
			"SELECT * FROM a, LATERAL (WITH x AS (SELECT a.id) SELECT * FROM x) t",
			"SELECT * FROM a, LATERAL (WITH x AS (SELECT a.id) SELECT * FROM x) t",
		},
		{
			"scalar subquery",
			"SELECT (WITH x AS (SELECT 1) SELECT * FROM x)",
			"SELECT (WITH x AS (SELECT 1) SELECT * FROM x)",
		},
		{
			"in subquery",
			"SELECT 1 WHERE a IN (WITH x AS (SELECT 1) SELECT * FROM x)",
			"SELECT 1 WHERE a IN (WITH x AS (SELECT 1) SELECT * FROM x)",
		},
		{
			"exists subquery",
			"SELECT 1 WHERE EXISTS (WITH x AS (SELECT 1) SELECT * FROM x)",
			"SELECT 1 WHERE EXISTS (WITH x AS (SELECT 1) SELECT * FROM x)",
		},
		{
			"subquery with set operation",
			"SELECT * FROM (WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2) t",
			"SELECT * FROM (WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2) t",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"with scopes whole set operation",
			"WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2 ORDER BY 1",
			"WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2 ORDER BY 1",
		},
		{
			"with on parenthesized operand",
			"(WITH x AS (SELECT 1) SELECT * FROM x) UNION SELECT 2",
			"(WITH x AS (SELECT 1) SELECT * FROM x) UNION SELECT 2",
		},
		{
			"with on right operand",
			"SELECT 1 EXCEPT (WITH x AS (SELECT 2) SELECT * FROM x)",
			"SELECT 1 EXCEPT (WITH x AS (SELECT 2) SELECT * FROM x)",
		},
		{
			"outer and operand with",
			"WITH a AS (SELECT 1) (WITH b AS (SELECT 2) SELECT * FROM b) UNION SELECT * FROM a",
			"WITH a AS (SELECT 1) (WITH b AS (SELECT 2) SELECT * FROM b) UNION SELECT * FROM a",
		},
		{
			"with before parenthesized set operation",
			"WITH x AS (SELECT 1) (SELECT * FROM x UNION SELECT 2)",
			"WITH x AS (SELECT 1) SELECT * FROM x UNION SELECT 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSetOpRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_withAttachesToOutermost(t *testing.T) {
	setOp := mustParseSetOp(t, "WITH x AS (SELECT 1) SELECT 1 UNION SELECT 2 INTERSECT SELECT 3")

	if setOp.With == nil || len(setOp.With.CTEs) != 1 || setOp.With.CTEs[0].Name != "x" {
		t.Fatalf("outermost With = %#v, want one CTE named x", setOp.With)
	}

	if left := asSelect(t, setOp.Left); left.With != nil {
		t.Errorf("left operand With = %#v, want nil", left.With)
	}
}

func TestParseInsert_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"leading with",
			"WITH src AS (SELECT 1 AS a) INSERT INTO t (a) SELECT a FROM src",
			"WITH src AS (SELECT 1 AS a) INSERT INTO t (a) SELECT a FROM src",
		},
		{
			"delete body feeding insert",
			"WITH deleted AS (DELETE FROM logs WHERE created_at < now() RETURNING *) INSERT INTO archive SELECT * FROM deleted",
			"WITH deleted AS (DELETE FROM logs WHERE created_at < now() RETURNING *) INSERT INTO archive SELECT * FROM deleted",
		},
		{
			"with on select row source",
			"INSERT INTO t WITH src AS (SELECT 1) SELECT * FROM src",
			"INSERT INTO t WITH src AS (SELECT 1) SELECT * FROM src",
		},
		{
			"with on statement and row source",
			"WITH a AS (SELECT 1) INSERT INTO t WITH b AS (SELECT * FROM a) SELECT * FROM b RETURNING *",
			"WITH a AS (SELECT 1) INSERT INTO t WITH b AS (SELECT * FROM a) SELECT * FROM b RETURNING *",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseUpdate_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"leading with",
			"WITH src AS (SELECT id, v FROM s) UPDATE t SET v = src.v FROM src WHERE t.id = src.id",
			"WITH src AS (SELECT id, v FROM s) UPDATE t SET v = src.v FROM src WHERE t.id = src.id",
		},
		{
			"insert body feeding update",
			"WITH ins AS (INSERT INTO log (msg) VALUES ('x') RETURNING id) UPDATE t SET log_id = ins.id FROM ins",
			"WITH ins AS (INSERT INTO log (msg) VALUES ('x') RETURNING id) UPDATE t SET log_id = ins.id FROM ins",
		},
		{
			"with in multi-column set source",
			"UPDATE t SET (a, b) = (WITH x AS (SELECT 1, 2) SELECT * FROM x)",
			"UPDATE t SET (a, b) = (WITH x AS (SELECT 1, 2) SELECT * FROM x)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertUpdateRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDelete_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"leading with",
			"WITH old AS (SELECT id FROM t WHERE expired) DELETE FROM t USING old WHERE t.id = old.id",
			"WITH old AS (SELECT id FROM t WHERE expired) DELETE FROM t USING old WHERE t.id = old.id",
		},
		{
			"update body feeding delete",
			"WITH RECURSIVE moved AS (UPDATE a SET b = 1 RETURNING id) DELETE FROM c WHERE id IN (SELECT id FROM moved)",
			"WITH RECURSIVE moved AS (UPDATE a SET b = 1 RETURNING id) DELETE FROM c WHERE id IN (SELECT id FROM moved)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDeleteRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseInsert_withStructure(t *testing.T) {
	ins, err := pgparser.ParseInsert(
		"WITH deleted AS MATERIALIZED (DELETE FROM logs RETURNING id) INSERT INTO archive SELECT id FROM deleted")
	if err != nil {
		t.Fatalf("ParseInsert error = %v", err)
	}

	if ins.With == nil || len(ins.With.CTEs) != 1 {
		t.Fatalf("With = %#v, want one CTE", ins.With)
	}

	cte := ins.With.CTEs[0]
	if cte.Materialization != pgast.Materialized {
		t.Errorf("Materialization = %v, want Materialized", cte.Materialization)
	}

	if _, ok := cte.Subquery.(*pgast.Delete); !ok {
		t.Errorf("CTE body = %T, want *pgast.Delete", cte.Subquery)
	}

	sel, ok := ins.Rows.(*pgast.Select)
	if !ok {
		t.Fatalf("Rows = %T, want *pgast.Select", ins.Rows)
	}

	if sel.With != nil {
		t.Errorf("row source With = %#v, want nil", sel.With)
	}
}

func TestParseSelect_withErrors(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"with without cte", "WITH SELECT 1"},
		{"missing as", "WITH x (SELECT 1) SELECT * FROM x"},
		{"missing body parens", "WITH x AS SELECT 1 SELECT * FROM x"},
		{"empty body", "WITH x AS () SELECT * FROM x"},
		{"empty column list", "WITH x () AS (SELECT 1) SELECT * FROM x"},
		{"not without materialized", "WITH x AS NOT (SELECT 1) SELECT * FROM x"},
		{"trailing comma", "WITH x AS (SELECT 1), SELECT * FROM x"},
		{"missing main statement", "WITH x AS (SELECT 1)"},
		{"values body", "WITH x AS (VALUES (1)) SELECT * FROM x"},
		{"search clause", "WITH RECURSIVE x AS (SELECT 1) SEARCH DEPTH FIRST BY a SET o SELECT * FROM x"},
		{"cycle clause", "WITH RECURSIVE x AS (SELECT 1) CYCLE a SET c USING p SELECT * FROM x"},
		{"with on bare set operand", "SELECT 1 FROM (SELECT 1 UNION WITH x AS (SELECT 2) SELECT * FROM x) t"},
		{"dml in subquery", "SELECT * FROM (DELETE FROM t RETURNING *) d"},
		{"with followed by dml", "WITH x AS (SELECT 1) DELETE FROM t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectError(t, tt.in)
		})
	}
}

func TestParseDML_withErrors(t *testing.T) {
	tests := []struct {
		name  string
		parse func(string) error
		in    string
	}{
		{"insert: repeated with", parseInsertErr, "WITH a AS (SELECT 1) WITH b AS (SELECT 2) INSERT INTO t SELECT 1"},
		{"update: with followed by select", parseUpdateErr, "WITH a AS (SELECT 1) SELECT 1"},
		{"delete: dangling with", parseDeleteErr, "WITH a AS (SELECT 1) FROM t"},
		{"delete: repeated with in body", parseDeleteErr,
			"WITH a AS (WITH b AS (SELECT 1) WITH c AS (SELECT 2) DELETE FROM t) DELETE FROM u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.parse(tt.in)
			if _, ok := errors.AsType[*pgparser.ParseError](err); !ok {
				t.Errorf("parse(%q) error = %v, want *pgparser.ParseError", tt.in, err)
			}
		})
	}
}

func TestParse_multipleWithClauses(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		column int
	}{
		{
			"cte body",
			"WITH a AS (WITH b AS (SELECT 1) (WITH c AS (SELECT 2) SELECT 3)) SELECT 1",
			12,
		},
		{
			"derived table",
			"SELECT * FROM (WITH b AS (SELECT 1) (WITH c AS (SELECT 2) SELECT 3)) t",
			16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pgparser.ParseSelect(tt.in)

			parseErr, ok := errors.AsType[*pgparser.ParseError](err)
			if !ok {
				t.Fatalf("ParseSelect(%q) error = %v, want *pgparser.ParseError", tt.in, err)
			}

			const msg = "multiple WITH clauses not allowed"
			if parseErr.Msg != msg || parseErr.Pos.Column != tt.column {
				t.Errorf("ParseSelect(%q) error = %q at column %d, want %q at column %d",
					tt.in, parseErr.Msg, parseErr.Pos.Column, msg, tt.column)
			}
		})
	}
}

func parseInsertErr(in string) error {
	_, err := pgparser.ParseInsert(in)

	return err
}

func parseUpdateErr(in string) error {
	_, err := pgparser.ParseUpdate(in)

	return err
}

func parseDeleteErr(in string) error {
	_, err := pgparser.ParseDelete(in)

	return err
}
