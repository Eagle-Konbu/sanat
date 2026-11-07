package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestParseSelect_fromItems(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"function", "SELECT g FROM generate_series(1, 3) g", "SELECT g FROM generate_series(1, 3) g"},
		{"function with column aliases", "SELECT * FROM unnest(a, b) AS u(x, y)", "SELECT * FROM unnest(a, b) u(x, y)"},
		{"qualified function", "SELECT * FROM pg_catalog.generate_series(1, 2)", "SELECT * FROM pg_catalog.generate_series(1, 2)"},
		{"lateral function", "SELECT * FROM t, LATERAL f(t.a) x", "SELECT * FROM t, LATERAL f(t.a) x"},
		{"with ordinality", "SELECT * FROM unnest(arr) WITH ORDINALITY AS u(v, n)", "SELECT * FROM unnest(arr) WITH ORDINALITY u(v, n)"},
		{"rows from", "SELECT * FROM ROWS FROM (f(1), g(2)) WITH ORDINALITY r", "SELECT * FROM ROWS FROM (f(1), g(2)) WITH ORDINALITY r"},
		{"lateral rows from", "SELECT * FROM t, LATERAL ROWS FROM (f(t.a))", "SELECT * FROM t, LATERAL ROWS FROM (f(t.a))"},
		{"table column aliases", "SELECT * FROM t AS x(a, b)", "SELECT * FROM t x(a, b)"},
		{"derived table column aliases", "SELECT * FROM (SELECT 1, 2) AS s(a, b)", "SELECT * FROM (SELECT 1, 2) s(a, b)"},
		{"lateral derived table column aliases", "SELECT * FROM t, LATERAL (SELECT 1) s(a)", "SELECT * FROM t, LATERAL (SELECT 1) s(a)"},
		{"parenthesized join alias", "SELECT * FROM (a JOIN b USING (id)) AS j(x)", "SELECT * FROM (a JOIN b USING (id)) j(x)"},
		{"tablesample", "SELECT * FROM t TABLESAMPLE bernoulli (10)", "SELECT * FROM t TABLESAMPLE bernoulli (10)"},
		{
			"tablesample after alias with repeatable",
			"SELECT * FROM t AS x TABLESAMPLE system (5.5) REPEATABLE (42)",
			"SELECT * FROM t x TABLESAMPLE system (5.5) REPEATABLE (42)",
		},
		{"only", "SELECT * FROM ONLY t", "SELECT * FROM ONLY t"},
		{"only parenthesized", "SELECT * FROM ONLY (s.t) x", "SELECT * FROM ONLY s.t x"},
		{"descendants", "SELECT * FROM t * x", "SELECT * FROM t * x"},
		{"nested joins", "SELECT * FROM a JOIN b JOIN c ON b.id = c.id ON a.id = b.id", "SELECT * FROM a JOIN b JOIN c ON b.id = c.id ON a.id = b.id"},
		{"nested join after cross join", "SELECT * FROM a JOIN b CROSS JOIN c ON TRUE", "SELECT * FROM a JOIN b CROSS JOIN c ON TRUE"},
		{"left-deep joins", "SELECT * FROM a JOIN b ON x JOIN c ON y", "SELECT * FROM a JOIN b ON x JOIN c ON y"},
		{"join using alias", "SELECT * FROM a JOIN b USING (id) AS j", "SELECT * FROM a JOIN b USING (id) AS j"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{
		"SELECT * FROM t AS tablesample",
		"SELECT * FROM t TABLESAMPLE",
		"SELECT * FROM t TABLESAMPLE bernoulli",
		"SELECT * FROM ROWS FROM (t)",
		"SELECT * FROM a JOIN b JOIN c ON x",
		"SELECT * FROM a JOIN b USING (id) AS",
		"SELECT * FROM t x(",
	} {
		t.Run("error "+in, func(t *testing.T) {
			assertSelectError(t, in)
		})
	}
}

func TestParseSelect_nestedJoinStructure(t *testing.T) {
	sel, err := pgparser.ParseSelect("SELECT * FROM a JOIN b JOIN c ON y ON x")
	if err != nil {
		t.Fatal(err)
	}

	outer, ok := sel.From[0].(*pgast.JoinTableExpr)
	if !ok {
		t.Fatalf("From[0] = %#v, want *pgast.JoinTableExpr", sel.From[0])
	}

	if _, ok := outer.Right.(*pgast.JoinTableExpr); !ok {
		t.Errorf("outer.Right = %#v, want the inner join", outer.Right)
	}

	if got := outer.Condition.On.String(); got != "x" {
		t.Errorf("outer condition = %q, want x", got)
	}
}
