package gofile

import "testing"

func TestFindSQLLiterals_OnlyRawStrings(t *testing.T) {
	src := []byte(`package main

import "database/sql"

func example(db *sql.DB) {
	db.Exec(` + "`select id from users where id = ?`" + `, 1)
	db.Exec("select name from users", 1)
	name := "hello world"
	_ = name
	db.Query(` + "`select * from orders`" + `)
}
`)

	_, _, literals, err := FindSQLLiterals(src, "test.go")
	if err != nil {
		t.Fatal(err)
	}

	// Should only find backtick raw string literals, not double-quoted
	found := map[string]bool{}
	for _, lit := range literals {
		found[lit.Original] = true
	}

	wantFound := []string{
		"select id from users where id = ?",
		"select * from orders",
	}
	for _, w := range wantFound {
		if !found[w] {
			t.Errorf("expected to find literal %q", w)
		}
	}

	wantNotFound := []string{
		"select name from users",
		"hello world",
		"database/sql",
	}
	for _, w := range wantNotFound {
		if found[w] {
			t.Errorf("should NOT find double-quoted literal %q", w)
		}
	}
}
