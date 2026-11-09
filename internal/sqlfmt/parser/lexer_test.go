package parser_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

type wantToken struct {
	typ     parser.TokenType
	literal string
}

func lexAllWithMode(t *testing.T, input string, mode parser.SQLMode) []wantToken {
	t.Helper()

	l := parser.NewWithMode(input, mode)

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

	assertTokensWithMode(t, input, parser.ModeDefault, want)
}

func assertTokensWithMode(t *testing.T, input string, mode parser.SQLMode, want []wantToken) {
	t.Helper()

	got := lexAllWithMode(t, input, mode)

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

func TestLexer_AtVariable(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		assertTokens(t, "@rank", []wantToken{
			{parser.AtVariable, "rank"},
			{parser.EOF, ""},
		})
	})

	t.Run("with underscore and digits", func(t *testing.T) {
		assertTokens(t, "@my_var1", []wantToken{
			{parser.AtVariable, "my_var1"},
			{parser.EOF, ""},
		})
	})

	t.Run("missing identifier", func(t *testing.T) {
		l := parser.New("@ 1")

		_, err := l.Next()

		if _, ok := errors.AsType[*parser.LexError](err); !ok {
			t.Fatalf("Next() error = %v, want *parser.LexError", err)
		}
	})

	t.Run("missing identifier at eof", func(t *testing.T) {
		l := parser.New("@")

		_, err := l.Next()

		if _, ok := errors.AsType[*parser.LexError](err); !ok {
			t.Fatalf("Next() error = %v, want *parser.LexError", err)
		}
	})

	t.Run("at-at system variable syntax is rejected", func(t *testing.T) {
		l := parser.New("@@global.sql_mode")

		_, err := l.Next()

		if _, ok := errors.AsType[*parser.LexError](err); !ok {
			t.Fatalf("Next() error = %v, want *parser.LexError", err)
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
		{"float with no leading digit", ".5", parser.FLOAT},
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
		{"hex mixed digits", "0xDEADbeef"},
		{"binary lowercase b", "0b101"},
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

	t.Run("uppercase 0X prefix is not recognized, unlike lowercase 0x", func(t *testing.T) {
		assertTokens(t, "0X1A", []wantToken{
			{parser.INT, "0"},
			{parser.IDENT, "X1A"},
			{parser.EOF, ""},
		})
	})

	t.Run("uppercase 0B prefix is not recognized, unlike lowercase 0b", func(t *testing.T) {
		assertTokens(t, "0B110", []wantToken{
			{parser.INT, "0"},
			{parser.IDENT, "B110"},
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

// TestLexer_StringLiterals_NoBackslashEscapes covers ModeNoBackslashEscapes,
// under which backslash has no special meaning in a string literal: it reads
// like any other character instead of introducing an escape sequence. Quote
// doubling still works, since that isn't backslash-based.
func TestLexer_StringLiterals_NoBackslashEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "'hello'", "hello"},
		{"doubled quote escape", "'it''s'", "it's"},
		{"backslash is a literal character, not an escape", `'a\nb'`, `a\nb`},
		{"backslash before quote does not escape it", `'a\'`, "a\\"},
		{"literal newline is preserved as-is", "'a\nb'", "a\nb"},
		{"trailing backslash before closing quote", `'a\\'`, `a\\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokensWithMode(t, tt.in, parser.ModeNoBackslashEscapes, []wantToken{
				{parser.STRING, tt.want},
				{parser.EOF, ""},
			})
		})
	}
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

	t.Run("hex string with invalid digit", func(t *testing.T) {
		l := parser.New("x'0G'")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for non-hex digit in hex string literal")
		}
	})

	t.Run("bit string with invalid digit", func(t *testing.T) {
		l := parser.New("b'102'")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for non-binary digit in bit string literal")
		}
	})

	t.Run("hex string with odd number of digits", func(t *testing.T) {
		l := parser.New("x'F'")
		if _, err := l.Next(); err == nil {
			t.Fatal("expected error for odd number of digits in hex string literal")
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
	assertTokens(t, "  id\t\n\f\v  ,\r name  ", []wantToken{
		{parser.IDENT, "id"},
		{parser.COMMA, ","},
		{parser.IDENT, "name"},
		{parser.EOF, ""},
	})
}

// TestLexer_Comments covers comment rejection: the AST cannot keep
// comments, so the lexer reports them as errors instead of dropping them.
func TestLexer_Comments(t *testing.T) {
	leading := []struct {
		name string
		in   string
	}{
		{"leading double dash", "-- comment\nid"},
		{"leading hash", "# comment\nid"},
		{"leading block comment", "/* comment */ id"},
	}

	for _, tt := range leading {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.New(tt.in).Next()

			if _, ok := errors.AsType[*parser.LexError](err); !ok {
				t.Fatalf("Next() error = %v, want *parser.LexError for comment", err)
			}
		})
	}

	rejected := []struct {
		name string
		in   string
	}{
		{"double dash to end of line", "id -- this is a comment\n, name"},
		{"double dash then tab", "id --\tcomment"},
		{"double dash then form feed", "id --\fcomment"},
		{"double dash then vertical tab", "id --\vcomment"},
		{"double dash to EOF", "id -- trailing comment"},
		{"hash to end of line", "id # comment\n, name"},
		{"block comment", "id /* comment\nspanning lines */, name"},
		{"unterminated block comment", "id /* unterminated"},
		{"optimizer hint", "id /*+ MAX_EXECUTION_TIME(1000) */ 1"},
		{"executable comment", "id /*!50700 1 */"},
	}

	for _, tt := range rejected {
		t.Run(tt.name, func(t *testing.T) {
			l := parser.New(tt.in)
			if _, err := l.Next(); err != nil {
				t.Fatalf("Next() error = %v, want nil for first token", err)
			}

			_, err := l.Next()

			if _, ok := errors.AsType[*parser.LexError](err); !ok {
				t.Fatalf("Next() error = %v, want *parser.LexError for comment", err)
			}
		})
	}

	accepted := []struct {
		name string
		in   string
		want []wantToken
	}{
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
		{
			name: "trailing double dash at EOF is not a comment",
			in:   "a--",
			want: []wantToken{
				{parser.IDENT, "a"},
				{parser.MINUS, "-"},
				{parser.MINUS, "-"},
				{parser.EOF, ""},
			},
		},
		{
			name: "comment markers inside a string literal",
			in:   "'-- # /* */'",
			want: []wantToken{
				{parser.STRING, "-- # /* */"},
				{parser.EOF, ""},
			},
		},
		{
			name: "comment markers inside a quoted identifier",
			in:   "`a -- #b /*c*/`",
			want: []wantToken{
				{parser.QuotedIdent, "a -- #b /*c*/"},
				{parser.EOF, ""},
			},
		},
	}

	for _, tt := range accepted {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, tt.want)
		})
	}
}

// TestLexer_Raw covers Token.Raw, the exact source text of each token,
// which the parser uses to keep string literals as written.
func TestLexer_Raw(t *testing.T) {
	tests := []struct {
		name string
		mode parser.SQLMode
		in   string
		want []string
	}{
		{"plain string", parser.ModeDefault, "'abc'", []string{"'abc'"}},
		{"doubled quote", parser.ModeDefault, "'a''b'", []string{"'a''b'"}},
		{"backslash escapes", parser.ModeDefault, `'it\'s \n \\'`, []string{`'it\'s \n \\'`}},
		{"literal newline", parser.ModeDefault, "'a\nb'", []string{"'a\nb'"}},
		{"no backslash escapes", parser.ModeNoBackslashEscapes, `'a\'`, []string{`'a\'`}},
		{"hex and bit strings", parser.ModeDefault, "X'1a' b'01'", []string{"X'1a'", "b'01'"}},
		{"multibyte", parser.ModeDefault, "'café' = x", []string{"'café'", "=", "x"}},
		{"quoted identifier", parser.ModeDefault, "`a``b`", []string{"`a``b`"}},
		{"surrounding whitespace excluded", parser.ModeDefault, "  'a'  ,  1 ", []string{"'a'", ",", "1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := parser.NewWithMode(tt.in, tt.mode)

			var got []string

			for {
				tok, err := l.Next()
				if err != nil {
					t.Fatalf("Next() error = %v", err)
				}

				if tok.Type == parser.EOF {
					break
				}

				got = append(got, tok.Raw)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf("Raw = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLexer_IllegalCharacter(t *testing.T) {
	l := parser.New("~")

	_, err := l.Next()
	if err == nil {
		t.Fatal("expected error for illegal character")
	}

	if _, ok := errors.AsType[*parser.LexError](err); !ok {
		t.Fatalf("Next() error type = %T, want *parser.LexError", err)
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
