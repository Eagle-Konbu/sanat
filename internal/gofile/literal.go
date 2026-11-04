package gofile

import (
	"go/ast"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

type Options struct {
	Indent      int
	Newline     bool
	KeywordCase string
	CommaStyle  string
	SQLMode     string
	Dialect     string
}

// FormattedLiteral is a literal whose source text would change under
// formatting. Node is never modified; Formatted is the complete replacement
// for Node.Value, including the enclosing backticks.
type FormattedLiteral struct {
	Node      *ast.BasicLit
	Formatted string
}

// FormatLiteral returns the replacement source text for lit, including the
// enclosing backticks, and whether it differs from the current source text.
// Strings that are not SQL, or that the formatter cannot parse, are reported
// as unchanged rather than as needing formatting.
func FormatLiteral(lit SQLLiteral, opts *Options) (string, bool) {
	if !sqlfmt.MightBeSQL(lit.Original) {
		return lit.Node.Value, false
	}

	sql, ok := sqlfmt.FormatSQLWithOptions(lit.Original, sqlfmt.Options{
		Indent:      opts.Indent,
		KeywordCase: opts.KeywordCase,
		CommaStyle:  opts.CommaStyle,
		SQLMode:     opts.SQLMode,
		Dialect:     opts.Dialect,
	})
	if !ok {
		return lit.Node.Value, false
	}

	sql = strings.TrimRight(sql, "\n")
	if opts.Newline {
		sql = "\n" + sql + "\n"
	}

	formatted := "`" + sql + "`"

	return formatted, formatted != lit.Node.Value
}

// FormatLiterals returns the raw string literals in file that would change
// under formatting, without modifying the AST.
func FormatLiterals(file *ast.File, opts *Options) []FormattedLiteral {
	var results []FormattedLiteral

	for _, lit := range CollectSQLLiterals(file) {
		if formatted, changed := FormatLiteral(lit, opts); changed {
			results = append(results, FormattedLiteral{Node: lit.Node, Formatted: formatted})
		}
	}

	return results
}
