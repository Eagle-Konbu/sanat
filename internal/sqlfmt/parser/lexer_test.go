package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

type wantToken struct {
	typ     parser.TokenType
	literal string
}

func lexAll(t *testing.T, input string) []wantToken {
	t.Helper()

	l := parser.New(input)

	var got []wantToken

	for {
		tok, err := l.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}

		got = append(got, wantToken{typ: tok.Type, literal: tok.Literal})

		if tok.Type == parser.EOF {
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

func TestLexer_Keywords(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want parser.TokenType
	}{
		{"uppercase", "SELECT", parser.SELECT},
		{"lowercase", "select", parser.SELECT},
		{"mixedcase", "SeLeCt", parser.SELECT},
		{"from", "from", parser.FROM},
		{"where", "WHERE", parser.WHERE},
		{"join", "join", parser.JOIN},
		{"left", "left", parser.LEFT},
		{"group", "GROUP", parser.GROUP},
		{"recursive", "recursive", parser.RECURSIVE},
		{"index", "INDEX", parser.INDEX},
		{"regexp", "REGEXP", parser.REGEXP},
		{"rlike", "rlike", parser.RLIKE},
		{"rollup", "ROLLUP", parser.ROLLUP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.want, tt.in},
				{parser.EOF, ""},
			})
		})
	}
}

func TestLexer_Identifiers(t *testing.T) {
	t.Run("unquoted", func(t *testing.T) {
		assertTokens(t, "user_id", []wantToken{
			{parser.IDENT, "user_id"},
			{parser.EOF, ""},
		})
	})

	t.Run("with dollar sign", func(t *testing.T) {
		assertTokens(t, "col$1", []wantToken{
			{parser.IDENT, "col$1"},
			{parser.EOF, ""},
		})
	})

	t.Run("not a keyword substring", func(t *testing.T) {
		assertTokens(t, "selection", []wantToken{
			{parser.IDENT, "selection"},
			{parser.EOF, ""},
		})
	})

	t.Run("backtick quoted", func(t *testing.T) {
		assertTokens(t, "`table_name`", []wantToken{
			{parser.QuotedIdent, "table_name"},
			{parser.EOF, ""},
		})
	})

	t.Run("backtick quoted keyword-like", func(t *testing.T) {
		assertTokens(t, "`select`", []wantToken{
			{parser.QuotedIdent, "select"},
			{parser.EOF, ""},
		})
	})

	t.Run("backtick with escaped backtick", func(t *testing.T) {
		assertTokens(t, "`a``b`", []wantToken{
			{parser.QuotedIdent, "a`b"},
			{parser.EOF, ""},
		})
	})

	t.Run("unterminated backtick", func(t *testing.T) {
		l := parser.New("`abc")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated quoted identifier")
		}
	})
}

func TestLexer_NumberLiterals(t *testing.T) {
	tests := []struct {
		name string
		in   string
		typ  parser.TokenType
	}{
		{"integer", "123", parser.INT},
		{"float", "123.45", parser.FLOAT},
		{"float leading digit only", "0.5", parser.FLOAT},
		{"exponent", "1e10", parser.FLOAT},
		{"exponent with sign", "1.5e-10", parser.FLOAT},
		{"exponent uppercase", "2E+3", parser.FLOAT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.typ, tt.in},
				{parser.EOF, ""},
			})
		})
	}

	t.Run("integer followed by dot identifier", func(t *testing.T) {
		assertTokens(t, "123.col", []wantToken{
			{parser.INT, "123"},
			{parser.DOT, "."},
			{parser.IDENT, "col"},
			{parser.EOF, ""},
		})
	})

	t.Run("trailing e without digits is not an exponent", func(t *testing.T) {
		assertTokens(t, "1e", []wantToken{
			{parser.INT, "1"},
			{parser.IDENT, "e"},
			{parser.EOF, ""},
		})
	})

	t.Run("trailing e with sign but no digits is not an exponent", func(t *testing.T) {
		assertTokens(t, "1e+", []wantToken{
			{parser.INT, "1"},
			{parser.IDENT, "e"},
			{parser.PLUS, "+"},
			{parser.EOF, ""},
		})
	})
}

