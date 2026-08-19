package sqlast

import (
	"fmt"
	"strings"
)

// CreateTable represents a CREATE TABLE statement.
type CreateTable struct {
	IfNotExists bool
	Table       TableName
	Elements    []TableElement
	Options     []TableOption
}

// String returns CreateTable's SQL text.
func (c *CreateTable) String() string {
	elems := make([]string, len(c.Elements))
	for i, e := range c.Elements {
		elems[i] = e.String()
	}

	var b strings.Builder

	b.WriteString("CREATE TABLE ")

	if c.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}

	b.WriteString(c.Table.String())
	b.WriteString(" (")
	b.WriteString(strings.Join(elems, ", "))
	b.WriteString(")")

	for _, opt := range c.Options {
		b.WriteString(" ")
		b.WriteString(opt.String())
	}

	return b.String()
}

// AlterTable represents an ALTER TABLE statement with one or more
// comma-separated actions.
type AlterTable struct {
	Table   TableName
	Actions []AlterAction
}

// String returns AlterTable's SQL text.
func (a *AlterTable) String() string {
	actions := make([]string, len(a.Actions))
	for i, act := range a.Actions {
		actions[i] = act.String()
	}

	return "ALTER TABLE " + a.Table.String() + " " + strings.Join(actions, ", ")
}

// CreateIndex represents a CREATE INDEX statement.
type CreateIndex struct {
	Unique  bool
	Name    TableIdent
	Table   TableName
	Columns []IndexColumn
}

// String returns CreateIndex's SQL text.
func (c *CreateIndex) String() string {
	s := "CREATE "
	if c.Unique {
		s += "UNIQUE "
	}

	return s + "INDEX " + c.Name.String() + " ON " + c.Table.String() + " (" + indexColumnsString(c.Columns) + ")"
}

// DropIndex represents a DROP INDEX statement.
type DropIndex struct {
	Name  TableIdent
	Table TableName
}

// String returns DropIndex's SQL text.
func (d *DropIndex) String() string {
	return "DROP INDEX " + d.Name.String() + " ON " + d.Table.String()
}

// DropTable represents a DROP TABLE statement, optionally naming multiple
// tables.
type DropTable struct {
	IfExists bool
	Tables   []TableName
}

// String returns DropTable's SQL text.
func (d *DropTable) String() string {
	names := make([]string, len(d.Tables))
	for i, t := range d.Tables {
		names[i] = t.String()
	}

	s := "DROP TABLE "
	if d.IfExists {
		s += "IF EXISTS "
	}

	return s + strings.Join(names, ", ")
}

// TruncateTable represents a TRUNCATE TABLE statement.
type TruncateTable struct {
	Table TableName
}

// String returns TruncateTable's SQL text.
func (t *TruncateTable) String() string {
	return "TRUNCATE TABLE " + t.Table.String()
}

// ColumnDef represents a single column definition inside CREATE TABLE, or
// ALTER TABLE's ADD/MODIFY COLUMN actions.
type ColumnDef struct {
	Name          ColIdent
	Type          DataType
	Nullability   Nullability
	Default       Expr // nil if unspecified
	AutoIncrement bool
	// OnUpdate is a TIMESTAMP/DATETIME column's ON UPDATE <expr> clause
	// (typically CURRENT_TIMESTAMP); nil if unspecified.
	OnUpdate   Expr
	PrimaryKey bool // inline PRIMARY KEY
	Unique     bool // inline UNIQUE [KEY]
	Comment    Expr // nil if unspecified
}

// String returns ColumnDef's SQL text.
func (c *ColumnDef) String() string {
	var b strings.Builder

	b.WriteString(c.Name.String())
	b.WriteString(" ")
	b.WriteString(c.Type.String())

	switch c.Nullability {
	case NotNullConstraint:
		b.WriteString(" NOT NULL")
	case NullConstraint:
		b.WriteString(" NULL")
	case NullabilityUnspecified:
	}

	if c.Default != nil {
		b.WriteString(" DEFAULT ")
		b.WriteString(c.Default.String())
	}

	if c.AutoIncrement {
		b.WriteString(" AUTO_INCREMENT")
	}

	if c.OnUpdate != nil {
		b.WriteString(" ON UPDATE ")
		b.WriteString(c.OnUpdate.String())
	}

	if c.PrimaryKey {
		b.WriteString(" PRIMARY KEY")
	}

	if c.Unique {
		b.WriteString(" UNIQUE")
	}

	if c.Comment != nil {
		b.WriteString(" COMMENT ")
		b.WriteString(c.Comment.String())
	}

	return b.String()
}

