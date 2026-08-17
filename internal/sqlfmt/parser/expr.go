package parser

import (
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// parseExpr parses a SQL expression, starting from the loosest-binding
// precedence level (OR) down to the tightest (primary expressions).
func (p *Parser) parseExpr() sqlast.Expr {
	return p.parseOrExpr()
}

func (p *Parser) parseOrExpr() sqlast.Expr {
	left := p.parseAndExpr()

	for p.consume(OR) {
		left = &sqlast.OrExpr{Left: left, Right: p.parseAndExpr()}
	}

	return left
}

func (p *Parser) parseAndExpr() sqlast.Expr {
	left := p.parseNotExpr()

	for p.consume(AND) {
		left = &sqlast.AndExpr{Left: left, Right: p.parseNotExpr()}
	}

	return left
}

func (p *Parser) parseNotExpr() sqlast.Expr {
	if !p.consume(NOT) {
		return p.parseComparisonExpr()
	}

	return &sqlast.NotExpr{Expr: p.parseNotExpr()}
}

var simpleComparisonOps = map[TokenType]sqlast.ComparisonOperator{
	EQ:  sqlast.EqualOp,
	NE:  sqlast.NotEqualOp,
	NSE: sqlast.NullSafeEqualOp,
	LT:  sqlast.LessThanOp,
	GT:  sqlast.GreaterThanOp,
	LE:  sqlast.LessEqualOp,
	GE:  sqlast.GreaterEqualOp,
}

// parseComparisonExpr parses a single predicate: a plain comparison, or one
// of the [NOT] IN / [NOT] BETWEEN / [NOT] LIKE / IS [NOT] NULL forms.
func (p *Parser) parseComparisonExpr() sqlast.Expr {
	return p.parsePredicateSuffix(p.parseAdditiveExpr())
}

func (p *Parser) parsePredicateSuffix(left sqlast.Expr) sqlast.Expr {
	not := p.consumeNegatedPredicateKeyword()

	// not is always consistent with one of the four predicate cases below:
	// consumeNegatedPredicateKeyword only consumes NOT when the following
	// token is already IN/BETWEEN/LIKE/REGEXP/RLIKE, so there is no case
	// where not is true but none of them match.
	switch {
	case p.at(IN):
		return p.parseInExpr(left, not)
	case p.at(BETWEEN):
		return p.parseBetweenExpr(left, not)
	case p.at(LIKE):
		return p.parseLikeExpr(left, not)
	case p.at(REGEXP) || p.at(RLIKE):
		return p.parseRegexpExpr(left, not)
	case p.at(IS):
		return p.parseIsExpr(left)
	default:
		return p.parseOptionalSimpleComparison(left)
	}
}

// consumeNegatedPredicateKeyword consumes a leading NOT that belongs to a
// [NOT] IN / [NOT] BETWEEN / [NOT] LIKE / [NOT] REGEXP / [NOT] RLIKE
// predicate, as opposed to a prefix logical NOT (already handled by
// parseNotExpr, above this precedence level).
func (p *Parser) consumeNegatedPredicateKeyword() bool {
	if !p.at(NOT) ||
		(!p.peekAt(IN) && !p.peekAt(BETWEEN) && !p.peekAt(LIKE) && !p.peekAt(REGEXP) && !p.peekAt(RLIKE)) {
		return false
	}

	p.advance()

	return true
}

func (p *Parser) parseOptionalSimpleComparison(left sqlast.Expr) sqlast.Expr {
	op, ok := simpleComparisonOps[p.tok.Type]
	if !ok {
		return left
	}

	return p.parseSimpleComparison(left, op)
}

func (p *Parser) parseSimpleComparison(left sqlast.Expr, op sqlast.ComparisonOperator) sqlast.Expr {
	p.advance()

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: p.parseAdditiveExpr()}
}

func (p *Parser) parseInExpr(left sqlast.Expr, not bool) sqlast.Expr {
	p.advance() // consume IN
	p.expect(LPAREN)

	op := sqlast.InOp
	if not {
		op = sqlast.NotInOp
	}

	if p.at(SELECT) || p.at(WITH) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: &sqlast.Subquery{Select: sel}}
	}

	values := p.parseExprList()
	p.expect(RPAREN)

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: sqlast.ValTuple(values)}
}

