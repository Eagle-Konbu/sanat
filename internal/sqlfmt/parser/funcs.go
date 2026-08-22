package parser

import (
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

type simpleAggCtor func(arg sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr

type distinctAggCtor func(arg sqlast.Expr, distinct bool, oc *sqlast.OverClause) sqlast.Expr

// simpleAggConstructors builds aggregate/window function nodes that take a
// single argument, no DISTINCT modifier, and an optional OVER clause.
var simpleAggConstructors = map[string]simpleAggCtor{
	"BIT_AND": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.BitAnd{Arg: a, OverClause: oc}
	},
	"BIT_OR": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.BitOr{Arg: a, OverClause: oc}
	},
	"BIT_XOR": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.BitXor{Arg: a, OverClause: oc}
	},
	"STD": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Std{Arg: a, OverClause: oc}
	},
	"STDDEV": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.StdDev{Arg: a, OverClause: oc}
	},
	"STDDEV_POP": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.StdPop{Arg: a, OverClause: oc}
	},
	"STDDEV_SAMP": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.StdSamp{Arg: a, OverClause: oc}
	},
	"VARIANCE": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Variance{Arg: a, OverClause: oc}
	},
	"VAR_POP": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.VarPop{Arg: a, OverClause: oc}
	},
	"VAR_SAMP": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.VarSamp{Arg: a, OverClause: oc}
	},
	"JSON_ARRAYAGG": func(a sqlast.Expr, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.JSONArrayAgg{Expr: a, OverClause: oc}
	},
}

// distinctAggConstructors builds aggregate function nodes that take a single
// argument, an optional DISTINCT modifier, and an optional OVER clause.
var distinctAggConstructors = map[string]distinctAggCtor{
	"SUM": func(a sqlast.Expr, d bool, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Sum{Arg: a, Distinct: d, OverClause: oc}
	},
	"AVG": func(a sqlast.Expr, d bool, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Avg{Arg: a, Distinct: d, OverClause: oc}
	},
	"MIN": func(a sqlast.Expr, d bool, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Min{Arg: a, Distinct: d, OverClause: oc}
	},
	"MAX": func(a sqlast.Expr, d bool, oc *sqlast.OverClause) sqlast.Expr {
		return &sqlast.Max{Arg: a, Distinct: d, OverClause: oc}
	},
}

var argumentLessWindowTypes = map[string]sqlast.ArgumentLessWindowExprType{
	"ROW_NUMBER":   sqlast.RowNumberExprType,
	"RANK":         sqlast.RankExprType,
	"DENSE_RANK":   sqlast.DenseRankExprType,
	"PERCENT_RANK": sqlast.PercentRankExprType,
	"CUME_DIST":    sqlast.CumeDistExprType,
}

// parseFuncCall parses the argument list and trailing clauses of a function
// call whose name has already been seen; the current token is the opening
// '(', which this function consumes before parsing arguments. It dispatches
// to the specific AST node the function name maps to, falling back to a
// generic FuncExpr.
func (p *Parser) parseFuncCall(name string) sqlast.Expr {
	p.advance() // consume '('

	upper := strings.ToUpper(name)

	switch upper {
	case "COUNT":
		return p.parseCountCall()
	case "NTILE":
		return p.parseNtileCall()
	case "NTH_VALUE":
		return p.parseNthValueCall()
	case "LAG", "LEAD":
		return p.parseLagLeadCall(upper == "LEAD")
	case "FIRST_VALUE", "LAST_VALUE":
		return p.parseFirstOrLastValueCall(upper == "LAST_VALUE")
	case "JSON_OBJECTAGG":
		return p.parseJSONObjectAggCall()
	default:
		return p.parseMappedOrGenericFuncCall(upper, name)
	}
}

