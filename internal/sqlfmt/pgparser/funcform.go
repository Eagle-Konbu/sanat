package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// This file parses the SQL-standard function forms whose arguments are
// separated by keywords (EXTRACT, SUBSTRING, POSITION, TRIM, OVERLAY) and the
// row OVERLAPS predicate. Their names are col_name keywords in PostgreSQL,
// which this lexer leaves as identifiers, so they are recognized by text
// when "(" follows.

// specialFuncForm returns the parser for a keyword-argument function's
// arguments, or nil if name is not one.
func specialFuncForm(name string) func(*Parser) pgast.Expr {
	switch name {
	case "extract":
		return (*Parser).parseExtract
	case "substring":
		return (*Parser).parseSubstring
	case "position":
		return (*Parser).parsePosition
	case "trim":
		return (*Parser).parseTrim
	case "overlay":
		return (*Parser).parseOverlay
	default:
		return nil
	}
}

// tryParseSpecialFuncForm parses one of the keyword-argument function forms,
// reporting ok=false without consuming anything if the current tokens don't
// start one.
func (p *Parser) tryParseSpecialFuncForm() (pgast.Expr, bool) {
	if !p.at(IDENT) || !p.peekAt(LPAREN) {
		return nil, false
	}

	parse := specialFuncForm(p.tok.Literal)
	if parse == nil {
		return nil, false
	}

	p.advance() // consume the name
	p.advance() // consume '('

	expr := parse(p)

	p.expect(RPAREN)

	return expr, true
}

// parseExtract parses "field FROM source". The field is an identifier or a
// string constant.
func (p *Parser) parseExtract() pgast.Expr {
	var field string

	if p.at(STRING) {
		field = p.readStringLiteralText()
	} else {
		field = p.readColLabel()
	}

	p.expect(FROM)

	return &pgast.ExtractExpr{Field: field, Source: p.parseExpr()}
}

// parseSubstring parses "s [FROM start] [FOR count]" (in either order),
// "s SIMILAR pattern ESCAPE escape", or a plain argument list.
func (p *Parser) parseSubstring() pgast.Expr {
	if p.at(RPAREN) {
		return &pgast.FuncExpr{Name: "substring"}
	}

	sub := &pgast.SubstringExpr{Expr: p.parseExpr()}

	switch {
	case p.consume(SIMILAR):
		sub.Similar = p.parseExpr()
		p.expect(ESCAPE)
		sub.Escape = p.parseExpr()
	case p.consume(FROM):
		sub.From = p.parseExpr()

		if p.consume(FOR) {
			sub.For = p.parseExpr()
		}
	case p.consume(FOR):
		sub.For = p.parseExpr()

		if p.consume(FROM) {
			sub.From = p.parseExpr()
		}
	default:
		return p.finishPlainFuncArgs("substring", sub.Expr)
	}

	return sub
}

// parsePosition parses "substring IN source". Both operands stop short of
// IN and the other predicates, as PostgreSQL's b_expr does.
func (p *Parser) parsePosition() pgast.Expr {
	substring := p.parseOtherOpExpr()
	p.expect(IN)

	return &pgast.PositionExpr{Substring: substring, Source: p.parseOtherOpExpr()}
}

var trimSides = map[TokenType]pgast.TrimSide{
	BOTH:     pgast.TrimBoth,
	LEADING:  pgast.TrimLeading,
	TRAILING: pgast.TrimTrailing,
}

// parseTrim parses "[side] [characters] FROM source, ...",
// "side source, ...", or a plain argument list.
func (p *Parser) parseTrim() pgast.Expr {
	trim := &pgast.TrimExpr{}

	if side, ok := trimSides[p.tok.Type]; ok {
		trim.Side = side
		p.advance()
	}

	if p.consume(FROM) {
		trim.From = true
		trim.Args = p.parseExprList()

		return trim
	}

	first := p.parseExpr()

	switch {
	case p.consume(FROM):
		trim.Characters = first
		trim.From = true
		trim.Args = p.parseExprList()
	case trim.Side != pgast.TrimDefault:
		trim.Args = append([]pgast.Expr{first}, p.parseOptionalCommaExprs()...)
	default:
		return p.finishPlainFuncArgs("trim", first)
	}

	return trim
}

// parseOverlay parses "s PLACING replacement FROM start [FOR count]" or a
// plain argument list.
func (p *Parser) parseOverlay() pgast.Expr {
	first := p.parseExpr()

	if !p.consume(PLACING) {
		return p.finishPlainFuncArgs("overlay", first)
	}

	overlay := &pgast.OverlayExpr{Expr: first, Placing: p.parseExpr()}

	p.expect(FROM)

	overlay.From = p.parseExpr()

	if p.consume(FOR) {
		overlay.For = p.parseExpr()
	}

	return overlay
}

// finishPlainFuncArgs finishes the ordinary comma-separated spelling of a
// keyword-argument function whose first argument has been parsed.
func (p *Parser) finishPlainFuncArgs(name string, first pgast.Expr) pgast.Expr {
	return &pgast.FuncExpr{Name: pgast.ColIdent(name), Args: append([]pgast.Expr{first}, p.parseOptionalCommaExprs()...)}
}

// parseOptionalCommaExprs parses ", expr" repeatedly.
func (p *Parser) parseOptionalCommaExprs() []pgast.Expr {
	if !p.consume(COMMA) {
		return nil
	}

	return p.parseExprList()
}

// parseOptionalOverlaps parses an "OVERLAPS (start, end)" following left,
// returning left unchanged if none follows. Both sides must be explicit
// two-element rows, as PostgreSQL requires.
func (p *Parser) parseOptionalOverlaps(left pgast.Expr) pgast.Expr {
	if !p.at(OVERLAPS) {
		return left
	}

	p.advance()

	right := p.parsePrimaryExpr()

	if !isOverlapsRow(left) || !isOverlapsRow(right) {
		p.failf("OVERLAPS requires a two-element row on each side")
	}

	return &pgast.OverlapsExpr{Left: left, Right: right}
}

func isOverlapsRow(expr pgast.Expr) bool {
	switch e := expr.(type) {
	case pgast.ValTuple:
		return len(e) == 2
	case *pgast.RowExpr:
		return len(e.Exprs) == 2
	default:
		return false
	}
}