func (p *Parser) parseBetweenExpr(left sqlast.Expr, not bool) sqlast.Expr {
	p.advance() // consume BETWEEN

	from := p.parseAdditiveExpr()
	p.expect(AND)

	to := p.parseAdditiveExpr()

	return &sqlast.RangeCond{Not: not, Left: left, From: from, To: to}
}

func (p *Parser) parseLikeExpr(left sqlast.Expr, not bool) sqlast.Expr {
	p.advance() // consume LIKE

	right := p.parseAdditiveExpr()

	op := sqlast.LikeOp
	if not {
		op = sqlast.NotLikeOp
	}

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: right}
}

func (p *Parser) parseRegexpExpr(left sqlast.Expr, not bool) sqlast.Expr {
	p.advance() // consume REGEXP/RLIKE

	right := p.parseAdditiveExpr()

	op := sqlast.RegexpOp
	if not {
		op = sqlast.NotRegexpOp
	}

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: right}
}

func (p *Parser) parseIsExpr(left sqlast.Expr) sqlast.Expr {
	p.advance() // consume IS

	not := p.consume(NOT)
	p.expect(NULL)

	return &sqlast.IsExpr{Not: not, Expr: left}
}

var arithmeticOps = map[TokenType]sqlast.ArithmeticOperator{
	PLUS:    sqlast.PlusOp,
	MINUS:   sqlast.MinusOp,
	STAR:    sqlast.MultOp,
	SLASH:   sqlast.DivOp,
	PERCENT: sqlast.ModOp,
}

func (p *Parser) parseAdditiveExpr() sqlast.Expr {
	left := p.parseMultiplicativeExpr()

	for p.at(PLUS) || p.at(MINUS) {
		op := arithmeticOps[p.tok.Type]
		p.advance()

		left = &sqlast.ArithmeticExpr{Left: left, Operator: op, Right: p.parseMultiplicativeExpr()}
	}

	return left
}

func (p *Parser) parseMultiplicativeExpr() sqlast.Expr {
	left := p.parseUnaryExpr()

	for p.at(STAR) || p.at(SLASH) || p.at(PERCENT) {
		op := arithmeticOps[p.tok.Type]
		p.advance()

		left = &sqlast.ArithmeticExpr{Left: left, Operator: op, Right: p.parseUnaryExpr()}
	}

	return left
}

func (p *Parser) parseUnaryExpr() sqlast.Expr {
	var op sqlast.UnaryOperator

	switch {
	case p.at(PLUS):
		op = sqlast.UPlusOp
	case p.at(MINUS):
		op = sqlast.UMinusOp
	default:
		return p.parsePrimaryExpr()
	}

	p.advance()

	return &sqlast.UnaryExpr{Operator: op, Expr: p.parseUnaryExpr()}
}

// keywordLiterals holds the keyword tokens that stand for a literal value.
var keywordLiterals = map[TokenType]string{
	NULL:  "NULL",
	TRUE:  "TRUE",
	FALSE: "FALSE",
}

func (p *Parser) parsePrimaryExpr() sqlast.Expr {
	if text, ok := keywordLiterals[p.tok.Type]; ok {
		return p.parseKeywordLiteral(text)
	}

	switch p.tok.Type {
	case INT, FLOAT, QUESTION, HexStr, BitStr:
		return p.parseLiteralToken()
	case STRING:
		return p.parseStringLiteral()
	case COLON:
		return p.parseColonPlaceholder()
	case LPAREN:
		return p.parseParenExprOrSubquery()
	case CASE:
		return p.parseCaseExpr()
	case EXISTS:
		return p.parseExistsExpr()
	case IDENT, QuotedIdent, VALUES:
		return p.parseIdentOrValuesExpr()
	default:
		return p.parseNonReservedKeywordAsIdentExpr()
	}
}

