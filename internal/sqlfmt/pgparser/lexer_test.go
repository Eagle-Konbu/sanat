package pgparser_test

import (
	"errors"
	"strings"
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
			if _, ok := errors.AsType[*pgparser.LexError](err); !ok {
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
		{"recursive", "recursive", pgparser.RECURSIVE},
		{"materialized", "MATERIALIZED", pgparser.MATERIALIZED},
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

	// PostgreSQL's scanner accepts any byte from 0x80 up in an identifier,
	// not just letters.
	t.Run("non-ASCII identifier characters", func(t *testing.T) {
		assertTokens(t, "café €uro x·y", []wantToken{
			{pgparser.IDENT, "café"},
			{pgparser.IDENT, "€uro"},
			{pgparser.IDENT, "x·y"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("identifier starting with a combining mark", func(t *testing.T) {
		assertTokens(t, "́a", []wantToken{
			{pgparser.IDENT, "́a"},
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
		{"zero code point", `E'\u0000'`},
	}

	// PostgreSQL text values cannot hold a zero byte.
	for _, in := range []string{`E'\0'`, `E'a\000b'`, `E'\400'`, `E'\x00'`, `E'\x0'`} {
		t.Run("escape string zero byte "+in, func(t *testing.T) {
			assertLexError(t, in)
		})
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

	t.Run("trailing junk", func(t *testing.T) {
		assertLexError(t, "$1abc")
	})
}

func TestLexer_prefixedStrings(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want wantToken
	}{
		{"bit string", "B'0101'", wantToken{pgparser.BitString, "B'0101'"}},
		{"lower-case bit string", "b'01'", wantToken{pgparser.BitString, "B'01'"}},
		{"empty bit string", "B''", wantToken{pgparser.BitString, "B''"}},
		{"hex bit string", "X'1F'", wantToken{pgparser.BitString, "X'1F'"}},
		{"lower-case hex bit string", "x'af'", wantToken{pgparser.BitString, "X'af'"}},
		{"national string", "N'abc'", wantToken{pgparser.NationalString, "abc"}},
		{"national string with doubled quote", "n'it''s'", wantToken{pgparser.NationalString, "it's"}},
		{"bit string continuation", "B'01'\n'10'", wantToken{pgparser.BitString, "B'0110'"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, []wantToken{tt.want, {pgparser.EOF, ""}})
		})
	}

	for _, in := range []string{"B'012'", "X'1G'", "B'0", "N'abc", "B'0''1'"} {
		t.Run("invalid "+in, func(t *testing.T) {
			assertLexError(t, in)
		})
	}

	t.Run("prefix letter followed by a space is an identifier", func(t *testing.T) {
		assertTokens(t, "b '1'", []wantToken{
			{pgparser.IDENT, "b"},
			{pgparser.STRING, "1"},
			{pgparser.EOF, ""},
		})
	})

	t.Run("longer identifier ending in a prefix letter", func(t *testing.T) {
		assertTokens(t, "ab'1'", []wantToken{
			{pgparser.IDENT, "ab"},
			{pgparser.STRING, "1"},
			{pgparser.EOF, ""},
		})
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

	// PostgreSQL 15+ rejects an identifier character directly after a
	// number rather than reading it as an alias.
	for _, in := range []string{"0x1", "1e", "1.col", "123abc", "1.5e3x", ".5a", "1é"} {
		t.Run("trailing junk "+in, func(t *testing.T) {
			assertLexError(t, in)
		})
	}

	t.Run("number followed by an operator or space", func(t *testing.T) {
		assertTokens(t, "1+2 3", []wantToken{
			{pgparser.INT, "1"},
			{pgparser.PLUS, "+"},
			{pgparser.INT, "2"},
			{pgparser.INT, "3"},
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
		{"^", pgparser.CARET},
		{"::", pgparser.DoubleColon},
		{"=>", pgparser.EqualsGreater},
		{":=", pgparser.ColonEquals},
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
}

func TestLexer_genericOperators(t *testing.T) {
	for _, in := range []string{
		"||", "->", "->>", "#>", "#>>", "#-", "@>", "<@", "@?", "@@", "?", "?|", "?&",
		"~", "~*", "!~", "!~*", "~~", "&", "&&", "|", "<<", ">>", "#", "@", "!", "<->",
		"|/", "||/", "`", "?-", "@-", "%+", "<=>", "=~",
	} {
		t.Run(in, func(t *testing.T) {
			assertTokens(t, in, []wantToken{
				{pgparser.Op, in},
				{pgparser.EOF, ""},
			})
		})
	}
}

// A run of operator characters is one operator, except that it stops at a
// comment start, and a trailing + or - is split off unless the run contains
// a character SQL's own operators never use (scan.l's rule).
func TestLexer_operatorBoundaries(t *testing.T) {
	tests := []struct {
		name, in string
		want     []wantToken
	}{
		{"trailing minus split off", "a=-1", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.EQ, "="}, {pgparser.MINUS, "-"}, {pgparser.INT, "1"},
		}},
		{"trailing minus after less-than", "a<-1", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.LT, "<"}, {pgparser.MINUS, "-"}, {pgparser.INT, "1"},
		}},
		{"several trailing signs split off", "a*+-b", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.STAR, "*"}, {pgparser.PLUS, "+"}, {pgparser.MINUS, "-"}, {pgparser.IDENT, "b"},
		}},
		{"split leaves a two-character fixed operator", "a<>-1", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.NE, "<>"}, {pgparser.MINUS, "-"}, {pgparser.INT, "1"},
		}},
		{"trailing minus kept after a non-SQL character", "a@-1", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.Op, "@-"}, {pgparser.INT, "1"},
		}},
		{"distance operator stays whole", "a<->b", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.Op, "<->"}, {pgparser.IDENT, "b"},
		}},
		{"stops at line comment", "a+--c\nb", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.PLUS, "+"}, {pgparser.IDENT, "b"},
		}},
		{"stops at block comment", "a||/* c */b", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.Op, "||"}, {pgparser.IDENT, "b"},
		}},
		{"named argument arrow", "a=>1", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.EqualsGreater, "=>"}, {pgparser.INT, "1"},
		}},
		{"split-off signs stop at a comment", "a<+-+--c\nb", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.LT, "<"}, {pgparser.PLUS, "+"}, {pgparser.MINUS, "-"},
			{pgparser.PLUS, "+"}, {pgparser.IDENT, "b"},
		}},
		{"operator after split-off signs is lexed afresh", "a*+ +@b", []wantToken{
			{pgparser.IDENT, "a"}, {pgparser.STAR, "*"}, {pgparser.PLUS, "+"}, {pgparser.Op, "+@"}, {pgparser.IDENT, "b"},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertTokens(t, tt.in, append(tt.want, wantToken{pgparser.EOF, ""}))
		})
	}
}