func TestLexer_PrefixedIntLiterals(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"hex lowercase x", "0x1a"},
		{"hex uppercase X", "0X1A"},
		{"hex mixed digits", "0xDEADbeef"},
		{"binary lowercase b", "0b101"},
		{"binary uppercase B", "0B110"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{parser.INT, tt.in},
				{parser.EOF, ""},
			})
		})
	}

	t.Run("bare zero is not treated as a prefix", func(t *testing.T) {
		assertTokens(t, "0", []wantToken{
			{parser.INT, "0"},
			{parser.EOF, ""},
		})
	})

	t.Run("0x without hex digits falls back to plain 0 then ident", func(t *testing.T) {
		assertTokens(t, "0xy", []wantToken{
			{parser.INT, "0"},
			{parser.IDENT, "xy"},
			{parser.EOF, ""},
		})
	})

	t.Run("0b without binary digits falls back to plain 0 then ident", func(t *testing.T) {
		assertTokens(t, "0b2", []wantToken{
			{parser.INT, "0"},
			{parser.IDENT, "b2"},
			{parser.EOF, ""},
		})
	})

	t.Run("0x followed by dot identifier stops at non-hex-digit", func(t *testing.T) {
		assertTokens(t, "0x1a.col", []wantToken{
			{parser.INT, "0x1a"},
			{parser.DOT, "."},
			{parser.IDENT, "col"},
			{parser.EOF, ""},
		})
	})
}