// parseNonReservedKeywordAsIdentExpr handles the fallback case in
// parsePrimaryExpr's switch: a non-reserved keyword token (e.g. COMMENT,
// ENGINE — see token.go's nonReservedKeywords) used as an ordinary column
// reference or function name, split out to keep parsePrimaryExpr's
// cyclomatic complexity down.
func (p *Parser) parseNonReservedKeywordAsIdentExpr() sqlast.Expr {
	if p.tok.Type.IsNonReservedKeyword() {
		return p.parseIdentOrValuesExpr()
	}

	return failReturn[sqlast.Expr](p, "unexpected token %s in expression", p.tok.Type)
}

// parseIdentOrValuesExpr parses a column reference or function call, or the
// deprecated VALUES(col) reference used in an ON DUPLICATE KEY UPDATE clause
// to read the value that would have been inserted for col. VALUES is a
// keyword everywhere else, and MySQL only accepts it as a function name
// inside that clause, so it's rejected anywhere p.inOnDupUpdate is false.
func (p *Parser) parseIdentOrValuesExpr() sqlast.Expr {
	if !p.at(VALUES) {
		return p.parseIdentExpr()
	}

	if !p.inOnDupUpdate {
		return failReturn[sqlast.Expr](p, "VALUES() is only valid in an ON DUPLICATE KEY UPDATE clause")
	}

	name := p.tok.Literal
	p.advance() // consume VALUES
	p.expect(LPAREN)

	return p.parseGenericFuncCall(name)
}

func (p *Parser) parseLiteralToken() sqlast.Expr {
	tok := p.tok
	p.advance()

	return &sqlast.Literal{Val: tok.Literal}
}

func (p *Parser) parseKeywordLiteral(text string) sqlast.Expr {
	p.advance()

	return &sqlast.Literal{Val: text}
}

func (p *Parser) parseStringLiteral() sqlast.Expr {
	tok := p.tok
	p.advance()

	return &sqlast.Literal{Val: "'" + escapeStringLiteral(tok.Literal) + "'"}
}

var stringEscapes = map[rune]string{
	'\'':   `\'`,
	'\\':   `\\`,
	'\x00': `\0`,
	'\n':   `\n`,
	'\r':   `\r`,
	'\x1a': `\Z`,
}

// escapeStringLiteral re-encodes a lexer-decoded string literal so it can be
// embedded between single quotes again. \% and \_ are round-tripped as-is
// since the lexer deliberately leaves them un-decoded (they are only
// meaningful within LIKE pattern matching).
func escapeStringLiteral(s string) string {
	runes := []rune(s)

	var b strings.Builder

	for i := 0; i < len(runes); i++ {
		if consumed := writeLikeEscape(&b, runes, i); consumed > 0 {
			i += consumed - 1

			continue
		}

		writeEscapedRune(&b, runes[i])
	}

	return b.String()
}

// writeLikeEscape writes runes[i:] to b if it starts a \% or \_ sequence,
// returning how many runes it consumed (0 if it did not match).
func writeLikeEscape(b *strings.Builder, runes []rune, i int) int {
	if runes[i] != '\\' || i+1 >= len(runes) || (runes[i+1] != '%' && runes[i+1] != '_') {
		return 0
	}

	b.WriteByte('\\')
	b.WriteRune(runes[i+1])

	return 2
}

func writeEscapedRune(b *strings.Builder, r rune) {
	if esc, ok := stringEscapes[r]; ok {
		b.WriteString(esc)

		return
	}

	b.WriteRune(r)
}

func (p *Parser) parseColonPlaceholder() sqlast.Expr {
	p.advance() // consume ':'

	if p.tok.Type != IDENT && p.tok.Type != INT {
		p.failf("expected identifier or number after ':'")
	}

	name := p.tok.Literal
	p.advance()

	return &sqlast.Literal{Val: ":" + name}
}

func (p *Parser) parseParenExprOrSubquery() sqlast.Expr {
	p.advance() // consume '('

	if p.at(SELECT) || p.at(WITH) {
		sel := p.parseSubqueryStatement()
		p.expect(RPAREN)

		return &sqlast.Subquery{Select: sel}
	}

	expr := p.parseExpr()
	p.expect(RPAREN)

	return &sqlast.ParenExpr{Expr: expr}
}

