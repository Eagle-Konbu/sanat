package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestComparisonExpr_String(t *testing.T) {
	e := &sqlast.ComparisonExpr{
		Left:     lit("a"),
		Operator: sqlast.EqualOp,
		Right:    lit("1"),
	}

	assertEqual(t, "a = 1", e.String())
}

func TestAndExpr_String(t *testing.T) {
	e := &sqlast.AndExpr{Left: lit("a"), Right: lit("b")}
	assertEqual(t, "a AND b", e.String())
}

func TestOrExpr_String(t *testing.T) {
	e := &sqlast.OrExpr{Left: lit("a"), Right: lit("b")}
	assertEqual(t, "a OR b", e.String())
}

func TestNotExpr_String(t *testing.T) {
	e := &sqlast.NotExpr{Expr: lit("a")}
	assertEqual(t, "NOT a", e.String())
}

func TestCaseExpr_String(t *testing.T) {
	t.Run("searched", func(t *testing.T) {
		e := &sqlast.CaseExpr{
			Whens: []*sqlast.When{
				{Cond: lit("a = 1"), Val: lit("'x'")},
				{Cond: lit("a = 2"), Val: lit("'y'")},
			},
			Else: lit("'z'"),
		}

		assertEqual(t, "CASE WHEN a = 1 THEN 'x' WHEN a = 2 THEN 'y' ELSE 'z' END", e.String())
	})

	t.Run("simple", func(t *testing.T) {
		e := &sqlast.CaseExpr{
			Expr: lit("status"),
			Whens: []*sqlast.When{
				{Cond: lit("1"), Val: lit("'active'")},
			},
		}

		assertEqual(t, "CASE status WHEN 1 THEN 'active' END", e.String())
	})
}

func TestComparisonExpr_String_wordOperators(t *testing.T) {
	tests := []struct {
		op   sqlast.ComparisonOperator
		want string
	}{
		{sqlast.InOp, "a IN 1"},
		{sqlast.NotInOp, "a NOT IN 1"},
		{sqlast.LikeOp, "a LIKE 1"},
		{sqlast.NotLikeOp, "a NOT LIKE 1"},
	}

	for _, tt := range tests {
		e := &sqlast.ComparisonExpr{Left: lit("a"), Operator: tt.op, Right: lit("1")}
		assertEqual(t, tt.want, e.String())
	}
}

func TestRangeCond_String(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		e := &sqlast.RangeCond{Left: lit("a"), From: lit("1"), To: lit("10")}
		assertEqual(t, "a BETWEEN 1 AND 10", e.String())
	})

	t.Run("negated", func(t *testing.T) {
		e := &sqlast.RangeCond{Not: true, Left: lit("a"), From: lit("1"), To: lit("10")}
		assertEqual(t, "a NOT BETWEEN 1 AND 10", e.String())
	})
}

func TestIsExpr_String(t *testing.T) {
	t.Run("is null", func(t *testing.T) {
		e := &sqlast.IsExpr{Expr: lit("a")}
		assertEqual(t, "a IS NULL", e.String())
	})

	t.Run("is not null", func(t *testing.T) {
		e := &sqlast.IsExpr{Not: true, Expr: lit("a")}
		assertEqual(t, "a IS NOT NULL", e.String())
	})
}

func TestValTuple_String(t *testing.T) {
	v := sqlast.ValTuple{lit("1"), lit("2"), lit("3")}
	assertEqual(t, "(1, 2, 3)", v.String())
}

func TestArithmeticExpr_String(t *testing.T) {
	tests := []struct {
		op   sqlast.ArithmeticOperator
		want string
	}{
		{sqlast.PlusOp, "a + b"},
		{sqlast.MinusOp, "a - b"},
		{sqlast.MultOp, "a * b"},
		{sqlast.DivOp, "a / b"},
		{sqlast.ModOp, "a % b"},
	}

	for _, tt := range tests {
		e := &sqlast.ArithmeticExpr{Left: lit("a"), Operator: tt.op, Right: lit("b")}
		assertEqual(t, tt.want, e.String())
	}
}

func TestArithmeticOperator_ToString(t *testing.T) {
	assertEqual(t, "+", sqlast.ArithmeticOperator(99).ToString())
	assertEqual(t, "+", sqlast.ArithmeticOperator(-1).ToString())
}

