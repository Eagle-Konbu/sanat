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
// OR > AND > NOT > IS/ISNULL/NOTNULL > plain comparison > BETWEEN/IN/LIKE/
// ILIKE > any other operator > + - > * / % > ^ > unary +/- > :: cast >
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
		return p.parseIsPredicateExpr()
	}

	return &pgast.NotExpr{Expr: p.parseNotExpr()}
}

// parseIsPredicateExpr parses a comparison followed by any number of
// postfix IS [NOT] NULL / TRUE / FALSE / UNKNOWN, IS [NOT] DISTINCT FROM,
// ISNULL, or NOTNULL tests, so "a = b IS NULL" is "(a = b) IS NULL".
func (p *Parser) parseIsPredicateExpr() pgast.Expr {
	left := p.parseComparisonExpr()

	for {
		switch {
		case p.at(IS):
			left = p.parseIsSuffix(left)
		case p.consume(ISNULL):
			left = &pgast.IsExpr{Expr: left}
		case p.consume(NOTNULL):
			left = &pgast.IsExpr{Not: true, Expr: left}
		default:
			return left
		}
	}
}

var truthValues = map[TokenType]pgast.TruthValue{
	TRUE:  pgast.TruthTrue,
	FALSE: pgast.TruthFalse,
}

// parseIsSuffix parses the IS-prefixed predicate forms applied to left.
// p.tok must be IS.
func (p *Parser) parseIsSuffix(left pgast.Expr) pgast.Expr {
	p.advance() // consume IS

	not := p.consume(NOT)

	if p.consume(DISTINCT) {
		p.expect(FROM)

		return &pgast.IsDistinctFromExpr{Not: not, Left: left, Right: p.parseComparisonExpr()}
	}

	if p.consume(NULL) {
		return &pgast.IsExpr{Not: not, Expr: left}
	}

	return &pgast.IsTruthExpr{Not: not, Expr: left, Value: p.readTruthValue()}
}

// readTruthValue reads the TRUE / FALSE / UNKNOWN operand of an IS test.
// UNKNOWN is matched as a plain identifier because the lexer has no keyword
// token for it.
func (p *Parser) readTruthValue() pgast.TruthValue {
	if value, ok := truthValues[p.tok.Type]; ok {
		p.advance()

		return value
	}

	if p.at(IDENT) && p.tok.Literal == "unknown" {
		p.advance()

		return pgast.TruthUnknown
	}

	return failReturn[pgast.TruthValue](p,
		"expected NULL, TRUE, FALSE, UNKNOWN, or DISTINCT FROM after IS, got %s", p.tok.Type)
}

var simpleComparisonOps = map[TokenType]pgast.ComparisonOperator{
	EQ: pgast.EqualOp,
	NE: pgast.NotEqualOp,
	LT: pgast.LessThanOp,
	GT: pgast.GreaterThanOp,
	LE: pgast.LessEqualOp,
	GE: pgast.GreaterEqualOp,
}

// parseComparisonExpr parses an optional plain comparison. Like PostgreSQL,
// it is non-associative: "a = b = c" is a syntax error.
func (p *Parser) parseComparisonExpr() pgast.Expr {
	left := p.parseRangePredicateExpr()

	op, ok := simpleComparisonOps[p.tok.Type]
	if !ok {
		return left
	}

	p.advance()

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: p.parseRangePredicateExpr()}
}

// parseRangePredicateExpr parses an optional [NOT] IN / [NOT] BETWEEN /
// [NOT] LIKE / [NOT] ILIKE predicate, which binds tighter than a plain
// comparison, so "a IN (1) = b" is "(a IN (1)) = b".
func (p *Parser) parseRangePredicateExpr() pgast.Expr {
	left := p.parseOtherOpExpr()
	not := p.consumeNegatedPredicateKeyword()

	// not is always consistent with one of the cases below:
	// consumeNegatedPredicateKeyword only consumes NOT when the following
	// token is already IN/BETWEEN/LIKE/ILIKE.
	switch {
	case p.at(IN):
		return p.parseInExpr(left, not)
	case p.at(BETWEEN):
		return p.parseBetweenExpr(left, not)
	case p.at(LIKE):
		return p.parseLikeExpr(left, not)
	case p.at(ILIKE):
		return p.parseILikeExpr(left, not)
	default:
		return left
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

// parseInExpr parses the "IN (...)" suffix of a [NOT] IN predicate applied to
// left: a subquery or set operation, or a parenthesized value list.
func (p *Parser) parseInExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume IN
	p.expect(LPAREN)

	op := pgast.InOp
	if not {
		op = pgast.NotInOp
	}

	if isQueryStart(p.tok.Type) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &pgast.ComparisonExpr{Left: left, Operator: op, Right: &pgast.Subquery{Select: sel}}
	}

	values := p.parseExprList()

	if len(values) == 1 {
		if query := p.tryContinueQueryFromExpr(values[0]); query != nil {
			p.expect(RPAREN)

			return &pgast.ComparisonExpr{Left: left, Operator: op, Right: &pgast.Subquery{Select: query}}
		}

		// "IN ((SELECT ...))" is still a subquery, not a one-item value list.
		if sub := unwrapSubquery(values[0]); sub != nil {
			p.expect(RPAREN)

			return &pgast.ComparisonExpr{Left: left, Operator: op, Right: sub}
		}
	}

	p.expect(RPAREN)

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: pgast.ValTuple(values)}
}

