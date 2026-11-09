package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestLexer_tokenSpan(t *testing.T) {
	input := "SELECT UserId, \"Us\"\"er\", E'a\\nb', 'a'\n  'b', $body$it's$body$, $1, x::int, 1.5e3 =-1"
	want := []string{
		"SELECT", "UserId", ",", `"Us""er"`, ",", `E'a\nb'`, ",", "'a'\n  'b'", ",", "$body$it's$body$", ",",
		"$1", ",", "x", "::", "int", ",", "1.5e3", "=", "-", "1", "",
	}

	l := pgparser.New(input)

	for i, wantRaw := range want {
		tok, err := l.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}

		if got := input[tok.Pos.Offset:tok.End.Offset]; got != wantRaw {
			t.Errorf("token[%d] source = %q, want %q", i, got, wantRaw)
		}
	}
}

func TestLexer_tokenEndPosition(t *testing.T) {
	l := pgparser.New("x\n  'a'\n'b' y")

	if _, err := l.Next(); err != nil {
		t.Fatal(err)
	}

	tok, err := l.Next()
	if err != nil {
		t.Fatal(err)
	}

	want := pgparser.Position{Offset: 11, Line: 3, Column: 4}
	if tok.End != want {
		t.Errorf("End = %+v, want %+v", tok.End, want)
	}
}

func TestParseExpr_literalRaw(t *testing.T) {
	tests := []struct {
		name, in, wantVal, wantRaw string
	}{
		{"plain", `'abc'`, `'abc'`, `'abc'`},
		{"doubled quote", `'it''s'`, `'it''s'`, `'it''s'`},
		{"escape string", `E'a\nb'`, "'a\nb'", `E'a\nb'`},
		{"lower-case escape prefix", `e'it\'s'`, `'it''s'`, `e'it\'s'`},
		{"dollar-quoted with tag", "$body$it's$body$", `'it''s'`, "$body$it's$body$"},
		{"empty-tag dollar-quoted", "$$x$$", `'x'`, "$$x$$"},
		{"joined across a newline", "'a'\n'b'", `'ab'`, "'a'\n'b'"},
		{"joined across indented newlines", "'a' \n\t 'b'\n'c'", `'abc'`, "'a' \n\t 'b'\n'c'"},
		{"national string", `n'x'`, `N'x'`, `n'x'`},
		{"bit string", `b'01'`, `B'01'`, `b'01'`},
		{"hex bit string", `x'1F'`, `X'1F'`, `x'1F'`},
		{"number", `1.5E3`, `1.5E3`, `1.5E3`},
		{"placeholder", `$1`, `$1`, `$1`},
		{"keyword literal", `null`, `NULL`, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lit, ok := parseExpr(t, tt.in).(*pgast.Literal)
			if !ok {
				t.Fatalf("ParseExpr(%q) is not a *pgast.Literal", tt.in)
			}

			if lit.Val != tt.wantVal || lit.Raw != tt.wantRaw {
				t.Errorf("ParseExpr(%q) = {Val: %q, Raw: %q}, want {Val: %q, Raw: %q}", tt.in, lit.Val, lit.Raw, tt.wantVal, tt.wantRaw)
			}
		})
	}
}

func TestParseExpr_literalRawInOtherNodes(t *testing.T) {
	t.Run("typed literal", func(t *testing.T) {
		lit, ok := parseExpr(t, "DATE $d$2020-01-01$d$").(*pgast.TypedLiteral)
		if !ok || lit.Value != "'2020-01-01'" || lit.ValueRaw != "$d$2020-01-01$d$" {
			t.Errorf("got %#v, want Value '2020-01-01' and ValueRaw $d$2020-01-01$d$", lit)
		}
	})

	t.Run("extract field identifier", func(t *testing.T) {
		e, ok := parseExpr(t, "EXTRACT(YEAR FROM d)").(*pgast.ExtractExpr)
		if !ok || e.Field != "year" || e.FieldRaw != "YEAR" {
			t.Errorf("got %#v, want Field year and FieldRaw YEAR", e)
		}
	})

	t.Run("extract field string", func(t *testing.T) {
		e, ok := parseExpr(t, "EXTRACT(E'epoch' FROM d)").(*pgast.ExtractExpr)
		if !ok || e.Field != "'epoch'" || e.FieldRaw != "E'epoch'" {
			t.Errorf("got %#v, want Field 'epoch' and FieldRaw E'epoch'", e)
		}
	})

	t.Run("String keeps the normalized text", func(t *testing.T) {
		if got := parseExpr(t, "E'a''b' || $$c$$").String(); got != "'a''b' || 'c'" {
			t.Errorf("String() = %q, want %q", got, "'a''b' || 'c'")
		}
	})
}

