package sqlast

// ShowTables represents a SHOW TABLES statement.
type ShowTables struct {
	Database TableIdent // empty if no FROM database clause
	Like     Expr       // nil if no LIKE pattern
}

// String returns ShowTables's SQL text.
func (s *ShowTables) String() string {
	str := "SHOW TABLES"

	if !s.Database.IsEmpty() {
		str += " FROM " + s.Database.String()
	}

	if s.Like != nil {
		str += " LIKE " + s.Like.String()
	}

	return str
}

// ShowCreateTable represents a SHOW CREATE TABLE statement.
type ShowCreateTable struct {
	Table TableName
}

// String returns ShowCreateTable's SQL text.
func (s *ShowCreateTable) String() string {
	return "SHOW CREATE TABLE " + s.Table.String()
}

// ShowColumns represents a SHOW COLUMNS FROM statement.
type ShowColumns struct {
	Table TableName
}

// String returns ShowColumns's SQL text.
func (s *ShowColumns) String() string {
	return "SHOW COLUMNS FROM " + s.Table.String()
}

// ShowIndex represents a SHOW INDEX FROM statement.
type ShowIndex struct {
	Table TableName
}

// String returns ShowIndex's SQL text.
func (s *ShowIndex) String() string {
	return "SHOW INDEX FROM " + s.Table.String()
}

// ShowDatabases represents a SHOW DATABASES statement.
type ShowDatabases struct{}

// String returns ShowDatabases's SQL text.
func (*ShowDatabases) String() string { return "SHOW DATABASES" }

// ShowVariables represents a SHOW VARIABLES statement.
type ShowVariables struct {
	Like Expr // nil if no LIKE pattern
}

// String returns ShowVariables's SQL text.
func (s *ShowVariables) String() string {
	str := "SHOW VARIABLES"

	if s.Like != nil {
		str += " LIKE " + s.Like.String()
	}

	return str
}

// ShowStatus represents a SHOW STATUS statement.
type ShowStatus struct {
	Like Expr // nil if no LIKE pattern
}

// String returns ShowStatus's SQL text.
func (s *ShowStatus) String() string {
	str := "SHOW STATUS"

	if s.Like != nil {
		str += " LIKE " + s.Like.String()
	}

	return str
}

// Describe represents a DESCRIBE statement.
type Describe struct {
	Table  TableName
	Column ColIdent // empty if no column name is given
}

// String returns Describe's SQL text.
func (d *Describe) String() string {
	str := "DESCRIBE " + d.Table.String()

	if !d.Column.IsEmpty() {
		str += " " + d.Column.String()
	}

	return str
}

// ExplainFormat represents EXPLAIN's optional FORMAT modifier.
type ExplainFormat int8

const (
	NoExplainFormat ExplainFormat = iota
	TraditionalFormat
	JSONFormat
	TreeFormat
)

// String returns ExplainFormat's SQL text, or "" for NoExplainFormat.
func (f ExplainFormat) String() string {
	switch f {
	case TraditionalFormat:
		return "TRADITIONAL"
	case JSONFormat:
		return "JSON"
	case TreeFormat:
		return "TREE"
	default:
		return ""
	}
}

// Explain represents an EXPLAIN statement wrapping a SELECT statement (or a
// UNION of SELECT branches, optionally preceded by a WITH clause).
type Explain struct {
	Format    ExplainFormat
	Statement Statement
}

// String returns Explain's SQL text.
func (e *Explain) String() string {
	str := "EXPLAIN"

	if format := e.Format.String(); format != "" {
		str += " FORMAT = " + format
	}

	return str + " " + e.Statement.String()
}

// Use represents a USE statement.
type Use struct {
	Database TableIdent
}

// String returns Use's SQL text.
func (u *Use) String() string {
	return "USE " + u.Database.String()
}
