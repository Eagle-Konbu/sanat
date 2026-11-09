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

// MergeAction represents the action of a MERGE WHEN clause: *MergeUpdate,
// MergeDelete, *MergeInsert, or MergeDoNothing.
type MergeAction interface {
	SQLNode
	iMergeAction()
}

// ColIdent represents a column identifier.
//
// PostgreSQL folds unquoted identifiers to lower case but keeps a
// double-quoted identifier's case, so "Users" and users name different
// objects. Name therefore holds the identifier as normalized SQL text:
// pgparser folds an unquoted identifier to lower case and keeps a quoted
// identifier's surrounding double quotes (with embedded quotes doubled), and
// String renders Name as-is.
//
// Raw is the identifier exactly as written in the source (UserId, "UserId"),
// for a renderer that must reproduce the user's spelling. It is display-only:
// it never affects String or what the identifier names, and it is empty for
// an identifier that was not parsed from source text.
type ColIdent struct {
	Name string
	Raw  string
}

// String returns ColIdent's SQL text.
func (c ColIdent) String() string { return c.Name }

// IsEmpty reports whether the ColIdent is empty.
func (c ColIdent) IsEmpty() bool { return c.Name == "" }

// TableIdent represents a table identifier. Its fields mean the same as
// ColIdent's.
type TableIdent struct {
	Name string
	Raw  string
}

// String returns TableIdent's SQL text.
func (t TableIdent) String() string { return t.Name }

// IsEmpty reports whether the TableIdent is empty.
func (t TableIdent) IsEmpty() bool { return t.Name == "" }

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

// AnyName represents a dotted name of any length, such as a collation's
// [database.][schema.]name.
type AnyName []ColIdent

// String returns AnyName's SQL text.
func (n AnyName) String() string {
	strs := make([]string, len(n))
	for i, part := range n {
		strs[i] = part.String()
	}

	return strings.Join(strs, ".")
}

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