func TestUnaryExpr_String(t *testing.T) {
	assertEqual(t, "-a", (&sqlast.UnaryExpr{Operator: sqlast.UMinusOp, Expr: lit("a")}).String())
	assertEqual(t, "+a", (&sqlast.UnaryExpr{Operator: sqlast.UPlusOp, Expr: lit("a")}).String())
}

func TestUnaryOperator_ToString(t *testing.T) {
	assertEqual(t, "-", sqlast.UMinusOp.ToString())
	assertEqual(t, "+", sqlast.UPlusOp.ToString())
}

func TestExistsExpr_String(t *testing.T) {
	e := &sqlast.ExistsExpr{
		Subquery: &sqlast.Subquery{
			Select: &sqlast.Select{
				SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}},
				From:        []sqlast.TableExpr{&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "t"}}},
			},
		},
	}

	assertEqual(t, "EXISTS (SELECT 1 FROM t)", e.String())
}

func TestSubquery_String(t *testing.T) {
	s := &sqlast.Subquery{
		Select: &sqlast.Select{
			SelectExprs: []sqlast.SelectExpr{&sqlast.AliasedExpr{Expr: lit("1")}},
		},
	}

	assertEqual(t, "(SELECT 1)", s.String())
}

func TestColName_String(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		c := &sqlast.ColName{Name: "id"}
		assertEqual(t, "id", c.String())
	})

	t.Run("qualified", func(t *testing.T) {
		c := &sqlast.ColName{Name: "id", Qualifier: sqlast.TableName{Name: "users"}}
		assertEqual(t, "users.id", c.String())
	})
}

func TestLiteral_String(t *testing.T) {
	assertEqual(t, "42", lit("42").String())
}

func TestFuncExpr_String(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		f := &sqlast.FuncExpr{Name: "COALESCE", Exprs: exprs("a", "0")}
		assertEqual(t, "COALESCE(a, 0)", f.String())
	})

	t.Run("qualified", func(t *testing.T) {
		f := &sqlast.FuncExpr{Qualifier: "mydb", Name: "func1", Exprs: exprs("x")}
		assertEqual(t, "mydb.func1(x)", f.String())
	})
}

func TestParenExpr_String(t *testing.T) {
	e := &sqlast.ParenExpr{Expr: lit("a + b")}
	assertEqual(t, "(a + b)", e.String())
}

func TestCount_String(t *testing.T) {
	t.Run("without distinct", func(t *testing.T) {
		c := &sqlast.Count{Args: exprs("id")}
		assertEqual(t, "COUNT(id)", c.String())
	})

	t.Run("with distinct", func(t *testing.T) {
		c := &sqlast.Count{Args: exprs("id"), Distinct: true}
		assertEqual(t, "COUNT(DISTINCT id)", c.String())
	})

	t.Run("with over", func(t *testing.T) {
		c := &sqlast.Count{
			Args:       exprs("id"),
			OverClause: &sqlast.OverClause{},
		}

		assertEqual(t, "COUNT(id) OVER ()", c.String())
	})
}

func TestCountStar_String(t *testing.T) {
	assertEqual(t, "COUNT(*)", (&sqlast.CountStar{}).String())
	assertEqual(t, "COUNT(*) OVER ()", (&sqlast.CountStar{OverClause: &sqlast.OverClause{}}).String())
}