// Nullability represents an explicit NULL/NOT NULL column constraint;
// NullabilityUnspecified means neither was written.
type Nullability int8

const (
	NullabilityUnspecified Nullability = iota
	NullConstraint
	NotNullConstraint
)

// DataType represents a column's data type. It carries the type name and
// parameters generically (rather than an enum of known MySQL types), plus
// the modifiers common to numeric and string types.
type DataType struct {
	Name     string
	Params   []string // e.g. ["10", "2"] for DECIMAL(10, 2), ["'a'", "'b'"] for ENUM('a', 'b')
	Unsigned bool
	Zerofill bool
	Charset  string
	Collate  string
}

// String returns DataType's SQL text.
func (d DataType) String() string {
	var b strings.Builder

	b.WriteString(d.Name)

	if len(d.Params) > 0 {
		b.WriteString("(")
		b.WriteString(strings.Join(d.Params, ", "))
		b.WriteString(")")
	}

	if d.Unsigned {
		b.WriteString(" UNSIGNED")
	}

	if d.Zerofill {
		b.WriteString(" ZEROFILL")
	}

	if d.Charset != "" {
		b.WriteString(" CHARACTER SET ")
		b.WriteString(d.Charset)
	}

	if d.Collate != "" {
		b.WriteString(" COLLATE ")
		b.WriteString(d.Collate)
	}

	return b.String()
}

// IndexColumn represents one entry in an index's column list — shared by
// CREATE INDEX and the PRIMARY KEY/UNIQUE/INDEX table constraints, which all
// accept an optional prefix length and ASC/DESC direction per column.
type IndexColumn struct {
	Column ColIdent
	Length int // key prefix length, e.g. the 20 in col(20); 0 means unspecified
	Desc   bool
}

// String returns IndexColumn's SQL text.
func (i IndexColumn) String() string {
	s := i.Column.String()
	if i.Length > 0 {
		s += fmt.Sprintf("(%d)", i.Length)
	}

	if i.Desc {
		s += " DESC"
	}

	return s
}

func indexColumnsString(cols []IndexColumn) string {
	strs := make([]string, len(cols))
	for i, c := range cols {
		strs[i] = c.String()
	}

	return strings.Join(strs, ", ")
}

// PrimaryKeyConstraint represents a table-level PRIMARY KEY (cols) constraint.
type PrimaryKeyConstraint struct {
	ConstraintName TableIdent // optional CONSTRAINT symbol
	Columns        []IndexColumn
}

// String returns PrimaryKeyConstraint's SQL text.
func (c *PrimaryKeyConstraint) String() string {
	return constraintNamePrefix(c.ConstraintName) + "PRIMARY KEY (" + indexColumnsString(c.Columns) + ")"
}

// UniqueConstraint represents a table-level UNIQUE [KEY] [name] (cols)
// constraint.
type UniqueConstraint struct {
	ConstraintName TableIdent // optional CONSTRAINT symbol
	IndexName      TableIdent
	Columns        []IndexColumn
}

// String returns UniqueConstraint's SQL text.
func (c *UniqueConstraint) String() string {
	s := constraintNamePrefix(c.ConstraintName) + "UNIQUE KEY"
	if !c.IndexName.IsEmpty() {
		s += " " + c.IndexName.String()
	}

	return s + " (" + indexColumnsString(c.Columns) + ")"
}

// IndexConstraint represents a table-level INDEX/KEY [name] (cols) secondary
// index (no CONSTRAINT symbol — MySQL doesn't allow one on a plain index).
type IndexConstraint struct {
	IndexName TableIdent
	Columns   []IndexColumn
}

// String returns IndexConstraint's SQL text.
func (c *IndexConstraint) String() string {
	s := "INDEX"
	if !c.IndexName.IsEmpty() {
		s += " " + c.IndexName.String()
	}

	return s + " (" + indexColumnsString(c.Columns) + ")"
}

