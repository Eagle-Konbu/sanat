package parser

import (
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// parseExpr parses a SQL expression, starting from the loosest-binding
// precedence level (OR) down to the tightest (primary expressions).
func (p *Parser) parseExpr() (sqlast.Expr, error) {
	return p.parseOrExpr()
}

func (p *Parser) parseOrExpr() (sqlast.Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}

	for p.at(OR) {
		if err := p.advance(); err != nil {
			return nil, err
		}

		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}

		left = &sqlast.OrExpr{Left: left, Right: right}
	}

	return left, nil
}

func (p *Parser) parseAndExpr() (sqlast.Expr, error) {
	left, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	for p.at(AND) {
		if err := p.advance(); err != nil {
			return nil, err
		}

		right, err := p.parseNotExpr()
		if err != nil {
			return nil, err
		}

		left = &sqlast.AndExpr{Left: left, Right: right}
	}

	return left, nil
}

func (p *Parser) parseNotExpr() (sqlast.Expr, error) {
	if !p.at(NOT) {
		return p.parseComparisonExpr()
	}

	if err := p.advance(); err != nil {
		return nil, err
	}

	expr, err := p.parseNotExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.NotExpr{Expr: expr}, nil
}

var simpleComparisonOps = map[TokenType]sqlast.ComparisonOperator{
	EQ: sqlast.EqualOp,
	NE: sqlast.NotEqualOp,
	LT: sqlast.LessThanOp,
	GT: sqlast.GreaterThanOp,
	LE: sqlast.LessEqualOp,
	GE: sqlast.GreaterEqualOp,
}

// parseComparisonExpr parses a single predicate: a plain comparison, or one
// of the [NOT] IN / [NOT] BETWEEN / [NOT] LIKE / IS [NOT] NULL forms.
func (p *Parser) parseComparisonExpr() (sqlast.Expr, error) {
	left, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	return p.parsePredicateSuffix(left)
}

func (p *Parser) parsePredicateSuffix(left sqlast.Expr) (sqlast.Expr, error) {
	not, err := p.consumeNegatedPredicateKeyword()
	if err != nil {
		return nil, err
	}

	switch {
	case p.at(IN):
		return p.parseInExpr(left, not)
	case p.at(BETWEEN):
		return p.parseBetweenExpr(left, not)
	case p.at(LIKE):
		return p.parseLikeExpr(left, not)
	case not:
		return nil, p.errorf("expected IN, BETWEEN, or LIKE after NOT")
	case p.at(IS):
		return p.parseIsExpr(left)
	default:
		return p.parseOptionalSimpleComparison(left)
	}
}

// consumeNegatedPredicateKeyword consumes a leading NOT that belongs to a
// [NOT] IN / [NOT] BETWEEN / [NOT] LIKE predicate, as opposed to a prefix
// logical NOT (already handled by parseNotExpr, above this precedence level).
func (p *Parser) consumeNegatedPredicateKeyword() (bool, error) {
	if !p.at(NOT) || (!p.peekAt(IN) && !p.peekAt(BETWEEN) && !p.peekAt(LIKE)) {
		return false, nil
	}

	return true, p.advance()
}

func (p *Parser) parseOptionalSimpleComparison(left sqlast.Expr) (sqlast.Expr, error) {
	op, ok := simpleComparisonOps[p.tok.Type]
	if !ok {
		return left, nil
	}

	return p.parseSimpleComparison(left, op)
}

func (p *Parser) parseSimpleComparison(left sqlast.Expr, op sqlast.ComparisonOperator) (sqlast.Expr, error) {
	if err := p.advance(); err != nil {
		return nil, err
	}

	right, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: right}, nil
}

func (p *Parser) parseInExpr(left sqlast.Expr, not bool) (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume IN
		return nil, err
	}

	if err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	op := sqlast.InOp
	if not {
		op = sqlast.NotInOp
	}

	if p.at(SELECT) || p.at(WITH) {
		sel, err := p.parseSelectStatement()
		if err != nil {
			return nil, err
		}

		if err := p.expect(RPAREN); err != nil {
			return nil, err
		}

		return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: &sqlast.Subquery{Select: sel}}, nil
	}

	values, err := p.parseExprList()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: sqlast.ValTuple(values)}, nil
}

