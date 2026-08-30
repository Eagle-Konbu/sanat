package pgparser_test

import (
	"reflect"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestParseError_Error(t *testing.T) {
	err := &pgparser.ParseError{Pos: pgparser.Position{Line: 3, Column: 7}, Msg: "expected RPAREN, got EOF"}

	want := "3:7: expected RPAREN, got EOF"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func parseExpr(t *testing.T, input string) pgast.Expr {
	t.Helper()

	e, err := pgparser.ParseExpr(input)
	if err != nil {
		t.Fatalf("ParseExpr(%q) error = %v", input, err)
	}

	return e
}

func assertExpr(t *testing.T, input string, want pgast.Expr) {
	t.Helper()

	got := parseExpr(t, input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseExpr(%q) =\n  %#v\nwant\n  %#v", input, got, want)
	}
}

func assertExprError(t *testing.T, input string) {
	t.Helper()

	if _, err := pgparser.ParseExpr(input); err == nil {
		t.Errorf("ParseExpr(%q) error = nil, want error", input)
	}
}

func col(name string) *pgast.ColName { return &pgast.ColName{Name: pgast.ColIdent(name)} }
func num(s string) *pgast.Literal    { return &pgast.Literal{Val: s} }

func TestParseExpr_literals(t *testing.T) {
	assertExpr(t, "123", num("123"))
	assertExpr(t, "1.5", num("1.5"))
	assertExpr(t, "NULL", &pgast.Literal{Val: "NULL"})
	assertExpr(t, "null", &pgast.Literal{Val: "NULL"})
	assertExpr(t, "TRUE", &pgast.Literal{Val: "TRUE"})
	assertExpr(t, "FALSE", &pgast.Literal{Val: "FALSE"})
	assertExpr(t, "$1", &pgast.Literal{Val: "$1"})
}

func TestParseExpr_stringLiteral(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", `'abc'`, `'abc'`},
		{"doubled quote escape", `'it''s'`, `'it''s'`},
		{"escape string re-encodes to plain", `E'a\nb'`, "'a\nb'"},
		{"escape string embedded quote", `E'it\'s'`, "'it''s'"},
		{"dollar-quoted re-encodes to plain", "$$it's a test$$", "'it''s a test'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, &pgast.Literal{Val: tt.want})
		})
	}
}

func TestParseExpr_arithmetic(t *testing.T) {
	assertExpr(t, "1 + 2", &pgast.ArithmeticExpr{Operator: pgast.PlusOp, Left: num("1"), Right: num("2")})
	assertExpr(t, "1 - 2", &pgast.ArithmeticExpr{Operator: pgast.MinusOp, Left: num("1"), Right: num("2")})
	assertExpr(t, "1 * 2", &pgast.ArithmeticExpr{Operator: pgast.MultOp, Left: num("1"), Right: num("2")})
	assertExpr(t, "1 / 2", &pgast.ArithmeticExpr{Operator: pgast.DivOp, Left: num("1"), Right: num("2")})
	assertExpr(t, "1 % 2", &pgast.ArithmeticExpr{Operator: pgast.ModOp, Left: num("1"), Right: num("2")})

	t.Run("precedence: * binds tighter than +", func(t *testing.T) {
		assertExpr(t, "1 + 2 * 3", &pgast.ArithmeticExpr{
			Operator: pgast.PlusOp,
			Left:     num("1"),
			Right:    &pgast.ArithmeticExpr{Operator: pgast.MultOp, Left: num("2"), Right: num("3")},
		})
	})

	t.Run("left associative", func(t *testing.T) {
		assertExpr(t, "1 - 2 - 3", &pgast.ArithmeticExpr{
			Operator: pgast.MinusOp,
			Left:     &pgast.ArithmeticExpr{Operator: pgast.MinusOp, Left: num("1"), Right: num("2")},
			Right:    num("3"),
		})
	})
}

