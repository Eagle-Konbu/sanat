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

	t.Run("exponentiation", func(t *testing.T) {
		assertExpr(t, "2 ^ 3", &pgast.ArithmeticExpr{Operator: pgast.ExpOp, Left: num("2"), Right: num("3")})
	})

	t.Run("exponentiation is left associative", func(t *testing.T) {
		assertExpr(t, "2 ^ 3 ^ 2", &pgast.ArithmeticExpr{
			Operator: pgast.ExpOp,
			Left:     &pgast.ArithmeticExpr{Operator: pgast.ExpOp, Left: num("2"), Right: num("3")},
			Right:    num("2"),
		})
	})

	t.Run("exponentiation binds tighter than *", func(t *testing.T) {
		assertExpr(t, "2 * 3 ^ 2", &pgast.ArithmeticExpr{
			Operator: pgast.MultOp,
			Left:     num("2"),
			Right:    &pgast.ArithmeticExpr{Operator: pgast.ExpOp, Left: num("3"), Right: num("2")},
		})
	})

	t.Run("unary minus binds tighter than exponentiation", func(t *testing.T) {
		assertExpr(t, "-2 ^ 2", &pgast.ArithmeticExpr{
			Operator: pgast.ExpOp,
			Left:     &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: num("2")},
			Right:    num("2"),
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
	assertExpr(t, "a LIKE 'x!%' ESCAPE '!'", &pgast.ComparisonExpr{
		Left: col("a"), Operator: pgast.LikeOp, Right: num("'x!%'"), Escape: num("'!'"),
	})
	assertExpr(t, "a NOT ILIKE 'x!%' ESCAPE '!'", &pgast.ILikeExpr{
		Not: true, Left: col("a"), Right: num("'x!%'"), Escape: num("'!'"),
	})
}

func TestParseExpr_is(t *testing.T) {
	assertExpr(t, "a IS NULL", &pgast.IsExpr{Expr: col("a")})
	assertExpr(t, "a IS NOT NULL", &pgast.IsExpr{Not: true, Expr: col("a")})
	assertExpr(t, "a ISNULL", &pgast.IsExpr{Expr: col("a")})
	assertExpr(t, "a NOTNULL", &pgast.IsExpr{Not: true, Expr: col("a")})
	assertExpr(t, "a IS DISTINCT FROM b", &pgast.IsDistinctFromExpr{Left: col("a"), Right: col("b")})
	assertExpr(t, "a IS NOT DISTINCT FROM b", &pgast.IsDistinctFromExpr{Not: true, Left: col("a"), Right: col("b")})
	assertExpr(t, "a IS TRUE", &pgast.IsTruthExpr{Expr: col("a"), Value: pgast.TruthTrue})
	assertExpr(t, "a IS NOT FALSE", &pgast.IsTruthExpr{Not: true, Expr: col("a"), Value: pgast.TruthFalse})
	assertExpr(t, "a IS UNKNOWN", &pgast.IsTruthExpr{Expr: col("a"), Value: pgast.TruthUnknown})
	assertExpr(t, "a IS NOT unknown", &pgast.IsTruthExpr{Not: true, Expr: col("a"), Value: pgast.TruthUnknown})
}

func TestParseExpr_predicatePrecedence(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgast.Expr
	}{
		{
			"IS binds looser than comparison",
			"a = b IS NULL",
			&pgast.IsExpr{Expr: &pgast.ComparisonExpr{Left: col("a"), Operator: pgast.EqualOp, Right: col("b")}},
		},
		{
			"chained IS tests",
			"a IS NULL IS NOT TRUE",
			&pgast.IsTruthExpr{Not: true, Expr: &pgast.IsExpr{Expr: col("a")}, Value: pgast.TruthTrue},
		},
		{
			"IN binds tighter than comparison",
			"a IN (1) = b",
			&pgast.ComparisonExpr{
				Left:     &pgast.ComparisonExpr{Left: col("a"), Operator: pgast.InOp, Right: pgast.ValTuple{num("1")}},
				Operator: pgast.EqualOp,
				Right:    col("b"),
			},
		},
		{
			"LIKE on the right of a comparison",
			"TRUE = a LIKE 'x'",
			&pgast.ComparisonExpr{
				Left:     &pgast.Literal{Val: "TRUE"},
				Operator: pgast.EqualOp,
				Right:    &pgast.ComparisonExpr{Left: col("a"), Operator: pgast.LikeOp, Right: num("'x'")},
			},
		},
		{
			"BETWEEN then comparison then IS",
			"a BETWEEN 1 AND 2 = b IS NOT NULL",
			&pgast.IsExpr{Not: true, Expr: &pgast.ComparisonExpr{
				Left:     &pgast.RangeCond{Left: col("a"), From: num("1"), To: num("2")},
				Operator: pgast.EqualOp,
				Right:    col("b"),
			}},
		},
		{
			"IS DISTINCT FROM right operand includes a comparison",
			"a IS DISTINCT FROM b = c",
			&pgast.IsDistinctFromExpr{
				Left:  col("a"),
				Right: &pgast.ComparisonExpr{Left: col("b"), Operator: pgast.EqualOp, Right: col("c")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, tt.want)
		})
	}
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
	exists := func(sel *pgast.Select) pgast.Expr {
		return &pgast.ExistsExpr{Subquery: &pgast.Subquery{Select: sel}}
	}
	tableT := []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}}

	tests := []struct {
		name string
		in   string
		want pgast.Expr
	}{
		{
			"basic",
			"EXISTS (SELECT * FROM t)",
			exists(&pgast.Select{SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{}}, From: tableT}),
		},
		{
			"qualified star",
			"EXISTS (SELECT t.* FROM t)",
			exists(&pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.StarExpr{TableName: pgast.TableName{Name: "t"}}},
				From:        tableT,
			}),
		},
		{
			"explicit aliases",
			"EXISTS (SELECT a AS x FROM t AS u)",
			exists(&pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("a"), As: "x"}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}, As: "u"}},
			}),
		},
		{
			"bare aliases",
			"EXISTS (SELECT a x FROM t u)",
			exists(&pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("a"), As: "x"}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}, As: "u"}},
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, tt.want)
		})
	}
}