func TestParse_identifierRaw(t *testing.T) {
	sel, err := pgparser.ParseSelect(
		`SELECT UserId, "UserId", T.Col AS Total, Pg_Catalog.Now(), x::Public."MyType", SUBSTRING(s, 1) ` +
			`FROM Schema.Tbl AS T JOIN "Other" O USING (Id)`)
	if err != nil {
		t.Fatal(err)
	}

	item := func(i int) pgast.Expr { return mustBe[*pgast.AliasedExpr](t, sel.SelectExprs[i]).Expr }
	userID := mustBe[*pgast.ColName](t, item(0))
	quoted := mustBe[*pgast.ColName](t, item(1))
	aliased := mustBe[*pgast.AliasedExpr](t, sel.SelectExprs[2])
	qualifiedCol := mustBe[*pgast.ColName](t, aliased.Expr)
	call := mustBe[*pgast.FuncExpr](t, item(3))
	castType := mustBe[*pgast.CastExpr](t, item(4)).Type
	plainForm := mustBe[*pgast.FuncExpr](t, item(5))
	join := mustBe[*pgast.JoinTableExpr](t, sel.From[0])
	left := mustBe[*pgast.AliasedTableExpr](t, join.Left)
	leftTable := mustBe[pgast.TableName](t, left.Expr)
	right := mustBe[*pgast.AliasedTableExpr](t, join.Right)
	rightTable := mustBe[pgast.TableName](t, right.Expr)

	tests := []struct {
		name          string
		got           pgast.ColIdent
		wantName, raw string
	}{
		{"unquoted column", userID.Name, "userid", "UserId"},
		{"quoted column", quoted.Name, `"UserId"`, `"UserId"`},
		{"column qualifier", pgast.ColIdent(qualifiedCol.Qualifier.Name), "t", "T"},
		{"qualified column", qualifiedCol.Name, "col", "Col"},
		{"column alias", aliased.As, "total", "Total"},
		{"function schema", pgast.ColIdent(call.Qualifier), "pg_catalog", "Pg_Catalog"},
		{"function name", call.Name, "now", "Now"},
		{"type schema", castType.Names[0], "public", "Public"},
		{"quoted type name", castType.Names[1], `"MyType"`, `"MyType"`},
		{"plain-form special function", plainForm.Name, "substring", "SUBSTRING"},
		{"table schema", pgast.ColIdent(leftTable.Qualifier), "schema", "Schema"},
		{"table name", pgast.ColIdent(leftTable.Name), "tbl", "Tbl"},
		{"table alias", pgast.ColIdent(left.As), "t", "T"},
		{"quoted table", pgast.ColIdent(rightTable.Name), `"Other"`, `"Other"`},
		{"bare table alias", pgast.ColIdent(right.As), "o", "O"},
		{"USING column", join.Condition.Using[0], "id", "Id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got.Name != tt.wantName || tt.got.Raw != tt.raw {
				t.Errorf("got %#v, want {Name: %q, Raw: %q}", tt.got, tt.wantName, tt.raw)
			}
		})
	}

	want := `SELECT userid, "UserId", t.col AS total, pg_catalog.now(), x::public."MyType", substring(s, 1) ` +
		`FROM schema.tbl t JOIN "Other" o USING (id)`
	if got := sel.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestContainsComment(t *testing.T) {
	tests := []struct {
		name, in string
		want     bool
	}{
		{"no comment", "SELECT 1", false},
		{"line comment", "SELECT 1 -- note", true},
		{"block comment", "SELECT /* note */ 1", true},
		{"nested block comment", "SELECT /* a /* b */ c */ 1", true},
		{"comment cutting an operator", "SELECT 1 +/* note */ 2", true},
		{"comment between joined strings", "SELECT 'a' -- note\n'b'", true},
		{"dashes in a string", "SELECT '-- not a comment'", false},
		{"slash-star in a dollar-quoted string", "SELECT $$/* not a comment */$$", false},
		{"slash-star in a tagged dollar-quoted string", "SELECT $x$a /* b$x$", false},
		{"dashes in a quoted identifier", `SELECT "a--b"`, false},
		{"dashes in an escape string", `SELECT E'\'--'`, false},
		{"comment before a lex error", "SELECT 1 -- note\n'unterminated", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pgparser.ContainsComment(tt.in); got != tt.want {
				t.Errorf("ContainsComment(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
