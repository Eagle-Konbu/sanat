package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// The built-in type names below are keywords in PostgreSQL's grammar, but
// unreserved or col_name ones that this lexer leaves as plain identifiers, so
// they are matched by their (folded) text where a type name is expected.

// characterTypeWords are the first words of PostgreSQL's character types,
// each of which may be followed by VARYING.
var characterTypeWords = map[string]bool{"character": true, "char": true, "nchar": true}

// intervalFieldWords are the units an interval type's fields may name.
var intervalFieldWords = map[string]bool{
	"year": true, "month": true, "day": true, "hour": true, "minute": true, "second": true,
}

// intervalRanges are the "field to field" ranges PostgreSQL's opt_interval
// allows.
var intervalRanges = map[string]bool{
	"year to month":    true,
	"day to hour":      true,
	"day to minute":    true,
	"day to second":    true,
	"hour to minute":   true,
	"hour to second":   true,
	"minute to second": true,
}

// parseTypeName parses a cast's target type: a type name followed by any
// number of array bounds (int[], int[3]). The ARRAY keyword spelling
// (int ARRAY[3]) and SETOF are not supported.
func (p *Parser) parseTypeName() *pgast.TypeName {
	t := p.parseSimpleTypeName()

	for p.at(LBRACKET) {
		t.ArrayBounds = append(t.ArrayBounds, p.readArrayBound())
	}

	return t
}

// parseSimpleTypeName parses a type name without array bounds: a built-in
// multi-word name (double precision, character varying, timestamp with time
// zone, interval day to second, ...) or a possibly qualified generic name,
// with optional modifiers. A dotted name is always generic, so
// interval.mytype names a type in the interval schema.
func (p *Parser) parseSimpleTypeName() *pgast.TypeName {
	if p.at(IDENT) && !p.peekAt(DOT) {
		if t := p.tryParseBuiltinTypeName(); t != nil {
			return t
		}
	}

	t := &pgast.TypeName{Names: []pgast.ColIdent{pgast.ColIdent(p.readTypeIdent())}}

	for p.consume(DOT) {
		t.Names = append(t.Names, pgast.ColIdent(p.readColLabel()))
	}

	t.Modifiers = p.parseOptionalTypeModifiers()

	return t
}

// tryParseBuiltinTypeName parses one of the built-in type names whose
// spelling is more than a single identifier, returning nil without consuming
// anything if the current IDENT doesn't start one.
func (p *Parser) tryParseBuiltinTypeName() *pgast.TypeName {
	word := p.tok.Literal

	switch {
	case word == "double" && p.peekIsWord("precision"):
		p.advance()
		p.advance()

		return builtinType("double precision")
	case word == "national" && p.peekTok.Type == IDENT && characterTypeWords[p.peekTok.Literal]:
		p.advance()

		name := "national " + p.tok.Literal
		p.advance()

		return p.finishCharacterType(name)
	case characterTypeWords[word], word == "bit":
		p.advance()

		return p.finishCharacterType(word)
	case word == "time", word == "timestamp":
		p.advance()

		return p.finishDateTimeType(word)
	case word == "interval":
		p.advance()

		return p.finishIntervalType()
	default:
		return nil
	}
}

func builtinType(name string) *pgast.TypeName {
	return &pgast.TypeName{Names: []pgast.ColIdent{pgast.ColIdent(name)}}
}

// finishCharacterType parses the optional VARYING and length modifier after
// a character or bit type's leading word(s).
func (p *Parser) finishCharacterType(name string) *pgast.TypeName {
	if p.atWord("varying") {
		p.advance()

		name += " varying"
	}

	t := builtinType(name)
	t.Modifiers = p.parseOptionalBuiltinTypeModifier()

	return t
}

// finishDateTimeType parses the optional precision and WITH / WITHOUT TIME
// ZONE suffix after time or timestamp.
func (p *Parser) finishDateTimeType(name string) *pgast.TypeName {
	t := builtinType(name)
	t.Modifiers = p.parseOptionalBuiltinTypeModifier()

	switch {
	case p.at(WITH) && p.peekIsWord("time") && p.peek2IsWord("zone"):
		t.TimeZone = pgast.WithTimeZone
	case p.atWord("without") && p.peekIsWord("time") && p.peek2IsWord("zone"):
		t.TimeZone = pgast.WithoutTimeZone
	default:
		return t
	}

	p.advance() // consume WITH / WITHOUT
	p.advance() // consume TIME
	p.advance() // consume ZONE

	return t
}

// finishIntervalType parses an interval type's optional precision or field
// restriction; PostgreSQL accepts one or the other, not both.
func (p *Parser) finishIntervalType() *pgast.TypeName {
	t := builtinType("interval")

	if p.at(LPAREN) {
		t.Modifiers = p.parseOptionalBuiltinTypeModifier()

		return t
	}

	t.IntervalFields = p.parseOptionalIntervalFields()

	return t
}

// parseOptionalIntervalFields parses an interval's optional field
// restriction, "field [TO field]", where a trailing SECOND may carry a
// precision, returning it as SQL text, or "" if there is none.
func (p *Parser) parseOptionalIntervalFields() string {
	if !p.atIntervalField() {
		return ""
	}

	start := p.tok.Literal
	fields := p.readIntervalField()

	if p.consume(TO) {
		if !p.atIntervalField() {
			p.failf("expected interval field after TO, got %s", p.tok.Type)
		}

		if !intervalRanges[start+" to "+p.tok.Literal] {
			p.failf("invalid interval range %s TO %s", fields, p.tok.Literal)
		}

		fields += " to " + p.readIntervalField()
	}

	return fields
}

