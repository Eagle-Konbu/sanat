package parser

import (
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

// ParseShowTables parses a SHOW TABLES statement from input.
func ParseShowTables(input string) (*sqlast.ShowTables, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseShowTablesStatement)
}

// ParseShowCreateTable parses a SHOW CREATE TABLE statement from input.
func ParseShowCreateTable(input string) (*sqlast.ShowCreateTable, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseShowCreateTableStatement)
}

// ParseShowColumns parses a SHOW COLUMNS FROM statement from input.
func ParseShowColumns(input string) (*sqlast.ShowColumns, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseShowColumnsStatement)
}

// ParseShowIndex parses a SHOW INDEX FROM statement from input.
func ParseShowIndex(input string) (*sqlast.ShowIndex, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseShowIndexStatement)
}

// ParseShowDatabases parses a SHOW DATABASES statement from input.
func ParseShowDatabases(input string) (*sqlast.ShowDatabases, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseShowDatabasesStatement)
}

// ParseShowVariables parses a SHOW VARIABLES statement from input, using
// ModeDefault.
func ParseShowVariables(input string) (*sqlast.ShowVariables, error) {
	return ParseShowVariablesWithMode(input, ModeDefault)
}

// ParseShowVariablesWithMode parses a SHOW VARIABLES statement from input,
// decoding string literals (the optional LIKE pattern) per mode.
func ParseShowVariablesWithMode(input string, mode SQLMode) (*sqlast.ShowVariables, error) {
	return parseDDLEntry(input, mode, (*Parser).parseShowVariablesStatement)
}

// ParseShowStatus parses a SHOW STATUS statement from input, using
// ModeDefault.
func ParseShowStatus(input string) (*sqlast.ShowStatus, error) {
	return ParseShowStatusWithMode(input, ModeDefault)
}

// ParseShowStatusWithMode parses a SHOW STATUS statement from input,
// decoding string literals (the optional LIKE pattern) per mode.
func ParseShowStatusWithMode(input string, mode SQLMode) (*sqlast.ShowStatus, error) {
	return parseDDLEntry(input, mode, (*Parser).parseShowStatusStatement)
}

// ParseDescribe parses a DESCRIBE statement from input, using ModeDefault.
func ParseDescribe(input string) (*sqlast.Describe, error) {
	return ParseDescribeWithMode(input, ModeDefault)
}

// ParseDescribeWithMode parses a DESCRIBE statement from input, decoding
// string literals per mode.
func ParseDescribeWithMode(input string, mode SQLMode) (*sqlast.Describe, error) {
	return parseDDLEntry(input, mode, (*Parser).parseDescribeStatement)
}

// ParseExplain parses an EXPLAIN statement from input, using ModeDefault.
func ParseExplain(input string) (*sqlast.Explain, error) {
	return ParseExplainWithMode(input, ModeDefault)
}

// ParseExplainWithMode parses an EXPLAIN statement from input, decoding
// string literals in the wrapped statement per mode.
func ParseExplainWithMode(input string, mode SQLMode) (*sqlast.Explain, error) {
	return parseDDLEntry(input, mode, (*Parser).parseExplainStatement)
}

// ParseUse parses a USE statement from input.
func ParseUse(input string) (*sqlast.Use, error) {
	return parseDDLEntry(input, ModeDefault, (*Parser).parseUseStatement)
}

// parseAdminStatement dispatches a leading SHOW, DESCRIBE, EXPLAIN, or USE
// to its statement parser, split out of parseStatement to keep that switch's
// cyclomatic complexity down. It reports whether the current token started
// a recognized admin/utility statement.
func (p *Parser) parseAdminStatement() (sqlast.Statement, bool) {
	switch {
	case p.at(SHOW):
		return p.parseShowStatement(), true
	case p.at(DESCRIBE):
		return p.parseDescribeStatement(), true
	case p.at(EXPLAIN):
		return p.parseExplainStatement(), true
	case p.at(USE):
		return p.parseUseStatement(), true
	default:
		return nil, false
	}
}

// parseShowStatement dispatches a leading SHOW to the specific SHOW variant
// based on the following keyword. The current token must be SHOW.
func (p *Parser) parseShowStatement() sqlast.Statement {
	switch {
	case p.peekAt(TABLES):
		return p.parseShowTablesStatement()
	case p.peekAt(CREATE):
		return p.parseShowCreateTableStatement()
	case p.peekAt(COLUMNS):
		return p.parseShowColumnsStatement()
	case p.peekAt(INDEX):
		return p.parseShowIndexStatement()
	case p.peekAt(DATABASES):
		return p.parseShowDatabasesStatement()
	case p.peekAt(VARIABLES):
		return p.parseShowVariablesStatement()
	case p.peekAt(STATUS):
		return p.parseShowStatusStatement()
	default:
		return failReturn[sqlast.Statement](p,
			"expected TABLES, CREATE, COLUMNS, INDEX, DATABASES, VARIABLES, or STATUS after SHOW, got %s", p.peekTok.Type)
	}
}