// parseMappedOrGenericFuncCall handles the function names whose parsing
// shape is looked up from a constructor table, falling back to a generic
// FuncExpr for anything not recognized.
func (p *Parser) parseMappedOrGenericFuncCall(upper, name string) sqlast.Expr {
	if _, ok := argumentLessWindowTypes[upper]; ok {
		return p.parseArgumentLessWindowCall(upper)
	}

	if ctor, ok := distinctAggConstructors[upper]; ok {
		return p.parseDistinctAggCall(ctor)
	}

	if ctor, ok := simpleAggConstructors[upper]; ok {
		return p.parseSimpleAggCall(ctor)
	}

	return p.parseGenericFuncCall(name)
}

func (p *Parser) parseGenericFuncCall(name string) sqlast.Expr {
	var args []sqlast.Expr

	if !p.at(RPAREN) {
		args = p.parseExprList()
	}

	p.expect(RPAREN)

	return &sqlast.FuncExpr{Name: sqlast.ColIdent(name), Exprs: args}
}

func (p *Parser) parseCountCall() sqlast.Expr {
	if p.consume(STAR) {
		p.expect(RPAREN)

		return &sqlast.CountStar{OverClause: p.parseOptionalOverClause()}
	}

	distinct := p.consumeDistinct()
	args := p.parseExprList()
	p.expect(RPAREN)

	return &sqlast.Count{Args: args, Distinct: distinct, OverClause: p.parseOptionalOverClause()}
}

func (p *Parser) parseDistinctAggCall(ctor distinctAggCtor) sqlast.Expr {
	distinct := p.consumeDistinct()
	arg := p.parseExpr()
	p.expect(RPAREN)

	return ctor(arg, distinct, p.parseOptionalOverClause())
}

func (p *Parser) parseSimpleAggCall(ctor simpleAggCtor) sqlast.Expr {
	arg := p.parseExpr()
	p.expect(RPAREN)

	return ctor(arg, p.parseOptionalOverClause())
}

func (p *Parser) parseArgumentLessWindowCall(name string) sqlast.Expr {
	p.expect(RPAREN)

	return &sqlast.ArgumentLessWindowExpr{Type: argumentLessWindowTypes[name], OverClause: p.parseOptionalOverClause()}
}

func (p *Parser) parseFirstOrLastValueCall(isLast bool) sqlast.Expr {
	expr := p.parseExpr()
	p.expect(RPAREN)

	nt := p.parseOptionalNullTreatment()
	oc := p.parseOptionalOverClause()

	typ := sqlast.FirstValueExprType
	if isLast {
		typ = sqlast.LastValueExprType
	}

	return &sqlast.FirstOrLastValueExpr{Type: typ, Expr: expr, NullTreatmentClause: nt, OverClause: oc}
}

func (p *Parser) parseNtileCall() sqlast.Expr {
	n := p.parseExpr()
	p.expect(RPAREN)

	return &sqlast.NtileExpr{N: n, OverClause: p.parseOptionalOverClause()}
}

func (p *Parser) parseNthValueCall() sqlast.Expr {
	expr := p.parseExpr()
	p.expect(COMMA)

	n := p.parseExpr()
	p.expect(RPAREN)

	fl := p.parseOptionalFromFirstLast()
	nt := p.parseOptionalNullTreatment()
	oc := p.parseOptionalOverClause()

	return &sqlast.NTHValueExpr{Expr: expr, N: n, FromFirstLastClause: fl, NullTreatmentClause: nt, OverClause: oc}
}

func (p *Parser) parseLagLeadCall(isLead bool) sqlast.Expr {
	expr := p.parseExpr()
	n, def := p.parseOptionalLagLeadArgs()
	p.expect(RPAREN)

	nt := p.parseOptionalNullTreatment()
	oc := p.parseOptionalOverClause()

	typ := sqlast.LagExprType
	if isLead {
		typ = sqlast.LeadExprType
	}

	return &sqlast.LagLeadExpr{Type: typ, Expr: expr, N: n, Default: def, NullTreatmentClause: nt, OverClause: oc}
}

func (p *Parser) parseOptionalLagLeadArgs() (sqlast.Expr, sqlast.Expr) {
	if !p.consume(COMMA) {
		return nil, nil
	}

	n := p.parseExpr()

	if !p.consume(COMMA) {
		return n, nil
	}

	return n, p.parseExpr()
}

