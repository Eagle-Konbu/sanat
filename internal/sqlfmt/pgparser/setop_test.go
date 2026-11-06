package pgparser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertSetOpRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertSetOpRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	setOp, err := pgparser.ParseSetOp(input)
	if err != nil {
		t.Fatalf("ParseSetOp(%q) error = %v", input, err)
	}

	if got := setOp.String(); got != want {
		t.Errorf("ParseSetOp(%q).String() = %q, want %q", input, got, want)
	}
}

func mustParseSetOp(t *testing.T, input string) *pgast.SetOp {
	t.Helper()

	setOp, err := pgparser.ParseSetOp(input)
	if err != nil {
		t.Fatalf("ParseSetOp(%q) error = %v", input, err)
	}

	return setOp
}

func asSetOp(t *testing.T, stmt pgast.Statement) *pgast.SetOp {
	t.Helper()

	setOp, ok := stmt.(*pgast.SetOp)
	if !ok {
		t.Fatalf("got %#v, want *pgast.SetOp", stmt)
	}

	return setOp
}

func asSelect(t *testing.T, stmt pgast.Statement) *pgast.Select {
	t.Helper()

	sel, ok := stmt.(*pgast.Select)
	if !ok {
		t.Fatalf("got %#v, want *pgast.Select", stmt)
	}

	return sel
}

func TestParseSetOp_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"union", "SELECT 1 UNION SELECT 2", "SELECT 1 UNION SELECT 2"},
		{"union all", "SELECT 1 UNION ALL SELECT 2", "SELECT 1 UNION ALL SELECT 2"},
		{"union distinct renders as bare union", "SELECT 1 UNION DISTINCT SELECT 2", "SELECT 1 UNION SELECT 2"},
		{"intersect", "SELECT 1 INTERSECT SELECT 2", "SELECT 1 INTERSECT SELECT 2"},
		{"intersect all", "SELECT 1 INTERSECT ALL SELECT 2", "SELECT 1 INTERSECT ALL SELECT 2"},
		{"intersect distinct", "SELECT 1 INTERSECT DISTINCT SELECT 2", "SELECT 1 INTERSECT SELECT 2"},
		{"except", "SELECT 1 EXCEPT SELECT 2", "SELECT 1 EXCEPT SELECT 2"},
		{"except all", "SELECT 1 EXCEPT ALL SELECT 2", "SELECT 1 EXCEPT ALL SELECT 2"},
		{"lower-case keywords", "select 1 union all select 2", "SELECT 1 UNION ALL SELECT 2"},
		{"trailing semicolon", "SELECT 1 UNION SELECT 2;", "SELECT 1 UNION SELECT 2"},
		{
			"operands with full select cores",
			"SELECT DISTINCT a FROM x JOIN y ON x.id = y.id WHERE a > 1 GROUP BY a HAVING count(*) > 1 UNION SELECT b FROM z",
			"SELECT DISTINCT a FROM x JOIN y ON x.id = y.id WHERE a > 1 GROUP BY a HAVING count(*) > 1 UNION SELECT b FROM z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSetOpRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_precedence(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"chained unions", "SELECT 1 UNION SELECT 2 UNION ALL SELECT 3", "SELECT 1 UNION SELECT 2 UNION ALL SELECT 3"},
		{"union then except", "SELECT 1 UNION SELECT 2 EXCEPT SELECT 3", "SELECT 1 UNION SELECT 2 EXCEPT SELECT 3"},
		{"intersect binds tighter on the right", "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3", "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3"},
		{"intersect binds tighter on the left", "SELECT 1 INTERSECT SELECT 2 EXCEPT SELECT 3", "SELECT 1 INTERSECT SELECT 2 EXCEPT SELECT 3"},
		{
			"parens override precedence",
			"(SELECT 1 UNION SELECT 2) INTERSECT SELECT 3",
			"(SELECT 1 UNION SELECT 2) INTERSECT SELECT 3",
		},
		{
			"parens grouping right operand",
			"SELECT 1 EXCEPT (SELECT 2 UNION SELECT 3)",
			"SELECT 1 EXCEPT (SELECT 2 UNION SELECT 3)",
		},
		{"redundant operand parens dropped", "(SELECT 1) UNION (SELECT 2)", "SELECT 1 UNION SELECT 2"},
		{"redundant nested parens dropped", "((SELECT 1)) UNION ((SELECT 2))", "SELECT 1 UNION SELECT 2"},
		{"redundant left-group parens dropped", "(SELECT 1 UNION SELECT 2) UNION SELECT 3", "SELECT 1 UNION SELECT 2 UNION SELECT 3"},
		{
			"redundant intersect group parens dropped",
			"SELECT 1 UNION (SELECT 2 INTERSECT SELECT 3)",
			"SELECT 1 UNION SELECT 2 INTERSECT SELECT 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSetOpRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_leftAssociative(t *testing.T) {
	// a UNION b EXCEPT c groups as (a UNION b) EXCEPT c.
	outer := mustParseSetOp(t, "SELECT 1 UNION SELECT 2 EXCEPT ALL SELECT 3")

	if outer.Op != pgast.Except || !outer.All {
		t.Errorf("outer = %v ALL=%v, want EXCEPT ALL", outer.Op, outer.All)
	}

	inner := asSetOp(t, outer.Left)
	if inner.Op != pgast.Union || inner.All {
		t.Errorf("inner = %v ALL=%v, want UNION", inner.Op, inner.All)
	}

	asSelect(t, inner.Left)
	asSelect(t, inner.Right)
	asSelect(t, outer.Right)
}

func TestParseSetOp_intersectBindsTighter(t *testing.T) {
	tests := []struct {
		name, in    string
		outerOp     pgast.SetOpType
		intersectOn string
	}{
		{"a UNION b INTERSECT c is a UNION (b INTERSECT c)", "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3", pgast.Union, "right"},
		{"a EXCEPT b INTERSECT c is a EXCEPT (b INTERSECT c)", "SELECT 1 EXCEPT SELECT 2 INTERSECT SELECT 3", pgast.Except, "right"},
		{"a INTERSECT b UNION c is (a INTERSECT b) UNION c", "SELECT 1 INTERSECT SELECT 2 UNION SELECT 3", pgast.Union, "left"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outer := mustParseSetOp(t, tt.in)
			if outer.Op != tt.outerOp {
				t.Fatalf("outer op = %v, want %v", outer.Op, tt.outerOp)
			}

			grouped, single := outer.Right, outer.Left
			if tt.intersectOn == "left" {
				grouped, single = outer.Left, outer.Right
			}

			if inner := asSetOp(t, grouped); inner.Op != pgast.Intersect {
				t.Errorf("grouped op = %v, want INTERSECT", inner.Op)
			}

			asSelect(t, single)
		})
	}
}

