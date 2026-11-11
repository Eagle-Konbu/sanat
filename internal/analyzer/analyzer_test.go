package analyzer_test

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Eagle-Konbu/sanat/internal/analyzer"
	"github.com/Eagle-Konbu/sanat/internal/gofile"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

func defaultOptions() *gofile.Options {
	return &gofile.Options{
		Indent:      2,
		Newline:     true,
		KeywordCase: sqlfmt.KeywordCaseUpper,
		CommaStyle:  sqlfmt.CommaStyleTrailing,
		SQLMode:     sqlfmt.SQLModeDefault,
	}
}

func TestAnalyzer_ReportsOnlyLiteralsThatWouldChange(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer.New(defaultOptions()), "basic")
}

func TestAnalyzer_RespectsFormatterOptions(t *testing.T) {
	opts := defaultOptions()
	opts.Newline = false
	opts.CommaStyle = sqlfmt.CommaStyleLeading

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer.New(opts), "options")
}

func TestAnalyzer_FormatsPostgreSQLDialect(t *testing.T) {
	opts := defaultOptions()
	opts.Newline = false
	opts.Dialect = sqlfmt.DialectPostgreSQL

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer.New(opts), "postgresql")
}

func TestAnalyzer_SkipsGeneratedFiles(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzer.New(defaultOptions()), "generated")
}

func TestAnalyzer_DiagnosticCoversWholeLiteral(t *testing.T) {
	results := analysistest.Run(t, analysistest.TestData(), analyzer.New(defaultOptions()), "position")

	if len(results) != 1 || len(results[0].Diagnostics) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %+v", results)
	}

	fset := results[0].Pass.Fset
	diag := results[0].Diagnostics[0]

	start := fset.Position(diag.Pos)
	end := fset.Position(diag.End)

	const literal = "`select id from users`"

	if start.Line != 4 || start.Column != 9 {
		t.Errorf("start = %d:%d, want 4:9", start.Line, start.Column)
	}

	if end.Line != 4 || end.Column != start.Column+len(literal) {
		t.Errorf("end = %d:%d, want 4:%d", end.Line, end.Column, start.Column+len(literal))
	}
}