func TestParseExpr_existsErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"AS without select alias", "EXISTS (SELECT a AS FROM t)"},
		{"AS without table alias", "EXISTS (SELECT a FROM t AS)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExprError(t, tt.in)
		})
	}
}

func TestParseExpr_paren(t *testing.T) {
	assertExpr(t, "(a)", &pgast.ParenExpr{Expr: col("a")})

	t.Run("row constructor", func(t *testing.T) {
		assertExpr(t, "(a, b)", pgast.ValTuple{col("a"), col("b")})
	})

	t.Run("explicit ROW constructor", func(t *testing.T) {
		assertExpr(t, "ROW(a, 1)", &pgast.RowExpr{Exprs: []pgast.Expr{col("a"), num("1")}})
		assertExpr(t, "ROW()", &pgast.RowExpr{})
	})

	t.Run("row constructor IN subquery", func(t *testing.T) {
		assertExpr(t, "(a, b) IN (SELECT a, b FROM t)", &pgast.ComparisonExpr{
			Left:     pgast.ValTuple{col("a"), col("b")},
			Operator: pgast.InOp,
			Right: &pgast.Subquery{Select: &pgast.Select{
				SelectExprs: []pgast.SelectExpr{&pgast.AliasedExpr{Expr: col("a")}, &pgast.AliasedExpr{Expr: col("b")}},
				From:        []pgast.TableExpr{&pgast.AliasedTableExpr{Expr: pgast.TableName{Name: "t"}}},
			}},
		})
	})

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

func TestParseExpr_funcCallOver(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgast.Expr
	}{
		{
			"empty window",
			"row_number() OVER ()",
			&pgast.FuncExpr{Name: "row_number", Over: &pgast.WindowSpec{}},
		},
		{
			"partition by",
			"sum(a) OVER (PARTITION BY b, c)",
			&pgast.FuncExpr{
				Name: "sum", Args: []pgast.Expr{col("a")},
				Over: &pgast.WindowSpec{PartitionBy: []pgast.Expr{col("b"), col("c")}},
			},
		},
		{
			"order by",
			"row_number() OVER (ORDER BY a DESC NULLS LAST, b)",
			&pgast.FuncExpr{Name: "row_number", Over: &pgast.WindowSpec{OrderBy: pgast.OrderBy{
				{Expr: col("a"), Direction: pgast.DescOrder, Nulls: pgast.NullsLast},
				{Expr: col("b")},
			}}},
		},
		{
			"partition by and order by",
			"rank() OVER (PARTITION BY a ORDER BY b ASC NULLS FIRST)",
			&pgast.FuncExpr{Name: "rank", Over: &pgast.WindowSpec{
				PartitionBy: []pgast.Expr{col("a")},
				OrderBy:     pgast.OrderBy{{Expr: col("b"), Nulls: pgast.NullsFirst}},
			}},
		},
		{
			"named window",
			"row_number() OVER w",
			&pgast.FuncExpr{Name: "row_number", Over: &pgast.WindowSpec{Name: "w"}, OverRef: true},
		},
		{
			"refined named window",
			"row_number() OVER (w ORDER BY a)",
			&pgast.FuncExpr{Name: "row_number", Over: &pgast.WindowSpec{
				Name: "w", OrderBy: pgast.OrderBy{{Expr: col("a")}},
			}},
		},
		{
			"after filter",
			"count(*) FILTER (WHERE a) OVER w",
			&pgast.FuncExpr{
				Name: "count", Star: true, Filter: &pgast.Where{Expr: col("a")},
				Over: &pgast.WindowSpec{Name: "w"}, OverRef: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, tt.want)
		})
	}
}

