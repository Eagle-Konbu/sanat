package gofile

import (
	"strings"
	"testing"
)

func TestRewriteFile_WithNewline(t *testing.T) {
	src := []byte("package main\n\nimport \"database/sql\"\n\nfunc example(db *sql.DB) {\n\tdb.Exec(`select id from users where id = ?`, 1)\n\tmsg := \"hello world\"\n\t_ = msg\n}\n")

	file, fset, literals, err := FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	out, err := RewriteFile(fset, file, literals, Options{Indent: 2, Newline: true})
	if err != nil {
		t.Fatal(err)
	}

	result := string(out)

	// SQL should be reformatted with leading newline
	if !strings.Contains(result, "`\nSELECT") {
		t.Errorf("expected newline after opening backtick, got:\n%s", result)
	}

	// Non-SQL strings should remain unchanged
	if !strings.Contains(result, `"hello world"`) {
		t.Error("expected non-SQL string to remain unchanged")
	}

	// Double-quoted SQL should NOT be changed
	if strings.Contains(result, `"SELECT`) {
		t.Error("double-quoted strings should not be reformatted")
	}
}

func TestRewriteFile_WithoutNewline(t *testing.T) {
	src := []byte("package main\n\nvar q = `select id from users`\n")

	file, fset, literals, err := FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	out, err := RewriteFile(fset, file, literals, Options{Indent: 2, Newline: false})
	if err != nil {
		t.Fatal(err)
	}

	result := string(out)
	if !strings.Contains(result, "`SELECT") {
		t.Errorf("expected SELECT right after backtick (no newline), got:\n%s", result)
	}
}

func TestRewriteFile_DoubleQuotedSQLNotChanged(t *testing.T) {
	src := []byte("package main\n\nvar q = \"select id from users\"\n")

	file, fset, literals, err := FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	out, err := RewriteFile(fset, file, literals, Options{Indent: 2, Newline: true})
	if err != nil {
		t.Fatal(err)
	}

	result := string(out)
	if !strings.Contains(result, `"select id from users"`) {
		t.Errorf("double-quoted SQL should remain unchanged, got:\n%s", result)
	}
}

func TestRewriteFile_BacktickIdentifiersStripped(t *testing.T) {
	// "status" is a MySQL keyword that vitess backtick-quotes
	src := []byte("package main\n\nvar q = `select status from users`\n")

	file, fset, literals, err := FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	out, err := RewriteFile(fset, file, literals, Options{Indent: 2, Newline: true})
	if err != nil {
		t.Fatal(err)
	}

	result := string(out)
	// Output should be a raw string without MySQL backticks
	if strings.Contains(result, "``") {
		t.Errorf("should not contain MySQL backtick-quoted identifiers in raw string, got:\n%s", result)
	}
	if !strings.Contains(result, "status") {
		t.Errorf("should contain status column name, got:\n%s", result)
	}
}
