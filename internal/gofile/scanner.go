package gofile

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type SQLLiteral struct {
	Node     *ast.BasicLit
	Original string
}

func isRawStringLit(value string) bool {
	return len(value) >= 2 && value[0] == '`' && value[len(value)-1] == '`'
}

func FindSQLLiterals(src []byte, filename string) (*ast.File, *token.FileSet, []SQLLiteral, error) {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, err
	}

	return file, fset, CollectSQLLiterals(file), nil
}

// CollectSQLLiterals returns the raw string literals in an already-parsed
// file, so callers such as go/analysis passes can reuse their own AST.
func CollectSQLLiterals(file *ast.File) []SQLLiteral {
	var literals []SQLLiteral

	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		if !isRawStringLit(lit.Value) {
			return true
		}

		val := lit.Value[1 : len(lit.Value)-1]
		literals = append(literals, SQLLiteral{Node: lit, Original: val})

		return true
	})

	return literals
}
