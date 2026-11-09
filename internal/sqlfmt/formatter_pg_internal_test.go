package sqlfmt

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// TestFormatParsedPGStatement_UnhandledNode covers the recover that turns a
// node the renderer doesn't handle into an ok=false failure.
func TestFormatParsedPGStatement_UnhandledNode(t *testing.T) {
	if got, ok := formatParsedPGStatement(Options{Indent: 2}, nil); ok || got != "" {
		t.Errorf("formatParsedPGStatement(nil) = %q, %v, want \"\", false", got, ok)
	}
}

// TestPGFormatter_NodesWithoutRaw covers nodes built without source text,
// which render their normalized SQL text instead of Raw.
func TestPGFormatter_NodesWithoutRaw(t *testing.T) {
	f := &pgFormatter{renderer: newRenderer(Options{Indent: 2})}

	tests := []struct {
		name string
		expr pgast.Expr
		want string
	}{
		{"literal", &pgast.Literal{Val: "'x'"}, "'x'"},
		{
			"qualified column",
			&pgast.ColName{Qualifier: pgast.TableName{Name: pgast.TableIdent{Name: "t"}}, Name: pgast.ColIdent{Name: "c"}},
			"t.c",
		},
		{
			"typed literal",
			&pgast.TypedLiteral{Type: &pgast.TypeName{Names: []pgast.ColIdent{{Name: "date"}}}, Value: "'2020-01-01'"},
			"date '2020-01-01'",
		},
		{"extract", &pgast.ExtractExpr{Field: "year", Source: &pgast.ColName{Name: pgast.ColIdent{Name: "d"}}}, "EXTRACT(year FROM d)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.expr(tt.expr, 0); got != tt.want {
				t.Errorf("expr() = %q, want %q", got, tt.want)
			}
		})
	}
}