func TestParseExpr_unary(t *testing.T) {
	assertExpr(t, "-1", &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: num("1")})
	assertExpr(t, "+1", &pgast.UnaryExpr{Operator: pgast.UPlusOp, Expr: num("1")})
	// "--1" is not a double negative here: "--" is PostgreSQL's line-comment
	// marker (no boundary character required after it, unlike MySQL), so a
	// space is needed to get two separate MINUS tokens.
	assertExpr(t, "- -1", &pgast.UnaryExpr{
		Operator: pgast.UMinusOp,
		Expr:     &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: num("1")},
	})
}

func TestParseExpr_comparison(t *testing.T) {
	tests := []struct {
		in string
		op pgast.ComparisonOperator
	}{
		{"a = b", pgast.EqualOp},
		{"a <> b", pgast.NotEqualOp},
		{"a != b", pgast.NotEqualOp},
		{"a < b", pgast.LessThanOp},
		{"a > b", pgast.GreaterThanOp},
		{"a <= b", pgast.LessEqualOp},
		{"a >= b", pgast.GreaterEqualOp},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assertExpr(t, tt.in, &pgast.ComparisonExpr{Operator: tt.op, Left: col("a"), Right: col("b")})
		})
	}
}

func TestParseExpr_logical(t *testing.T) {
	assertExpr(t, "a AND b", &pgast.AndExpr{Left: col("a"), Right: col("b")})
	assertExpr(t, "a OR b", &pgast.OrExpr{Left: col("a"), Right: col("b")})
	assertExpr(t, "NOT a", &pgast.NotExpr{Expr: col("a")})

	t.Run("AND binds tighter than OR", func(t *testing.T) {
		assertExpr(t, "a OR b AND c", &pgast.OrExpr{
			Left:  col("a"),
			Right: &pgast.AndExpr{Left: col("b"), Right: col("c")},
		})
	})
}

func TestParseExpr_between(t *testing.T) {
	assertExpr(t, "a BETWEEN 1 AND 10", &pgast.RangeCond{Left: col("a"), From: num("1"), To: num("10")})
	assertExpr(t, "a NOT BETWEEN 1 AND 10", &pgast.RangeCond{Not: true, Left: col("a"), From: num("1"), To: num("10")})
}

func TestParseExpr_in(t *testing.T) {
	t.Run("value list", func(t *testing.T) {
		assertExpr(t, "a IN (1, 2)", &pgast.ComparisonExpr{
			Left: col("a"), Operator: pgast.InOp, Right: pgast.ValTuple{num("1"), num("2")},
		})
	})

	t.Run("negated", func(t *testing.T) {
		assertExpr(t, "a NOT IN (1, 2)", &pgast.ComparisonExpr{
			Left: col("a"), Operator: pgast.NotInOp, Right: pgast.ValTuple{num("1"), num("2")},
		})
	})

	t.Run("subquery", func(t *testing.T) {
		assertExpr(t, "a IN (SELECT id FROM t)", &pgast.ComparisonExpr{
			Left:     col("a"),
			Operator: pgast.InOp,
			Right: &pgast.Subquery{Select: &pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("id")}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}},
			}},
		})
	})
}

func TestParseExpr_like(t *testing.T) {
	assertExpr(t, "a LIKE 'x%'", &pgast.ComparisonExpr{Left: col("a"), Operator: pgast.LikeOp, Right: num("'x%'")})
	assertExpr(t, "a NOT LIKE 'x%'", &pgast.ComparisonExpr{Left: col("a"), Operator: pgast.NotLikeOp, Right: num("'x%'")})
	assertExpr(t, "a ILIKE 'x%'", &pgast.ILikeExpr{Left: col("a"), Right: num("'x%'")})
	assertExpr(t, "a NOT ILIKE 'x%'", &pgast.ILikeExpr{Not: true, Left: col("a"), Right: num("'x%'")})
}

