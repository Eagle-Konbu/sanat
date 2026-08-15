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
// call whose name and opening '(' have already been seen; the current token
// is the one immediately after '('. It dispatches to the specific AST node
// the function name maps to, falling back to a generic FuncExpr.
func (p *Parser) parseFuncCall(name string) (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume '('
		return nil, err
	}

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
func (p *Parser) parseMappedOrGenericFuncCall(upper, name string) (sqlast.Expr, error) {
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

func (p *Parser) parseGenericFuncCall(name string) (sqlast.Expr, error) {
	var args []sqlast.Expr

	if !p.at(RPAREN) {
		var err error

		args, err = p.parseExprList()
		if err != nil {
			return nil, err
		}
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.FuncExpr{Name: sqlast.ColIdent(name), Exprs: args}, nil
}

func (p *Parser) parseCountCall() (sqlast.Expr, error) {
	if ok, err := p.consume(STAR); err != nil {
		return nil, err
	} else if ok {
		if err := p.expect(RPAREN); err != nil {
			return nil, err
		}

		oc, err := p.parseOptionalOverClause()
		if err != nil {
			return nil, err
		}

		return &sqlast.CountStar{OverClause: oc}, nil
	}

	distinct, err := p.consumeDistinct()
	if err != nil {
		return nil, err
	}

	args, err := p.parseExprList()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return &sqlast.Count{Args: args, Distinct: distinct, OverClause: oc}, nil
}

func (p *Parser) parseDistinctAggCall(ctor distinctAggCtor) (sqlast.Expr, error) {
	distinct, err := p.consumeDistinct()
	if err != nil {
		return nil, err
	}

	arg, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return ctor(arg, distinct, oc), nil
}

func (p *Parser) parseSimpleAggCall(ctor simpleAggCtor) (sqlast.Expr, error) {
	arg, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return ctor(arg, oc), nil
}

func (p *Parser) parseArgumentLessWindowCall(name string) (sqlast.Expr, error) {
	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return &sqlast.ArgumentLessWindowExpr{Type: argumentLessWindowTypes[name], OverClause: oc}, nil
}

func (p *Parser) parseFirstOrLastValueCall(isLast bool) (sqlast.Expr, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	nt, err := p.parseOptionalNullTreatment()
	if err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	typ := sqlast.FirstValueExprType
	if isLast {
		typ = sqlast.LastValueExprType
	}

	return &sqlast.FirstOrLastValueExpr{Type: typ, Expr: expr, NullTreatmentClause: nt, OverClause: oc}, nil
}

func (p *Parser) parseNtileCall() (sqlast.Expr, error) {
	n, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return &sqlast.NtileExpr{N: n, OverClause: oc}, nil
}

func (p *Parser) parseNthValueCall() (sqlast.Expr, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(COMMA); err != nil {
		return nil, err
	}

	n, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	fl, err := p.parseOptionalFromFirstLast()
	if err != nil {
		return nil, err
	}

	nt, err := p.parseOptionalNullTreatment()
	if err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return &sqlast.NTHValueExpr{Expr: expr, N: n, FromFirstLastClause: fl, NullTreatmentClause: nt, OverClause: oc}, nil
}

func (p *Parser) parseLagLeadCall(isLead bool) (sqlast.Expr, error) {
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	n, def, err := p.parseOptionalLagLeadArgs()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	nt, err := p.parseOptionalNullTreatment()
	if err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	typ := sqlast.LagExprType
	if isLead {
		typ = sqlast.LeadExprType
	}

	return &sqlast.LagLeadExpr{Type: typ, Expr: expr, N: n, Default: def, NullTreatmentClause: nt, OverClause: oc}, nil
}

func (p *Parser) parseOptionalLagLeadArgs() (sqlast.Expr, sqlast.Expr, error) {
	ok, err := p.consume(COMMA)
	if err != nil || !ok {
		return nil, nil, err
	}

	n, err := p.parseExpr()
	if err != nil {
		return nil, nil, err
	}

	ok, err = p.consume(COMMA)
	if err != nil || !ok {
		return n, nil, err
	}

	def, err := p.parseExpr()
	if err != nil {
		return nil, nil, err
	}

	return n, def, nil
}

func (p *Parser) parseJSONObjectAggCall() (sqlast.Expr, error) {
	key, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(COMMA); err != nil {
		return nil, err
	}

	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	oc, err := p.parseOptionalOverClause()
	if err != nil {
		return nil, err
	}

	return &sqlast.JSONObjectAgg{Key: key, Value: val, OverClause: oc}, nil
}

func (p *Parser) parseOptionalNullTreatment() (*sqlast.NullTreatmentClause, error) {
	switch {
	case p.at(RESPECT):
		if err := p.advance(); err != nil {
			return nil, err
		}

		if err := p.expect(NULLS); err != nil {
			return nil, err
		}

		return &sqlast.NullTreatmentClause{Type: sqlast.RespectNullsType}, nil
	case p.at(IGNORE):
		if err := p.advance(); err != nil {
			return nil, err
		}

		if err := p.expect(NULLS); err != nil {
			return nil, err
		}

		return &sqlast.NullTreatmentClause{Type: sqlast.IgnoreNullsType}, nil
	default:
		return nil, nil
	}
}

func (p *Parser) parseOptionalFromFirstLast() (*sqlast.FromFirstLastClause, error) {
	if ok, err := p.consume(FROM); err != nil || !ok {
		return nil, err
	}

	switch {
	case p.at(FIRST):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FromFirstLastClause{Type: sqlast.FromFirstType}, nil
	case p.at(LAST):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FromFirstLastClause{Type: sqlast.FromLastType}, nil
	default:
		return nil, p.errorf("expected FIRST or LAST after FROM")
	}
}

func (p *Parser) parseOptionalOverClause() (*sqlast.OverClause, error) {
	if ok, err := p.consume(OVER); err != nil || !ok {
		return nil, err
	}

	if p.at(IDENT) {
		name := p.tok.Literal

		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.OverClause{WindowName: sqlast.ColIdent(name)}, nil
	}

	if err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	spec, err := p.parseWindowSpecification()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.OverClause{WindowSpec: spec}, nil
}

func (p *Parser) parseWindowSpecification() (*sqlast.WindowSpecification, error) {
	spec := &sqlast.WindowSpecification{}

	if ok, err := p.consume(PARTITION); err != nil {
		return nil, err
	} else if ok {
		if err := p.expect(BY); err != nil {
			return nil, err
		}

		exprs, err := p.parseExprList()
		if err != nil {
			return nil, err
		}

		spec.PartitionClause = exprs
	}

	if p.at(ORDER) {
		ob, err := p.parseOrderByClause()
		if err != nil {
			return nil, err
		}

		spec.OrderClause = ob
	}

	if p.at(ROWS) || p.at(RANGE) {
		fc, err := p.parseFrameClause()
		if err != nil {
			return nil, err
		}

		spec.FrameClause = fc
	}

	return spec, nil
}

func (p *Parser) parseFrameClause() (*sqlast.FrameClause, error) {
	unit := sqlast.FrameRowsType
	if p.at(RANGE) {
		unit = sqlast.FrameRangeType
	}

	if err := p.advance(); err != nil { // consume ROWS/RANGE
		return nil, err
	}

	if ok, err := p.consume(BETWEEN); err != nil {
		return nil, err
	} else if ok {
		start, err := p.parseFramePoint()
		if err != nil {
			return nil, err
		}

		if err := p.expect(AND); err != nil {
			return nil, err
		}

		end, err := p.parseFramePoint()
		if err != nil {
			return nil, err
		}

		return &sqlast.FrameClause{Unit: unit, Start: start, End: end}, nil
	}

	start, err := p.parseFramePoint()
	if err != nil {
		return nil, err
	}

	return &sqlast.FrameClause{Unit: unit, Start: start}, nil
}

func (p *Parser) parseFramePoint() (*sqlast.FramePoint, error) {
	switch {
	case p.at(CURRENT):
		if err := p.advance(); err != nil {
			return nil, err
		}

		if err := p.expect(ROW); err != nil {
			return nil, err
		}

		return &sqlast.FramePoint{Type: sqlast.CurrentRowType}, nil
	case p.at(UNBOUNDED):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return p.parseUnboundedFramePoint()
	default:
		return p.parseExprFramePoint()
	}
}

func (p *Parser) parseUnboundedFramePoint() (*sqlast.FramePoint, error) {
	switch {
	case p.at(PRECEDING):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FramePoint{Type: sqlast.UnboundedPrecedingType}, nil
	case p.at(FOLLOWING):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FramePoint{Type: sqlast.UnboundedFollowingType}, nil
	default:
		return nil, p.errorf("expected PRECEDING or FOLLOWING after UNBOUNDED")
	}
}

func (p *Parser) parseExprFramePoint() (*sqlast.FramePoint, error) {
	expr, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	switch {
	case p.at(PRECEDING):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FramePoint{Type: sqlast.ExprPrecedingType, Expr: expr}, nil
	case p.at(FOLLOWING):
		if err := p.advance(); err != nil {
			return nil, err
		}

		return &sqlast.FramePoint{Type: sqlast.ExprFollowingType, Expr: expr}, nil
	default:
		return nil, p.errorf("expected PRECEDING or FOLLOWING")
	}
}