// parseShowTablesStatement parses a SHOW TABLES [FROM database] [LIKE
// pattern] statement. The current token must be SHOW.
func (p *Parser) parseShowTablesStatement() *sqlast.ShowTables {
	p.expect(SHOW)
	p.expect(TABLES)

	st := &sqlast.ShowTables{}

	if p.consume(FROM) {
		st.Database = sqlast.TableIdent(p.readIdent())
	}

	st.Like = p.parseOptionalLike()

	return st
}

// parseShowCreateTableStatement parses a SHOW CREATE TABLE table_name
// statement. The current token must be SHOW.
func (p *Parser) parseShowCreateTableStatement() *sqlast.ShowCreateTable {
	p.expect(SHOW)
	p.expect(CREATE)
	p.expect(TABLE)

	return &sqlast.ShowCreateTable{Table: p.parseTableName()}
}

// parseShowColumnsStatement parses a SHOW COLUMNS FROM table_name statement.
// The current token must be SHOW.
func (p *Parser) parseShowColumnsStatement() *sqlast.ShowColumns {
	p.expect(SHOW)
	p.expect(COLUMNS)
	p.expect(FROM)

	return &sqlast.ShowColumns{Table: p.parseTableName()}
}

// parseShowIndexStatement parses a SHOW INDEX FROM table_name statement. The
// current token must be SHOW.
func (p *Parser) parseShowIndexStatement() *sqlast.ShowIndex {
	p.expect(SHOW)
	p.expect(INDEX)
	p.expect(FROM)

	return &sqlast.ShowIndex{Table: p.parseTableName()}
}

// parseShowDatabasesStatement parses a SHOW DATABASES statement. The current
// token must be SHOW.
func (p *Parser) parseShowDatabasesStatement() *sqlast.ShowDatabases {
	p.expect(SHOW)
	p.expect(DATABASES)

	return &sqlast.ShowDatabases{}
}

// parseShowVariablesStatement parses a SHOW VARIABLES [LIKE pattern]
// statement. The current token must be SHOW.
func (p *Parser) parseShowVariablesStatement() *sqlast.ShowVariables {
	p.expect(SHOW)
	p.expect(VARIABLES)

	return &sqlast.ShowVariables{Like: p.parseOptionalLike()}
}

// parseShowStatusStatement parses a SHOW STATUS [LIKE pattern] statement.
// The current token must be SHOW.
func (p *Parser) parseShowStatusStatement() *sqlast.ShowStatus {
	p.expect(SHOW)
	p.expect(STATUS)

	return &sqlast.ShowStatus{Like: p.parseOptionalLike()}
}

// parseOptionalLike parses an optional LIKE pattern clause: MySQL only
// accepts a quoted string literal here, never the full expression grammar.
func (p *Parser) parseOptionalLike() sqlast.Expr {
	if !p.consume(LIKE) {
		return nil
	}

	if !p.at(STRING) {
		p.failf("expected string literal after LIKE, got %s", p.tok.Type)
	}

	return p.parseStringLiteral()
}

// parseDescribeStatement parses a DESCRIBE table_name [col_name] statement.
// The current token must be DESCRIBE.
func (p *Parser) parseDescribeStatement() *sqlast.Describe {
	p.expect(DESCRIBE)

	d := &sqlast.Describe{Table: p.parseTableName()}

	if p.at(IDENT) || p.at(QuotedIdent) || p.tok.Type.IsNonReservedKeyword() {
		d.Column = sqlast.ColIdent(p.readIdent())
	}

	return d
}

// parseExplainStatement parses an EXPLAIN [FORMAT = {TRADITIONAL|JSON|TREE}]
// select_stmt statement, where select_stmt is a SELECT statement (or a UNION
// of SELECT branches, optionally preceded by a WITH clause). The current
// token must be EXPLAIN.
func (p *Parser) parseExplainStatement() *sqlast.Explain {
	p.expect(EXPLAIN)

	e := &sqlast.Explain{}

	if p.consume(FORMAT) {
		p.expect(EQ)

		e.Format = p.parseExplainFormat()
	}

	e.Statement = p.parseSelectOrUnionAfterWith(p.parseOptionalWith())

	return e
}

// parseExplainFormat parses EXPLAIN's FORMAT value: one of TRADITIONAL,
// JSON, or TREE, matched case-insensitively.
func (p *Parser) parseExplainFormat() sqlast.ExplainFormat {
	name := p.readIdent()

	switch strings.ToUpper(name) {
	case "TRADITIONAL":
		return sqlast.TraditionalFormat
	case "JSON":
		return sqlast.JSONFormat
	case "TREE":
		return sqlast.TreeFormat
	default:
		return failReturn[sqlast.ExplainFormat](p, "expected TRADITIONAL, JSON, or TREE, got %q", name)
	}
}

// parseUseStatement parses a USE database_name statement. The current token
// must be USE.
func (p *Parser) parseUseStatement() *sqlast.Use {
	p.expect(USE)

	return &sqlast.Use{Database: sqlast.TableIdent(p.readIdent())}
}