func TestParseSetOp_trailingClauses(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"order by limit offset",
			"SELECT a FROM x UNION SELECT b FROM y ORDER BY 1 DESC LIMIT 10 OFFSET 5",
			"SELECT a FROM x UNION SELECT b FROM y ORDER BY 1 DESC LIMIT 10 OFFSET 5",
		},
		{
			"offset before limit",
			"SELECT 1 INTERSECT SELECT 2 OFFSET 5 LIMIT 10",
			"SELECT 1 INTERSECT SELECT 2 LIMIT 10 OFFSET 5",
		},
		{
			"fetch",
			"SELECT 1 EXCEPT SELECT 2 ORDER BY 1 OFFSET 1 ROWS FETCH NEXT 3 ROWS WITH TIES",
			"SELECT 1 EXCEPT SELECT 2 ORDER BY 1 OFFSET 1 FETCH NEXT 3 ROWS WITH TIES",
		},
		{
			"clauses after a parenthesized set operation",
			"(SELECT 1 UNION SELECT 2) ORDER BY 1 LIMIT 1",
			"SELECT 1 UNION SELECT 2 ORDER BY 1 LIMIT 1",
		},
		{
			"operand clauses stay with their operand",
			"(SELECT a FROM x ORDER BY a LIMIT 1) UNION ALL (SELECT b FROM y FETCH FIRST 2 ROWS ONLY) ORDER BY 1",
			"(SELECT a FROM x ORDER BY a LIMIT 1) UNION ALL (SELECT b FROM y FETCH FIRST 2 ROW ONLY) ORDER BY 1",
		},
		{
			"clauses split across operand parens",
			"((SELECT 1 ORDER BY 1) LIMIT 2) UNION SELECT 3",
			"(SELECT 1 ORDER BY 1 LIMIT 2) UNION SELECT 3",
		},
		{
			"locked operand",
			"(SELECT * FROM t FOR UPDATE SKIP LOCKED) UNION SELECT * FROM u",
			"(SELECT * FROM t FOR UPDATE SKIP LOCKED) UNION SELECT * FROM u",
		},
		{
			"nested set operation with its own clauses",
			"(SELECT 1 UNION SELECT 2 LIMIT 1) INTERSECT SELECT 3 ORDER BY 1",
			"(SELECT 1 UNION SELECT 2 LIMIT 1) INTERSECT SELECT 3 ORDER BY 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSetOpRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_trailingClausesApplyToWholeChain(t *testing.T) {
	outer := mustParseSetOp(t, "SELECT 1 UNION SELECT 2 INTERSECT SELECT 3 ORDER BY 1 LIMIT 5")

	if len(outer.OrderBy) != 1 || outer.Limit == nil {
		t.Errorf("outer OrderBy = %v, Limit = %v, want both set", outer.OrderBy, outer.Limit)
	}

	inner := asSetOp(t, outer.Right)
	if len(inner.OrderBy) != 0 || inner.Limit != nil {
		t.Errorf("inner OrderBy = %v, Limit = %v, want both unset", inner.OrderBy, inner.Limit)
	}

	last := asSelect(t, inner.Right)
	if len(last.OrderBy) != 0 || last.Limit != nil {
		t.Errorf("last operand OrderBy = %v, Limit = %v, want both unset", last.OrderBy, last.Limit)
	}
}

