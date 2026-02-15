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

	return file, fset, literals, nil
}