func TestParseExpr_quotedIdentifiers(t *testing.T) {
	assertExpr(t, `"MyCol"`, col(`"MyCol"`))
	assertExpr(t, `"My""Col"`, col(`"My""Col"`))
	assertExpr(t, `"T"."C"`, &pgast.ColName{Qualifier: pgast.TableName{Name: `"T"`}, Name: `"C"`})

	for _, in := range []string{`"MyCol"`, `"T"."C"`, `a::"MyType"`, `CAST(a AS "MyType")`, `"Fn"(a)`} {
		if got := parseExpr(t, in).String(); got != in {
			t.Errorf("ParseExpr(%q).String() = %q", in, got)
		}
	}
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
		assertExpr(t, "a[1:3]", &pgast.ArraySubscriptExpr{Expr: col("a"), From: num("1"), To: num("3"), Slice: true})
	})

	t.Run("open-ended slice", func(t *testing.T) {
		assertExpr(t, "a[:3]", &pgast.ArraySubscriptExpr{Expr: col("a"), To: num("3"), Slice: true})
		assertExpr(t, "a[1:]", &pgast.ArraySubscriptExpr{Expr: col("a"), From: num("1"), Slice: true})
	})

	t.Run("slice round trip", func(t *testing.T) {
		for _, in := range []string{"a[1]", "a[1:3]", "a[:3]", "a[1:]", "a[:]"} {
			if got := parseExpr(t, in).String(); got != in {
				t.Errorf("ParseExpr(%q).String() = %q", in, got)
			}
		}
	})
}

func opExpr(left pgast.Expr, symbol string, right pgast.Expr) *pgast.OperatorExpr {
	return &pgast.OperatorExpr{Left: left, Operator: pgast.Operator{Symbol: symbol}, Right: right}
}

func TestParseExpr_genericOperators(t *testing.T) {
	tests := []struct {
		in    string
		op    string
		right pgast.Expr
	}{
		{"a -> 'k'", "->", num("'k'")},
		{"a ->> 'k'", "->>", num("'k'")},
		{"a #> 'k'", "#>", num("'k'")},
		{"a #>> 'k'", "#>>", num("'k'")},
		{"a @> b", "@>", col("b")},
		{"a <@ b", "<@", col("b")},
		{"a ? 'k'", "?", num("'k'")},
		{"a ?| b", "?|", col("b")},
		{"a ?& b", "?&", col("b")},
		{"a @? b", "@?", col("b")},
		{"a @@ b", "@@", col("b")},
		{"a || b", "||", col("b")},
		{"a ~ 'x'", "~", num("'x'")},
		{"a ~* 'x'", "~*", num("'x'")},
		{"a !~ 'x'", "!~", num("'x'")},
		{"a !~* 'x'", "!~*", num("'x'")},
		{"a & b", "&", col("b")},
		{"a && b", "&&", col("b")},
		{"a | b", "|", col("b")},
		{"a << b", "<<", col("b")},
		{"a >> b", ">>", col("b")},
		{"a # b", "#", col("b")},
		{"a <-> b", "<->", col("b")},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assertExpr(t, tt.in, opExpr(col("a"), tt.op, tt.right))
		})
	}
}

