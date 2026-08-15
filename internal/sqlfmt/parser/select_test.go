package parser_test

import (
	"reflect"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// assertSelectRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want. This is enough
// for grammar breadth (clause presence/order/keywords) but not for
// expression precedence, which is covered separately via structural checks.
func assertSelectRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	sel, err := parser.ParseSelect(input)
	if err != nil {
		t.Fatalf("ParseSelect(%q) error = %v", input, err)
	}

	if got := sel.String(); got != want {
		t.Errorf("ParseSelect(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseSelect_basic(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "SELECT id, name FROM users", "SELECT id, name FROM users"},
		{"star", "SELECT * FROM t", "SELECT * FROM t"},
		{"qualified star", "SELECT t.* FROM t", "SELECT t.* FROM t"},
		{"distinct", "SELECT DISTINCT id FROM t", "SELECT DISTINCT id FROM t"},
		{"select all is a no-op", "SELECT ALL id FROM t", "SELECT id FROM t"},
		{"explicit alias", "SELECT id AS user_id FROM t", "SELECT id AS user_id FROM t"},
		{"implicit alias", "SELECT id user_id FROM t", "SELECT id AS user_id FROM t"},
		{"table alias", "SELECT * FROM users AS u", "SELECT * FROM users u"},
		{"implicit table alias", "SELECT * FROM users u", "SELECT * FROM users u"},
		{"qualified table", "SELECT * FROM mydb.users", "SELECT * FROM mydb.users"},
		{"comma tables", "SELECT * FROM a, b", "SELECT * FROM a, b"},
		{"where", "SELECT * FROM t WHERE a = 1", "SELECT * FROM t WHERE a = 1"},
		{"group by having", "SELECT dept, COUNT(*) FROM emp GROUP BY dept HAVING COUNT(*) > 5",
			"SELECT dept, COUNT(*) FROM emp GROUP BY dept HAVING COUNT(*) > 5"},
		{"order by limit offset", "SELECT * FROM t ORDER BY id DESC LIMIT 10 OFFSET 5",
			"SELECT * FROM t ORDER BY id DESC LIMIT 10 OFFSET 5"},
		{"order by asc default", "SELECT * FROM t ORDER BY id ASC", "SELECT * FROM t ORDER BY id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_joins(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"inner join on", "SELECT * FROM a JOIN b ON a.id = b.a_id", "SELECT * FROM a JOIN b ON a.id = b.a_id"},
		{"explicit inner", "SELECT * FROM a INNER JOIN b ON a.id = b.id", "SELECT * FROM a JOIN b ON a.id = b.id"},
		{"left join", "SELECT * FROM a LEFT JOIN b ON a.id = b.id", "SELECT * FROM a LEFT JOIN b ON a.id = b.id"},
		{"left outer join", "SELECT * FROM a LEFT OUTER JOIN b ON a.id = b.id", "SELECT * FROM a LEFT JOIN b ON a.id = b.id"},
		{"right join", "SELECT * FROM a RIGHT JOIN b ON a.id = b.id", "SELECT * FROM a RIGHT JOIN b ON a.id = b.id"},
		{"cross join", "SELECT * FROM a CROSS JOIN b", "SELECT * FROM a CROSS JOIN b"},
		{"natural join", "SELECT * FROM a NATURAL JOIN b", "SELECT * FROM a NATURAL JOIN b"},
		{"natural left join", "SELECT * FROM a NATURAL LEFT JOIN b", "SELECT * FROM a NATURAL LEFT JOIN b"},
		{"straight_join", "SELECT * FROM a STRAIGHT_JOIN b", "SELECT * FROM a STRAIGHT_JOIN b"},
		{"chained joins", "SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id",
			"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id"},
		{"parenthesized join", "SELECT * FROM (a JOIN b ON a.id = b.id)", "SELECT * FROM (a JOIN b ON a.id = b.id)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_indexHints(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"use index", "SELECT * FROM t USE INDEX (idx1)", "SELECT * FROM t USE INDEX (idx1)"},
		{"force index for join", "SELECT * FROM t FORCE INDEX FOR JOIN (idx1)", "SELECT * FROM t FORCE INDEX FOR JOIN (idx1)"},
		{"ignore index multi", "SELECT * FROM t IGNORE INDEX (idx1, idx2)", "SELECT * FROM t IGNORE INDEX (idx1, idx2)"},
		{"index for group by", "SELECT * FROM t USE INDEX FOR GROUP BY (idx1)", "SELECT * FROM t USE INDEX FOR GROUP BY (idx1)"},
		{"index for order by", "SELECT * FROM t USE INDEX FOR ORDER BY (idx1)", "SELECT * FROM t USE INDEX FOR ORDER BY (idx1)"},
		{"two hints", "SELECT * FROM t USE INDEX (a) IGNORE INDEX (b)", "SELECT * FROM t USE INDEX (a) IGNORE INDEX (b)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_derivedTable(t *testing.T) {
	assertSelectRoundTrip(t,
		"SELECT id FROM (SELECT id FROM t) AS sub",
		"SELECT id FROM (SELECT id FROM t) sub")
}

func TestParseSelect_subqueryInWhere(t *testing.T) {
	assertSelectRoundTrip(t,
		"SELECT id FROM t WHERE id IN (SELECT id FROM u)",
		"SELECT id FROM t WHERE id IN (SELECT id FROM u)")
}

func TestParseSelect_cte(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"simple", "WITH c AS (SELECT 1) SELECT * FROM c", "WITH c AS (SELECT 1) SELECT * FROM c"},
		{"recursive", "WITH RECURSIVE c AS (SELECT 1) SELECT * FROM c", "WITH RECURSIVE c AS (SELECT 1) SELECT * FROM c"},
		{"with columns", "WITH c (a, b) AS (SELECT 1, 2) SELECT * FROM c", "WITH c (a, b) AS (SELECT 1, 2) SELECT * FROM c"},
		{"multiple ctes", "WITH c1 AS (SELECT 1), c2 AS (SELECT 2) SELECT * FROM c1, c2",
			"WITH c1 AS (SELECT 1), c2 AS (SELECT 2) SELECT * FROM c1, c2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_locking(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"for update", "SELECT * FROM t FOR UPDATE", "SELECT * FROM t FOR UPDATE"},
		{"for share", "SELECT * FROM t FOR SHARE", "SELECT * FROM t FOR SHARE"},
		{"lock in share mode", "SELECT * FROM t LOCK IN SHARE MODE", "SELECT * FROM t LOCK IN SHARE MODE"},
		{"for update nowait", "SELECT * FROM t FOR UPDATE NOWAIT", "SELECT * FROM t FOR UPDATE NOWAIT"},
		{"for share skip locked", "SELECT * FROM t FOR SHARE SKIP LOCKED", "SELECT * FROM t FOR SHARE SKIP LOCKED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_structural(t *testing.T) {
	sel, err := parser.ParseSelect("SELECT id FROM users WHERE active = 1")
	if err != nil {
		t.Fatalf("ParseSelect error = %v", err)
	}

	want := &sqlast.Select{
		SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: col("id")}},
		From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}}},
		Where: &sqlast.Where{Expr: &sqlast.ComparisonExpr{
			Left: col("active"), Operator: sqlast.EqualOp, Right: num("1"),
		}},
	}

	if !reflect.DeepEqual(sel, want) {
		t.Errorf("ParseSelect =\n  %#v\nwant\n  %#v", sel, want)
	}
}

func TestParseSelect_errors(t *testing.T) {
	tests := []string{
		"",
		"SELECT",
		"SELECT * FROM",
		"SELECT * FROM t WHERE",
		"SELECT * FROM t JOIN",
		"SELECT * FROM t GROUP",
		"SELECT * FROM t GROUP BY",
		"SELECT * FROM t ORDER",
		"SELECT * FROM t LIMIT",
		"SELECT * FROM t USE",
		"SELECT * FROM t USE INDEX",
		"SELECT * FROM t USE INDEX FOR",
		"SELECT * FROM t USE INDEX FOR GROUP",
		"SELECT * FROM t USE INDEX (",
		"SELECT * FROM t FOR",
		"SELECT * FROM t FOR UPDATE SKIP",
		"SELECT * FROM t LOCK",
		"SELECT * FROM t LOCK IN",
		"WITH",
		"WITH c",
		"WITH c AS",
		"WITH c (a",
		"SELECT * FROM (",
		"SELECT * FROM (SELECT 1",
		"FROM t",
		"SELECT * FROM t extra tokens FROM u",
	}

	for _, in := range tests {
		if _, err := parser.ParseSelect(in); err == nil {
			t.Errorf("ParseSelect(%q) expected error, got nil", in)
		}
	}
}