func (p *Parser) atIntervalField() bool {
	return p.at(IDENT) && intervalFieldWords[p.tok.Literal]
}

// readIntervalField reads one interval field name. Only SECOND may carry a
// fractional-seconds precision, as in "interval minute to second(3)".
func (p *Parser) readIntervalField() string {
	field := p.tok.Literal
	p.advance()

	if field == "second" && p.at(LPAREN) {
		p.advance()

		if !p.at(INT) {
			p.failf("expected integer precision, got %s", p.tok.Type)
		}

		field += "(" + p.tok.Literal + ")"
		p.advance()
		p.expect(RPAREN)
	}

	return field
}

// readTypeIdent reads the first word of a generic type name, which may be
// any type_function_name; a quoted name keeps its quoting (see identText).
func (p *Parser) readTypeIdent() string {
	if !isTypeFuncName(p.tok) {
		return failReturn[string](p, "expected type name, got %s", p.tok.Type)
	}

	text := identText(p.tok)
	p.advance()

	return text
}

// parseOptionalBuiltinTypeModifier parses a built-in character, bit,
// datetime, or interval type's optional length or precision, which is a
// single integer.
func (p *Parser) parseOptionalBuiltinTypeModifier() []pgast.Expr {
	if !p.consume(LPAREN) {
		return nil
	}

	if !p.at(INT) {
		p.failf("expected integer type modifier, got %s", p.tok.Type)
	}

	mod := p.parseLiteralToken()

	p.expect(RPAREN)

	return []pgast.Expr{mod}
}

// parseOptionalTypeModifiers parses an optional parenthesized type modifier
// list. PostgreSQL rejects anything but constants and plain identifiers as
// a modifier, so the general expression grammar is not needed here; that
// also keeps speculative type-name parsing (see tryParseTypedLiteral) cheap.
func (p *Parser) parseOptionalTypeModifiers() []pgast.Expr {
	if !p.consume(LPAREN) {
		return nil
	}

	var mods []pgast.Expr

	for {
		mods = append(mods, p.parseTypeModifier())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return mods
}

// parseTypeModifier parses one generic type modifier: a possibly negative
// number, as in numeric(10, -2), a string, or a column-name identifier.
func (p *Parser) parseTypeModifier() pgast.Expr {
	switch {
	case p.at(INT), p.at(FLOAT):
		return p.parseLiteralToken()
	case p.at(MINUS) && (p.peekAt(INT) || p.peekAt(FLOAT)):
		p.advance()

		return &pgast.UnaryExpr{Operator: pgast.UMinusOp, Expr: p.parseLiteralToken()}
	case p.at(STRING):
		return p.parseStringLiteral()
	case isColID(p.tok):
		return &pgast.ColName{Name: pgast.ColIdent(p.readColID())}
	default:
		return failReturn[pgast.Expr](p, "expected constant or identifier in type modifier, got %s", p.tok.Type)
	}
}

// readArrayBound consumes an array bound, "[]" or "[n]", returning n or "".
func (p *Parser) readArrayBound() string {
	p.advance() // consume '['

	bound := ""
	if p.tok.Type == INT {
		bound = p.tok.Literal
		p.advance()
	}

	p.expect(RBRACKET)

	return bound
}

// tryParseTypedLiteral parses a string constant preceded by its type, such
// as date '2020-01-01', varchar(3) 'abc', or interval '1' day, reporting
// ok=false without consuming anything if the upcoming tokens don't spell
// one. A type name can't be told apart from a column reference or function
// call until the string after it, so the type name is parsed speculatively.
func (p *Parser) tryParseTypedLiteral() (pgast.Expr, bool) {
	if !p.mayStartTypedLiteral() {
		return nil, false
	}

	saved := p.save()

	t, ok := speculate(p, (*Parser).parseSimpleTypeName)
	if !ok || !p.at(STRING) {
		p.restore(&saved)

		return nil, false
	}

	lit := &pgast.TypedLiteral{Type: t, Value: p.readStringLiteralText()}

	if isPlainInterval(t) {
		t.IntervalFields = p.parseOptionalIntervalFields()
	}

	return lit, true
}

// mayStartTypedLiteral reports whether the current token could start a
// typed literal's type name, ruling out the common cases cheaply before
// speculating.
func (p *Parser) mayStartTypedLiteral() bool {
	if !isTypeFuncName(p.tok) {
		return false
	}

	return p.peekAt(STRING) || p.peekAt(DOT) || p.peekAt(LPAREN) || p.at(IDENT) && multiWordTypeStarts[p.tok.Literal]
}

// multiWordTypeStarts are the first words of the built-in type names that
// span more than one token.
var multiWordTypeStarts = map[string]bool{
	"double": true, "national": true, "character": true, "char": true, "nchar": true, "bit": true,
	"time": true, "timestamp": true,
}

// isPlainInterval reports whether t is a bare interval type, whose typed
// literal may carry a field restriction after the string.
func isPlainInterval(t *pgast.TypeName) bool {
	return len(t.Names) == 1 && t.Names[0] == "interval" && len(t.Modifiers) == 0 && t.IntervalFields == ""
}

// atWord reports whether the current token is the unquoted identifier
// word. Quoted identifiers never match, so "zone" in double quotes is not
// mistaken for the word ZONE.
func (p *Parser) atWord(word string) bool { return p.tok.Type == IDENT && p.tok.Literal == word }

// peekIsWord is atWord for the next token.
func (p *Parser) peekIsWord(word string) bool {
	return p.peekTok.Type == IDENT && p.peekTok.Literal == word
}

// peek2IsWord is atWord for the token after next.
func (p *Parser) peek2IsWord(word string) bool {
	return p.peek2Tok.Type == IDENT && p.peek2Tok.Literal == word
}
