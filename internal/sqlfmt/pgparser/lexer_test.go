package pgparser_test

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

type wantToken struct {
	typ     pgparser.TokenType
	literal string
}

func lexAll(t *testing.T, input string) []wantToken {
	t.Helper()

	l := pgparser.New(input)

	var got []wantToken

	for {
		tok, err := l.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}

		got = append(got, wantToken{typ: tok.Type, literal: tok.Literal})

		if tok.Type == pgparser.EOF {
			return got
		}
	}
}

func assertTokens(t *testing.T, input string, want []wantToken) {
	t.Helper()

	got := lexAll(t, input)

	if len(got) != len(want) {
		t.Fatalf("token count = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("token[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func assertLexError(t *testing.T, input string) {
	t.Helper()

	l := pgparser.New(input)

	for {
		tok, err := l.Next()
		if err != nil {
			var lexErr *pgparser.LexError
			if !errors.As(err, &lexErr) {
				t.Fatalf("error type = %T, want *pgparser.LexError", err)
			}

			return
		}

		if tok.Type == pgparser.EOF {
			t.Fatalf("Next() reached EOF without error for input %q", input)
		}
	}
}

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgparser.TokenType
	}{
		{"uppercase", "SELECT", pgparser.SELECT},
		{"lowercase", "select", pgparser.SELECT},
		{"mixedcase", "SeLeCt", pgparser.SELECT},
		{"from", "from", pgparser.FROM},
		{"where", "WHERE", pgparser.WHERE},
		{"ilike", "ILIKE", pgparser.ILIKE},
		{"isnull", "ISNULL", pgparser.ISNULL},
		{"notnull", "NOTNULL", pgparser.NOTNULL},
		{"filter", "FILTER", pgparser.FILTER},
		{"array", "ARRAY", pgparser.ARRAY},
		{"lateral", "LATERAL", pgparser.LATERAL},
		{"grouping", "GROUPING", pgparser.GROUPING},
		{"rollup", "ROLLUP", pgparser.ROLLUP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.want, tt.in},
				{pgparser.EOF, ""},
			})
		})
	}
}

