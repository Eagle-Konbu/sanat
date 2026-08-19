package sqlast

import "strings"

// TransactionMode represents START TRANSACTION's optional READ ONLY /
// READ WRITE characteristic.
type TransactionMode int8

const (
	NoTransactionMode TransactionMode = iota
	ReadOnlyMode
	ReadWriteMode
)

// String returns TransactionMode's SQL text, or "" for NoTransactionMode.
func (m TransactionMode) String() string {
	switch m {
	case ReadOnlyMode:
		return "READ ONLY"
	case ReadWriteMode:
		return "READ WRITE"
	default:
		return ""
	}
}

// StartTransaction represents a START TRANSACTION statement.
type StartTransaction struct {
	Mode TransactionMode
}

// String returns StartTransaction's SQL text.
func (s *StartTransaction) String() string {
	str := "START TRANSACTION"

	if mode := s.Mode.String(); mode != "" {
		str += " " + mode
	}

	return str
}

// Begin represents a BEGIN statement.
type Begin struct{}

// String returns Begin's SQL text.
func (*Begin) String() string { return "BEGIN" }

// Commit represents a COMMIT statement.
type Commit struct{}

// String returns Commit's SQL text.
func (*Commit) String() string { return "COMMIT" }

// Rollback represents a ROLLBACK statement, or a ROLLBACK TO [SAVEPOINT]
// statement when SavepointName is non-empty.
type Rollback struct {
	SavepointName TableIdent
}

// String returns Rollback's SQL text.
func (r *Rollback) String() string {
	if r.SavepointName.IsEmpty() {
		return "ROLLBACK"
	}

	return "ROLLBACK TO SAVEPOINT " + r.SavepointName.String()
}

// Savepoint represents a SAVEPOINT statement.
type Savepoint struct {
	Name TableIdent
}

// String returns Savepoint's SQL text.
func (s *Savepoint) String() string {
	return "SAVEPOINT " + s.Name.String()
}

// ReleaseSavepoint represents a RELEASE SAVEPOINT statement.
type ReleaseSavepoint struct {
	Name TableIdent
}

// String returns ReleaseSavepoint's SQL text.
func (r *ReleaseSavepoint) String() string {
	return "RELEASE SAVEPOINT " + r.Name.String()
}

// VariableScope represents the optional scope prefix on a SetVariable
// statement's assignment.
type VariableScope int8

const (
	NoScope VariableScope = iota
	SessionScope
	GlobalScope
)

// String returns VariableScope's SQL text, or "" for NoScope.
func (s VariableScope) String() string {
	switch s {
	case SessionScope:
		return "SESSION"
	case GlobalScope:
		return "GLOBAL"
	default:
		return ""
	}
}

// SetVariable represents a SET statement assigning a user-defined variable
// (Name starting with "@") or a session/global system variable.
type SetVariable struct {
	Scope VariableScope
	Name  string
	Value Expr
}

// String returns SetVariable's SQL text.
func (s *SetVariable) String() string {
	var b strings.Builder

	b.WriteString("SET ")

	if scope := s.Scope.String(); scope != "" {
		b.WriteString(scope)
		b.WriteString(" ")
	}

	b.WriteString(s.Name)
	b.WriteString(" = ")
	b.WriteString(s.Value.String())

	return b.String()
}

// SetNames represents a SET NAMES statement.
type SetNames struct {
	Charset Expr
	Collate Expr // nil if no COLLATE clause is present
}

// String returns SetNames's SQL text.
func (s *SetNames) String() string {
	str := "SET NAMES " + s.Charset.String()
	if s.Collate != nil {
		str += " COLLATE " + s.Collate.String()
	}

	return str
}
