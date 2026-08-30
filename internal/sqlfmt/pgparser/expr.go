package pgparser

import (
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// parseExpr parses a SQL expression, starting from the loosest-binding
// precedence level (OR) down to the tightest (primary expressions).
//
// Precedence, loosest to tightest (per
// https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-PRECEDENCE):
// OR > AND > NOT > IS/ISNULL/NOTNULL/IS DISTINCT FROM > BETWEEN/IN/LIKE/ILIKE
// and plain comparison (flattened into one predicate tier, matching how
// internal/sqlfmt/parser's MySQL grammar already flattens its own predicate
// forms) > JSON/JSONB operators > + - > * / % > unary +/- > :: cast >
// primary (including array subscript, which is resolved inside cast's type-
// name grammar rather than as a separate postfix tier — see parseTypeName).
func (p *Parser) parseExpr() pgast.Expr {
	return p.parseOrExpr()
}

func (p *Parser) parseOrExpr() pgast.Expr {
	left := p.parseAndExpr()

	for p.consume(OR) {
		left = &pgast.OrExpr{Left: left, Right: p.parseAndExpr()}
	}

	return left
}

func (p *Parser) parseAndExpr() pgast.Expr {
	left := p.parseNotExpr()

	for p.consume(AND) {
		left = &pgast.AndExpr{Left: left, Right: p.parseNotExpr()}
	}

	return left
}

func (p *Parser) parseNotExpr() pgast.Expr {
	if !p.consume(NOT) {
		return p.parseComparisonExpr()
	}

	return &pgast.NotExpr{Expr: p.parseNotExpr()}
}

var simpleComparisonOps = map[TokenType]pgast.ComparisonOperator{
	EQ: pgast.EqualOp,
	NE: pgast.NotEqualOp,
	LT: pgast.LessThanOp,
	GT: pgast.GreaterThanOp,
	LE: pgast.LessEqualOp,
	GE: pgast.GreaterEqualOp,
}

// parseComparisonExpr parses a single predicate: a plain comparison, or one
// of the [NOT] IN / [NOT] BETWEEN / [NOT] LIKE / [NOT] ILIKE / IS [NOT] NULL
// / IS [NOT] DISTINCT FROM / ISNULL / NOTNULL forms.
func (p *Parser) parseComparisonExpr() pgast.Expr {
	return p.parsePredicateSuffix(p.parseJSONExpr())
}

func (p *Parser) parsePredicateSuffix(left pgast.Expr) pgast.Expr {
	not := p.consumeNegatedPredicateKeyword()

	// not is always consistent with one of the three negatable predicate
	// cases below: consumeNegatedPredicateKeyword only consumes NOT when the
	// following token is already IN/BETWEEN/LIKE/ILIKE, so there is no case
	// where not is true but none of them match.
	switch {
	case p.at(IN):
		return p.parseInExpr(left, not)
	case p.at(BETWEEN):
		return p.parseBetweenExpr(left, not)
	case p.at(LIKE):
		return p.parseLikeExpr(left, not)
	case p.at(ILIKE):
		return p.parseILikeExpr(left, not)
	case p.at(IS):
		return p.parseIsExpr(left)
	case p.at(ISNULL):
		p.advance()

		return &pgast.IsExpr{Expr: left}
	case p.at(NOTNULL):
		p.advance()

		return &pgast.IsExpr{Not: true, Expr: left}
	default:
		return p.parseOptionalSimpleComparison(left)
	}
}

// consumeNegatedPredicateKeyword consumes a leading NOT that belongs to a
// [NOT] IN / [NOT] BETWEEN / [NOT] LIKE / [NOT] ILIKE predicate, as opposed
// to a prefix logical NOT (already handled by parseNotExpr, above this
// precedence level).
func (p *Parser) consumeNegatedPredicateKeyword() bool {
	if !p.at(NOT) || (!p.peekAt(IN) && !p.peekAt(BETWEEN) && !p.peekAt(LIKE) && !p.peekAt(ILIKE)) {
		return false
	}

	p.advance()

	return true
}

func (p *Parser) parseOptionalSimpleComparison(left pgast.Expr) pgast.Expr {
	op, ok := simpleComparisonOps[p.tok.Type]
	if !ok {
		return left
	}

	p.advance()

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: p.parseJSONExpr()}
}

func (p *Parser) parseInExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume IN
	p.expect(LPAREN)

	op := pgast.InOp
	if not {
		op = pgast.NotInOp
	}

	if p.at(SELECT) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &pgast.ComparisonExpr{Left: left, Operator: op, Right: &pgast.Subquery{Select: sel}}
	}

	values := p.parseExprList()
	p.expect(RPAREN)

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: pgast.ValTuple(values)}
}

