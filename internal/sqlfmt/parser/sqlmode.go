package parser

// SQLMode selects which MySQL sql_mode-dependent lexing and parsing
// behavior the lexer and parser use for string literals.
type SQLMode int

const (
	// ModeDefault matches MySQL's behavior without NO_BACKSLASH_ESCAPES set:
	// backslash is a string-literal escape character. It is the zero value,
	// so a Lexer or Parser constructed without an explicit mode behaves the
	// same as before SQLMode was introduced.
	ModeDefault SQLMode = iota

	// ModeNoBackslashEscapes matches MySQL's NO_BACKSLASH_ESCAPES sql_mode:
	// backslash has no special meaning in a string literal, and a literal
	// quote can only be embedded by doubling it.
	ModeNoBackslashEscapes
)