func TestParseExpr_is(t *testing.T) {
	assertExpr(t, "a IS NULL", &pgast.IsExpr{Expr: col("a")})
	assertExpr(t, "a IS NOT NULL", &pgast.IsExpr{Not: true, Expr: col("a")})
	assertExpr(t, "a ISNULL", &pgast.IsExpr{Expr: col("a")})
	assertExpr(t, "a NOTNULL", &pgast.IsExpr{Not: true, Expr: col("a")})
	assertExpr(t, "a IS DISTINCT FROM b", &pgast.IsDistinctFromExpr{Left: col("a"), Right: col("b")})
	assertExpr(t, "a IS NOT DISTINCT FROM b", &pgast.IsDistinctFromExpr{Not: true, Left: col("a"), Right: col("b")})
}

func TestParseExpr_case(t *testing.T) {
	t.Run("searched, no else", func(t *testing.T) {
		assertExpr(t, "CASE WHEN a THEN 1 END", &pgast.CaseExpr{
			Whens: []*pgast.When{{Cond: col("a"), Val: num("1")}},
		})
	})

	t.Run("simple with else", func(t *testing.T) {
		assertExpr(t, "CASE a WHEN 1 THEN 'one' ELSE 'other' END", &pgast.CaseExpr{
			Expr:  col("a"),
			Whens: []*pgast.When{{Cond: num("1"), Val: num("'one'")}},
			Else:  num("'other'"),
		})
	})
}

func TestParseExpr_exists(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgast.Expr
	}{
		{
			"basic",
			"EXISTS (SELECT * FROM t)",
			&pgast.ExistsExpr{Subquery: &pgast.Subquery{Select: &pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}},
			}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, tt.want)
		})
	}
}

func TestParseExpr_paren(t *testing.T) {
	assertExpr(t, "(a)", &pgast.ParenExpr{Expr: col("a")})

	t.Run("scalar subquery", func(t *testing.T) {
		assertExpr(t, "(SELECT 1)", &pgast.Subquery{Select: &pgast.Select{
			SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: num("1")}},
		}})
	})
}

func TestParseExpr_funcCall(t *testing.T) {
	t.Run("plain", func(t *testing.T) {
		assertExpr(t, "lower(a)", &pgast.FuncExpr{Name: "lower", Args: []pgast.Expr{col("a")}})
	})

	t.Run("qualified", func(t *testing.T) {
		assertExpr(t, "pg_catalog.lower(a)", &pgast.FuncExpr{Qualifier: "pg_catalog", Name: "lower", Args: []pgast.Expr{col("a")}})
	})

	t.Run("no args", func(t *testing.T) {
		assertExpr(t, "now()", &pgast.FuncExpr{Name: "now"})
	})

	t.Run("star", func(t *testing.T) {
		assertExpr(t, "count(*)", &pgast.FuncExpr{Name: "count", Star: true})
	})

	t.Run("distinct", func(t *testing.T) {
		assertExpr(t, "count(DISTINCT a)", &pgast.FuncExpr{Name: "count", Distinct: true, Args: []pgast.Expr{col("a")}})
	})

	t.Run("filter", func(t *testing.T) {
		assertExpr(t, "count(*) FILTER (WHERE a)", &pgast.FuncExpr{
			Name: "count", Star: true, Filter: &pgast.Where{Expr: col("a")},
		})
	})
}

func TestParseExpr_cast(t *testing.T) {
	t.Run("shorthand", func(t *testing.T) {
		assertExpr(t, "a::int", &pgast.CastExpr{Expr: col("a"), Type: "int"})
	})

	t.Run("explicit", func(t *testing.T) {
		assertExpr(t, "CAST(a AS int)", &pgast.CastExpr{Expr: col("a"), Type: "int", Explicit: true})
	})

	t.Run("chained", func(t *testing.T) {
		assertExpr(t, "a::text::int", &pgast.CastExpr{
			Expr: &pgast.CastExpr{Expr: col("a"), Type: "text"},
			Type: "int",
		})
	})

	t.Run("type with precision modifier", func(t *testing.T) {
		assertExpr(t, "a::numeric(10, 2)", &pgast.CastExpr{Expr: col("a"), Type: "numeric(10, 2)"})
	})

	t.Run("type with single modifier", func(t *testing.T) {
		assertExpr(t, "a::varchar(255)", &pgast.CastExpr{Expr: col("a"), Type: "varchar(255)"})
	})

	t.Run("array type, unsized", func(t *testing.T) {
		assertExpr(t, "a::int[]", &pgast.CastExpr{Expr: col("a"), Type: "int[]"})
	})
}