// ForeignKeyConstraint represents a table-level FOREIGN KEY constraint.
type ForeignKeyConstraint struct {
	ConstraintName TableIdent // optional CONSTRAINT symbol
	IndexName      TableIdent
	Columns        Columns
	RefTable       TableName
	RefColumns     Columns
	OnDelete       ReferenceAction
	OnUpdate       ReferenceAction
}

// String returns ForeignKeyConstraint's SQL text.
func (c *ForeignKeyConstraint) String() string {
	s := constraintNamePrefix(c.ConstraintName) + "FOREIGN KEY"
	if !c.IndexName.IsEmpty() {
		s += " " + c.IndexName.String()
	}

	s += " (" + c.Columns.String() + ") REFERENCES " + c.RefTable.String() + " (" + c.RefColumns.String() + ")"

	if c.OnDelete != NoRefAction {
		s += " ON DELETE " + c.OnDelete.String()
	}

	if c.OnUpdate != NoRefAction {
		s += " ON UPDATE " + c.OnUpdate.String()
	}

	return s
}

func constraintNamePrefix(name TableIdent) string {
	if name.IsEmpty() {
		return ""
	}

	return "CONSTRAINT " + name.String() + " "
}

// ReferenceAction represents a FOREIGN KEY's ON DELETE/ON UPDATE action;
// NoRefAction means the clause was omitted.
type ReferenceAction int8

const (
	NoRefAction ReferenceAction = iota
	CascadeAction
	SetNullAction
	RestrictAction
	NoActionAction
	SetDefaultAction
)

// String returns ReferenceAction's SQL text, or "" for NoRefAction.
func (r ReferenceAction) String() string {
	switch r {
	case CascadeAction:
		return "CASCADE"
	case SetNullAction:
		return "SET NULL"
	case RestrictAction:
		return "RESTRICT"
	case NoActionAction:
		return "NO ACTION"
	case SetDefaultAction:
		return "SET DEFAULT"
	default:
		return ""
	}
}

// TableOption represents a single CREATE TABLE trailing option (ENGINE,
// DEFAULT CHARSET, COMMENT, ...). Modeled generically — a name/value pair —
// rather than as an enum of known option names, since MySQL has dozens of
// them and the formatter only needs to round-trip whichever ones appear.
type TableOption struct {
	Name  string
	Value Expr
}

// String returns TableOption's SQL text.
func (o TableOption) String() string {
	return o.Name + "=" + o.Value.String()
}

// AddColumnAction represents an ALTER TABLE ADD [COLUMN] col_def action.
type AddColumnAction struct {
	Column *ColumnDef
}

// String returns AddColumnAction's SQL text.
func (a *AddColumnAction) String() string {
	return "ADD COLUMN " + a.Column.String()
}

// AddConstraintAction represents an ALTER TABLE action that adds a table
// constraint or secondary index: ADD INDEX, ADD CONSTRAINT, ADD PRIMARY KEY,
// ADD UNIQUE, or ADD FOREIGN KEY all produce this, wrapping whichever
// TableConstraint variant matches.
type AddConstraintAction struct {
	Constraint TableConstraint
}

// String returns AddConstraintAction's SQL text.
func (a *AddConstraintAction) String() string {
	return "ADD " + a.Constraint.String()
}

// DropColumnAction represents an ALTER TABLE DROP [COLUMN] col_name action.
type DropColumnAction struct {
	Name ColIdent
}

// String returns DropColumnAction's SQL text.
func (a *DropColumnAction) String() string {
	return "DROP COLUMN " + a.Name.String()
}

// DropIndexAction represents an ALTER TABLE DROP INDEX/KEY index_name action.
type DropIndexAction struct {
	Name TableIdent
}

// String returns DropIndexAction's SQL text.
func (a *DropIndexAction) String() string {
	return "DROP INDEX " + a.Name.String()
}

// ModifyColumnAction represents an ALTER TABLE MODIFY [COLUMN] col_def action.
type ModifyColumnAction struct {
	Column *ColumnDef
}

// String returns ModifyColumnAction's SQL text.
func (a *ModifyColumnAction) String() string {
	return "MODIFY COLUMN " + a.Column.String()
}

// RenameTableAction represents an ALTER TABLE RENAME TO new_name action.
type RenameTableAction struct {
	NewName TableName
}

// String returns RenameTableAction's SQL text.
func (a *RenameTableAction) String() string {
	return "RENAME TO " + a.NewName.String()
}
