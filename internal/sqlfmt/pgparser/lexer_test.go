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

	t.Run("zero-length quoted identifier", func(t *testing.T) {
		assertLexError(t, `""`)
	})

	t.Run("unquoted folds only ASCII letters", func(t *testing.T) {
		assertTokens(t, "ÄRGER", []wantToken{
			{pgparser.IDENT, "Ärger"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("non-ASCII letters do not fold into keywords", func(t *testing.T) {
		assertTokens(t, "ſelect ınner", []wantToken{
			{pgparser.IDENT, "ſelect"},
			{pgparser.IDENT, "ınner"},
			{pgparser.EOF, ""},
		})
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

	t.Run("escape string octal and hex escapes are raw bytes", func(t *testing.T) {
		assertTokens(t, `E'\303\251\xc3\xa9'`, []wantToken{
			{pgparser.STRING, "éé"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string unicode surrogate pair", func(t *testing.T) {
		assertTokens(t, `E'\uD83D\uDE00\uD83D\U0000DE00'`, []wantToken{
			{pgparser.STRING, "😀😀"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("escape string max unicode code point", func(t *testing.T) {
		assertTokens(t, `E'\U0010FFFF'`, []wantToken{
			{pgparser.STRING, "\U0010FFFF"},
			{pgparser.EOF, ""},
		})
	})

	invalidUnicodeEscapes := []struct {
		name string
		in   string
	}{
		{"too few digits", `E'\u12'`},
		{"beyond max code point", `E'\U00110000'`},
		{"beyond int32", `E'\UFFFFFFFF'`},
		{"lone low surrogate", `E'\uDE00'`},
		{"high surrogate at end", `E'\uD83D'`},
		{"high surrogate followed by text", `E'\uD83Dx'`},
		{"high surrogate followed by non-surrogate", `E'\uD83D\u0041'`},
	}

	for _, tt := range invalidUnicodeEscapes {
		t.Run("escape string invalid unicode escape, "+tt.name, func(t *testing.T) {
			assertLexError(t, tt.in)
		})
	}

	t.Run("escape string unterminated", func(t *testing.T) {
		assertLexError(t, `E'abc`)
	})
}

func TestLexer_StringContinuation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []wantToken
	}{
		{
			name: "newline-separated strings are joined",
			in:   "'foo'\n  'bar'",
			want: []wantToken{{pgparser.STRING, "foobar"}, {pgparser.EOF, ""}},
		},
		{
			name: "line comment before the newline still joins",
			in:   "'foo' -- note\n'bar'",
			want: []wantToken{{pgparser.STRING, "foobar"}, {pgparser.EOF, ""}},
		},
		{
			name: "escape string continuation keeps escapes",
			in:   "E'a\\n'\n'\\t'",
			want: []wantToken{{pgparser.STRING, "a\n\t"}, {pgparser.EOF, ""}},
		},
		{
			name: "same-line strings stay separate",
			in:   "'foo' 'bar'",
			want: []wantToken{{pgparser.STRING, "foo"}, {pgparser.STRING, "bar"}, {pgparser.EOF, ""}},
		},
		{
			name: "block comment separates strings",
			in:   "'foo' /* c */\n'bar'",
			want: []wantToken{{pgparser.STRING, "foo"}, {pgparser.STRING, "bar"}, {pgparser.EOF, ""}},
		},
		{
			name: "newline followed by a non-string token",
			in:   "'foo'\nbar",
			want: []wantToken{{pgparser.STRING, "foo"}, {pgparser.IDENT, "bar"}, {pgparser.EOF, ""}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, tt.want)
		})
	}
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

	t.Run("multibyte tag", func(t *testing.T) {
		assertTokens(t, "$é$x$é$ y", []wantToken{
			{pgparser.STRING, "x"},
			{pgparser.IDENT, "y"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("unterminated tag", func(t *testing.T) {
		assertLexError(t, "$tag")
	})

	t.Run("tag with hyphen is rejected", func(t *testing.T) {
		assertLexError(t, "$a-b$x$a-b$")
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
		{"trailing dot", "1.", pgparser.FLOAT},
		{"trailing dot with exponent", "1.e5", pgparser.FLOAT},
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

	t.Run("trailing-dot number followed by identifier", func(t *testing.T) {
		assertTokens(t, "1.col", []wantToken{
			{pgparser.FLOAT, "1."},
			{pgparser.IDENT, "col"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("double dot is not part of the number", func(t *testing.T) {
		assertTokens(t, "1..2", []wantToken{
			{pgparser.INT, "1"},
			{pgparser.DOT, "."},
			{pgparser.FLOAT, ".2"},
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
		{"||", pgparser.CONCAT},
		{"^", pgparser.CARET},
		{"::", pgparser.DoubleColon},
		{"->", pgparser.Arrow},
		{"->>", pgparser.ArrowText},
		{"#>", pgparser.HashArrow},
		{"#>>", pgparser.HashArrowText},
		{"@>", pgparser.Contains},
		{"<@", pgparser.ContainedBy},
		{"@?", pgparser.JSONPathExists},
		{"@@", pgparser.JSONPathMatch},
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

	t.Run("unexpected pipe", func(t *testing.T) {
		assertLexError(t, "|a")
	})

	t.Run("unexpected character", func(t *testing.T) {
		assertLexError(t, "~")
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

	t.Run("form feed is whitespace", func(t *testing.T) {
		assertTokens(t, "a\fb", []wantToken{
			{pgparser.IDENT, "a"},
			{pgparser.IDENT, "b"},
			{pgparser.EOF, ""},
		})
	})
}

func TestLexer_Positions(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []pgparser.Position
	}{
		{
			name: "multibyte input counts byte offset and rune column",
			in:   "'é' x",
			want: []pgparser.Position{
				{Offset: 0, Line: 1, Column: 1},
				{Offset: 5, Line: 1, Column: 5},
				{Offset: 6, Line: 1, Column: 6},
			},
		},
		{
			name: "newline advances line and resets column",
			in:   "a\n  b",
			want: []pgparser.Position{
				{Offset: 0, Line: 1, Column: 1},
				{Offset: 4, Line: 2, Column: 3},
				{Offset: 5, Line: 2, Column: 4},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := pgparser.New(tt.in)

			for i, want := range tt.want {
				tok, err := l.Next()
				if err != nil {
					t.Fatalf("Next() error = %v", err)
				}

				if tok.Pos != want {
					t.Errorf("token[%d] %v Pos = %+v, want %+v", i, tok.Type, tok.Pos, want)
				}
			}
		})
	}
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