// A long run of signs splits into one-character operators; lexing it must
// not scan the rest of the run again for each one.
func TestLexer_longSignRun(t *testing.T) {
	const n = 1 << 16

	l := pgparser.New("a" + strings.Repeat("+", n) + "1")

	count := 0

	for {
		tok, err := l.Next()
		if err != nil {
			t.Fatalf("Next() error = %v", err)
		}

		if tok.Type == pgparser.EOF {
			break
		}

		count++
	}

	if want := n + 2; count != want {
		t.Errorf("token count = %d, want %d", count, want)
	}
}

func TestLexer_unicodeEscapeConstantUnsupported(t *testing.T) {
	for _, in := range []string{"U&'x'", `u&"x"`} {
		t.Run(in, func(t *testing.T) {
			assertLexError(t, in)
		})
	}

	t.Run("u followed by the & operator", func(t *testing.T) {
		assertTokens(t, "u & 'x'", []wantToken{
			{pgparser.IDENT, "u"}, {pgparser.Op, "&"}, {pgparser.STRING, "x"}, {pgparser.EOF, ""},
		})
	})
}

func TestLexer_unexpectedCharacter(t *testing.T) {
	for _, in := range []string{"{", "}", "\\", "a \\ b"} {
		t.Run(in, func(t *testing.T) {
			assertLexError(t, in)
		})
	}
}

func TestLexer_invalidUTF8Identifier(t *testing.T) {
	for _, in := range []string{"\xffname", "name\xff", "a\xc3"} {
		t.Run(in, func(t *testing.T) {
			assertLexError(t, in)
		})
	}

	t.Run("U+FFFD is an identifier character", func(t *testing.T) {
		assertTokens(t, "a\uFFFDb", []wantToken{{pgparser.IDENT, "a\uFFFDb"}, {pgparser.EOF, ""}})
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

	t.Run("form feed and vertical tab are whitespace", func(t *testing.T) {
		assertTokens(t, "a\fb\vc", []wantToken{
			{pgparser.IDENT, "a"},
			{pgparser.IDENT, "b"},
			{pgparser.IDENT, "c"},
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
