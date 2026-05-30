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

// ColIdent represents a column identifier.
type ColIdent string

func (c ColIdent) String() string { return string(c) }
func (c ColIdent) IsEmpty() bool  { return c == "" }

// TableIdent represents a table identifier.
type TableIdent string

func (t TableIdent) String() string { return string(t) }
func (t TableIdent) IsEmpty() bool  { return t == "" }

// TableName represents a possibly qualified table name.
type TableName struct {
	Name      TableIdent
	Qualifier TableIdent
}

func (t TableName) String() string {
	if t.Qualifier.IsEmpty() {
		return t.Name.String()
	}

	return t.Qualifier.String() + "." + t.Name.String()
}

func (t TableName) IsEmpty() bool { return t.Name.IsEmpty() }

func (TableName) iSimpleTableExpr() {}