func TestLexer_Identifiers(t *testing.T) {
	t.Run("unquoted folds to lower case", func(t *testing.T) {
		assertTokens(t, "User_Id", []wantToken{
			{pgparser.IDENT, "user_id"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unquoted with trailing dollar sign", func(t *testing.T) {
		assertTokens(t, "col$1", []wantToken{
			{pgparser.IDENT, "col$1"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("quoted preserves case", func(t *testing.T) {
		assertTokens(t, `"Users"`, []wantToken{
			{pgparser.QuotedIdent, "Users"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("quoted with doubled quote escape", func(t *testing.T) {
		assertTokens(t, `"a""b"`, []wantToken{
			{pgparser.QuotedIdent, `a"b`},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unterminated quoted identifier", func(t *testing.T) {
		assertLexError(t, `"users`)
	})
}

func TestLexer_Strings(t *testing.T) {
	t.Run("plain string", func(t *testing.T) {
		assertTokens(t, "'abc'", []wantToken{
			{pgparser.STRING, "abc"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("doubled quote escape", func(t *testing.T) {
		assertTokens(t, "'it''s'", []wantToken{
			{pgparser.STRING, "it's"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("plain string has no backslash escapes", func(t *testing.T) {
		assertTokens(t, `'a\nb'`, []wantToken{
			{pgparser.STRING, `a\nb`},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unterminated string", func(t *testing.T) {
		assertLexError(t, "'abc")
	})

	t.Run("escape string simple escapes", func(t *testing.T) {
		assertTokens(t, `E'a\nb\tc\r\\d\'e\bf\fg'`, []wantToken{
			{pgparser.STRING, "a\nb\tc\r\\d'e\bf\fg"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string trailing backslash", func(t *testing.T) {
		assertLexError(t, `E'abc\`)
	})

	t.Run("escape string is case-insensitive prefix", func(t *testing.T) {
		assertTokens(t, `e'\n'`, []wantToken{
			{pgparser.STRING, "\n"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string hex escape", func(t *testing.T) {
		assertTokens(t, `E'\x41\x9'`, []wantToken{
			{pgparser.STRING, "A\t"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string unicode escape", func(t *testing.T) {
		assertTokens(t, `E'A\U00000042'`, []wantToken{
			{pgparser.STRING, "AB"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string octal escape, three digits", func(t *testing.T) {
		assertTokens(t, `E'\101'`, []wantToken{
			{pgparser.STRING, "A"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string octal escape, fewer than three digits", func(t *testing.T) {
		assertTokens(t, `E'\7!'`, []wantToken{
			{pgparser.STRING, "\a!"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string unrecognized char keeps char, drops backslash", func(t *testing.T) {
		assertTokens(t, `E'\q'`, []wantToken{
			{pgparser.STRING, "q"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string invalid hex escape", func(t *testing.T) {
		assertLexError(t, `E'\x'`)
	})

	t.Run("escape string invalid unicode escape", func(t *testing.T) {
		assertLexError(t, `E'\u12'`)
	})

	t.Run("escape string unterminated", func(t *testing.T) {
		assertLexError(t, `E'abc`)
	})
}

func TestLexer_DollarQuotedStrings(t *testing.T) {
	t.Run("empty tag", func(t *testing.T) {
		assertTokens(t, "$$hello$$", []wantToken{
			{pgparser.STRING, "hello"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("named tag", func(t *testing.T) {
		assertTokens(t, "$tag$hello world$tag$", []wantToken{
			{pgparser.STRING, "hello world"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("no internal escaping", func(t *testing.T) {
		assertTokens(t, `$$it's a \n test$$`, []wantToken{
			{pgparser.STRING, `it's a \n test`},
			{pgparser.EOF, ""},
		})
	})

	t.Run("nested via distinct tags", func(t *testing.T) {
		assertTokens(t, "$outer$a $$inner$$ b$outer$", []wantToken{
			{pgparser.STRING, "a $$inner$$ b"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unterminated tag", func(t *testing.T) {
		assertLexError(t, "$tag")
	})

	t.Run("unterminated body", func(t *testing.T) {
		assertLexError(t, "$$hello")
	})
}

func TestLexer_Placeholders(t *testing.T) {
	assertTokens(t, "$1, $12", []wantToken{
		{pgparser.Placeholder, "1"},
		{pgparser.COMMA, ","},
		{pgparser.Placeholder, "12"},
		{pgparser.EOF, ""},
	})
}

func TestLexer_Numbers(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want pgparser.TokenType
	}{
		{"int", "123", pgparser.INT},
		{"float", "123.45", pgparser.FLOAT},
		{"leading dot", ".5", pgparser.FLOAT},
		{"exponent", "1e10", pgparser.FLOAT},
		{"exponent with sign", "1e-10", pgparser.FLOAT},
		{"float with exponent", "1.5e+10", pgparser.FLOAT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.want, tt.in},
				{pgparser.EOF, ""},
			})
		})
	}

	t.Run("no hex prefix support", func(t *testing.T) {
		assertTokens(t, "0x1", []wantToken{
			{pgparser.INT, "0"},
			{pgparser.IDENT, "x1"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("trailing e without digits is not an exponent", func(t *testing.T) {
		assertTokens(t, "1e", []wantToken{
			{pgparser.INT, "1"},
			{pgparser.IDENT, "e"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("number followed by dot-qualified identifier", func(t *testing.T) {
		assertTokens(t, "1.col", []wantToken{
			{pgparser.INT, "1"},
			{pgparser.DOT, "."},
			{pgparser.IDENT, "col"},
			{pgparser.EOF, ""},
		})
	})
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		in   string
		want pgparser.TokenType
	}{
		{"=", pgparser.EQ},
		{"<>", pgparser.NE},
		{"!=", pgparser.NE},
		{"<", pgparser.LT},
		{">", pgparser.GT},
		{"<=", pgparser.LE},
		{">=", pgparser.GE},
		{"+", pgparser.PLUS},
		{"-", pgparser.MINUS},
		{"*", pgparser.STAR},
		{"/", pgparser.SLASH},
		{"%", pgparser.PERCENT},
		{"::", pgparser.DoubleColon},
		{"->", pgparser.Arrow},
		{"->>", pgparser.ArrowText},
		{"#>", pgparser.HashArrow},
		{"#>>", pgparser.HashArrowText},
		{"@>", pgparser.Contains},
		{"<@", pgparser.ContainedBy},
		{"?", pgparser.JSONExists},
		{"?|", pgparser.JSONExistsAny},
		{"?&", pgparser.JSONExistsAll},
		{"(", pgparser.LPAREN},
		{")", pgparser.RPAREN},
		{"[", pgparser.LBRACKET},
		{"]", pgparser.RBRACKET},
		{",", pgparser.COMMA},
		{".", pgparser.DOT},
		{":", pgparser.COLON},
		{";", pgparser.SEMICOLON},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.want, tt.in},
				{pgparser.EOF, ""},
			})
		})
	}

	t.Run("unexpected bang", func(t *testing.T) {
		assertLexError(t, "!a")
	})

	t.Run("unexpected hash", func(t *testing.T) {
		assertLexError(t, "#a")
	})

	t.Run("unexpected at", func(t *testing.T) {
		assertLexError(t, "@a")
	})

	t.Run("unexpected character", func(t *testing.T) {
		assertLexError(t, "^")
	})
}

func TestLexer_Comments(t *testing.T) {
	t.Run("line comment", func(t *testing.T) {
		assertTokens(t, "a --comment\nb", []wantToken{
			{pgparser.IDENT, "a"},
			{pgparser.IDENT, "b"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("block comment", func(t *testing.T) {
		assertTokens(t, "a /* comment */ b", []wantToken{
			{pgparser.IDENT, "a"},
			{pgparser.IDENT, "b"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("nested block comment", func(t *testing.T) {
		assertTokens(t, "a /* outer /* inner */ still outer */ b", []wantToken{
			{pgparser.IDENT, "a"},
			{pgparser.IDENT, "b"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unterminated block comment", func(t *testing.T) {
		assertLexError(t, "/* comment")
	})
}

func TestLexError_Error(t *testing.T) {
	err := &pgparser.LexError{Pos: pgparser.Position{Line: 2, Column: 5}, Msg: "boom"}
	want := "2:5: boom"

	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestTokenType_String(t *testing.T) {
	if got := pgparser.SELECT.String(); got != "SELECT" {
		t.Errorf("String() = %q, want %q", got, "SELECT")
	}

	if got := pgparser.TokenType(-1).String(); got != "UNKNOWN" {
		t.Errorf("String() = %q, want %q", got, "UNKNOWN")
	}
}

func TestTokenType_IsKeyword(t *testing.T) {
	if !pgparser.SELECT.IsKeyword() {
		t.Error("SELECT.IsKeyword() = false, want true")
	}

	if pgparser.IDENT.IsKeyword() {
		t.Error("IDENT.IsKeyword() = true, want false")
	}
}