func (p *Parser) parseJSONObjectAggCall() sqlast.Expr {
	key := p.parseExpr()
	p.expect(COMMA)

	val := p.parseExpr()
	p.expect(RPAREN)

	return &sqlast.JSONObjectAgg{Key: key, Value: val, OverClause: p.parseOptionalOverClause()}
}

func (p *Parser) parseOptionalNullTreatment() *sqlast.NullTreatmentClause {
	switch {
	case p.consume(RESPECT):
		p.expect(NULLS)

		return &sqlast.NullTreatmentClause{Type: sqlast.RespectNullsType}
	case p.consume(IGNORE):
		p.expect(NULLS)

		return &sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType}
	default:
		return nil
	}
}

func (p *Parser) parseOptionalFromFirstLast() *sqlast.FromFirstLastClause {
	if !p.consume(FROM) {
		return nil
	}

	switch {
	case p.consume(FIRST):
		return &sqlast.FromFirstLastClause{Type: sqlast.FromFirstType}
	case p.consume(LAST):
		return &sqlast.FromFirstLastClause{Type: sqlast.FromLastType}
	default:
		return failReturn[*sqlast.FromFirstLastClause](p, "expected FIRST or LAST after FROM")
	}
}

func (p *Parser) parseOptionalOverClause() *sqlast.OverClause {
	if !p.consume(OVER) {
		return nil
	}

	if p.at(IDENT) {
		name := p.tok.Literal
		p.advance()

		return &sqlast.OverClause{WindowName: sqlast.ColIdent(name)}
	}

	p.expect(LPAREN)

	spec := p.parseWindowSpecification()

	p.expect(RPAREN)

	return &sqlast.OverClause{WindowSpec: spec}
}

func (p *Parser) parseWindowSpecification() *sqlast.WindowSpecification {
	spec := &sqlast.WindowSpecification{}

	if p.consume(PARTITION) {
		p.expect(BY)

		spec.PartitionClause = p.parseExprList()
	}

	if p.at(ORDER) {
		spec.OrderClause = p.parseOrderByClause()
	}

	if p.at(ROWS) || p.at(RANGE) {
		spec.FrameClause = p.parseFrameClause()
	}

	return spec
}

func (p *Parser) parseFrameClause() *sqlast.FrameClause {
	unit := sqlast.FrameRowsType
	if p.at(RANGE) {
		unit = sqlast.FrameRangeType
	}

	p.advance() // consume ROWS/RANGE

	if p.consume(BETWEEN) {
		start := p.parseFramePoint()
		p.expect(AND)

		end := p.parseFramePoint()

		return &sqlast.FrameClause{Unit: unit, Start: start, End: end}
	}

	return &sqlast.FrameClause{Unit: unit, Start: p.parseFramePoint()}
}

func (p *Parser) parseFramePoint() *sqlast.FramePoint {
	switch {
	case p.consume(CURRENT):
		p.expect(ROW)

		return &sqlast.FramePoint{Type: sqlast.CurrentRowType}
	case p.consume(UNBOUNDED):
		return p.parseUnboundedFramePoint()
	default:
		return p.parseExprFramePoint()
	}
}

func (p *Parser) parseUnboundedFramePoint() *sqlast.FramePoint {
	switch {
	case p.consume(PRECEDING):
		return &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType}
	case p.consume(FOLLOWING):
		return &sqlast.FramePoint{Type: sqlast.UnboundedFollowingType}
	default:
		return failReturn[*sqlast.FramePoint](p, "expected PRECEDING or FOLLOWING after UNBOUNDED")
	}
}

func (p *Parser) parseExprFramePoint() *sqlast.FramePoint {
	expr := p.parseAdditiveExpr()

	switch {
	case p.consume(PRECEDING):
		return &sqlast.FramePoint{Type: sqlast.ExprPrecedingType, Expr: expr}
	case p.consume(FOLLOWING):
		return &sqlast.FramePoint{Type: sqlast.ExprFollowingType, Expr: expr}
	default:
		return failReturn[*sqlast.FramePoint](p, "expected PRECEDING or FOLLOWING")
	}
}
