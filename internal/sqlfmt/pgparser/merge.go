package pgparser

import "github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"

// MERGE is PostgreSQL 15+, unlike the rest of the 13+ grammar this package
// accepts. Within it, WHEN NOT MATCHED BY SOURCE, an explicit BY TARGET,
// RETURNING, and MERGE as a CTE body are PostgreSQL 17+, and
// RETURNING WITH (OLD/NEW AS ...) is PostgreSQL 18+. The parser accepts all
// of them regardless of version.

// ParseMerge parses a MERGE statement, optionally led by a WITH clause, from
// input.
func ParseMerge(input string) (*pgast.Merge, error) {
	return parseStatementInput(input, (*Parser).parseMergeAfterWith)
}

// parseMergeAfterWith parses a MERGE statement whose optional WITH clause has
// already been consumed. The current token must be MERGE. The data source is
// a single FROM item, which may itself be a join.
func (p *Parser) parseMergeAfterWith() *pgast.Merge {
	p.expect(MERGE)
	p.expect(INTO)

	merge := &pgast.Merge{Target: p.parseRelationExpr()}

	p.expect(USING)

	merge.Source = p.parseTableReference()

	p.expect(ON)

	merge.On = p.parseExpr()
	merge.Whens = p.parseMergeWhenList()
	merge.Returning = p.parseOptionalMergeReturning()

	return merge
}

// parseMergeWhenList parses one or more WHEN clauses.
func (p *Parser) parseMergeWhenList() []*pgast.MergeWhen {
	var whens []*pgast.MergeWhen

	for {
		whens = append(whens, p.parseMergeWhen())

		if !p.at(WHEN) {
			return whens
		}
	}
}

// parseMergeWhen parses "WHEN match [AND condition] THEN action".
func (p *Parser) parseMergeWhen() *pgast.MergeWhen {
	p.expect(WHEN)

	when := &pgast.MergeWhen{Match: p.parseMergeMatchKind()}

	if p.consume(AND) {
		when.Condition = p.parseExpr()
	}

	p.expect(THEN)

	if mergeMatchTargetsTargetRow(when.Match) {
		when.Action = p.parseMergeTargetRowAction()
	} else {
		when.Action = p.parseMergeInsertAction()
	}

	return when
}

// parseMergeMatchKind parses "MATCHED" or "NOT MATCHED [BY {SOURCE | TARGET}]".
func (p *Parser) parseMergeMatchKind() pgast.MergeMatchKind {
	if p.consume(MATCHED) {
		return pgast.MergeMatched
	}

	p.expect(NOT)
	p.expect(MATCHED)

	if !p.consume(BY) {
		return pgast.MergeNotMatched
	}

	switch {
	case p.consume(SOURCE):
		return pgast.MergeNotMatchedBySource
	case p.consume(TARGET):
		return pgast.MergeNotMatchedByTarget
	default:
		return failReturn[pgast.MergeMatchKind](p, "expected SOURCE or TARGET after NOT MATCHED BY, got %s", p.tok.Type)
	}
}

// mergeMatchTargetsTargetRow reports whether a WHEN clause of kind match acts
// on an existing target row (MATCHED and NOT MATCHED BY SOURCE), and so takes
// UPDATE or DELETE, rather than on a source row with no target row, which
// takes INSERT.
func mergeMatchTargetsTargetRow(match pgast.MergeMatchKind) bool {
	return match == pgast.MergeMatched || match == pgast.MergeNotMatchedBySource
}

// parseMergeTargetRowAction parses the action of a WHEN clause that acts on
// an existing target row: UPDATE SET ..., DELETE, or DO NOTHING.
func (p *Parser) parseMergeTargetRowAction() pgast.MergeAction {
	switch {
	case p.consume(UPDATE):
		p.expect(SET)

		return &pgast.MergeUpdate{Set: p.parseSetClauseList()}
	case p.consume(DELETE):
		return pgast.MergeDelete{}
	case p.at(DO):
		return p.parseMergeDoNothing()
	default:
		return failReturn[pgast.MergeAction](p, "expected UPDATE, DELETE, or DO NOTHING, got %s", p.tok.Type)
	}
}

