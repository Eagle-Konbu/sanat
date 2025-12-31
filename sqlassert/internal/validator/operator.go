package validator

import (
	"strings"

	"github.com/pingcap/tidb/pkg/parser/opcode"
)

// MatchesOperator checks if an opcode matches the given operator string.
func MatchesOperator(op opcode.Op, opStr string) bool {
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