func (p *Parser) parseBetweenExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume BETWEEN

	from := p.parseJSONExpr()
	p.expect(AND)

	to := p.parseJSONExpr()

	return &pgast.RangeCond{Not: not, Left: left, From: from, To: to}
}

func (p *Parser) parseLikeExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume LIKE

	right := p.parseJSONExpr()

	op := pgast.LikeOp
	if not {
		op = pgast.NotLikeOp
	}

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: right}
}

func (p *Parser) parseILikeExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume ILIKE

	return &pgast.ILikeExpr{Not: not, Left: left, Right: p.parseJSONExpr()}
}

// parseIsExpr parses the IS-prefixed predicate forms: IS [NOT] NULL and
// IS [NOT] DISTINCT FROM expr. l.tok must be IS.
func (p *Parser) parseIsExpr(left pgast.Expr) pgast.Expr {
	p.advance() // consume IS

	not := p.consume(NOT)

	if p.consume(DISTINCT) {
		p.expect(FROM)

		return &pgast.IsDistinctFromExpr{Not: not, Left: left, Right: p.parseJSONExpr()}
	}

	p.expect(NULL)

	return &pgast.IsExpr{Not: not, Expr: left}
}

// jsonOps maps each JSON/JSONB operator token to its pgast.JSONOp value.
var jsonOps = map[TokenType]pgast.JSONOp{
	Arrow:         pgast.JSONArrow,
	ArrowText:     pgast.JSONArrowText,
	HashArrow:     pgast.JSONHashArrow,
	HashArrowText: pgast.JSONHashArrowText,
	Contains:      pgast.JSONContains,
	ContainedBy:   pgast.JSONContainedBy,
	JSONExists:    pgast.JSONExists,
	JSONExistsAny: pgast.JSONExistsAny,
	JSONExistsAll: pgast.JSONExistsAll,
}

// parseJSONExpr parses PostgreSQL's JSON/JSONB operators (->, ->>, #>, #>>,
// @>, <@, ?, ?|, ?&), which bind tighter than any predicate but looser than
// the arithmetic operators below them.
func (p *Parser) parseJSONExpr() pgast.Expr {
	left := p.parseAdditiveExpr()

	for {
		op, ok := jsonOps[p.tok.Type]
		if !ok {
			return left
		}

		p.advance()

		left = &pgast.JSONOpExpr{Left: left, Op: op, Right: p.parseAdditiveExpr()}
	}
}

var arithmeticOps = map[TokenType]pgast.ArithmeticOperator{
	PLUS:    pgast.PlusOp,
	MINUS:   pgast.MinusOp,
	STAR:    pgast.MultOp,
	SLASH:   pgast.DivOp,
	PERCENT: pgast.ModOp,
}

func (p *Parser) parseAdditiveExpr() pgast.Expr {
	left := p.parseMultiplicativeExpr()

	for p.at(PLUS) || p.at(MINUS) {
		op := arithmeticOps[p.tok.Type]
		p.advance()

		left = &pgast.ArithmeticExpr{Left: left, Operator: op, Right: p.parseMultiplicativeExpr()}
	}

	return left
}

func (p *Parser) parseMultiplicativeExpr() pgast.Expr {
	left := p.parseUnaryExpr()

	for p.at(STAR) || p.at(SLASH) || p.at(PERCENT) {
		op := arithmeticOps[p.tok.Type]
		p.advance()

		left = &pgast.ArithmeticExpr{Left: left, Operator: op, Right: p.parseUnaryExpr()}
	}

	return left
}

func (p *Parser) parseUnaryExpr() pgast.Expr {
	var op pgast.UnaryOperator

	switch {
	case p.at(PLUS):
		op = pgast.UPlusOp
	case p.at(MINUS):
		op = pgast.UMinusOp
	default:
		return p.parseCastExpr()
	}

	p.advance()

	return &pgast.UnaryExpr{Operator: op, Expr: p.parseUnaryExpr()}
}

// parseCastExpr parses zero or more trailing "::type" casts. Array
// subscripting (a[1], a[1:3]) is not a separate postfix tier here: a bare
// "a::int[1]" is PostgreSQL type syntax for a cast to an array type, so the
// "[1]" is consumed by parseTypeName as part of the type operand, not as a
// subscript of the cast's result. An explicit subscript of a cast result,
// e.g. "(a::int)[1]", still works via the parens forcing parsePrimaryExpr to
// finish the cast before parseSubscriptExpr sees the "[1]".
func (p *Parser) parseCastExpr() pgast.Expr {
	left := p.parseSubscriptExpr()

	for p.consume(DoubleColon) {
		left = &pgast.CastExpr{Expr: left, Type: p.parseTypeName()}
	}

	return left
}

func (p *Parser) parseSubscriptExpr() pgast.Expr {
	left := p.parsePrimaryExpr()

	for p.at(LBRACKET) {
		left = p.parseSubscript(left)
	}

	return left
}

