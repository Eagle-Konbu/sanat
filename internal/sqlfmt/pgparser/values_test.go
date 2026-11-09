package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestParseStatement_valuesAndTableQueries(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"values", "VALUES (1, 'a'), (2, 'b')", "VALUES (1, 'a'), (2, 'b')"},
		{"values with trailing clauses", "VALUES (2), (1) ORDER BY 1 LIMIT 1", "VALUES (2), (1) ORDER BY 1 LIMIT 1"},
		{"set operation operand", "SELECT 1 UNION VALUES (2)", "SELECT 1 UNION VALUES (2)"},
		{"leading values operand", "VALUES (1) EXCEPT SELECT 2", "VALUES (1) EXCEPT SELECT 2"},
		{"derived table", "SELECT * FROM (VALUES (1), (2)) v(a)", "SELECT * FROM (VALUES (1), (2)) v(a)"},
		{"cte body", "WITH c AS (VALUES (1)) SELECT * FROM c", "WITH c AS (VALUES (1)) SELECT * FROM c"},
		{"with values", "WITH c AS (SELECT 1) VALUES (2)", "WITH c AS (SELECT 1) VALUES (2)"},
		{"in subquery", "SELECT 1 WHERE a IN (VALUES (1), (2))", "SELECT 1 WHERE a IN (VALUES (1), (2))"},
		{"values column name", "SELECT values FROM t", "SELECT values FROM t"},
		{"table", "TABLE t", "TABLE t"},
		{"table only", "TABLE ONLY s.t ORDER BY 1", "TABLE ONLY s.t ORDER BY 1"},
		{"table descendants", "TABLE t *", "TABLE t *"},
		{"table operand", "SELECT 1 UNION TABLE t", "SELECT 1 UNION TABLE t"},
		{"table subquery", "SELECT * FROM (TABLE t) x", "SELECT * FROM (TABLE t) x"},
		{"no target list", "SELECT FROM t WHERE a", "SELECT FROM t WHERE a"},
		{"bare select", "SELECT;", "SELECT"},
		{"no target list operand", "SELECT UNION SELECT", "SELECT UNION SELECT"},
		{"insert values set operation", "INSERT INTO t VALUES (1) UNION SELECT 2", "INSERT INTO t VALUES (1) UNION SELECT 2"},
		{"insert parenthesized query", "INSERT INTO t (SELECT 1)", "INSERT INTO t SELECT 1"},
		{"insert parenthesized values", "INSERT INTO t (VALUES (1))", "INSERT INTO t VALUES (1)"},
		{"insert columns then parenthesized query", "INSERT INTO t (a) ((SELECT 1))", "INSERT INTO t (a) SELECT 1"},
		{"insert table", "INSERT INTO t TABLE u", "INSERT INTO t TABLE u"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{"VALUES", "VALUES ()", "TABLE", "TABLE t x", "SELECT DISTINCT FROM t", "table + 1"} {
		t.Run("error "+in, func(t *testing.T) {
			if _, err := pgparser.ParseStatement(in); err == nil {
				t.Errorf("ParseStatement(%q) error = nil, want error", in)
			}
		})
	}
}

func TestParseSelect_valuesStructure(t *testing.T) {
	sel, err := pgparser.ParseSelect("VALUES (1) LIMIT 1")
	if err != nil {
		t.Fatal(err)
	}

	if len(sel.Values) != 1 || sel.Limit == nil {
		t.Errorf("ParseSelect = %#v, want one VALUES row and a LIMIT", sel)
	}

	if _, ok := any(sel).(pgast.InsertRows); !ok {
		t.Error("a VALUES query must be usable as INSERT rows")
	}
}
