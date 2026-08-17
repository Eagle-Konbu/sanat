package parser

import (
	"strconv"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// parseDDLEntry runs parse to completion over input, then verifies input was
// fully consumed — the same shape every ParseXxx DDL entry point below
// needs, factored out to avoid repeating it six times.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func parseDDLEntry[T sqlast.Statement](input string, parse func(*Parser) T) (result T, err error) {
	defer recoverParseError(&err)

	p := NewParser(input)
	result = parse(p)

	if !p.at(EOF) {
		p.failf("unexpected token %s after statement", p.tok.Type)
	}

	return result, nil
}

// ParseCreateTable parses a CREATE TABLE statement from input.
func ParseCreateTable(input string) (*sqlast.CreateTable, error) {
	return parseDDLEntry(input, (*Parser).parseCreateTableStatement)
}

// ParseAlterTable parses an ALTER TABLE statement from input.
func ParseAlterTable(input string) (*sqlast.AlterTable, error) {
	return parseDDLEntry(input, (*Parser).parseAlterTableStatement)
}

// ParseCreateIndex parses a CREATE INDEX statement from input.
func ParseCreateIndex(input string) (*sqlast.CreateIndex, error) {
	return parseDDLEntry(input, (*Parser).parseCreateIndexStatement)
}

// ParseDropIndex parses a DROP INDEX statement from input.
func ParseDropIndex(input string) (*sqlast.DropIndex, error) {
	return parseDDLEntry(input, (*Parser).parseDropIndexStatement)
}

// ParseDropTable parses a DROP TABLE statement from input.
func ParseDropTable(input string) (*sqlast.DropTable, error) {
	return parseDDLEntry(input, (*Parser).parseDropTableStatement)
}

// ParseTruncateTable parses a TRUNCATE TABLE statement from input.
func ParseTruncateTable(input string) (*sqlast.TruncateTable, error) {
	return parseDDLEntry(input, (*Parser).parseTruncateTableStatement)
}

// parseCreateStatement dispatches a leading CREATE to CREATE TABLE or
// CREATE [UNIQUE] INDEX based on the following token. The current token
// must be CREATE.
func (p *Parser) parseCreateStatement() sqlast.Statement {
	switch {
	case p.peekAt(TABLE):
		return p.parseCreateTableStatement()
	case p.peekAt(INDEX) || p.peekAt(UNIQUE):
		return p.parseCreateIndexStatement()
	default:
		return failReturn[sqlast.Statement](p, "expected TABLE, INDEX, or UNIQUE after CREATE, got %s", p.peekTok.Type)
	}
}

// parseDropStatement dispatches a leading DROP to DROP TABLE or DROP INDEX
// based on the following token. The current token must be DROP.
func (p *Parser) parseDropStatement() sqlast.Statement {
	switch {
	case p.peekAt(TABLE):
		return p.parseDropTableStatement()
	case p.peekAt(INDEX):
		return p.parseDropIndexStatement()
	default:
		return failReturn[sqlast.Statement](p, "expected TABLE or INDEX after DROP, got %s", p.peekTok.Type)
	}
}