// parseSubscript parses a single "[expr]" or "[expr:expr]" (slice, with
// either bound optional) array subscript applied to left. l.tok must be '['.
func (p *Parser) parseSubscript(left pgast.Expr) pgast.Expr {
	p.advance() // consume '['

	var from, to pgast.Expr

	if !p.at(COLON) {
		from = p.parseExpr()
	}

	if !p.consume(COLON) {
		p.expect(RBRACKET)

		return &pgast.ArraySubscriptExpr{Expr: left, From: from}
	}

	if !p.at(RBRACKET) {
		to = p.parseExpr()
	}

	p.expect(RBRACKET)

	return &pgast.ArraySubscriptExpr{Expr: left, From: from, To: to}
}

// parseTypeName parses a cast's target type: an identifier, optionally
// followed by a parenthesized numeric modifier list (numeric(10, 2),
// varchar(255)) and/or array-bounds suffixes (int[], int[3]). PostgreSQL's
// full type-name grammar also supports multi-word built-in names ("double
// precision", "timestamp with time zone") and schema qualification; those
// are out of scope here; the formatter treats a cast's type as an opaque
// string it round-trips, not a structure it validates.
func (p *Parser) parseTypeName() pgast.ColIdent {
	var b strings.Builder

	b.WriteString(p.readTypeIdent())

	if p.at(LPAREN) {
		b.WriteString(p.readTypeModifier())
	}

	for p.at(LBRACKET) {
		b.WriteString(p.readArrayBound())
	}

	return pgast.ColIdent(b.String())
}

func (p *Parser) readTypeIdent() string {
	if p.tok.Type != IDENT && p.tok.Type != QuotedIdent {
		return failReturn[string](p, "expected type name, got %s", p.tok.Type)
	}

	lit := p.tok.Literal
	p.advance()

	return lit
}

func (p *Parser) readTypeModifier() string {
	p.advance() // consume '('

	var parts []string

	for {
		if p.tok.Type != INT {
			p.failf("expected integer in type modifier, got %s", p.tok.Type)
		}

		parts = append(parts, p.tok.Literal)
		p.advance()

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return "(" + strings.Join(parts, ", ") + ")"
}

func (p *Parser) readArrayBound() string {
	p.advance() // consume '['

	bound := ""
	if p.tok.Type == INT {
		bound = p.tok.Literal
		p.advance()
	}

	p.expect(RBRACKET)

	return "[" + bound + "]"
}

// keywordLiterals holds the keyword tokens that stand for a literal value.
var keywordLiterals = map[TokenType]string{
	NULL:  "NULL",
	TRUE:  "TRUE",
	FALSE: "FALSE",
}

func (p *Parser) parsePrimaryExpr() pgast.Expr {
	if expr, ok := p.tryParseLiteral(); ok {
		return expr
	}

	switch p.tok.Type {
	case LPAREN:
		return p.parseParenExprOrSubquery()
	case CASE:
		return p.parseCaseExpr()
	case EXISTS:
		return p.parseExistsExpr()
	case CAST:
		return p.parseCastCall()
	case ARRAY:
		return p.parseArrayLiteral()
	case IDENT, QuotedIdent:
		return p.parseIdentExpr()
	default:
		return failReturn[pgast.Expr](p, "unexpected token %s in expression", p.tok.Type)
	}
}

// tryParseLiteral parses p.tok if it starts one of the literal forms (a
// keyword literal, number, placeholder, or string), reporting ok=false
// without consuming anything otherwise. Split out of parsePrimaryExpr to
// keep its cyclomatic complexity down.
func (p *Parser) tryParseLiteral() (pgast.Expr, bool) {
	if text, ok := keywordLiterals[p.tok.Type]; ok {
		p.advance()

		return &pgast.Literal{Val: text}, true
	}

	switch p.tok.Type {
	case INT, FLOAT:
		return p.parseLiteralToken(), true
	case Placeholder:
		return p.parsePlaceholder(), true
	case STRING:
		return p.parseStringLiteral(), true
	default:
		return nil, false
	}
}

func (p *Parser) parseLiteralToken() pgast.Expr {
	tok := p.tok
	p.advance()

	return &pgast.Literal{Val: tok.Literal}
}

func (p *Parser) parsePlaceholder() pgast.Expr {
	tok := p.tok
	p.advance()

	return &pgast.Literal{Val: "$" + tok.Literal}
}

// parseStringLiteral re-encodes a lexer-decoded string (which may have come
// from a plain '...', an E'...' escape string, or a $tag$...$tag$
// dollar-quoted string) as a plain '...' literal with doubled-quote
// escaping. standard_conforming_strings=on means no backslash escaping is
// ever needed on the way back out, unlike internal/sqlfmt/parser's MySQL
// literal re-encoding, which has a ModeDefault/ModeNoBackslashEscapes split.
func (p *Parser) parseStringLiteral() pgast.Expr {
	tok := p.tok
	p.advance()

	return &pgast.Literal{Val: "'" + strings.ReplaceAll(tok.Literal, "'", "''") + "'"}
}

func (p *Parser) parseParenExprOrSubquery() pgast.Expr {
	p.advance() // consume '('

	if p.at(SELECT) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &pgast.Subquery{Select: sel}
	}

	expr := p.parseExpr()
	p.expect(RPAREN)

	return &pgast.ParenExpr{Expr: expr}
}

