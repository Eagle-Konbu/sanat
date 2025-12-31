package sqlassert

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pingcap/tidb/pkg/parser/opcode"
)

// Matcher is an interface for matching AST expression nodes.
type Matcher interface {
	Match(ast.ExprNode) bool
	Describe() string
}

// RequireExprMatch asserts that the expression matches the given matcher.
func RequireExprMatch(t *testing.T, expr ast.ExprNode, m Matcher) {
	t.Helper()

	if expr == nil {
		t.Fatalf("expression is nil, expected: %s", m.Describe())
	}

	if !m.Match(expr) {
		t.Fatalf("expression does not match: %s", m.Describe())
	}
}

// colMatcher matches a column reference.
type colMatcher struct {
	tableAlias string
	column     string
}

// Col creates a Matcher for a column reference.
// tableAlias can be empty to match any table.
func Col(tableAlias, column string) Matcher {
	return &colMatcher{
		tableAlias: tableAlias,
		column:     column,
	}
}

func (m *colMatcher) Match(expr ast.ExprNode) bool {
	colExpr, ok := expr.(*ast.ColumnNameExpr)
	if !ok {
		return false
	}

	if !strings.EqualFold(colExpr.Name.Name.L, m.column) {
		return false
	}

	if m.tableAlias != "" {
		return strings.EqualFold(colExpr.Name.Table.L, m.tableAlias)
	}

	return true
}

func (m *colMatcher) Describe() string {
	if m.tableAlias != "" {
		return fmt.Sprintf("column %s.%s", m.tableAlias, m.column)
	}
	return fmt.Sprintf("column %s", m.column)
}

// funcMatcher matches a function call.
type funcMatcher struct {
	name string
	args []Matcher
}

// Func creates a Matcher for a function call with optional argument matchers.
func Func(name string, args ...Matcher) Matcher {
	return &funcMatcher{
		name: name,
		args: args,
	}
}

func (m *funcMatcher) Match(expr ast.ExprNode) bool {
	funcExpr, ok := expr.(*ast.FuncCallExpr)
	if !ok {
		// Also try aggregate function
		aggExpr, ok := expr.(*ast.AggregateFuncExpr)
		if !ok {
			return false
		}
		if !strings.EqualFold(aggExpr.F, m.name) {
			return false
		}
		// Match arguments if specified
		if len(m.args) > 0 {
			if len(aggExpr.Args) != len(m.args) {
				return false
			}
			for i, argMatcher := range m.args {
				if !argMatcher.Match(aggExpr.Args[i]) {
					return false
				}
			}
		}
		return true
	}

	if !strings.EqualFold(funcExpr.FnName.L, m.name) {
		return false
	}

	// Match arguments if specified
	if len(m.args) > 0 {
		if len(funcExpr.Args) != len(m.args) {
			return false
		}
		for i, argMatcher := range m.args {
			if !argMatcher.Match(funcExpr.Args[i]) {
				return false
			}
		}
	}

	return true
}

func (m *funcMatcher) Describe() string {
	if len(m.args) > 0 {
		argDescs := make([]string, len(m.args))
		for i, arg := range m.args {
			argDescs[i] = arg.Describe()
		}
		return fmt.Sprintf("function %s(%s)", m.name, strings.Join(argDescs, ", "))
	}
	return fmt.Sprintf("function %s", m.name)
}

// binaryMatcher matches a binary operation.
type binaryMatcher struct {
	op    string
	left  Matcher
	right Matcher
}

// Binary creates a Matcher for a binary operation.
// op should be a string like "=", "<", ">", "AND", "OR", etc.
func Binary(op string, left, right Matcher) Matcher {
	return &binaryMatcher{
		op:    op,
		left:  left,
		right: right,
	}
}

func (m *binaryMatcher) Match(expr ast.ExprNode) bool {
	binExpr, ok := expr.(*ast.BinaryOperationExpr)
	if !ok {
		return false
	}

	// Match operator
	if !matchesOperator(binExpr.Op, m.op) {
		return false
	}

	// Match left and right operands
	return m.left.Match(binExpr.L) && m.right.Match(binExpr.R)
}

func (m *binaryMatcher) Describe() string {
	return fmt.Sprintf("binary op %s (%s %s %s)", m.op, m.left.Describe(), m.op, m.right.Describe())
}

// matchesOperator checks if an opcode matches the given operator string.
func matchesOperator(op opcode.Op, opStr string) bool {
	opStr = strings.ToUpper(opStr)
	switch opStr {
	case "=", "EQ":
		return op == opcode.EQ
	case "!=", "<>", "NE":
		return op == opcode.NE
	case "<", "LT":
		return op == opcode.LT
	case "<=", "LE":
		return op == opcode.LE
	case ">", "GT":
		return op == opcode.GT
	case ">=", "GE":
		return op == opcode.GE
	case "AND", "LOGICAND":
		return op == opcode.LogicAnd
	case "OR", "LOGICOR":
		return op == opcode.LogicOr
	case "+", "PLUS":
		return op == opcode.Plus
	case "-", "MINUS":
		return op == opcode.Minus
	case "*", "MUL":
		return op == opcode.Mul
	case "/", "DIV":
		return op == opcode.Div
	default:
		return false
	}
}

// subqueryMatcher matches a subquery.
type subqueryMatcher struct {
	validator func(*ast.SelectStmt) error
}

// Subquery creates a Matcher for a subquery expression.
// The validator function receives the inner SELECT statement and should return
// an error if it doesn't match expectations.
func Subquery(validator func(*ast.SelectStmt) error) Matcher {
	return &subqueryMatcher{
		validator: validator,
	}
}

func (m *subqueryMatcher) Match(expr ast.ExprNode) bool {
	subExpr, ok := expr.(*ast.SubqueryExpr)
	if !ok {
		return false
	}

	selStmt, ok := subExpr.Query.(*ast.SelectStmt)
	if !ok {
		return false
	}

	if m.validator != nil {
		return m.validator(selStmt) == nil
	}

	return true
}

func (m *subqueryMatcher) Describe() string {
	return "subquery"
}

// anyMatcher matches any expression.
type anyMatcher struct{}

// Any creates a Matcher that matches any expression.
func Any() Matcher {
	return &anyMatcher{}
}

func (m *anyMatcher) Match(expr ast.ExprNode) bool {
	return expr != nil
}

func (m *anyMatcher) Describe() string {
	return "any expression"
}