func (p *Parser) parseBetweenExpr(left sqlast.Expr, not bool) (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume BETWEEN
		return nil, err
	}

	from, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(AND); err != nil {
		return nil, err
	}

	to, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.RangeCond{Not: not, Left: left, From: from, To: to}, nil
}

func (p *Parser) parseLikeExpr(left sqlast.Expr, not bool) (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume LIKE
		return nil, err
	}

	right, err := p.parseAdditiveExpr()
	if err != nil {
		return nil, err
	}

	op := sqlast.LikeOp
	if not {
		op = sqlast.NotLikeOp
	}

	return &sqlast.ComparisonExpr{Left: left, Operator: op, Right: right}, nil
}

func (p *Parser) parseIsExpr(left sqlast.Expr) (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume IS
		return nil, err
	}

	not, err := p.consume(NOT)
	if err != nil {
		return nil, err
	}

	if err := p.expect(NULL); err != nil {
		return nil, err
	}

	return &sqlast.IsExpr{Not: not, Expr: left}, nil
}

var arithmeticOps = map[TokenType]sqlast.ArithmeticOperator{
	PLUS:    sqlast.PlusOp,
	MINUS:   sqlast.MinusOp,
	STAR:    sqlast.MultOp,
	SLASH:   sqlast.DivOp,
	PERCENT: sqlast.ModOp,
}

func (p *Parser) parseAdditiveExpr() (sqlast.Expr, error) {
	left, err := p.parseMultiplicativeExpr()
	if err != nil {
		return nil, err
	}

	for p.at(PLUS) || p.at(MINUS) {
		op := arithmeticOps[p.tok.Type]

		if err := p.advance(); err != nil {
			return nil, err
		}

		right, err := p.parseMultiplicativeExpr()
		if err != nil {
			return nil, err
		}

		left = &sqlast.ArithmeticExpr{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

func (p *Parser) parseMultiplicativeExpr() (sqlast.Expr, error) {
	left, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}

	for p.at(STAR) || p.at(SLASH) || p.at(PERCENT) {
		op := arithmeticOps[p.tok.Type]

		if err := p.advance(); err != nil {
			return nil, err
		}

		right, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}

		left = &sqlast.ArithmeticExpr{Left: left, Operator: op, Right: right}
	}

	return left, nil
}

func (p *Parser) parseUnaryExpr() (sqlast.Expr, error) {
	var op sqlast.UnaryOperator

	switch {
	case p.at(PLUS):
		op = sqlast.UPlusOp
	case p.at(MINUS):
		op = sqlast.UMinusOp
	default:
		return p.parsePrimaryExpr()
	}

	if err := p.advance(); err != nil {
		return nil, err
	}

	expr, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.UnaryExpr{Operator: op, Expr: expr}, nil
}

// keywordLiterals holds the keyword tokens that stand for a literal value.
var keywordLiterals = map[TokenType]string{
	NULL:  "NULL",
	TRUE:  "TRUE",
	FALSE: "FALSE",
}

func (p *Parser) parsePrimaryExpr() (sqlast.Expr, error) {
	if text, ok := keywordLiterals[p.tok.Type]; ok {
		return p.parseKeywordLiteral(text)
	}

	switch p.tok.Type {
	case INT, FLOAT, QUESTION:
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
	case IDENT, QuotedIdent:
		return p.parseIdentExpr()
	default:
		return nil, p.errorf("unexpected token %s in expression", p.tok.Type)
	}
}

func (p *Parser) parseLiteralToken() (sqlast.Expr, error) {
	tok := p.tok

	if err := p.advance(); err != nil {
		return nil, err
	}

	return &sqlast.Literal{Val: tok.Literal}, nil
}

func (p *Parser) parseKeywordLiteral(text string) (sqlast.Expr, error) {
	if err := p.advance(); err != nil {
		return nil, err
	}

	return &sqlast.Literal{Val: text}, nil
}

func (p *Parser) parseStringLiteral() (sqlast.Expr, error) {
	tok := p.tok

	if err := p.advance(); err != nil {
		return nil, err
	}

	return &sqlast.Literal{Val: "'" + escapeStringLiteral(tok.Literal) + "'"}, nil
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

func (p *Parser) parseColonPlaceholder() (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume ':'
		return nil, err
	}

	if p.tok.Type != IDENT && p.tok.Type != INT {
		return nil, p.errorf("expected identifier or number after ':'")
	}

	name := p.tok.Literal

	if err := p.advance(); err != nil {
		return nil, err
	}

	return &sqlast.Literal{Val: ":" + name}, nil
}

func (p *Parser) parseParenExprOrSubquery() (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume '('
		return nil, err
	}

	if p.at(SELECT) || p.at(WITH) {
		sel, err := p.parseSelectStatement()
		if err != nil {
			return nil, err
		}

		if err := p.expect(RPAREN); err != nil {
			return nil, err
		}

		return &sqlast.Subquery{Select: sel}, nil
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.ParenExpr{Expr: expr}, nil
}

func (p *Parser) parseCaseExpr() (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume CASE
		return nil, err
	}

	caseExpr, err := p.parseOptionalCaseValue()
	if err != nil {
		return nil, err
	}

	whens, err := p.parseWhenClauses()
	if err != nil {
		return nil, err
	}

	elseExpr, err := p.parseOptionalElse()
	if err != nil {
		return nil, err
	}

	if err := p.expect(END); err != nil {
		return nil, err
	}

	return &sqlast.CaseExpr{Expr: caseExpr, Whens: whens, Else: elseExpr}, nil
}

