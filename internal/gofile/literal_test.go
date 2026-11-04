package gofile_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/gofile"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

func rawLiteralSource(content string) []byte {
	return []byte("package main\n\nvar q = `" + content + "`\n")
}

func TestFormatLiteral(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		opts        gofile.Options
		wantContent string
		wantChanged bool
	}{
		{
			name:        "unformatted SQL with newline",
			content:     "select id from users",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "\nSELECT\n  id\nFROM\n  users\n",
			wantChanged: true,
		},
		{
			name:        "unformatted SQL without newline",
			content:     "select id from users",
			opts:        gofile.Options{Indent: 2, Newline: false},
			wantContent: "SELECT\n  id\nFROM\n  users",
			wantChanged: true,
		},
		{
			name:        "already formatted SQL",
			content:     "\nSELECT\n  id\nFROM\n  users\n",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "\nSELECT\n  id\nFROM\n  users\n",
			wantChanged: false,
		},
		{
			name:        "formatted SQL missing the configured newlines",
			content:     "SELECT\n  id\nFROM\n  users",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "\nSELECT\n  id\nFROM\n  users\n",
			wantChanged: true,
		},
		{
			name:        "non-SQL string",
			content:     "hello world",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "hello world",
			wantChanged: false,
		},
		{
			name:        "format verb in SQL-like string",
			content:     "select id from %s",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "select id from %s",
			wantChanged: false,
		},
		{
			name:        "invalid SQL",
			content:     "select id from users where (",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "select id from users where (",
			wantChanged: false,
		},
		{
			name:        "already formatted SQL with CRLF line endings",
			content:     "\r\nSELECT\r\n  id\r\nFROM\r\n  users\r\n",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "\nSELECT\n  id\nFROM\n  users\n",
			wantChanged: false,
		},
		{
			name:        "unformatted SQL with CRLF line endings",
			content:     "select id\r\nfrom users",
			opts:        gofile.Options{Indent: 2, Newline: true},
			wantContent: "\nSELECT\n  id\nFROM\n  users\n",
			wantChanged: true,
		},
		{
			name:    "lower keyword case",
			content: "select id from users where status = 1 and active = true",
			opts: gofile.Options{
				Indent:      2,
				Newline:     true,
				KeywordCase: sqlfmt.KeywordCaseLower,
			},
			wantContent: "\nSELECT\n  id\nFROM\n  users\nWHERE\n  status = 1\n  and active = true\n",
			wantChanged: true,
		},
		{
			name:    "leading comma style",
			content: "select id, name from users",
			opts: gofile.Options{
				Indent:     2,
				Newline:    true,
				CommaStyle: sqlfmt.CommaStyleLeading,
			},
			wantContent: "\nSELECT\n  id\n, name\nFROM\n  users\n",
			wantChanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, literals, err := gofile.FindSQLLiterals(rawLiteralSource(tt.content), "test.go")
			if err != nil {
				t.Fatal(err)
			}

			if len(literals) != 1 {
				t.Fatalf("expected 1 literal, got %d", len(literals))
			}

			originalValue := literals[0].Node.Value

			got, changed := gofile.FormatLiteral(literals[0], &tt.opts)

			if want := "`" + tt.wantContent + "`"; got != want {
				t.Errorf("formatted mismatch\ngot:  %q\nwant: %q", got, want)
			}

			if changed != tt.wantChanged {
				t.Errorf("changed = %v, want %v", changed, tt.wantChanged)
			}

			if literals[0].Node.Value != originalValue {
				t.Errorf("node was modified: %q", literals[0].Node.Value)
			}
		})
	}
}

func TestFormatLiterals_ReturnsOnlyChangedLiteralsWithoutModifyingAST(t *testing.T) {
	src := []byte("package main\n" +
		"\n" +
		"var formatted = `\nSELECT\n  id\nFROM\n  users\n`\n" +
		"\n" +
		"var unformatted = `select name from users`\n" +
		"\n" +
		"var notSQL = `hello world`\n" +
		"\n" +
		"var doubleQuoted = \"select id from users\"\n")

	file, fset, _, err := gofile.FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	valuesBefore := basicLitValues(file)

	results := gofile.FormatLiterals(file, &gofile.Options{Indent: 2, Newline: true})

	if len(results) != 1 {
		t.Fatalf("expected 1 changed literal, got %d: %+v", len(results), results)
	}

	if results[0].Node.Value != "`select name from users`" {
		t.Errorf("unexpected node reported: %q", results[0].Node.Value)
	}

	if want := "`\nSELECT\n  name\nFROM\n  users\n`"; results[0].Formatted != want {
		t.Errorf("formatted mismatch\ngot:  %q\nwant: %q", results[0].Formatted, want)
	}

	if line := fset.Position(results[0].Node.Pos()).Line; line != 10 {
		t.Errorf("reported literal at line %d, want 10", line)
	}

	valuesAfter := basicLitValues(file)
	for i := range valuesBefore {
		if valuesBefore[i] != valuesAfter[i] {
			t.Errorf("AST was modified: %q became %q", valuesBefore[i], valuesAfter[i])
		}
	}
}

func TestCollectSQLLiterals_ReturnsNodesOfGivenAST(t *testing.T) {
	file, _, _, err := gofile.FindSQLLiterals(rawLiteralSource("select id from users"), "test.go")
	if err != nil {
		t.Fatal(err)
	}

	var rawLit *ast.BasicLit

	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			rawLit = lit
		}

		return true
	})

	literals := gofile.CollectSQLLiterals(file)

	if len(literals) != 1 {
		t.Fatalf("expected 1 literal, got %d", len(literals))
	}

	if literals[0].Node != rawLit {
		t.Error("expected the collected node to be the node in the given AST")
	}

	if literals[0].Original != "select id from users" {
		t.Errorf("unexpected original: %q", literals[0].Original)
	}
}

func TestCollectSQLLiterals_DiscardsCarriageReturns(t *testing.T) {
	_, _, literals, err := gofile.FindSQLLiterals(rawLiteralSource("select id\r\nfrom users"), "test.go")
	if err != nil {
		t.Fatal(err)
	}

	if len(literals) != 1 {
		t.Fatalf("expected 1 literal, got %d", len(literals))
	}

	if literals[0].Original != "select id\nfrom users" {
		t.Errorf("unexpected original: %q", literals[0].Original)
	}
}

func basicLitValues(file *ast.File) []string {
	var values []string

	ast.Inspect(file, func(n ast.Node) bool {
		if lit, ok := n.(*ast.BasicLit); ok {
			values = append(values, lit.Value)
		}

		return true
	})

	return values
}
