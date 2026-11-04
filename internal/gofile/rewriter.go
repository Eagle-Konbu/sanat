package gofile

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
)

func RewriteFile(fset *token.FileSet, file *ast.File, literals []SQLLiteral, opts *Options) ([]byte, error) {
	for _, lit := range literals {
		if formatted, changed := FormatLiteral(lit, opts); changed {
			lit.Node.Value = formatted
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
