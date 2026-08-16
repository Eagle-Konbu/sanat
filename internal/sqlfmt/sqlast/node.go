package sqlast

// SQLNode is the base interface for all AST nodes.
type SQLNode interface {
	String() string
}

// Statement represents a SQL statement.
type Statement interface {
	SQLNode
	iStatement()
}

// Expr represents a SQL expression.
type Expr interface {
	SQLNode
	iExpr()
}

// TableExpr represents a table expression in a FROM clause.
type TableExpr interface {
	SQLNode
	iTableExpr()
}

// SelectExpr represents an expression in a SELECT clause.
type SelectExpr interface {
	SQLNode
	iSelectExpr()
}

// SimpleTableExpr represents a simple table expression (table name or derived table).
type SimpleTableExpr interface {
	SQLNode
	iSimpleTableExpr()
}

// InsertRows represents the rows for an INSERT statement.
type InsertRows interface {
	SQLNode
	iInsertRows()
}

// TableElement represents one comma-separated entry inside CREATE TABLE's
// column list: either a ColumnDef or a TableConstraint variant.
type TableElement interface {
	SQLNode
	iTableElement()
}

// TableConstraint represents a table-level constraint or secondary index
// definition, shared between CREATE TABLE and ALTER TABLE's ADD CONSTRAINT
// action.
type TableConstraint interface {
	SQLNode
	iTableConstraint()
}

// AlterAction represents a single action within an ALTER TABLE statement.
type AlterAction interface {
	SQLNode
	iAlterAction()
}

// ColIdent represents a column identifier.
type ColIdent string

// String returns ColIdent's SQL text.
func (c ColIdent) String() string { return string(c) }

// IsEmpty reports whether the ColIdent is empty.
func (c ColIdent) IsEmpty() bool { return c == "" }

// TableIdent represents a table identifier.
type TableIdent string

// String returns TableIdent's SQL text.
func (t TableIdent) String() string { return string(t) }

// IsEmpty reports whether the TableIdent is empty.
func (t TableIdent) IsEmpty() bool { return t == "" }

// TableName represents a possibly qualified table name.
type TableName struct {
	Name      TableIdent
	Qualifier TableIdent
}

// String returns TableName's SQL text.
func (t TableName) String() string {
	if t.Qualifier.IsEmpty() {
		return t.Name.String()
	}

	return t.Qualifier.String() + "." + t.Name.String()
}

// IsEmpty reports whether the TableName is empty.
func (t TableName) IsEmpty() bool { return t.Name.IsEmpty() }