// parseMergeInsertAction parses the action of a WHEN NOT MATCHED [BY TARGET]
// clause: INSERT ... or DO NOTHING.
func (p *Parser) parseMergeInsertAction() pgast.MergeAction {
	switch {
	case p.at(INSERT):
		return p.parseMergeInsert()
	case p.at(DO):
		return p.parseMergeDoNothing()
	default:
		return failReturn[pgast.MergeAction](p, "expected INSERT or DO NOTHING, got %s", p.tok.Type)
	}
}

func (p *Parser) parseMergeDoNothing() pgast.MergeAction {
	p.expect(DO)
	p.expect(NOTHING)

	return pgast.MergeDoNothing{}
}

// parseMergeInsert parses
// "INSERT [(column, ...)] [OVERRIDING {SYSTEM | USER} VALUE]
// {VALUES (...) | DEFAULT VALUES}". As in PostgreSQL's grammar, DEFAULT
// VALUES admits neither a column list nor OVERRIDING. The current token must
// be INSERT.
func (p *Parser) parseMergeInsert() *pgast.MergeInsert {
	p.expect(INSERT)

	ins := &pgast.MergeInsert{
		Columns:    p.parseOptionalParenColumnList(),
		Overriding: p.parseOptionalOverriding(),
	}

	if p.at(DEFAULT) {
		if ins.Columns != nil || ins.Overriding != pgast.OverridingNone {
			p.failf("DEFAULT VALUES cannot follow a column list or OVERRIDING")
		}

		p.advance()
		p.expect(VALUES)

		return ins
	}

	p.expect(VALUES)

	ins.Values = p.parseParenExprOrDefaultList()

	return ins
}

// parseOptionalOverriding parses an optional
// "OVERRIDING {SYSTEM | USER} VALUE" clause. USER is a reserved keyword in
// PostgreSQL that this package has no token for (its use as the current user
// name in expressions isn't modeled), so it is matched as a plain identifier.
func (p *Parser) parseOptionalOverriding() pgast.Overriding {
	if !p.consume(OVERRIDING) {
		return pgast.OverridingNone
	}

	var overriding pgast.Overriding

	switch {
	case p.consume(SYSTEM):
		overriding = pgast.OverridingSystemValue
	case p.at(IDENT) && p.tok.Literal == "user":
		p.advance()

		overriding = pgast.OverridingUserValue
	default:
		p.failf("expected SYSTEM or USER after OVERRIDING, got %s", p.tok.Type)
	}

	p.expect(VALUE)

	return overriding
}

// parseOptionalMergeReturning parses MERGE's optional RETURNING clause,
// including PostgreSQL 18's "WITH ({OLD | NEW} AS name, ...)" output aliases.
// INSERT, UPDATE, and DELETE accept the same aliases in PostgreSQL 18, but
// this package does not support them there yet.
func (p *Parser) parseOptionalMergeReturning() *pgast.Returning {
	if !p.consume(RETURNING) {
		return nil
	}

	returning := &pgast.Returning{}

	if p.consume(WITH) {
		returning.Aliases = p.parseReturningAliases()
	}

	returning.Exprs = p.parseSelectExprList()

	return returning
}

// parseReturningAliases parses "({OLD | NEW} AS name, ...)" after
// RETURNING WITH.
func (p *Parser) parseReturningAliases() []*pgast.ReturningAlias {
	p.expect(LPAREN)

	var aliases []*pgast.ReturningAlias

	for {
		aliases = append(aliases, p.parseReturningAlias())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return aliases
}

// parseReturningAlias parses one "{OLD | NEW} AS name" item of RETURNING
// WITH (...). Unlike most aliases, AS is required, as in PostgreSQL 18.
func (p *Parser) parseReturningAlias() *pgast.ReturningAlias {
	var kind pgast.ReturningAliasKind

	switch {
	case p.consume(OLD):
		kind = pgast.ReturningOld
	case p.consume(NEW):
		kind = pgast.ReturningNew
	default:
		p.failf("expected OLD or NEW, got %s", p.tok.Type)
	}

	p.expect(AS)

	return &pgast.ReturningAlias{Kind: kind, Name: pgast.TableIdent(p.readColID())}
}