func TestLexer_StringLiterals(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "'hello'", "hello"},
		{"empty", "''", ""},
		{"doubled quote escape", "'it''s'", "it's"},
		{"backslash escape newline", `'a\nb'`, "a\nb"},
		{"backslash escape quote", `'it\'s'`, "it's"},
		{"backslash escape backslash", `'a\\b'`, `a\b`},
		{"backslash escape percent retains backslash", `'50\%'`, `50\%`},
		{"backslash escape underscore retains backslash", `'a\_b'`, `a\_b`},
		{"backslash escape nul", `'a\0b'`, "a\x00b"},
		{"backslash escape backspace", `'a\bb'`, "a\bb"},
		{"backslash escape carriage return", `'a\rb'`, "a\rb"},
		{"backslash escape tab", `'a\tb'`, "a\tb"},
		{"backslash escape ctrl-z", `'a\Zb'`, "a\x1ab"},
		{"backslash unrecognized escape drops backslash", `'a\xb'`, "axb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{parser.STRING, tt.want},
				{parser.EOF, ""},
			})
		})
	}

	t.Run("unterminated", func(t *testing.T) {
		l := parser.New("'abc")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated string literal")
		}
	})

	t.Run("unterminated after backslash", func(t *testing.T) {
		l := parser.New(`'abc\`)
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated string literal")
		}
	})
}

func TestLexer_PrefixedStringLiterals(t *testing.T) {
	tests := []struct {
		name string
		in   string
		typ  parser.TokenType
	}{
		{"hex lowercase x", "x'1A'", parser.HexStr},
		{"hex uppercase X", "X'1a'", parser.HexStr},
		{"hex empty", "x''", parser.HexStr},
		{"bit lowercase b", "b'101'", parser.BitStr},
		{"bit uppercase B", "B'110'", parser.BitStr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.typ, tt.in},
				{parser.EOF, ""},
			})
		})
	}

	t.Run("x identifier followed by separate string is not a hex literal", func(t *testing.T) {
		assertTokens(t, "x = '1A'", []wantToken{
			{parser.IDENT, "x"},
			{parser.EQ, "="},
			{parser.STRING, "1A"},
			{parser.EOF, ""},
		})
	})

	t.Run("b identifier not immediately followed by quote stays an identifier", func(t *testing.T) {
		assertTokens(t, "boat", []wantToken{
			{parser.IDENT, "boat"},
			{parser.EOF, ""},
		})
	})

	t.Run("other identifier immediately followed by quote is not a prefixed literal", func(t *testing.T) {
		assertTokens(t, "n'x'", []wantToken{
			{parser.IDENT, "n"},
			{parser.STRING, "x"},
			{parser.EOF, ""},
		})
	})

	t.Run("unterminated hex string", func(t *testing.T) {
		l := parser.New("x'1A")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated hex string literal")
		}
	})

	t.Run("unterminated bit string", func(t *testing.T) {
		l := parser.New("b'101")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated bit string literal")
		}
	})
}

func TestLexer_Operators(t *testing.T) {
	tests := []struct {
		name string
		in   string
		typ  parser.TokenType
	}{
		{"eq", "=", parser.EQ},
		{"ne angle", "<>", parser.NE},
		{"ne bang", "!=", parser.NE},
		{"lt", "<", parser.LT},
		{"gt", ">", parser.GT},
		{"le", "<=", parser.LE},
		{"ge", ">=", parser.GE},
		{"null-safe equal", "<=>", parser.NSE},
		{"plus", "+", parser.PLUS},
		{"minus", "-", parser.MINUS},
		{"star", "*", parser.STAR},
		{"slash", "/", parser.SLASH},
		{"percent", "%", parser.PERCENT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.typ, tt.in},
				{parser.EOF, ""},
			})
		})
	}

	t.Run("bang alone is illegal", func(t *testing.T) {
		l := parser.New("!")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for lone '!'")
		}
	})
}

func TestLexer_Punctuation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		typ  parser.TokenType
	}{
		{"lparen", "(", parser.LPAREN},
		{"rparen", ")", parser.RPAREN},
		{"comma", ",", parser.COMMA},
		{"dot", ".", parser.DOT},
		{"colon", ":", parser.COLON},
		{"question", "?", parser.QUESTION},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{
				{tt.typ, tt.in},
				{parser.EOF, ""},
			})
		})
	}
}

func TestLexer_WhitespaceSkipped(t *testing.T) {
	assertTokens(t, "  id\t\n  ,  name  ", []wantToken{
		{parser.IDENT, "id"},
		{parser.COMMA, ","},
		{parser.IDENT, "name"},
		{parser.EOF, ""},
	})
}

func TestLexer_Comments(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []wantToken
	}{
		{
			name: "double dash to end of line",
			in:   "id -- this is a comment\n, name",
			want: []wantToken{
				{parser.IDENT, "id"},
				{parser.COMMA, ","},
				{parser.IDENT, "name"},
				{parser.EOF, ""},
			},
		},
		{
			name: "double dash to EOF",
			in:   "id -- trailing comment",
			want: []wantToken{
				{parser.IDENT, "id"},
				{parser.EOF, ""},
			},
		},
		{
			name: "hash to end of line",
			in:   "id # comment\n, name",
			want: []wantToken{
				{parser.IDENT, "id"},
				{parser.COMMA, ","},
				{parser.IDENT, "name"},
				{parser.EOF, ""},
			},
		},
		{
			name: "block comment",
			in:   "id /* comment\nspanning lines */, name",
			want: []wantToken{
				{parser.IDENT, "id"},
				{parser.COMMA, ","},
				{parser.IDENT, "name"},
				{parser.EOF, ""},
			},
		},
		{
			name: "double dash without trailing space is not a comment",
			in:   "a--b",
			want: []wantToken{
				{parser.IDENT, "a"},
				{parser.MINUS, "-"},
				{parser.MINUS, "-"},
				{parser.IDENT, "b"},
				{parser.EOF, ""},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, tt.want)
		})
	}

	t.Run("unterminated block comment", func(t *testing.T) {
		l := parser.New("id /* unterminated")
		if _, err := l.Next(); err != nil {
			t.Fatalf("Next() error = %v, want nil for first token", err)
		}

		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for unterminated block comment")
		}
	})
}

func TestLexer_IllegalCharacter(t *testing.T) {
	l := parser.New("@")
	if _, err := l.Next(); err == nil {
		t.Fatal("expected error for illegal character")
	}
}

func TestLexer_Position(t *testing.T) {
	l := parser.New("id\n  name")

	first, err := l.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}

	if want := (parser.Position{Offset: 0, Line: 1, Column: 1}); first.Pos != want {
		t.Errorf("first.Pos = %+v, want %+v", first.Pos, want)
	}

	second, err := l.Next()
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}

	if want := (parser.Position{Offset: 5, Line: 2, Column: 3}); second.Pos != want {
		t.Errorf("second.Pos = %+v, want %+v", second.Pos, want)
	}
}

func TestLexer_Statement(t *testing.T) {
	in := "SELECT id, `name` FROM users WHERE id = ? AND active = TRUE"

	assertTokens(t, in, []wantToken{
		{parser.SELECT, "SELECT"},
		{parser.IDENT, "id"},
		{parser.COMMA, ","},
		{parser.QuotedIdent, "name"},
		{parser.FROM, "FROM"},
		{parser.IDENT, "users"},
		{parser.WHERE, "WHERE"},
		{parser.IDENT, "id"},
		{parser.EQ, "="},
		{parser.QUESTION, "?"},
		{parser.AND, "AND"},
		{parser.IDENT, "active"},
		{parser.EQ, "="},
		{parser.TRUE, "TRUE"},
		{parser.EOF, ""},
	})
}