func (p *Parser) parseBetweenExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume BETWEEN

	from := p.parseOtherOpExpr()
	p.expect(AND)

	to := p.parseOtherOpExpr()

	return &pgast.RangeCond{Not: not, Left: left, From: from, To: to}
}

func (p *Parser) parseLikeExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume LIKE

	op := pgast.LikeOp
	if not {
		op = pgast.NotLikeOp
	}

	right := p.parseOtherOpExpr()

	return &pgast.ComparisonExpr{Left: left, Operator: op, Right: right, Escape: p.parseOptionalEscape()}
}

func (p *Parser) parseILikeExpr(left pgast.Expr, not bool) pgast.Expr {
	p.advance() // consume ILIKE

	right := p.parseOtherOpExpr()

	return &pgast.ILikeExpr{Not: not, Left: left, Right: right, Escape: p.parseOptionalEscape()}
}

// parseOptionalEscape parses a LIKE/ILIKE pattern's optional trailing
// "ESCAPE expr", returning nil if none is present.
func (p *Parser) parseOptionalEscape() pgast.Expr {
	if !p.consume(ESCAPE) {
		return nil
	}

	return p.parseOtherOpExpr()
}

// parseOtherOpExpr parses PostgreSQL's generic operator level: every
// operator without a precedence of its own (||, ~, @>, ->, <->, a
// user-defined operator, or OPERATOR(...)). They are left-associative and
// bind tighter than any predicate but looser than + and -.
func (p *Parser) parseOtherOpExpr() pgast.Expr {
	left := p.parseAdditiveExpr()

	for {
		op, ok := p.tryParseOperator()
		if !ok {
			return left
		}

		left = &pgast.OperatorExpr{Left: left, Operator: op, Right: p.parseAdditiveExpr()}
	}
}

// parsePrefixOperatorExpr parses a generic prefix operator and its operand,
// such as "~ a" or "@ -1". The current token must be an Op. As in
// PostgreSQL, the operand extends over + and - and tighter operators, so
// "~ a + b" is "~ (a + b)" while "~ a || b" is "(~ a) || b".
func (p *Parser) parsePrefixOperatorExpr() pgast.Expr {
	op := pgast.Operator{Symbol: p.tok.Literal}
	p.advance()

	return &pgast.OperatorExpr{Operator: op, Right: p.parseAdditiveExpr()}
}

// tryParseOperator consumes a generic infix operator, an Op token or
// "OPERATOR([schema.]op)", reporting ok=false without consuming anything if
// the current token is neither.
func (p *Parser) tryParseOperator() (pgast.Operator, bool) {
	switch {
	case p.at(Op):
		op := pgast.Operator{Symbol: p.tok.Literal}
		p.advance()

		return op, true
	case p.at(OPERATOR) && p.peekAt(LPAREN):
		return p.parseExplicitOperator(), true
	default:
		return pgast.Operator{}, false
	}
}

// parseExplicitOperator parses "OPERATOR([schema.]op)", where op may be any
// operator, including the fixed ones such as + or =. The current token must
// be OPERATOR.
func (p *Parser) parseExplicitOperator() pgast.Operator {
	p.advance() // consume OPERATOR
	p.advance() // consume '('

	op := pgast.Operator{Explicit: true}

	if isColID(p.tok) {
		op.Schema = pgast.TableIdent(p.readColID())
		p.expect(DOT)
	}

	if !isOperatorToken(p.tok.Type) {
		p.failf("expected operator, got %s", p.tok.Type)
	}

	op.Symbol = p.tok.Literal
	p.advance()
	p.expect(RPAREN)

	return op
}