func TestParseExpr_array(t *testing.T) {
	t.Run("literal", func(t *testing.T) {
		assertExpr(t, "ARRAY[1, 2]", &pgast.ArrayExpr{Elems: []pgast.Expr{num("1"), num("2")}})
	})

	t.Run("empty literal", func(t *testing.T) {
		assertExpr(t, "ARRAY[]", &pgast.ArrayExpr{})
	})

	t.Run("subscript", func(t *testing.T) {
		assertExpr(t, "a[1]", &pgast.ArraySubscriptExpr{Expr: col("a"), From: num("1")})
	})

	t.Run("slice", func(t *testing.T) {
		assertExpr(t, "a[1:3]", &pgast.ArraySubscriptExpr{Expr: col("a"), From: num("1"), To: num("3")})
	})

	t.Run("open-ended slice", func(t *testing.T) {
		assertExpr(t, "a[:3]", &pgast.ArraySubscriptExpr{Expr: col("a"), To: num("3")})
		assertExpr(t, "a[1:]", &pgast.ArraySubscriptExpr{Expr: col("a"), From: num("1")})
	})
}

func TestParseExpr_jsonOperators(t *testing.T) {
	tests := []struct {
		in    string
		op    pgast.JSONOp
		right pgast.Expr
	}{
		{"a -> 'k'", pgast.JSONArrow, num("'k'")},
		{"a ->> 'k'", pgast.JSONArrowText, num("'k'")},
		{"a #> 'k'", pgast.JSONHashArrow, num("'k'")},
		{"a #>> 'k'", pgast.JSONHashArrowText, num("'k'")},
		{"a @> b", pgast.JSONContains, col("b")},
		{"a <@ b", pgast.JSONContainedBy, col("b")},
		{"a ? 'k'", pgast.JSONExists, num("'k'")},
		{"a ?| b", pgast.JSONExistsAny, col("b")},
		{"a ?& b", pgast.JSONExistsAll, col("b")},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assertExpr(t, tt.in, &pgast.JSONOpExpr{Left: col("a"), Op: tt.op, Right: tt.right})
		})
	}
}

func TestParseExpr_precedenceInteraction(t *testing.T) {
	t.Run("subscript is part of the cast's type, not applied after it", func(t *testing.T) {
		assertExpr(t, "a::int[1]", &pgast.CastExpr{Expr: col("a"), Type: "int[1]"})
	})

	t.Run("explicit subscript of a parenthesized cast", func(t *testing.T) {
		assertExpr(t, "(a::int)[1]", &pgast.ArraySubscriptExpr{
			Expr: &pgast.ParenExpr{Expr: &pgast.CastExpr{Expr: col("a"), Type: "int"}},
			From: num("1"),
		})
	})

	t.Run("cast binds tighter than a JSON operator", func(t *testing.T) {
		assertExpr(t, "a @> b::jsonb", &pgast.JSONOpExpr{
			Left:  col("a"),
			Op:    pgast.JSONContains,
			Right: &pgast.CastExpr{Expr: col("b"), Type: "jsonb"},
		})
	})

	t.Run("JSON operator binds tighter than comparison", func(t *testing.T) {
		assertExpr(t, "a -> 'k' = b", &pgast.ComparisonExpr{
			Left:     &pgast.JSONOpExpr{Left: col("a"), Op: pgast.JSONArrow, Right: num("'k'")},
			Operator: pgast.EqualOp,
			Right:    col("b"),
		})
	})
}

func TestParseExpr_errors(t *testing.T) {
	tests := []string{
		"",
		"(",
		"a +",
		"a::",
		"CASE END",
		"a IN (",
		"1 1",
		"^",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			assertExprError(t, tt)
		})
	}
}