func (p *Parser) parseCaseExpr() sqlast.Expr {
	p.advance() // consume CASE

	caseExpr := p.parseOptionalCaseValue()
	whens := p.parseWhenClauses()
	elseExpr := p.parseOptionalElse()

	p.expect(END)

	return &sqlast.CaseExpr{Expr: caseExpr, Whens: whens, Else: elseExpr}
}

// parseOptionalCaseValue parses the optional `expr` in a simple CASE expr
// (`CASE expr WHEN ...`); a searched CASE (`CASE WHEN ...`) has none.
func (p *Parser) parseOptionalCaseValue() sqlast.Expr {
	if p.at(WHEN) {
		return nil
	}

	return p.parseExpr()
}

func (p *Parser) parseWhenClauses() []*sqlast.When {
	var whens []*sqlast.When

	for p.at(WHEN) {
		whens = append(whens, p.parseWhen())
	}

	if len(whens) == 0 {
		p.failf("expected WHEN in CASE expression")
	}

	return whens
}

func (p *Parser) parseOptionalElse() sqlast.Expr {
	if !p.consume(ELSE) {
		return nil
	}

	return p.parseExpr()
}

func (p *Parser) parseWhen() *sqlast.When {
	p.advance() // consume WHEN

	cond := p.parseExpr()
	p.expect(THEN)

	val := p.parseExpr()

	return &sqlast.When{Cond: cond, Val: val}
}

func (p *Parser) parseExistsExpr() sqlast.Expr {
	p.advance() // consume EXISTS
	p.expect(LPAREN)

	sel := p.parseSubqueryStatement()

	p.expect(RPAREN)

	return &sqlast.ExistsExpr{Subquery: &sqlast.Subquery{Select: sel}}
}

func (p *Parser) readIdent() string {
	if p.tok.Type != IDENT && p.tok.Type != QuotedIdent && !p.tok.Type.IsNonReservedKeyword() {
		p.failf("expected identifier, got %s", p.tok.Type)
	}

	lit := p.tok.Literal
	p.advance()

	return lit
}

func (p *Parser) parseIdentExpr() sqlast.Expr {
	name := p.readIdent()

	if p.at(LPAREN) {
		return p.parseFuncCall(name)
	}

	if p.consume(DOT) {
		col := p.readIdent()

		return &sqlast.ColName{Qualifier: sqlast.TableName{Name: sqlast.TableIdent(name)}, Name: sqlast.ColIdent(col)}
	}

	return &sqlast.ColName{Name: sqlast.ColIdent(name)}
}

// parseColName parses a possibly-qualified column reference (col or
// table.col) as an assignment target. Shared by UPDATE's SET clause and
// INSERT's SET / ON DUPLICATE KEY UPDATE clauses.
func (p *Parser) parseColName() *sqlast.ColName {
	name := p.readIdent()

	if p.consume(DOT) {
		col := p.readIdent()

		return &sqlast.ColName{Qualifier: sqlast.TableName{Name: sqlast.TableIdent(name)}, Name: sqlast.ColIdent(col)}
	}

	return &sqlast.ColName{Name: sqlast.ColIdent(name)}
}

// parseSetExpr parses a single "col = expr" assignment.
func (p *Parser) parseSetExpr() *sqlast.UpdateExpr {
	name := p.parseColName()
	p.expect(EQ)

	return &sqlast.UpdateExpr{Name: name, Expr: p.parseExpr()}
}

// parseSetExprList parses a comma-separated list of "col = expr"
// assignments, as used by UPDATE's SET clause, INSERT's SET rows, and
// ON DUPLICATE KEY UPDATE.
func (p *Parser) parseSetExprList() []*sqlast.UpdateExpr {
	var exprs []*sqlast.UpdateExpr

	for {
		exprs = append(exprs, p.parseSetExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

// parseExprList parses a comma-separated list of expressions.
func (p *Parser) parseExprList() []sqlast.Expr {
	var exprs []sqlast.Expr

	for {
		exprs = append(exprs, p.parseExpr())

		if !p.consume(COMMA) {
			return exprs
		}
	}
}

func (p *Parser) consumeDistinct() bool {
	return p.consume(DISTINCT)
}