// TestParseSelect_setOpSubqueries checks that every parenthesized subquery
// position accepts a set operation, not just a single SELECT.
func TestParseSelect_setOpSubqueries(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"in subquery",
			"SELECT * FROM t WHERE id IN (SELECT a FROM x UNION SELECT b FROM y)",
			"SELECT * FROM t WHERE id IN (SELECT a FROM x UNION SELECT b FROM y)",
		},
		{
			"not in subquery with leading parenthesized operand",
			"SELECT * FROM t WHERE id NOT IN ((SELECT a FROM x) EXCEPT SELECT b FROM y)",
			"SELECT * FROM t WHERE id NOT IN (SELECT a FROM x EXCEPT SELECT b FROM y)",
		},
		{
			"in subquery with only trailing clauses after parens",
			"SELECT * FROM t WHERE id IN ((SELECT a FROM x) LIMIT 1)",
			"SELECT * FROM t WHERE id IN (SELECT a FROM x LIMIT 1)",
		},
		{
			"in list of scalar subqueries is still a list",
			"SELECT * FROM t WHERE id IN ((SELECT 1), 2)",
			"SELECT * FROM t WHERE id IN ((SELECT 1), 2)",
		},
		{
			"scalar subquery",
			"SELECT (SELECT 1 UNION ALL SELECT 2 LIMIT 1) FROM t",
			"SELECT (SELECT 1 UNION ALL SELECT 2 LIMIT 1) FROM t",
		},
		{
			"scalar subquery with leading parenthesized operand",
			"SELECT ((SELECT 1) INTERSECT SELECT 2) FROM t",
			"SELECT (SELECT 1 INTERSECT SELECT 2) FROM t",
		},
		{
			"scalar subquery with redundantly parenthesized leading operand",
			"SELECT (((SELECT 1)) UNION SELECT 2) FROM t",
			"SELECT (SELECT 1 UNION SELECT 2) FROM t",
		},
		{
			"exists",
			"SELECT * FROM t WHERE EXISTS (SELECT 1 FROM x INTERSECT SELECT 1 FROM y)",
			"SELECT * FROM t WHERE EXISTS (SELECT 1 FROM x INTERSECT SELECT 1 FROM y)",
		},
		{
			"exists with leading parenthesized operand",
			"SELECT * FROM t WHERE EXISTS ((SELECT 1) UNION SELECT 2)",
			"SELECT * FROM t WHERE EXISTS (SELECT 1 UNION SELECT 2)",
		},
		{
			"derived table",
			"SELECT * FROM (SELECT a FROM x UNION SELECT b FROM y) AS u",
			"SELECT * FROM (SELECT a FROM x UNION SELECT b FROM y) u",
		},
		{
			"derived table with leading parenthesized operand",
			"SELECT * FROM ((SELECT a FROM x) UNION (SELECT b FROM y)) u",
			"SELECT * FROM (SELECT a FROM x UNION SELECT b FROM y) u",
		},
		{
			"parenthesized join with derived table is still a join",
			"SELECT * FROM ((SELECT 1) a JOIN b ON true)",
			"SELECT * FROM ((SELECT 1) a JOIN b ON TRUE)",
		},
		{
			"lateral derived table",
			"SELECT * FROM t, LATERAL (SELECT t.a UNION SELECT t.b) u",
			"SELECT * FROM t, LATERAL (SELECT t.a UNION SELECT t.b) u",
		},
		{
			"comparison with subquery",
			"SELECT * FROM t WHERE a = (SELECT 1 EXCEPT SELECT 2)",
			"SELECT * FROM t WHERE a = (SELECT 1 EXCEPT SELECT 2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSetOp_errors(t *testing.T) {
	tests := []struct {
		name, in string
	}{
		{"empty", ""},
		{"single select", "SELECT 1"},
		{"single parenthesized select", "(SELECT 1)"},
		{"missing right operand", "SELECT 1 UNION"},
		{"both quantifiers", "SELECT 1 UNION ALL DISTINCT SELECT 2"},
		{"unclosed operand paren", "SELECT 1 UNION (SELECT 2"},
		{"order by on bare left operand", "SELECT 1 ORDER BY 1 UNION SELECT 2"},
		{"limit on bare left operand", "SELECT 1 LIMIT 1 UNION SELECT 2"},
		{"locking on bare left operand", "SELECT 1 FOR UPDATE UNION SELECT 2"},
		{"locking a set operation", "SELECT 1 UNION SELECT 2 FOR UPDATE"},
		{"multiple order by", "(SELECT 1 UNION SELECT 2 ORDER BY 1) ORDER BY 2"},
		{"multiple limit", "(SELECT 1 UNION SELECT 2 LIMIT 1) LIMIT 2"},
		{"limit then fetch", "(SELECT 1 UNION SELECT 2 LIMIT 1) FETCH FIRST ROW ONLY"},
		{"multiple offset", "(SELECT 1 UNION SELECT 2 OFFSET 1) OFFSET 2"},
		{"multiple order by on parenthesized select operand", "((SELECT 1 ORDER BY 1) ORDER BY 2) UNION SELECT 3"},
		{"multiple limit on parenthesized select operand", "((SELECT 1 FETCH FIRST ROW ONLY) LIMIT 2) UNION SELECT 3"},
		{"multiple offset on parenthesized select operand", "((SELECT 1 OFFSET 1) OFFSET 2) UNION SELECT 3"},
		{"trailing garbage", "SELECT 1 UNION SELECT 2)"},
		{"values operand is not supported", "SELECT 1 UNION VALUES (2)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pgparser.ParseSetOp(tt.in)
			if err == nil {
				t.Fatalf("ParseSetOp(%q) error = nil, want error", tt.in)
			}

			if _, ok := errors.AsType[*pgparser.ParseError](err); !ok {
				t.Errorf("ParseSetOp(%q) error = %T, want *pgparser.ParseError", tt.in, err)
			}
		})
	}
}

