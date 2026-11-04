package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"

	"github.com/Eagle-Konbu/sanat/internal/gofile"
)

const (
	name    = "sanat"
	doc     = "reports embedded SQL raw string literals that are not formatted by sanat"
	message = "SQL literal is not formatted"
)

// New returns an analyzer that reports SQL literals whose source text would
// change under formatting with opts. Callers resolve configuration files and
// flags themselves; the analyzer never reads them.
func New(opts *gofile.Options) *analysis.Analyzer {
	return &analysis.Analyzer{
		Name: name,
		Doc:  doc,
		Run: func(pass *analysis.Pass) (any, error) {
			run(pass, opts)

			return nil, nil
		},
	}
}

func run(pass *analysis.Pass, opts *gofile.Options) {
	for _, file := range pass.Files {
		// Generated code such as sqlc output cannot be fixed by hand, so
		// reporting it would only produce noise.
		if ast.IsGenerated(file) {
			continue
		}

		for _, lit := range gofile.FormatLiterals(file, opts) {
			pass.Report(diagnostic(lit))
		}
	}
}

func diagnostic(lit gofile.FormattedLiteral) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos:     lit.Node.Pos(),
		End:     lit.Node.End(),
		Message: message,
		SuggestedFixes: []analysis.SuggestedFix{{
			Message: "Format SQL literal",
			TextEdits: []analysis.TextEdit{{
				Pos:     lit.Node.Pos(),
				End:     lit.Node.End(),
				NewText: []byte(lit.Formatted),
			}},
		}},
	}
}
