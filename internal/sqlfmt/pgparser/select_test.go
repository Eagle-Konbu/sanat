package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertSelectRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want. This is enough
// for grammar breadth (clause presence/order/keywords) but not for
// expression precedence, which internal/sqlfmt/pgparser's own expr_test.go
// covers via structural checks.
func assertSelectRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	sel, err := pgparser.ParseSelect(input)
	if err != nil {
		t.Fatalf("ParseSelect(%q) error = %v", input, err)
	}

	if got := sel.String(); got != want {
		t.Errorf("ParseSelect(%q).String() = %q, want %q", input, got, want)
	}
}

func assertSelectError(t *testing.T, input string) {
	t.Helper()

	if _, err := pgparser.ParseSelect(input); err == nil {
		t.Errorf("ParseSelect(%q) error = nil, want error", input)
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
		{"distinct on", "SELECT DISTINCT ON (a, b) * FROM t", "SELECT DISTINCT ON (a, b) * FROM t"},
		{"select all is a no-op", "SELECT ALL id FROM t", "SELECT id FROM t"},
		{"explicit alias", "SELECT id AS user_id FROM t", "SELECT id AS user_id FROM t"},
		{"implicit alias", "SELECT id user_id FROM t", "SELECT id AS user_id FROM t"},
		{"table alias", "SELECT * FROM users AS u", "SELECT * FROM users u"},
		{"implicit table alias", "SELECT * FROM users u", "SELECT * FROM users u"},
		{"qualified table", "SELECT * FROM public.users", "SELECT * FROM public.users"},
		{"comma tables", "SELECT * FROM a, b", "SELECT * FROM a, b"},
		{"where", "SELECT * FROM t WHERE a = 1", "SELECT * FROM t WHERE a = 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_groupBy(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"group by having",
			"SELECT dept, count(*) FROM emp GROUP BY dept HAVING count(*) > 5",
			"SELECT dept, count(*) FROM emp GROUP BY dept HAVING count(*) > 5",
		},
		{
			"grouping sets",
			"SELECT dept FROM emp GROUP BY GROUPING SETS ((a, b), (a), ())",
			"SELECT dept FROM emp GROUP BY GROUPING SETS ((a, b), (a), ())",
		},
		{
			"cube",
			"SELECT dept FROM emp GROUP BY CUBE (a, b)",
			"SELECT dept FROM emp GROUP BY CUBE (a, b)",
		},
		{
			"rollup",
			"SELECT dept FROM emp GROUP BY ROLLUP (a, b)",
			"SELECT dept FROM emp GROUP BY ROLLUP (a, b)",
		},
		{
			"mixed with plain expr",
			"SELECT dept FROM emp GROUP BY dept, ROLLUP (a)",
			"SELECT dept FROM emp GROUP BY dept, ROLLUP (a)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_orderBy(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"asc default", "SELECT * FROM t ORDER BY id ASC", "SELECT * FROM t ORDER BY id"},
		{"desc", "SELECT * FROM t ORDER BY id DESC", "SELECT * FROM t ORDER BY id DESC"},
		{"nulls first", "SELECT * FROM t ORDER BY id NULLS FIRST", "SELECT * FROM t ORDER BY id NULLS FIRST"},
		{
			"desc nulls last",
			"SELECT * FROM t ORDER BY id DESC NULLS LAST",
			"SELECT * FROM t ORDER BY id DESC NULLS LAST",
		},
		{"multi-column", "SELECT * FROM t ORDER BY a, b DESC", "SELECT * FROM t ORDER BY a, b DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_limitOffsetFetch(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"limit", "SELECT * FROM t LIMIT 10", "SELECT * FROM t LIMIT 10"},
		{"limit all", "SELECT * FROM t LIMIT ALL", "SELECT * FROM t LIMIT ALL"},
		{"offset", "SELECT * FROM t OFFSET 5", "SELECT * FROM t OFFSET 5"},
		{"offset with row noise word", "SELECT * FROM t OFFSET 5 ROW", "SELECT * FROM t OFFSET 5"},
		{"offset with rows noise word", "SELECT * FROM t OFFSET 5 ROWS", "SELECT * FROM t OFFSET 5"},
		{"limit offset", "SELECT * FROM t LIMIT 10 OFFSET 5", "SELECT * FROM t LIMIT 10 OFFSET 5"},
		{
			"fetch first row only",
			"SELECT * FROM t FETCH FIRST ROW ONLY",
			"SELECT * FROM t FETCH FIRST ROW ONLY",
		},
		{
			"fetch next n rows with ties",
			"SELECT * FROM t FETCH NEXT 5 ROWS WITH TIES",
			"SELECT * FROM t FETCH NEXT 5 ROWS WITH TIES",
		},
		{
			// pgast.Fetch doesn't track singular/plural ROW vs. ROWS
			// separately from WithTies (they're pure synonyms in PostgreSQL)
			// — it always renders "ROW ONLY" / "ROWS WITH TIES", the same
			// kind of canonicalization sqlast already applies to keyword
			// casing elsewhere.
			"offset and fetch together, ROWS canonicalizes to ROW with ONLY",
			"SELECT * FROM t OFFSET 5 FETCH FIRST 10 ROWS ONLY",
			"SELECT * FROM t OFFSET 5 FETCH FIRST 10 ROW ONLY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_lock(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"for update", "SELECT * FROM t FOR UPDATE", "SELECT * FROM t FOR UPDATE"},
		{"for no key update", "SELECT * FROM t FOR NO KEY UPDATE", "SELECT * FROM t FOR NO KEY UPDATE"},
		{"for share", "SELECT * FROM t FOR SHARE", "SELECT * FROM t FOR SHARE"},
		{"for key share", "SELECT * FROM t FOR KEY SHARE", "SELECT * FROM t FOR KEY SHARE"},
		{"for update of", "SELECT * FROM t FOR UPDATE OF t", "SELECT * FROM t FOR UPDATE OF t"},
		{
			"for update of multiple tables",
			"SELECT * FROM t, u FOR UPDATE OF t, u",
			"SELECT * FROM t, u FOR UPDATE OF t, u",
		},
		{"for update nowait", "SELECT * FROM t FOR UPDATE NOWAIT", "SELECT * FROM t FOR UPDATE NOWAIT"},
		{
			"for update skip locked",
			"SELECT * FROM t FOR UPDATE SKIP LOCKED",
			"SELECT * FROM t FOR UPDATE SKIP LOCKED",
		},
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
		{"inner join with on", "SELECT * FROM a JOIN b ON a.id = b.id", "SELECT * FROM a JOIN b ON a.id = b.id"},
		{
			"explicit inner join",
			"SELECT * FROM a INNER JOIN b ON a.id = b.id",
			"SELECT * FROM a JOIN b ON a.id = b.id",
		},
		{"left join", "SELECT * FROM a LEFT JOIN b ON a.id = b.id", "SELECT * FROM a LEFT JOIN b ON a.id = b.id"},
		{
			"left outer join",
			"SELECT * FROM a LEFT OUTER JOIN b ON a.id = b.id",
			"SELECT * FROM a LEFT JOIN b ON a.id = b.id",
		},
		{"right join", "SELECT * FROM a RIGHT JOIN b ON a.id = b.id", "SELECT * FROM a RIGHT JOIN b ON a.id = b.id"},
		{"full join", "SELECT * FROM a FULL JOIN b ON a.id = b.id", "SELECT * FROM a FULL JOIN b ON a.id = b.id"},
		{"cross join", "SELECT * FROM a CROSS JOIN b", "SELECT * FROM a CROSS JOIN b"},
		{"natural join", "SELECT * FROM a NATURAL JOIN b", "SELECT * FROM a NATURAL JOIN b"},
		{"natural left join", "SELECT * FROM a NATURAL LEFT JOIN b", "SELECT * FROM a NATURAL LEFT JOIN b"},
		{"natural right join", "SELECT * FROM a NATURAL RIGHT JOIN b", "SELECT * FROM a NATURAL RIGHT JOIN b"},
		{"natural full join", "SELECT * FROM a NATURAL FULL JOIN b", "SELECT * FROM a NATURAL FULL JOIN b"},
		{"using", "SELECT * FROM a JOIN b USING (id)", "SELECT * FROM a JOIN b USING (id)"},
		{
			"chained joins",
			"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id",
			"SELECT * FROM a JOIN b ON a.id = b.id JOIN c ON b.id = c.id",
		},
		{
			"parenthesized join",
			"SELECT * FROM (a JOIN b ON a.id = b.id)",
			"SELECT * FROM (a JOIN b ON a.id = b.id)",
		},
		{
			"derived table",
			"SELECT * FROM (SELECT id FROM t) AS sub",
			"SELECT * FROM (SELECT id FROM t) sub",
		},
		{
			"lateral derived table",
			"SELECT * FROM a, LATERAL (SELECT id FROM t) AS sub",
			"SELECT * FROM a, LATERAL (SELECT id FROM t) sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_window(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"top-level window clause",
			"SELECT row_number() OVER w FROM t WINDOW w AS (PARTITION BY dept ORDER BY salary)",
			"SELECT row_number() OVER w FROM t WINDOW w AS (PARTITION BY dept ORDER BY salary)",
		},
		{
			"over with parens",
			"SELECT row_number() OVER (PARTITION BY dept ORDER BY salary) FROM t",
			"SELECT row_number() OVER (PARTITION BY dept ORDER BY salary) FROM t",
		},
		{
			"over with frame clause",
			"SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM t",
			"SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) FROM t",
		},
		{
			"frame with single expr-preceding point, no BETWEEN",
			"SELECT sum(x) OVER (ORDER BY id RANGE 3 PRECEDING) FROM t",
			"SELECT sum(x) OVER (ORDER BY id RANGE 3 PRECEDING) FROM t",
		},
		{
			"frame with expr FOLLOWING bounds",
			"SELECT sum(x) OVER (ORDER BY id GROUPS BETWEEN 1 PRECEDING AND 2 FOLLOWING) FROM t",
			"SELECT sum(x) OVER (ORDER BY id GROUPS BETWEEN 1 PRECEDING AND 2 FOLLOWING) FROM t",
		},
		{
			"frame with unbounded following bound",
			"SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING) FROM t",
			"SELECT sum(x) OVER (ORDER BY id ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING) FROM t",
		},
		{
			"window refining a named window",
			"SELECT row_number() OVER (w ORDER BY id) FROM t WINDOW w AS (PARTITION BY dept)",
			"SELECT row_number() OVER (w ORDER BY id) FROM t WINDOW w AS (PARTITION BY dept)",
		},
		{
			"multiple named windows",
			"SELECT 1 FROM t WINDOW w1 AS (PARTITION BY a), w2 AS (PARTITION BY b)",
			"SELECT 1 FROM t WINDOW w1 AS (PARTITION BY a), w2 AS (PARTITION BY b)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseSelect_errors(t *testing.T) {
	tests := []string{
		"",
		"SELECT",
		"SELECT * FROM",
		"SELECT * FROM a LEFT JOIN b",
		"SELECT * FROM a NATURAL JOIN b ON a.id = b.id",
		"SELECT * FOR UPDATE OF",
		"SELECT * FROM t FETCH FIRST",
		"SELECT * FROM t GROUP BY GROUPING SETS",
		"SELECT * FROM t; garbage",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			assertSelectError(t, tt)
		})
	}
}
