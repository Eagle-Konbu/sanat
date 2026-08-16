// Package parser implements a lexer and parser for MySQL SQL, producing the
// sqlast AST that the formatter walks to render output.
package parser

import "strings"

// TokenType identifies the category of a lexical token.
type TokenType int

const (
	EOF TokenType = iota

	IDENT       // unquoted identifier: users, id
	QuotedIdent // backtick-quoted identifier: `users`
	INT         // 123 (also 0x1A, 0b101)
	FLOAT       // 123.45
	STRING      // 'abc'
	HexStr      // x'1A' or X'1A'
	BitStr      // b'101' or B'101'

	EQ      // =
	NE      // <> or !=
	NSE     // <=>
	LT      // <
	GT      // >
	LE      // <=
	GE      // >=
	PLUS    // +
	MINUS   // -
	STAR    // *
	SLASH   // /
	PERCENT // %

	LPAREN   // (
	RPAREN   // )
	COMMA    // ,
	DOT      // .
	COLON    // :
	QUESTION // ?

	keywordBegin
	SELECT
	FROM
	WHERE
	JOIN
	LEFT
	RIGHT
	INNER
	OUTER
	CROSS
	ON
	AS
	AND
	OR
	NOT
	IN
	BETWEEN
	LIKE
	REGEXP
	RLIKE
	IS
	NULL
	TRUE
	FALSE
	EXISTS
	CASE
	WHEN
	THEN
	ELSE
	END
	INSERT
	INTO
	VALUES
	UPDATE
	SET
	DELETE
	UNION
	ALL
	DISTINCT
	ORDER
	BY
	GROUP
	ROLLUP
	HAVING
	LIMIT
	OFFSET
	ASC
	DESC
	FOR
	LOCK
	SHARE
	NOWAIT
	SKIP
	LOCKED
	WITH
	RECURSIVE
	OVER
	PARTITION
	ROWS
	RANGE
	UNBOUNDED
	PRECEDING
	FOLLOWING
	CURRENT
	ROW
	IGNORE
	DUPLICATE
	KEY
	USE
	FORCE
	INDEX
	NATURAL
	StraightJoin
	RESPECT
	NULLS
	FIRST
	LAST
	MODE
	REPLACE
	USING
	keywordEnd
)

var tokenNames = map[TokenType]string{
	EOF: "EOF",

	IDENT:       "IDENT",
	QuotedIdent: "QUOTED_IDENT",
	INT:         "INT",
	FLOAT:       "FLOAT",
	STRING:      "STRING",
	HexStr:      "HEX_STRING",
	BitStr:      "BIT_STRING",

	EQ:      "=",
	NE:      "<>",
	NSE:     "<=>",
	LT:      "<",
	GT:      ">",
	LE:      "<=",
	GE:      ">=",
	PLUS:    "+",
	MINUS:   "-",
	STAR:    "*",
	SLASH:   "/",
	PERCENT: "%",

	LPAREN:   "(",
	RPAREN:   ")",
	COMMA:    ",",
	DOT:      ".",
	COLON:    ":",
	QUESTION: "?",

	SELECT:       "SELECT",
	FROM:         "FROM",
	WHERE:        "WHERE",
	JOIN:         "JOIN",
	LEFT:         "LEFT",
	RIGHT:        "RIGHT",
	INNER:        "INNER",
	OUTER:        "OUTER",
	CROSS:        "CROSS",
	ON:           "ON",
	AS:           "AS",
	AND:          "AND",
	OR:           "OR",
	NOT:          "NOT",
	IN:           "IN",
	BETWEEN:      "BETWEEN",
	LIKE:         "LIKE",
	REGEXP:       "REGEXP",
	RLIKE:        "RLIKE",
	IS:           "IS",
	NULL:         "NULL",
	TRUE:         "TRUE",
	FALSE:        "FALSE",
	EXISTS:       "EXISTS",
	CASE:         "CASE",
	WHEN:         "WHEN",
	THEN:         "THEN",
	ELSE:         "ELSE",
	END:          "END",
	INSERT:       "INSERT",
	INTO:         "INTO",
	VALUES:       "VALUES",
	UPDATE:       "UPDATE",
	SET:          "SET",
	DELETE:       "DELETE",
	UNION:        "UNION",
	ALL:          "ALL",
	DISTINCT:     "DISTINCT",
	ORDER:        "ORDER",
	BY:           "BY",
	GROUP:        "GROUP",
	ROLLUP:       "ROLLUP",
	HAVING:       "HAVING",
	LIMIT:        "LIMIT",
	OFFSET:       "OFFSET",
	ASC:          "ASC",
	DESC:         "DESC",
	FOR:          "FOR",
	LOCK:         "LOCK",
	SHARE:        "SHARE",
	NOWAIT:       "NOWAIT",
	SKIP:         "SKIP",
	LOCKED:       "LOCKED",
	WITH:         "WITH",
	RECURSIVE:    "RECURSIVE",
	OVER:         "OVER",
	PARTITION:    "PARTITION",
	ROWS:         "ROWS",
	RANGE:        "RANGE",
	UNBOUNDED:    "UNBOUNDED",
	PRECEDING:    "PRECEDING",
	FOLLOWING:    "FOLLOWING",
	CURRENT:      "CURRENT",
	ROW:          "ROW",
	IGNORE:       "IGNORE",
	DUPLICATE:    "DUPLICATE",
	KEY:          "KEY",
	USE:          "USE",
	FORCE:        "FORCE",
	INDEX:        "INDEX",
	NATURAL:      "NATURAL",
	StraightJoin: "STRAIGHT_JOIN",
	RESPECT:      "RESPECT",
	NULLS:        "NULLS",
	FIRST:        "FIRST",
	LAST:         "LAST",
	MODE:         "MODE",
	REPLACE:      "REPLACE",
	USING:        "USING",
}

// String returns the token type's display name, used in error messages.
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}

	return "UNKNOWN"
}

// IsKeyword reports whether t is one of the reserved SQL keywords.
func (t TokenType) IsKeyword() bool {
	return t > keywordBegin && t < keywordEnd
}

var keywords = buildKeywordTable()

func buildKeywordTable() map[string]TokenType {
	m := make(map[string]TokenType, keywordEnd-keywordBegin-1)
	for tt := keywordBegin + 1; tt < keywordEnd; tt++ {
		m[tokenNames[tt]] = tt
	}

	return m
}

// lookupIdent returns the keyword TokenType for literal (matched case-insensitively),
// or IDENT if literal is not a reserved keyword.
func lookupIdent(literal string) TokenType {
	if tt, ok := keywords[strings.ToUpper(literal)]; ok {
		return tt
	}

	return IDENT
}

// Position identifies a location within the lexer input.
type Position struct {
	Offset int // byte offset, 0-based
	Line   int // 1-based
	Column int // 1-based, in runes
}

// Token is a single lexical token produced by the Lexer.
type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}
