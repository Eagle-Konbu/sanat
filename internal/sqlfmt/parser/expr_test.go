package parser_test

import (
	"reflect"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestParseError_Error(t *testing.T) {
	err := &parser.ParseError{Pos: parser.Position{Line: 3, Column: 7}, Msg: "expected RPAREN, got EOF"}

	want := "3:7: expected RPAREN, got EOF"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func parseExpr(t *testing.T, input string) sqlast.Expr {
	t.Helper()

	e, err := parser.ParseExpr(input)
	if err != nil {
		t.Fatalf("ParseExpr(%q) error = %v", input, err)
	}

	return e
}

func assertExpr(t *testing.T, input string, want sqlast.Expr) {
	t.Helper()

	got := parseExpr(t, input)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseExpr(%q) =\n  %#v\nwant\n  %#v", input, got, want)
	}
}

func col(name string) *sqlast.ColName { return &sqlast.ColName{Name: sqlast.ColIdent(name)} }
func num(s string) *sqlast.Literal    { return &sqlast.Literal{Val: s} }

func TestParseExpr_literals(t *testing.T) {
	assertExpr(t, "123", num("123"))
	assertExpr(t, "1.5", num("1.5"))
	assertExpr(t, "NULL", &sqlast.Literal{Val: "NULL"})
	assertExpr(t, "TRUE", &sqlast.Literal{Val: "TRUE"})
	assertExpr(t, "FALSE", &sqlast.Literal{Val: "FALSE"})
	assertExpr(t, "?", &sqlast.Literal{Val: "?"})
	assertExpr(t, ":name", &sqlast.Literal{Val: ":name"})
	assertExpr(t, ":1", &sqlast.Literal{Val: ":1"})
}

func TestParseExpr_prefixedLiterals(t *testing.T) {
	assertExpr(t, "0x1A", num("0x1A"))
	assertExpr(t, "0X1a", num("0X1a"))
	assertExpr(t, "0b101", num("0b101"))
	assertExpr(t, "0B110", num("0B110"))
	assertExpr(t, "x'1A'", num("x'1A'"))
	assertExpr(t, "X'1a'", num("X'1a'"))
	assertExpr(t, "b'101'", num("b'101'"))
	assertExpr(t, "B'110'", num("B'110'"))
}

func TestParseExpr_stringLiteral(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"plain", `'abc'`, `'abc'`},
		{"doubled quote escape", `'it''s'`, `'it\'s'`},
		{"backslash escape", `'a\'b'`, `'a\'b'`},
		{"preserves like escapes", `'50\%'`, `'50\%'`},
		{"newline roundtrips as backslash-n", "'a\nb'", `'a\nb'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertExpr(t, tt.in, &sqlast.Literal{Val: tt.want})
		})
	}
}

func TestParseExpr_columnRefs(t *testing.T) {
	assertExpr(t, "id", col("id"))
	assertExpr(t, "t.id", &sqlast.ColName{Qualifier: sqlast.TableName{Name: "t"}, Name: "id"})
	assertExpr(t, "`select`", col("select"))
}

func TestParseExpr_arithmeticPrecedence(t *testing.T) {
	// 1 + 2 * 3 must be 1 + (2 * 3), not (1 + 2) * 3.
	want := &sqlast.ArithmeticExpr{
		Left:     num("1"),
		Operator: sqlast.PlusOp,
		Right:    &sqlast.ArithmeticExpr{Left: num("2"), Operator: sqlast.MultOp, Right: num("3")},
	}
	assertExpr(t, "1 + 2 * 3", want)
}

func TestParseExpr_parensOverridePrecedence(t *testing.T) {
	want := &sqlast.ArithmeticExpr{
		Left:     &sqlast.ParenExpr{Expr: &sqlast.ArithmeticExpr{Left: num("1"), Operator: sqlast.PlusOp, Right: num("2")}},
		Operator: sqlast.MultOp,
		Right:    num("3"),
	}
	assertExpr(t, "(1 + 2) * 3", want)
}

func TestParseExpr_unaryMinus(t *testing.T) {
	want := &sqlast.ArithmeticExpr{
		Left:     &sqlast.UnaryExpr{Operator: sqlast.UMinusOp, Expr: num("1")},
		Operator: sqlast.PlusOp,
		Right:    num("2"),
	}
	assertExpr(t, "-1 + 2", want)
}

func TestParseExpr_unaryPlus(t *testing.T) {
	assertExpr(t, "+1", &sqlast.UnaryExpr{Operator: sqlast.UPlusOp, Expr: num("1")})
}

func TestParseExpr_orAndNotPrecedence(t *testing.T) {
	// NOT > AND > OR: "a OR b AND NOT c" must parse as a OR (b AND (NOT c)).
	want := &sqlast.OrExpr{
		Left: col("a"),
		Right: &sqlast.AndExpr{
			Left:  col("b"),
			Right: &sqlast.NotExpr{Expr: col("c")},
		},
	}
	assertExpr(t, "a OR b AND NOT c", want)
}

func TestParseExpr_notBindsTighterThanComparisonIsWrong(t *testing.T) {
	// Comparisons bind tighter than NOT: "NOT a = b" is NOT (a = b).
	want := &sqlast.NotExpr{
		Expr: &sqlast.ComparisonExpr{Left: col("a"), Operator: sqlast.EqualOp, Right: col("b")},
	}
	assertExpr(t, "NOT a = b", want)
}

func TestParseExpr_comparisonOperators(t *testing.T) {
	tests := []struct {
		in string
		op sqlast.ComparisonOperator
	}{
		{"a = b", sqlast.EqualOp},
		{"a <> b", sqlast.NotEqualOp},
		{"a != b", sqlast.NotEqualOp},
		{"a < b", sqlast.LessThanOp},
		{"a > b", sqlast.GreaterThanOp},
		{"a <= b", sqlast.LessEqualOp},
		{"a >= b", sqlast.GreaterEqualOp},
		{"a <=> b", sqlast.NullSafeEqualOp},
	}

	for _, tt := range tests {
		want := &sqlast.ComparisonExpr{Left: col("a"), Operator: tt.op, Right: col("b")}
		assertExpr(t, tt.in, want)
	}
}

func TestParseExpr_likeExpr(t *testing.T) {
	assertExpr(t, "a LIKE '%x%'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.LikeOp, Right: &sqlast.Literal{Val: "'%x%'"},
	})

	assertExpr(t, "a NOT LIKE '%x%'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.NotLikeOp, Right: &sqlast.Literal{Val: "'%x%'"},
	})
}

func TestParseExpr_regexpExpr(t *testing.T) {
	assertExpr(t, "a REGEXP '^x'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.RegexpOp, Right: &sqlast.Literal{Val: "'^x'"},
	})

	assertExpr(t, "a NOT REGEXP '^x'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.NotRegexpOp, Right: &sqlast.Literal{Val: "'^x'"},
	})

	// RLIKE is a synonym for REGEXP; it parses to the same operator.
	assertExpr(t, "a RLIKE '^x'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.RegexpOp, Right: &sqlast.Literal{Val: "'^x'"},
	})

	assertExpr(t, "a NOT RLIKE '^x'", &sqlast.ComparisonExpr{
		Left: col("a"), Operator: sqlast.NotRegexpOp, Right: &sqlast.Literal{Val: "'^x'"},
	})
}

func TestParseExpr_inExpr(t *testing.T) {
	t.Run("value list", func(t *testing.T) {
		assertExpr(t, "a IN (1, 2, 3)", &sqlast.ComparisonExpr{
			Left: col("a"), Operator: sqlast.InOp, Right: sqlast.ValTuple{num("1"), num("2"), num("3")},
		})
	})

	t.Run("not in", func(t *testing.T) {
		assertExpr(t, "a NOT IN (1)", &sqlast.ComparisonExpr{
			Left: col("a"), Operator: sqlast.NotInOp, Right: sqlast.ValTuple{num("1")},
		})
	})

	t.Run("subquery", func(t *testing.T) {
		assertExpr(t, "a IN (SELECT id FROM t)", &sqlast.ComparisonExpr{
			Left:     col("a"),
			Operator: sqlast.InOp,
			Right: &sqlast.Subquery{Select: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: col("id")}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
			}},
		})
	})
}

func TestParseExpr_betweenExpr(t *testing.T) {
	assertExpr(t, "a BETWEEN 1 AND 10", &sqlast.RangeCond{Left: col("a"), From: num("1"), To: num("10")})
	assertExpr(t, "a NOT BETWEEN 1 AND 10", &sqlast.RangeCond{Not: true, Left: col("a"), From: num("1"), To: num("10")})
}

func TestParseExpr_isNullExpr(t *testing.T) {
	assertExpr(t, "a IS NULL", &sqlast.IsExpr{Expr: col("a")})
	assertExpr(t, "a IS NOT NULL", &sqlast.IsExpr{Not: true, Expr: col("a")})
}

func TestParseExpr_caseExpr(t *testing.T) {
	t.Run("searched", func(t *testing.T) {
		want := &sqlast.CaseExpr{
			Whens: []*sqlast.When{{Cond: col("a"), Val: num("1")}},
			Else:  num("0"),
		}
		assertExpr(t, "CASE WHEN a THEN 1 ELSE 0 END", want)
	})

	t.Run("simple", func(t *testing.T) {
		want := &sqlast.CaseExpr{
			Expr:  col("status"),
			Whens: []*sqlast.When{{Cond: num("1"), Val: &sqlast.Literal{Val: "'active'"}}},
		}
		assertExpr(t, "CASE status WHEN 1 THEN 'active' END", want)
	})
}

func TestParseExpr_existsExpr(t *testing.T) {
	want := &sqlast.ExistsExpr{Subquery: &sqlast.Subquery{Select: &sqlast.Select{
		SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: num("1")}},
		From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
	}}}
	assertExpr(t, "EXISTS (SELECT 1 FROM t)", want)
}

func TestParseExpr_scalarSubquery(t *testing.T) {
	want := &sqlast.Subquery{Select: &sqlast.Select{
		SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: num("1")}},
	}}
	assertExpr(t, "(SELECT 1)", want)
}

func TestParseExpr_genericFuncCall(t *testing.T) {
	assertExpr(t, "COALESCE(a, b)", &sqlast.FuncExpr{Name: "COALESCE", Exprs: []sqlast.Expr{col("a"), col("b")}})
	assertExpr(t, "NOW()", &sqlast.FuncExpr{Name: "NOW"})
}

func TestParseExpr_countCall(t *testing.T) {
	t.Run("star", func(t *testing.T) {
		assertExpr(t, "COUNT(*)", &sqlast.CountStar{})
	})

	t.Run("distinct", func(t *testing.T) {
		assertExpr(t, "count(DISTINCT a)", &sqlast.Count{Args: []sqlast.Expr{col("a")}, Distinct: true})
	})
}

func TestParseExpr_distinctAggCall(t *testing.T) {
	assertExpr(t, "SUM(DISTINCT a)", &sqlast.Sum{Arg: col("a"), Distinct: true})
	assertExpr(t, "AVG(a)", &sqlast.Avg{Arg: col("a")})
	assertExpr(t, "MIN(a)", &sqlast.Min{Arg: col("a")})
	assertExpr(t, "MAX(a)", &sqlast.Max{Arg: col("a")})
}

func TestParseExpr_simpleAggCall(t *testing.T) {
	assertExpr(t, "BIT_AND(a)", &sqlast.BitAnd{Arg: col("a")})
	assertExpr(t, "BIT_OR(a)", &sqlast.BitOr{Arg: col("a")})
	assertExpr(t, "BIT_XOR(a)", &sqlast.BitXor{Arg: col("a")})
	assertExpr(t, "STD(a)", &sqlast.Std{Arg: col("a")})
	assertExpr(t, "STDDEV(a)", &sqlast.StdDev{Arg: col("a")})
	assertExpr(t, "STDDEV_POP(a)", &sqlast.StdPop{Arg: col("a")})
	assertExpr(t, "STDDEV_SAMP(a)", &sqlast.StdSamp{Arg: col("a")})
	assertExpr(t, "VARIANCE(a)", &sqlast.Variance{Arg: col("a")})
	assertExpr(t, "VAR_POP(a)", &sqlast.VarPop{Arg: col("a")})
	assertExpr(t, "VAR_SAMP(a)", &sqlast.VarSamp{Arg: col("a")})
	assertExpr(t, "JSON_ARRAYAGG(a)", &sqlast.JSONArrayAgg{Expr: col("a")})
}

func TestParseExpr_jsonObjectAgg(t *testing.T) {
	assertExpr(t, "JSON_OBJECTAGG(k, v)", &sqlast.JSONObjectAgg{Key: col("k"), Value: col("v")})
}

func TestParseExpr_windowFunctions(t *testing.T) {
	t.Run("argument-less with over", func(t *testing.T) {
		want := &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.RowNumberExprType,
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{
				PartitionClause: []sqlast.Expr{col("dept")},
				OrderClause:     sqlast.OrderBy{{Expr: col("id"), Direction: sqlast.DescOrder}},
			}},
		}
		assertExpr(t, "ROW_NUMBER() OVER (PARTITION BY dept ORDER BY id DESC)", want)
	})

	t.Run("named window", func(t *testing.T) {
		assertExpr(t, "RANK() OVER w", &sqlast.ArgumentLessWindowExpr{
			Type:       sqlast.RankExprType,
			OverClause: &sqlast.OverClause{WindowName: "w"},
		})
	})

	t.Run("first value with null treatment", func(t *testing.T) {
		want := &sqlast.FirstOrLastValueExpr{
			Type:                sqlast.FirstValueExprType,
			Expr:                col("a"),
			NullTreatmentClause: &sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType},
			OverClause:          &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "FIRST_VALUE(a) IGNORE NULLS OVER ()", want)
	})

	t.Run("nth value from last", func(t *testing.T) {
		want := &sqlast.NTHValueExpr{
			Expr:                col("a"),
			N:                   num("2"),
			FromFirstLastClause: &sqlast.FromFirstLastClause{Type: sqlast.FromLastType},
			OverClause:          &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "NTH_VALUE(a, 2) FROM LAST OVER ()", want)
	})

	t.Run("lag with default", func(t *testing.T) {
		want := &sqlast.LagLeadExpr{
			Type:       sqlast.LagExprType,
			Expr:       col("a"),
			N:          num("1"),
			Default:    num("0"),
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "LAG(a, 1, 0) OVER ()", want)
	})

	t.Run("ntile", func(t *testing.T) {
		want := &sqlast.NtileExpr{N: num("4"), OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}}}
		assertExpr(t, "NTILE(4) OVER ()", want)
	})

	t.Run("respect nulls", func(t *testing.T) {
		want := &sqlast.FirstOrLastValueExpr{
			Type:                sqlast.LastValueExprType,
			Expr:                col("a"),
			NullTreatmentClause: &sqlast.NullTreatmentClause{Type: sqlast.RespectNullsType},
			OverClause:          &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "LAST_VALUE(a) RESPECT NULLS OVER ()", want)
	})

	t.Run("nth value from first", func(t *testing.T) {
		want := &sqlast.NTHValueExpr{
			Expr:                col("a"),
			N:                   num("1"),
			FromFirstLastClause: &sqlast.FromFirstLastClause{Type: sqlast.FromFirstType},
			OverClause:          &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "NTH_VALUE(a, 1) FROM FIRST OVER ()", want)
	})

	t.Run("lead without extra args", func(t *testing.T) {
		want := &sqlast.LagLeadExpr{
			Type:       sqlast.LeadExprType,
			Expr:       col("a"),
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{}},
		}
		assertExpr(t, "LEAD(a) OVER ()", want)
	})

	t.Run("expr frame points", func(t *testing.T) {
		want := &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.RowNumberExprType,
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{
				FrameClause: &sqlast.FrameClause{
					Unit:  sqlast.FrameRowsType,
					Start: &sqlast.FramePoint{Type: sqlast.ExprPrecedingType, Expr: num("3")},
					End:   &sqlast.FramePoint{Type: sqlast.ExprFollowingType, Expr: num("5")},
				},
			}},
		}
		assertExpr(t, "ROW_NUMBER() OVER (ROWS BETWEEN 3 PRECEDING AND 5 FOLLOWING)", want)
	})

	t.Run("frame clause", func(t *testing.T) {
		want := &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.RowNumberExprType,
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{
				FrameClause: &sqlast.FrameClause{
					Unit:  sqlast.FrameRangeType,
					Start: &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType},
					End:   &sqlast.FramePoint{Type: sqlast.CurrentRowType},
				},
			}},
		}
		assertExpr(t, "ROW_NUMBER() OVER (RANGE BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW)", want)
	})

	t.Run("unbounded following", func(t *testing.T) {
		want := &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.RowNumberExprType,
			OverClause: &sqlast.OverClause{WindowSpec: &sqlast.WindowSpecification{
				FrameClause: &sqlast.FrameClause{
					Unit:  sqlast.FrameRowsType,
					Start: &sqlast.FramePoint{Type: sqlast.CurrentRowType},
					End:   &sqlast.FramePoint{Type: sqlast.UnboundedFollowingType},
				},
			}},
		}
		assertExpr(t, "ROW_NUMBER() OVER (ROWS BETWEEN CURRENT ROW AND UNBOUNDED FOLLOWING)", want)
	})
}

func TestParseExpr_errors(t *testing.T) {
	tests := []string{
		"",
		"1 +",
		"(1",
		"a BETWEEN 1",
		"a BETWEEN 1 2",
		"a NOT XYZ",
		"a IS TRUE",
		"a IN 1",
		"a IN (1",
		"CASE END",
		"CASE WHEN a THEN 1",
		"NOT",
		"1 2",
		":",
		":+",
		"COUNT(",
		"COUNT(*",
		"SUM(a",
		"NTH_VALUE(a)",
		"NTH_VALUE(a, 1) FROM x",
		"LAG(a) RESPECT x",
		"ROW_NUMBER() OVER (ROWS x)",
		"ROW_NUMBER() OVER (ROWS UNBOUNDED x)",
		"ROW_NUMBER() OVER (ROWS 1 x)",
		"EXISTS (1)",
		"a IN (SELECT",
		"FIRST_VALUE(a",

		// Lexer-level errors (unterminated string, illegal byte) positioned
		// deep inside each construct, to exercise the advance()/expect()
		// error-propagation paths that a wrong-token syntax error can't reach.
		"a OR b OR 'unterminated",
		"a AND b AND 'unterminated",
		"NOT 'unterminated",
		"a = 'unterminated",
		"a LIKE 'unterminated",
		"a NOT LIKE 'unterminated",
		"a REGEXP 'unterminated",
		"a NOT REGEXP 'unterminated",
		"a RLIKE 'unterminated",
		"a <=> 'unterminated",
		"x'unterminated",
		"b'unterminated",
		"a IN (1, 2, 'unterminated)",
		"a NOT IN (1, 'unterminated)",
		"a BETWEEN 1 AND 'unterminated",
		"a NOT BETWEEN 1 AND 'unterminated",
		"1 + 2 + 'unterminated",
		"1 * 2 * 'unterminated",
		"- - 'unterminated",
		"+ + 'unterminated",
		"(1 + 'unterminated)",
		"(SELECT 'unterminated FROM t)",
		"CASE 'unterminated WHEN 1 THEN 2 END",
		"CASE WHEN 'unterminated THEN 1 END",
		"CASE WHEN a THEN 'unterminated END",
		"CASE WHEN a THEN 1 ELSE 'unterminated END",
		"EXISTS (SELECT 'unterminated FROM t)",
		"COALESCE(a, 'unterminated)",
		"COUNT(DISTINCT 'unterminated)",
		"ROW_NUMBER() OVER (PARTITION BY 'unterminated)",
		"ROW_NUMBER() OVER (ORDER BY 'unterminated)",
		"ROW_NUMBER() OVER (ROWS BETWEEN 'unterminated AND CURRENT ROW)",
		"ROW_NUMBER() OVER (ROWS BETWEEN UNBOUNDED PRECEDING AND 'unterminated FOLLOWING)",
		"LAG(a, 1, 'unterminated) OVER ()",
		"NTH_VALUE(a, 'unterminated) OVER ()",
		"JSON_OBJECTAGG('unterminated, v)",
		"JSON_OBJECTAGG(k, 'unterminated)",
		`a "illegal`,
		`a OR "illegal`,
		`t."illegal`,
	}

	for _, in := range tests {
		if _, err := parser.ParseExpr(in); err == nil {
			t.Errorf("ParseExpr(%q) expected error, got nil", in)
		}
	}
}