func TestAggregateTypes_String(t *testing.T) {
	tests := []struct {
		name string
		expr sqlast.Expr
		want string
	}{
		{"SUM", &sqlast.Sum{Arg: lit("amount")}, "SUM(amount)"},
		{"SUM DISTINCT", &sqlast.Sum{Arg: lit("amount"), Distinct: true}, "SUM(DISTINCT amount)"},
		{"AVG", &sqlast.Avg{Arg: lit("score")}, "AVG(score)"},
		{"MIN", &sqlast.Min{Arg: lit("price")}, "MIN(price)"},
		{"MAX", &sqlast.Max{Arg: lit("price")}, "MAX(price)"},
		{"BIT_AND", &sqlast.BitAnd{Arg: lit("flags")}, "BIT_AND(flags)"},
		{"BIT_OR", &sqlast.BitOr{Arg: lit("flags")}, "BIT_OR(flags)"},
		{"BIT_XOR", &sqlast.BitXor{Arg: lit("flags")}, "BIT_XOR(flags)"},
		{"STD", &sqlast.Std{Arg: lit("val")}, "STD(val)"},
		{"STDDEV", &sqlast.StdDev{Arg: lit("val")}, "STDDEV(val)"},
		{"STDDEV_POP", &sqlast.StdPop{Arg: lit("val")}, "STDDEV_POP(val)"},
		{"STDDEV_SAMP", &sqlast.StdSamp{Arg: lit("val")}, "STDDEV_SAMP(val)"},
		{"VARIANCE", &sqlast.Variance{Arg: lit("val")}, "VARIANCE(val)"},
		{"VAR_POP", &sqlast.VarPop{Arg: lit("val")}, "VAR_POP(val)"},
		{"VAR_SAMP", &sqlast.VarSamp{Arg: lit("val")}, "VAR_SAMP(val)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.expr.String())
		})
	}
}

func TestWindowFunctionTypes_String(t *testing.T) {
	over := &sqlast.OverClause{
		WindowSpec: &sqlast.WindowSpecification{
			OrderClause: sqlast.OrderBy{{Expr: lit("id")}},
		},
	}

	tests := []struct {
		name string
		expr sqlast.Expr
		want string
	}{
		{"ROW_NUMBER", &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.RowNumberExprType, OverClause: over,
		}, "ROW_NUMBER() OVER (ORDER BY id)"},
		{"DENSE_RANK", &sqlast.ArgumentLessWindowExpr{
			Type: sqlast.DenseRankExprType, OverClause: over,
		}, "DENSE_RANK() OVER (ORDER BY id)"},
		{"FIRST_VALUE", &sqlast.FirstOrLastValueExpr{
			Type: sqlast.FirstValueExprType, Expr: lit("salary"), OverClause: over,
		}, "FIRST_VALUE(salary) OVER (ORDER BY id)"},
		{"LAST_VALUE with null treatment", &sqlast.FirstOrLastValueExpr{
			Type: sqlast.LastValueExprType, Expr: lit("salary"),
			NullTreatmentClause: &sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType},
			OverClause:          over,
		}, "LAST_VALUE(salary) IGNORE NULLS OVER (ORDER BY id)"},
		{"NTILE", &sqlast.NtileExpr{N: lit("4"), OverClause: over}, "NTILE(4) OVER (ORDER BY id)"},
		{"NTH_VALUE", &sqlast.NTHValueExpr{
			Expr: lit("salary"), N: lit("2"), OverClause: over,
		}, "NTH_VALUE(salary, 2) OVER (ORDER BY id)"},
		{"NTH_VALUE with clauses", &sqlast.NTHValueExpr{
			Expr:                lit("salary"),
			N:                   lit("2"),
			FromFirstLastClause: &sqlast.FromFirstLastClause{Type: sqlast.FromLastType},
			NullTreatmentClause: &sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType},
			OverClause:          over,
		}, "NTH_VALUE(salary, 2) FROM LAST IGNORE NULLS OVER (ORDER BY id)"},
		{"LAG", &sqlast.LagLeadExpr{
			Type: sqlast.LagExprType, Expr: lit("val"), N: lit("1"), Default: lit("0"),
			OverClause: over,
		}, "LAG(val, 1, 0) OVER (ORDER BY id)"},
		{"LEAD", &sqlast.LagLeadExpr{
			Type: sqlast.LeadExprType, Expr: lit("val"), OverClause: over,
		}, "LEAD(val) OVER (ORDER BY id)"},
		{"LAG with null treatment", &sqlast.LagLeadExpr{
			Type: sqlast.LagExprType, Expr: lit("val"),
			NullTreatmentClause: &sqlast.NullTreatmentClause{Type: sqlast.RespectNullsType},
			OverClause:          over,
		}, "LAG(val) RESPECT NULLS OVER (ORDER BY id)"},
		{"JSON_ARRAYAGG", &sqlast.JSONArrayAgg{Expr: lit("col")}, "JSON_ARRAYAGG(col)"},
		{"JSON_OBJECTAGG", &sqlast.JSONObjectAgg{Key: lit("k"), Value: lit("v")}, "JSON_OBJECTAGG(k, v)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.expr.String())
		})
	}
}
