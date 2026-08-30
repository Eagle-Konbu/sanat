// Package pgparser implements a lexer (and, in later issues, a parser) for
// PostgreSQL 13+ DML syntax, producing the pgast AST the formatter walks to
// render output. It is a separate package from internal/sqlfmt/parser rather
// than a Dialect-gated branch inside it, because PostgreSQL and MySQL's
// lexical rules genuinely conflict (double-quoted identifiers vs. backtick
// quoting, dollar-quoting/placeholders, opposite default backslash-escape
// behavior). This issue implements only the lexer (tokenizer) — no grammar.
package pgparser

import (
	"fmt"
	"strings"
)

// TokenType identifies the category of a lexical token.
type TokenType int

const (
	EOF TokenType = iota

	IDENT       // unquoted identifier, folded to lower case: users, id
	QuotedIdent // double-quoted identifier, case preserved: "Users"
	INT         // 123
	FLOAT       // 123.45
	STRING      // 'abc', E'abc', or $tag$abc$tag$ (Literal is the decoded content)
	Placeholder // $1, $2, ... (Literal is the digits, without the leading $)

	EQ      // =
	NE      // <> or !=
	LT      // <
	GT      // >
	LE      // <=
	GE      // >=
	PLUS    // +
	MINUS   // -
	STAR    // *
	SLASH   // /
	PERCENT // %

	DoubleColon   // ::
	Arrow         // ->
	ArrowText     // ->>
	HashArrow     // #>
	HashArrowText // #>>
	Contains      // @>
	ContainedBy   // <@
	JSONExists    // ?
	JSONExistsAny // ?|
	JSONExistsAll // ?&

	LPAREN    // (
	RPAREN    // )
	LBRACKET  // [
	RBRACKET  // ]
	COMMA     // ,
	DOT       // .
	COLON     // :
	SEMICOLON // ;

	keywordBegin
	SELECT
	FROM
	WHERE
	AS
	AND
	OR
	NOT
	IN
	BETWEEN
	LIKE
	ILIKE
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
	DISTINCT
	ON
	ALL
	CAST
	ARRAY
	FILTER
	ISNULL
	NOTNULL
	ORDER
	BY
	GROUP
	HAVING
	WINDOW
	ASC
	DESC
	NULLS
	FIRST
	LAST
	LIMIT
	OFFSET
	FETCH
	NEXT
	ROW
	ROWS
	GROUPS
	ONLY
	WITH
	TIES
	FOR
	UPDATE
	NO
	KEY
	SHARE
	OF
	NOWAIT
	SKIP
	LOCKED
	JOIN
	INNER
	LEFT
	RIGHT
	FULL
	OUTER
	CROSS
	NATURAL
	LATERAL
	USING
	GROUPING
	SETS
	CUBE
	ROLLUP
	OVER
	PARTITION
	RANGE
	UNBOUNDED
	PRECEDING
	FOLLOWING
	CURRENT
	keywordEnd
)

var tokenNames = map[TokenType]string{
	EOF: "EOF",

	IDENT:       "IDENT",
	QuotedIdent: "QUOTED_IDENT",
	INT:         "INT",
	FLOAT:       "FLOAT",
	STRING:      "STRING",
	Placeholder: "PLACEHOLDER",

	EQ:      "=",
	NE:      "<>",
	LT:      "<",
	GT:      ">",
	LE:      "<=",
	GE:      ">=",
	PLUS:    "+",
	MINUS:   "-",
	STAR:    "*",
	SLASH:   "/",
	PERCENT: "%",

	DoubleColon:   "::",
	Arrow:         "->",
	ArrowText:     "->>",
	HashArrow:     "#>",
	HashArrowText: "#>>",
	Contains:      "@>",
	ContainedBy:   "<@",
	JSONExists:    "?",
	JSONExistsAny: "?|",
	JSONExistsAll: "?&",

	LPAREN:    "(",
	RPAREN:    ")",
	LBRACKET:  "[",
	RBRACKET:  "]",
	COMMA:     ",",
	DOT:       ".",
	COLON:     ":",
	SEMICOLON: ";",

	SELECT:    "SELECT",
	FROM:      "FROM",
	WHERE:     "WHERE",
	AS:        "AS",
	AND:       "AND",
	OR:        "OR",
	NOT:       "NOT",
	IN:        "IN",
	BETWEEN:   "BETWEEN",
	LIKE:      "LIKE",
	ILIKE:     "ILIKE",
	IS:        "IS",
	NULL:      "NULL",
	TRUE:      "TRUE",
	FALSE:     "FALSE",
	EXISTS:    "EXISTS",
	CASE:      "CASE",
	WHEN:      "WHEN",
	THEN:      "THEN",
	ELSE:      "ELSE",
	END:       "END",
	DISTINCT:  "DISTINCT",
	ON:        "ON",
	ALL:       "ALL",
	CAST:      "CAST",
	ARRAY:     "ARRAY",
	FILTER:    "FILTER",
	ISNULL:    "ISNULL",
	NOTNULL:   "NOTNULL",
	ORDER:     "ORDER",
	BY:        "BY",
	GROUP:     "GROUP",
	HAVING:    "HAVING",
	WINDOW:    "WINDOW",
	ASC:       "ASC",
	DESC:      "DESC",
	NULLS:     "NULLS",
	FIRST:     "FIRST",
	LAST:      "LAST",
	LIMIT:     "LIMIT",
	OFFSET:    "OFFSET",
	FETCH:     "FETCH",
	NEXT:      "NEXT",
	ROW:       "ROW",
	ROWS:      "ROWS",
	GROUPS:    "GROUPS",
	ONLY:      "ONLY",
	WITH:      "WITH",
	TIES:      "TIES",
	FOR:       "FOR",
	UPDATE:    "UPDATE",
	NO:        "NO",
	KEY:       "KEY",
	SHARE:     "SHARE",
	OF:        "OF",
	NOWAIT:    "NOWAIT",
	SKIP:      "SKIP",
	LOCKED:    "LOCKED",
	JOIN:      "JOIN",
	INNER:     "INNER",
	LEFT:      "LEFT",
	RIGHT:     "RIGHT",
	FULL:      "FULL",
	OUTER:     "OUTER",
	CROSS:     "CROSS",
	NATURAL:   "NATURAL",
	LATERAL:   "LATERAL",
	USING:     "USING",
	GROUPING:  "GROUPING",
	SETS:      "SETS",
	CUBE:      "CUBE",
	ROLLUP:    "ROLLUP",
	OVER:      "OVER",
	PARTITION: "PARTITION",
	RANGE:     "RANGE",
	UNBOUNDED: "UNBOUNDED",
	PRECEDING: "PRECEDING",
	FOLLOWING: "FOLLOWING",
	CURRENT:   "CURRENT",
}

// String returns the token type's display name, used in error messages.
func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}

	return "UNKNOWN"
}

// IsKeyword reports whether t is a SQL keyword.
func (t TokenType) IsKeyword() bool {
	return t > keywordBegin && t < keywordEnd
}

var keywords = buildKeywordTable()

func buildKeywordTable() map[string]TokenType {
	m := make(map[string]TokenType, keywordEnd-keywordBegin-1)

	for tt := keywordBegin + 1; tt < keywordEnd; tt++ {
		name, ok := tokenNames[tt]
		if !ok {
			panic(fmt.Sprintf("pgparser: keyword token %d has no tokenNames entry", tt))
		}

		m[name] = tt
	}

	return m
}

// lookupIdent returns the keyword TokenType for literal (matched
// case-insensitively, per PostgreSQL's default unquoted-identifier folding),
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
