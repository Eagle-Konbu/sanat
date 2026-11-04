// Package pgast defines the PostgreSQL abstract syntax tree, mirroring the
// shape of internal/sqlfmt/sqlast but for PostgreSQL's DML grammar. It holds
// type definitions and fallback String() serialization only — no parsing
// logic (see internal/sqlfmt/pgparser) and no DDL/session statement types
// (a separate epic).
package pgast

import "strings"

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

// SelectExpr represents an expression in a SELECT (or RETURNING) list.
type SelectExpr interface {
	SQLNode
	iSelectExpr()
}

// SimpleTableExpr represents a simple table expression (table name or derived table).
type SimpleTableExpr interface {
	SQLNode
	iSimpleTableExpr()
}

// InsertRows represents the source of rows for an INSERT statement.
type InsertRows interface {
	SQLNode
	iInsertRows()
}

// ColIdent represents a column identifier.
//
// Unlike sqlast.ColIdent (backtick-quoted in MySQL), PostgreSQL quotes
// identifiers with double quotes and folds unquoted identifiers to lower
// case. Neither concern is modeled here: like sqlast.ColIdent, this stores
// the identifier's resolved name and renders it unquoted — pgparser is
// responsible for resolving quoting/case-folding into that name, the same
// division of responsibility sqlast/parser already uses.
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

// Columns represents a list of column identifiers.
type Columns []ColIdent

// String returns Columns's SQL text.
func (c Columns) String() string {
	strs := make([]string, len(c))
	for i, col := range c {
		strs[i] = col.String()
	}

	return strings.Join(strs, ", ")
}