func TestParseSetOp_errorPositions(t *testing.T) {
	tests := []struct {
		name, in, msg string
		column        int
	}{
		{"not a set operation", "  SELECT 1", "expected UNION, INTERSECT, or EXCEPT", 3},
		{"locking a set operation", "SELECT 1 UNION SELECT 2 FOR UPDATE", "FOR UPDATE/SHARE is not allowed with UNION/INTERSECT/EXCEPT", 25},
		{"multiple order by", "(SELECT 1 UNION SELECT 2 ORDER BY 1) ORDER BY 2", "multiple ORDER BY clauses not allowed", 38},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pgparser.ParseSetOp(tt.in)

			parseErr, ok := errors.AsType[*pgparser.ParseError](err)
			if !ok {
				t.Fatalf("ParseSetOp(%q) error = %v, want *pgparser.ParseError", tt.in, err)
			}

			if parseErr.Msg != tt.msg || parseErr.Pos.Column != tt.column {
				t.Errorf("ParseSetOp(%q) error = %q at column %d, want %q at column %d",
					tt.in, parseErr.Msg, parseErr.Pos.Column, tt.msg, tt.column)
			}
		})
	}
}

func TestParseSelect_setOpErrors(t *testing.T) {
	for _, in := range []string{
		"SELECT 1 UNION SELECT 2",
		"SELECT * FROM t WHERE id IN ((SELECT 1 LIMIT 1) LIMIT 2)",
		"SELECT * FROM t WHERE id IN (1 UNION SELECT 2)",
		"SELECT ((SELECT 1) + 1 UNION SELECT 2)",
		"SELECT * FROM (a UNION SELECT 1) t",
		"SELECT * FROM ((SELECT 1) x UNION SELECT 2) t",
		"SELECT * FROM (a JOIN b ON TRUE UNION SELECT 1) t",
		"SELECT * FROM (LATERAL (SELECT 1) UNION SELECT 2) t",
		"SELECT * FROM (a, (SELECT 1) UNION SELECT 2) t",
	} {
		t.Run(in, func(t *testing.T) {
			assertSelectError(t, in)
		})
	}
}