func isOperatorToken(tt TokenType) bool {
	switch tt {
	case Op, EQ, NE, LT, GT, LE, GE, PLUS, MINUS, STAR, SLASH, PERCENT, CARET:
		return true
	default:
		return false
	}
}

var arithmeticOps = map[TokenType]pgast.ArithmeticOperator{
	PLUS:    pgast.PlusOp,
	MINUS:   pgast.MinusOp,
	STAR:    pgast.MultOp,
	SLASH:   pgast.DivOp,
	PERCENT: pgast.ModOp,
	CARET:   pgast.ExpOp,
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
	left := p.parseExponentExpr()

	for p.at(STAR) || p.at(SLASH) || p.at(PERCENT) {
		op := arithmeticOps[p.tok.Type]
		p.advance()

		left = &pgast.ArithmeticExpr{Left: left, Operator: op, Right: p.parseExponentExpr()}
	}

	return left
}

// parseExponentExpr parses '^', which PostgreSQL makes left-associative
// ("2 ^ 3 ^ 2" is "(2 ^ 3) ^ 2") and looser than unary minus ("-2 ^ 2" is
// "(-2) ^ 2"), unlike the usual mathematical convention.
func (p *Parser) parseExponentExpr() pgast.Expr {
	left := p.parseUnaryExpr()

	for p.consume(CARET) {
		left = &pgast.ArithmeticExpr{Left: left, Operator: pgast.ExpOp, Right: p.parseUnaryExpr()}
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

	return &pgast.ArraySubscriptExpr{Expr: left, From: from, To: to, Slice: true}
}

// identText returns an identifier token as SQL text: a QuotedIdent is
// re-quoted (doubling any embedded quotes) so its case and quoting survive
// into the AST, per the pgast.ColIdent contract.
func identText(tok Token) string {
	if tok.Type.IsKeyword() {
		return asciiLower(tok.Literal)
	}

	if tok.Type != QuotedIdent {
		return tok.Literal
	}

	return `"` + strings.ReplaceAll(tok.Literal, `"`, `""`) + `"`
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

	if expr, ok := p.tryParseKeywordLedExpr(); ok {
		return expr
	}

	switch {
	case p.at(LPAREN):
		return p.parseParenExprOrSubquery()
	case p.at(Op):
		return p.parsePrefixOperatorExpr()
	case p.atIdentExprStart():
		if lit, ok := p.tryParseTypedLiteral(); ok {
			return lit
		}

		return p.parseIdentExpr()
	default:
		return failReturn[pgast.Expr](p, "unexpected token %s in expression", p.tok.Type)
	}
}

// tryParseKeywordLedExpr parses an expression introduced by its own keyword
// (CASE, EXISTS, CAST, ARRAY, or ROW), reporting ok=false without consuming
// anything if the current token starts none. ROW and EXISTS are col_name
// keywords, so they start their constructs only when "(" follows; otherwise
// they are column names, as in "SELECT row FROM t" or "row.x". Split out of
// parsePrimaryExpr to keep its cyclomatic complexity down.
func (p *Parser) tryParseKeywordLedExpr() (pgast.Expr, bool) {
	switch {
	case p.at(CASE):
		return p.parseCaseExpr(), true
	case p.at(EXISTS) && p.peekAt(LPAREN):
		return p.parseExistsExpr(), true
	case p.at(CAST):
		return p.parseCastCall(), true
	case p.at(ARRAY):
		return p.parseArrayLiteral(), true
	case p.at(ROW) && p.peekAt(LPAREN):
		return p.parseRowConstructor(), true
	default:
		return nil, false
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
	case INT, FLOAT, BitString:
		return p.parseLiteralToken(), true
	case Placeholder:
		return p.parsePlaceholder(), true
	case STRING:
		return p.parseStringLiteral(), true
	case NationalString:
		return &pgast.Literal{Val: "N" + p.readStringLiteralText()}, true
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
	return &pgast.Literal{Val: p.readStringLiteralText()}
}

// readStringLiteralText consumes a STRING or NationalString token and returns
// its content re-encoded as a plain '...' literal.
func (p *Parser) readStringLiteralText() string {
	text := quoteString(p.tok.Literal)
	p.advance()

	return text
}

func quoteString(content string) string {
	return "'" + strings.ReplaceAll(content, "'", "''") + "'"
}

// parseParenExprOrSubquery parses a parenthesized expression, row value, or
// scalar subquery (which may be a set operation).
func (p *Parser) parseParenExprOrSubquery() pgast.Expr {
	p.advance() // consume '('

	if isQueryStart(p.tok.Type) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &pgast.Subquery{Select: sel}
	}

	expr := p.parseExpr()

	if query := p.tryContinueQueryFromExpr(expr); query != nil {
		p.expect(RPAREN)

		return &pgast.Subquery{Select: query}
	}

	if p.consume(COMMA) {
		row := append([]pgast.Expr{expr}, p.parseExprList()...)
		p.expect(RPAREN)

		return pgast.ValTuple(row)
	}

	p.expect(RPAREN)

	return &pgast.ParenExpr{Expr: expr}
}

func (p *Parser) parseRowConstructor() pgast.Expr {
	p.advance() // consume ROW
	p.expect(LPAREN)

	if p.consume(RPAREN) {
		return &pgast.RowExpr{}
	}

	exprs := p.parseExprList()
	p.expect(RPAREN)

	return &pgast.RowExpr{Exprs: exprs}
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

// parseExistsExpr parses an "EXISTS (query)" predicate.
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

// atIdentExprStart reports whether the current token starts a column
// reference or a function call.
func (p *Parser) atIdentExprStart() bool {
	return isColID(p.tok) || isTypeFuncName(p.tok) && p.peekAt(LPAREN)
}

// parseIdentExpr parses a column reference or function call, with an
// optional qualifier: name, qualifier.name, name(args...), or
// qualifier.name(args...). An unqualified function name is a
// type_function_name, so it may be a keyword such as LEFT that cannot name a
// column; a name after the dot may be any keyword.
func (p *Parser) parseIdentExpr() pgast.Expr {
	if isTypeFuncName(p.tok) && p.peekAt(LPAREN) {
		return p.parseFuncCall("", p.readName(isTypeFuncName))
	}

	name := p.readColID()

	if !p.consume(DOT) {
		return &pgast.ColName{Name: pgast.ColIdent(name)}
	}

	second := p.readColLabel()

	if p.at(LPAREN) {
		return p.parseFuncCall(name, second)
	}

	return &pgast.ColName{Qualifier: pgast.TableName{Name: pgast.TableIdent(name)}, Name: pgast.ColIdent(second)}
}

// parseFuncCall parses a function call's argument list, DISTINCT flag or
// star form, and trailing FILTER (WHERE ...) and OVER clauses. l.tok must be
// '('.
func (p *Parser) parseFuncCall(qualifier, name string) pgast.Expr {
	p.advance() // consume '('

	f := &pgast.FuncExpr{Qualifier: pgast.TableIdent(qualifier), Name: pgast.ColIdent(name)}

	switch {
	case p.consume(STAR):
		f.Star = true
	case p.at(RPAREN):
	default:
		f.Distinct = p.consume(DISTINCT)
		f.Args = p.parseFuncArgList()
	}

	p.expect(RPAREN)

	if p.consume(FILTER) {
		p.expect(LPAREN)
		p.expect(WHERE)

		f.Filter = &pgast.Where{Expr: p.parseExpr()}

		p.expect(RPAREN)
	}

	if p.consume(OVER) {
		p.parseOverClause(f)
	}

	return f
}

// parseOverClause parses the window after OVER into f: either a bare window
// name or a parenthesized window specification.
func (p *Parser) parseOverClause(f *pgast.FuncExpr) {
	if !p.consume(LPAREN) {
		f.Over = &pgast.WindowSpec{Name: pgast.ColIdent(p.readColID())}
		f.OverRef = true

		return
	}

	f.Over = p.parseWindowSpecBody()

	p.expect(RPAREN)
}

// parseFuncArgList parses a function call's comma-separated arguments, each
// an expression or a named "name => value" (or "name := value") argument.
func (p *Parser) parseFuncArgList() []pgast.Expr {
	var args []pgast.Expr

	for {
		args = append(args, p.parseFuncArg())

		if !p.consume(COMMA) {
			return args
		}
	}
}

func (p *Parser) parseFuncArg() pgast.Expr {
	if !isTypeFuncName(p.tok) || !p.peekAt(EqualsGreater) && !p.peekAt(ColonEquals) {
		return p.parseExpr()
	}

	name := p.readName(isTypeFuncName)
	colonEquals := p.at(ColonEquals)
	p.advance() // consume "=>" or ":="

	return &pgast.NamedArgExpr{Name: pgast.ColIdent(name), Value: p.parseExpr(), ColonEquals: colonEquals}
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
