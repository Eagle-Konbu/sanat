package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		name string
		typ  parser.TokenType
		want string
	}{
		{"keyword", parser.SELECT, "SELECT"},
		{"punctuation", parser.LPAREN, "("},
		{"eof", parser.EOF, "EOF"},
		{"unknown", parser.TokenType(-1), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTokenType_IsKeyword(t *testing.T) {
	tests := []struct {
		name string
		typ  parser.TokenType
		want bool
	}{
		{"keyword", parser.SELECT, true},
		{"ident", parser.IDENT, false},
		{"eof", parser.EOF, false},
		{"punctuation", parser.LPAREN, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.typ.IsKeyword(); got != tt.want {
				t.Errorf("IsKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLexError_Error(t *testing.T) {
	err := &parser.LexError{Pos: parser.Position{Line: 2, Column: 5}, Msg: "unexpected character"}

	want := "2:5: unexpected character"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