// parseOptionalCaseValue parses the optional `expr` in a simple CASE expr
// (`CASE expr WHEN ...`); a searched CASE (`CASE WHEN ...`) has none.
func (p *Parser) parseOptionalCaseValue() (sqlast.Expr, error) {
	if p.at(WHEN) {
		return nil, nil
	}

	return p.parseExpr()
}

func (p *Parser) parseWhenClauses() ([]*sqlast.When, error) {
	var whens []*sqlast.When

	for p.at(WHEN) {
		when, err := p.parseWhen()
		if err != nil {
			return nil, err
		}

		whens = append(whens, when)
	}

	if len(whens) == 0 {
		return nil, p.errorf("expected WHEN in CASE expression")
	}

	return whens, nil
}

func (p *Parser) parseOptionalElse() (sqlast.Expr, error) {
	if ok, err := p.consume(ELSE); err != nil || !ok {
		return nil, err
	}

	return p.parseExpr()
}

func (p *Parser) parseWhen() (*sqlast.When, error) {
	if err := p.advance(); err != nil { // consume WHEN
		return nil, err
	}

	cond, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	if err := p.expect(THEN); err != nil {
		return nil, err
	}

	val, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &sqlast.When{Cond: cond, Val: val}, nil
}

func (p *Parser) parseExistsExpr() (sqlast.Expr, error) {
	if err := p.advance(); err != nil { // consume EXISTS
		return nil, err
	}

	if err := p.expect(LPAREN); err != nil {
		return nil, err
	}

	sel, err := p.parseSelectStatement()
	if err != nil {
		return nil, err
	}

	if err := p.expect(RPAREN); err != nil {
		return nil, err
	}

	return &sqlast.ExistsExpr{Subquery: &sqlast.Subquery{Select: sel}}, nil
}

func (p *Parser) readIdent() (string, error) {
	if p.tok.Type != IDENT && p.tok.Type != QuotedIdent {
		return "", p.errorf("expected identifier, got %s", p.tok.Type)
	}

	lit := p.tok.Literal

	if err := p.advance(); err != nil {
		return "", err
	}

	return lit, nil
}

func (p *Parser) parseIdentExpr() (sqlast.Expr, error) {
	name, err := p.readIdent()
	if err != nil {
		return nil, err
	}

	if p.at(LPAREN) {
		return p.parseFuncCall(name)
	}

	if ok, err := p.consume(DOT); err != nil {
		return nil, err
	} else if ok {
		col, err := p.readIdent()
		if err != nil {
			return nil, err
		}

		return &sqlast.ColName{Qualifier: sqlast.TableName{Name: sqlast.TableIdent(name)}, Name: sqlast.ColIdent(col)}, nil
	}

	return &sqlast.ColName{Name: sqlast.ColIdent(name)}, nil
}

// parseExprList parses a comma-separated list of expressions.
func (p *Parser) parseExprList() ([]sqlast.Expr, error) {
	var exprs []sqlast.Expr

	for {
		e, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, e)

		if ok, err := p.consume(COMMA); err != nil {
			return nil, err
		} else if !ok {
			return exprs, nil
		}
	}
}

func (p *Parser) consumeDistinct() (bool, error) {
	return p.consume(DISTINCT)
}