func TestParseExpr_operatorPrecedence(t *testing.T) {
	t.Run("left-associative", func(t *testing.T) {
		assertExpr(t, "a || b || c", opExpr(opExpr(col("a"), "||", col("b")), "||", col("c")))
	})

	t.Run("mixed generic operators share one level", func(t *testing.T) {
		assertExpr(t, "a -> 'k' || b", opExpr(opExpr(col("a"), "->", num("'k'")), "||", col("b")))
	})

	t.Run("plus binds tighter", func(t *testing.T) {
		assertExpr(t, "a || b + 1", opExpr(col("a"), "||", &pgast.ArithmeticExpr{
			Left: col("b"), Operator: pgast.PlusOp, Right: num("1"),
		}))
	})

	t.Run("comparison binds looser", func(t *testing.T) {
		assertExpr(t, "a ~ b = c", &pgast.ComparisonExpr{
			Left: opExpr(col("a"), "~", col("b")), Operator: pgast.EqualOp, Right: col("c"),
		})
	})

	t.Run("like binds looser", func(t *testing.T) {
		assertExpr(t, "a || b LIKE c", &pgast.ComparisonExpr{
			Left: opExpr(col("a"), "||", col("b")), Operator: pgast.LikeOp, Right: col("c"),
		})
	})

	t.Run("prefix operand extends over plus", func(t *testing.T) {
		assertExpr(t, "~ a + b", &pgast.OperatorExpr{
			Operator: pgast.Operator{Symbol: "~"},
			Right:    &pgast.ArithmeticExpr{Left: col("a"), Operator: pgast.PlusOp, Right: col("b")},
		})
	})

	t.Run("prefix operand stops at a generic operator", func(t *testing.T) {
		assertExpr(t, "~ a || b", opExpr(&pgast.OperatorExpr{Operator: pgast.Operator{Symbol: "~"}, Right: col("a")}, "||", col("b")))
	})
}

func TestParseExpr_operatorRoundTrip(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"a||b", "a || b"},
		{"a<->b", "a <-> b"},
		{"a=-1", "a = -1"},
		{"@ -5", "@ -5"},
		{"!! 5", "!! 5"},
		{"- ~a", "- ~ a"},
		{"- -a", "- -a"},
		{"-+1", "- +1"},
		{"-(-a)", "-(-a)"},
		{"a OPERATOR(+) b", "a OPERATOR(+) b"},
		{"a operator(pg_catalog.~) 'x'", "a OPERATOR(pg_catalog.~) 'x'"},
		{"a OPERATOR(pg_catalog.<>) b", "a OPERATOR(pg_catalog.<>) b"},
		{"f(a => 1, b := 2)", "f(a => 1, b := 2)"},
		{"f(1, first => x || y)", "f(1, first => x || y)"},
		{"operator", "operator"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := parseExpr(t, tt.in).String(); got != tt.want {
				t.Errorf("ParseExpr(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseExpr_namedArguments(t *testing.T) {
	t.Run("both spellings", func(t *testing.T) {
		assertExpr(t, "f(a => 1, b := 2)", &pgast.FuncExpr{Name: "f", Args: []pgast.Expr{
			&pgast.NamedArgExpr{Name: "a", Value: num("1")},
			&pgast.NamedArgExpr{Name: "b", Value: num("2"), ColonEquals: true},
		}})
	})

	t.Run("keyword name and operator value", func(t *testing.T) {
		assertExpr(t, "f(1, first => x || y)", &pgast.FuncExpr{Name: "f", Args: []pgast.Expr{
			num("1"),
			&pgast.NamedArgExpr{Name: "first", Value: opExpr(col("x"), "||", col("y"))},
		}})
	})
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
		assertExpr(t, "a @> b::jsonb", opExpr(col("a"), "@>", &pgast.CastExpr{Expr: col("b"), Type: "jsonb"}))
	})

	t.Run("JSON operator binds tighter than comparison", func(t *testing.T) {
		assertExpr(t, "a -> 'k' = b", &pgast.ComparisonExpr{
			Left:     opExpr(col("a"), "->", num("'k'")),
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
		"a ^",
		"a = b = c",
		"a IS",
		"a IS foo",
		"a LIKE b ESCAPE",
		"f() OVER",
		"f() OVER (PARTITION a)",
		"a ||",
		"a OPERATOR(",
		"a OPERATOR(x)",
		"a OPERATOR(+",
		"f(a =>)",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			assertExprError(t, tt)
		})
	}
}