func (p *Parser) parseCaseExpr() pgast.Expr {
	p.advance() // consume CASE

	caseExpr := p.parseOptionalCaseValue()
	whens := p.parseWhenClauses()
	elseExpr := p.parseOptionalElse()

	p.expect(END)

	return &pgast.CaseExpr{Expr: caseExpr, Whens: whens, Else: elseExpr}
}

// parseOptionalCaseValue parses the optional `expr` in a simple CASE expr
// (`CASE expr WHEN ...`); a searched CASE (`CASE WHEN ...`) has none.
func (p *Parser) parseOptionalCaseValue() pgast.Expr {
	if p.at(WHEN) {
		return nil
	}

	return p.parseExpr()
}

func (p *Parser) parseWhenClauses() []*pgast.When {
	var whens []*pgast.When

	for p.at(WHEN) {
		whens = append(whens, p.parseWhen())
	}

	if len(whens) == 0 {
		p.failf("expected WHEN in CASE expression")
	}

	return whens
}

func (p *Parser) parseOptionalElse() pgast.Expr {
	if !p.consume(ELSE) {
		return nil
	}

	return p.parseExpr()
}

func (p *Parser) parseWhen() *pgast.When {
	p.advance() // consume WHEN

	cond := p.parseExpr()
	p.expect(THEN)

	val := p.parseExpr()

	return &pgast.When{Cond: cond, Val: val}
}

func (p *Parser) parseExistsExpr() pgast.Expr {
	p.advance() // consume EXISTS
	p.expect(LPAREN)

	sel := p.parseSubqueryStatement()

	p.expect(RPAREN)

	return &pgast.ExistsExpr{Subquery: &pgast.Subquery{Select: sel}}
}

func (p *Parser) parseCastCall() pgast.Expr {
	p.advance() // consume CAST
	p.expect(LPAREN)

	expr := p.parseExpr()
	p.expect(AS)

	typeName := p.parseTypeName()

	p.expect(RPAREN)

	return &pgast.CastExpr{Expr: expr, Type: typeName, Explicit: true}
}

func (p *Parser) parseArrayLiteral() pgast.Expr {
	p.advance() // consume ARRAY
	p.expect(LBRACKET)

	if p.consume(RBRACKET) {
		return &pgast.ArrayExpr{}
	}

	elems := p.parseExprList()
	p.expect(RBRACKET)

	return &pgast.ArrayExpr{Elems: elems}
}

// parseIdentExpr parses a column reference or function call, with an
// optional qualifier: name, qualifier.name, name(args...), or
// qualifier.name(args...).
func (p *Parser) parseIdentExpr() pgast.Expr {
	name := p.readIdent()

	if p.at(LPAREN) {
		return p.parseFuncCall("", name)
	}

	if !p.consume(DOT) {
		return &pgast.ColName{Name: pgast.ColIdent(name)}
	}

	second := p.readIdent()

	if p.at(LPAREN) {
		return p.parseFuncCall(name, second)
	}

	return &pgast.ColName{Qualifier: pgast.TableName{Name: pgast.TableIdent(name)}, Name: pgast.ColIdent(second)}
}

// parseFuncCall parses a function call's argument list, DISTINCT flag or
// star form, and trailing FILTER (WHERE ...) clause. l.tok must be '('.
func (p *Parser) parseFuncCall(qualifier, name string) pgast.Expr {
	p.advance() // consume '('

	f := &pgast.FuncExpr{Qualifier: pgast.TableIdent(qualifier), Name: pgast.ColIdent(name)}

	switch {
	case p.consume(STAR):
		f.Star = true
	case p.at(RPAREN):
	default:
		f.Distinct = p.consume(DISTINCT)
		f.Args = p.parseExprList()
	}

	p.expect(RPAREN)

	if p.consume(FILTER) {
		p.expect(LPAREN)
		p.expect(WHERE)

		f.Filter = &pgast.Where{Expr: p.parseExpr()}

		p.expect(RPAREN)
	}

	f.Over = p.parseOptionalOver()

	return f
}

// parseExprList parses a comma-separated list of expressions.
func (p *Parser) parseExprList() []pgast.Expr {
	var exprs []pgast.Expr

	for {
		exprs = append(exprs, p.parseExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}
