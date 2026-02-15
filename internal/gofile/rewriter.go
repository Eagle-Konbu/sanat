package gofile

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

type Options struct {
	Indent  int
	Newline bool
}

func RewriteFile(fset *token.FileSet, file *ast.File, literals []SQLLiteral, opts Options) ([]byte, error) {
	for _, lit := range literals {
		if !sqlfmt.MightBeSQL(lit.Original) {
			continue
		}
		formatted, ok := sqlfmt.FormatSQL(lit.Original, opts.Indent)
		if !ok {
			continue
		}
		formatted = strings.TrimRight(formatted, "\n")
		if opts.Newline {
			formatted = "\n" + formatted + "\n"
		}
		lit.Node.Value = "`" + formatted + "`"
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