// parseCreateTableStatement parses a CREATE TABLE statement. The current
// token must be CREATE.
func (p *Parser) parseCreateTableStatement() *sqlast.CreateTable {
	p.expect(CREATE)
	p.expect(TABLE)

	ct := &sqlast.CreateTable{IfNotExists: p.parseOptionalIfNotExists()}
	ct.Table = p.parseTableName()

	p.expect(LPAREN)

	for {
		ct.Elements = append(ct.Elements, p.parseTableElement())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	ct.Options = p.parseTableOptions()

	return ct
}

// parseTableElement parses one entry of CREATE TABLE's column list: a
// column definition, or a table-level constraint/index if the current token
// starts one.
func (p *Parser) parseTableElement() sqlast.TableElement {
	if p.isTableConstraintStart() {
		return p.parseTableConstraint()
	}

	return p.parseColumnDef()
}

func (p *Parser) isTableConstraintStart() bool {
	switch p.tok.Type {
	case CONSTRAINT, PRIMARY, UNIQUE, INDEX, KEY, FOREIGN:
		return true
	default:
		return false
	}
}

// parseColumnDef parses a single column definition: name, data type, and
// any column constraints.
func (p *Parser) parseColumnDef() *sqlast.ColumnDef {
	col := &sqlast.ColumnDef{Name: sqlast.ColIdent(p.readIdent())}
	col.Type = p.parseDataType()
	p.parseColumnConstraints(col)

	return col
}

// parseColumnConstraints parses the constraints following a column's data
// type (NOT NULL/NULL, DEFAULT, AUTO_INCREMENT, ON UPDATE, PRIMARY KEY,
// UNIQUE, COMMENT), in whatever order they appear, until none match.
func (p *Parser) parseColumnConstraints(col *sqlast.ColumnDef) {
	for p.parseColumnConstraint(col) {
	}
}

// parseColumnConstraint parses a single column constraint and reports
// whether it recognized one, so parseColumnConstraints can loop until none
// match. Split out to keep parseColumnConstraints's cyclomatic complexity
// down.
func (p *Parser) parseColumnConstraint(col *sqlast.ColumnDef) bool {
	switch {
	case p.consume(NOT):
		p.expect(NULL)

		col.Nullability = sqlast.NotNullConstraint
	case p.consume(NULL):
		col.Nullability = sqlast.NullConstraint
	case p.consume(DEFAULT):
		col.Default = p.parseExpr()
	case p.consume(AutoIncrement):
		col.AutoIncrement = true
	case p.at(ON):
		p.advance()
		p.expect(UPDATE)

		col.OnUpdate = p.parseExpr()
	case p.consume(COMMENT):
		col.Comment = p.parseExpr()
	case p.parseInlineKeyConstraint(col):
	default:
		return false
	}

	return true
}

// parseInlineKeyConstraint parses an inline UNIQUE [KEY] or PRIMARY KEY
// column constraint, reporting whether it matched one.
func (p *Parser) parseInlineKeyConstraint(col *sqlast.ColumnDef) bool {
	switch {
	case p.consume(UNIQUE):
		p.consume(KEY)

		col.Unique = true
	case p.consume(PRIMARY):
		p.expect(KEY)

		col.PrimaryKey = true
	default:
		return false
	}

	return true
}

// parseDataType parses a column's data type: name, optional parenthesized
// parameters, and any UNSIGNED/ZEROFILL/CHARACTER SET/COLLATE modifiers.
func (p *Parser) parseDataType() sqlast.DataType {
	dt := sqlast.DataType{Name: p.readTypeName()}

	if p.consume(LPAREN) {
		dt.Params = p.parseTypeParams()
		p.expect(RPAREN)
	}

	for {
		switch {
		case p.consume(UNSIGNED):
			dt.Unsigned = true
		case p.consume(ZEROFILL):
			dt.Zerofill = true
		case p.consume(CHARACTER):
			p.expect(SET)

			dt.Charset = p.readIdent()
		case p.consume(CHARSET):
			dt.Charset = p.readIdent()
		case p.consume(COLLATE):
			dt.Collate = p.readIdent()
		default:
			return dt
		}
	}
}

// readTypeName reads a data type's name. This is almost always a plain
// identifier, but MySQL's SET type collides lexically with the SET keyword
// (used elsewhere for UPDATE/INSERT's SET clause), so it's special-cased
// here as the one type name that isn't a plain IDENT token.
func (p *Parser) readTypeName() string {
	if p.at(SET) {
		name := p.tok.Literal
		p.advance()

		return name
	}

	return p.readIdent()
}

// parseTypeParams parses a data type's comma-separated parenthesized
// parameters, e.g. the 10, 2 in DECIMAL(10, 2) or the 'a', 'b' in
// ENUM('a', 'b'). The current token must be positioned just after '('.
func (p *Parser) parseTypeParams() []string {
	var params []string

	for {
		switch p.tok.Type {
		case INT:
			params = append(params, p.tok.Literal)
			p.advance()
		case STRING:
			params = append(params, "'"+escapeStringLiteral(p.tok.Literal)+"'")
			p.advance()
		default:
			return failReturn[[]string](p, "expected type parameter, got %s", p.tok.Type)
		}

		if !p.consume(COMMA) {
			return params
		}
	}
}

// parseTableConstraint parses a table-level constraint or secondary index:
// [CONSTRAINT [name]] PRIMARY KEY | UNIQUE | FOREIGN KEY, or a plain
// INDEX/KEY. The current token must be one that isTableConstraintStart
// recognizes.
//
// A plain INDEX/KEY is checked first and handled without ever consuming a
// CONSTRAINT keyword, since MySQL only allows a CONSTRAINT symbol on
// PRIMARY KEY/UNIQUE/FOREIGN KEY — IndexConstraint has no field to hold one.
func (p *Parser) parseTableConstraint() sqlast.TableConstraint {
	if p.at(INDEX) || p.at(KEY) {
		p.advance()

		name := p.parseOptionalIndexName()

		return &sqlast.IndexConstraint{IndexName: name, Columns: p.parseIndexColumnList()}
	}

	constraintName := p.parseOptionalConstraintName()

	switch {
	case p.consume(PRIMARY):
		p.expect(KEY)

		return &sqlast.PrimaryKeyConstraint{ConstraintName: constraintName, Columns: p.parseIndexColumnList()}
	case p.consume(UNIQUE):
		if !p.consume(INDEX) {
			p.consume(KEY)
		}

		name := p.parseOptionalIndexName()

		return &sqlast.UniqueConstraint{ConstraintName: constraintName, IndexName: name, Columns: p.parseIndexColumnList()}
	case p.consume(FOREIGN):
		p.expect(KEY)

		return p.parseForeignKeyConstraint(constraintName)
	default:
		msg := "expected PRIMARY KEY, UNIQUE, INDEX, KEY, or FOREIGN KEY, got %s"

		return failReturn[sqlast.TableConstraint](p, msg, p.tok.Type)
	}
}

// parseOptionalConstraintName parses an optional leading CONSTRAINT [name]
// clause, returning the empty TableIdent if none is present.
func (p *Parser) parseOptionalConstraintName() sqlast.TableIdent {
	if !p.consume(CONSTRAINT) {
		return ""
	}

	if p.at(IDENT) || p.at(QuotedIdent) {
		return sqlast.TableIdent(p.readIdent())
	}

	return ""
}

// parseOptionalIndexName parses an optional index name following
// INDEX/KEY/UNIQUE [INDEX|KEY], returning the empty TableIdent if the next
// token is the column list's opening paren instead.
func (p *Parser) parseOptionalIndexName() sqlast.TableIdent {
	if p.at(IDENT) || p.at(QuotedIdent) {
		return sqlast.TableIdent(p.readIdent())
	}

	return ""
}

// parseIndexColumnList parses a parenthesized, comma-separated list of
// IndexColumns (used by PRIMARY KEY/UNIQUE/INDEX constraints and CREATE
// INDEX, which all accept a per-column prefix length and ASC/DESC).
func (p *Parser) parseIndexColumnList() []sqlast.IndexColumn {
	p.expect(LPAREN)

	var cols []sqlast.IndexColumn

	for {
		cols = append(cols, p.parseIndexColumn())

		if !p.consume(COMMA) {
			break
		}
	}

	p.expect(RPAREN)

	return cols
}

func (p *Parser) parseIndexColumn() sqlast.IndexColumn {
	col := sqlast.IndexColumn{Column: sqlast.ColIdent(p.readIdent())}

	if p.consume(LPAREN) {
		col.Length = p.parseIntLiteral()
		p.expect(RPAREN)
	}

	if p.consume(DESC) {
		col.Desc = true
	} else {
		p.consume(ASC)
	}

	return col
}

func (p *Parser) parseIntLiteral() int {
	if !p.at(INT) {
		return failReturn[int](p, "expected integer, got %s", p.tok.Type)
	}

	n, err := strconv.Atoi(p.tok.Literal)
	if err != nil {
		p.failf("invalid integer literal %q", p.tok.Literal)
	}

	p.advance()

	return n
}

// parseForeignKeyConstraint parses the remainder of a FOREIGN KEY
// constraint after the FOREIGN KEY keywords have been consumed: an optional
// index name, the local column list, REFERENCES table (columns), and any
// ON DELETE/ON UPDATE actions.
func (p *Parser) parseForeignKeyConstraint(constraintName sqlast.TableIdent) *sqlast.ForeignKeyConstraint {
	fk := &sqlast.ForeignKeyConstraint{ConstraintName: constraintName, IndexName: p.parseOptionalIndexName()}
	fk.Columns = p.parseColumnList()

	p.expect(REFERENCES)

	fk.RefTable = p.parseTableName()
	fk.RefColumns = p.parseColumnList()
	fk.OnDelete, fk.OnUpdate = p.parseOptionalReferentialActions()

	return fk
}

// parseOptionalReferentialActions parses zero or more ON DELETE/ON UPDATE
// clauses following a FOREIGN KEY's REFERENCES clause. MySQL allows at most
// one of each, so a repeated ON DELETE or ON UPDATE is a syntax error rather
// than silently overwriting the one already parsed.
func (p *Parser) parseOptionalReferentialActions() (sqlast.ReferenceAction, sqlast.ReferenceAction) {
	var onDelete, onUpdate sqlast.ReferenceAction

	var haveDelete, haveUpdate bool

	for p.at(ON) {
		p.advance()

		switch {
		case p.consume(DELETE):
			if haveDelete {
				p.failf("duplicate ON DELETE clause")
			}

			onDelete = p.parseReferenceAction()
			haveDelete = true
		case p.consume(UPDATE):
			if haveUpdate {
				p.failf("duplicate ON UPDATE clause")
			}

			onUpdate = p.parseReferenceAction()
			haveUpdate = true
		default:
			p.failf("expected DELETE or UPDATE after ON, got %s", p.tok.Type)
		}
	}

	return onDelete, onUpdate
}

func (p *Parser) parseReferenceAction() sqlast.ReferenceAction {
	switch {
	case p.consume(CASCADE):
		return sqlast.CascadeAction
	case p.consume(RESTRICT):
		return sqlast.RestrictAction
	case p.consume(SET):
		if p.consume(NULL) {
			return sqlast.SetNullAction
		}

		p.expect(DEFAULT)

		return sqlast.SetDefaultAction
	case p.consume(NO):
		p.expect(ACTION)

		return sqlast.NoActionAction
	default:
		msg := "expected CASCADE, RESTRICT, SET NULL, SET DEFAULT, or NO ACTION, got %s"

		return failReturn[sqlast.ReferenceAction](p, msg, p.tok.Type)
	}
}

// parseTableOptions parses CREATE TABLE's trailing options (ENGINE,
// DEFAULT CHARSET, COMMENT, ...), running until EOF since options are
// always the last thing in the statement.
func (p *Parser) parseTableOptions() []sqlast.TableOption {
	var opts []sqlast.TableOption

	for p.isTableOptionStart() {
		opts = append(opts, p.parseTableOption())
		p.consume(COMMA) // MySQL allows an optional comma between table options
	}

	return opts
}

// isTableOptionStart reports whether the current token could begin a table
// option: one of the explicitly recognized option-name keywords, or a bare
// identifier (the generic fallback in parseTableOptionName). Bounding the
// parseTableOptions loop on this — rather than looping until EOF — means
// trailing tokens that aren't shaped like an option name (e.g. a stray
// literal, or the start of another statement) are left for the caller's own
// trailing-token check instead of being force-fed into parseTableOption.
func (p *Parser) isTableOptionStart() bool {
	switch p.tok.Type {
	case DEFAULT, ENGINE, CHARACTER, CHARSET, COLLATE, COMMENT, AutoIncrement, IDENT, QuotedIdent:
		return true
	default:
		return false
	}
}

func (p *Parser) parseTableOption() sqlast.TableOption {
	name := p.parseTableOptionName()
	p.consume(EQ)

	return sqlast.TableOption{Name: name, Value: p.parseExpr()}
}

// parseTableOptionName parses one table option's name. The common MySQL
// options (ENGINE, DEFAULT CHARACTER SET/CHARSET/COLLATE, COLLATE, COMMENT,
// AUTO_INCREMENT) are recognized explicitly; anything else falls back to a
// single bare identifier, covering options like ROW_FORMAT or MAX_ROWS
// without needing to enumerate every one MySQL supports.
func (p *Parser) parseTableOptionName() string {
	if p.consume(DEFAULT) {
		return "DEFAULT " + p.parseDefaultTableOptionName()
	}

	switch {
	case p.consume(ENGINE):
		return "ENGINE"
	case p.consume(CHARACTER):
		p.expect(SET)

		return "CHARACTER SET"
	case p.consume(CHARSET):
		return CHARSET.String()
	case p.consume(COLLATE):
		return COLLATE.String()
	case p.consume(COMMENT):
		return "COMMENT"
	case p.consume(AutoIncrement):
		return "AUTO_INCREMENT"
	default:
		return p.readIdent()
	}
}

// parseDefaultTableOptionName parses the option name following a leading
// DEFAULT keyword: CHARACTER SET, CHARSET, or COLLATE (MySQL doesn't allow
// DEFAULT before any other table option).
func (p *Parser) parseDefaultTableOptionName() string {
	switch {
	case p.consume(CHARACTER):
		p.expect(SET)

		return "CHARACTER SET"
	case p.consume(CHARSET):
		return CHARSET.String()
	case p.consume(COLLATE):
		return COLLATE.String()
	default:
		return failReturn[string](p, "expected CHARACTER SET, CHARSET, or COLLATE after DEFAULT, got %s", p.tok.Type)
	}
}

// parseAlterTableStatement parses an ALTER TABLE statement with one or more
// comma-separated actions. The current token must be ALTER.
func (p *Parser) parseAlterTableStatement() *sqlast.AlterTable {
	p.expect(ALTER)
	p.expect(TABLE)

	at := &sqlast.AlterTable{Table: p.parseTableName()}

	for {
		at.Actions = append(at.Actions, p.parseAlterAction())

		if !p.consume(COMMA) {
			break
		}
	}

	return at
}

func (p *Parser) parseAlterAction() sqlast.AlterAction {
	switch {
	case p.consume(ADD):
		return p.parseAddAction()
	case p.consume(DROP):
		return p.parseDropAction()
	case p.consume(MODIFY):
		p.consume(COLUMN)

		return &sqlast.ModifyColumnAction{Column: p.parseColumnDef()}
	case p.consume(RENAME):
		p.expect(TO)

		return &sqlast.RenameTableAction{NewName: p.parseTableName()}
	default:
		return failReturn[sqlast.AlterAction](p, "expected ADD, DROP, MODIFY, or RENAME, got %s", p.tok.Type)
	}
}

// parseAddAction parses the remainder of an ADD action after ADD has been
// consumed: ADD INDEX/CONSTRAINT/PRIMARY KEY/UNIQUE/FOREIGN KEY all reuse
// parseTableConstraint, since they all add a TableConstraint; anything else
// is ADD [COLUMN] col_def.
func (p *Parser) parseAddAction() sqlast.AlterAction {
	if p.isTableConstraintStart() {
		return &sqlast.AddConstraintAction{Constraint: p.parseTableConstraint()}
	}

	p.consume(COLUMN)

	return &sqlast.AddColumnAction{Column: p.parseColumnDef()}
}

// parseDropAction parses the remainder of a DROP action after DROP has been
// consumed: DROP INDEX/KEY name, or DROP [COLUMN] name.
func (p *Parser) parseDropAction() sqlast.AlterAction {
	if p.at(INDEX) || p.at(KEY) {
		p.advance()

		return &sqlast.DropIndexAction{Name: sqlast.TableIdent(p.readIdent())}
	}

	p.consume(COLUMN)

	return &sqlast.DropColumnAction{Name: sqlast.ColIdent(p.readIdent())}
}

// parseCreateIndexStatement parses a CREATE [UNIQUE] INDEX statement. The
// current token must be CREATE.
func (p *Parser) parseCreateIndexStatement() *sqlast.CreateIndex {
	p.expect(CREATE)

	unique := p.consume(UNIQUE)

	p.expect(INDEX)

	ci := &sqlast.CreateIndex{Unique: unique, Name: sqlast.TableIdent(p.readIdent())}

	p.expect(ON)

	ci.Table = p.parseTableName()
	ci.Columns = p.parseIndexColumnList()

	return ci
}

// parseDropIndexStatement parses a DROP INDEX statement. The current token
// must be DROP.
func (p *Parser) parseDropIndexStatement() *sqlast.DropIndex {
	p.expect(DROP)
	p.expect(INDEX)

	di := &sqlast.DropIndex{Name: sqlast.TableIdent(p.readIdent())}

	p.expect(ON)

	di.Table = p.parseTableName()

	return di
}

// parseDropTableStatement parses a DROP TABLE statement naming one or more
// tables. The current token must be DROP.
func (p *Parser) parseDropTableStatement() *sqlast.DropTable {
	p.expect(DROP)
	p.expect(TABLE)

	dt := &sqlast.DropTable{IfExists: p.parseOptionalIfExists()}

	for {
		dt.Tables = append(dt.Tables, p.parseTableName())

		if !p.consume(COMMA) {
			break
		}
	}

	return dt
}

// parseTruncateTableStatement parses a TRUNCATE [TABLE] statement. The
// current token must be TRUNCATE.
func (p *Parser) parseTruncateTableStatement() *sqlast.TruncateTable {
	p.expect(TRUNCATE)
	p.consume(TABLE)

	return &sqlast.TruncateTable{Table: p.parseTableName()}
}

func (p *Parser) parseOptionalIfNotExists() bool {
	if !p.consume(IF) {
		return false
	}

	p.expect(NOT)
	p.expect(EXISTS)

	return true
}

func (p *Parser) parseOptionalIfExists() bool {
	if !p.consume(IF) {
		return false
	}

	p.expect(EXISTS)

	return true
}
