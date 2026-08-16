package sqlast

import (
	"strings"
	"unicode"
)

// reservedWords are the keywords recognized by the parser's grammar (see
// parser/token.go and parser-spec.md). An identifier that case-insensitively
// matches one of these can't be written bare and must be backtick-quoted.
//
// sqlast can't import the parser package to share its keyword table (parser
// already depends on sqlast), so this list is maintained by hand and must be
// kept in sync with the keywords defined there.
var reservedWords = map[string]struct{}{
	"SELECT": {}, "FROM": {}, "WHERE": {}, "JOIN": {}, "LEFT": {}, "RIGHT": {},
	"INNER": {}, "OUTER": {}, "CROSS": {}, "ON": {}, "AS": {}, "AND": {}, "OR": {},
	"NOT": {}, "IN": {}, "BETWEEN": {}, "LIKE": {}, "REGEXP": {}, "RLIKE": {},
	"IS": {}, "NULL": {}, "TRUE": {}, "FALSE": {}, "EXISTS": {}, "CASE": {},
	"WHEN": {}, "THEN": {}, "ELSE": {}, "END": {}, "INSERT": {}, "INTO": {},
	"VALUES": {}, "UPDATE": {}, "SET": {}, "DELETE": {}, "UNION": {}, "ALL": {},
	"DISTINCT": {}, "ORDER": {}, "BY": {}, "GROUP": {}, "ROLLUP": {}, "HAVING": {},
	"LIMIT": {}, "OFFSET": {}, "ASC": {}, "DESC": {}, "FOR": {}, "LOCK": {},
	"SHARE": {}, "NOWAIT": {}, "SKIP": {}, "LOCKED": {}, "WITH": {}, "RECURSIVE": {},
	"OVER": {}, "PARTITION": {}, "ROWS": {}, "RANGE": {}, "UNBOUNDED": {},
	"PRECEDING": {}, "FOLLOWING": {}, "CURRENT": {}, "ROW": {}, "IGNORE": {},
	"DUPLICATE": {}, "KEY": {}, "USE": {}, "FORCE": {}, "INDEX": {}, "NATURAL": {},
	"STRAIGHT_JOIN": {}, "RESPECT": {}, "NULLS": {}, "FIRST": {}, "LAST": {},
	"MODE": {}, "REPLACE": {}, "USING": {},
}

func isReservedWord(s string) bool {
	_, ok := reservedWords[strings.ToUpper(s)]

	return ok
}

// isIdentStartRune mirrors the parser lexer's isIdentStart: identifiers may
// start with a letter, underscore, or dollar sign.
func isIdentStartRune(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r)
}

// isIdentPartRune mirrors the parser lexer's isIdentPart: identifier
// characters after the first may also be an ASCII digit.
func isIdentPartRune(r rune) bool {
	return isIdentStartRune(r) || (r >= '0' && r <= '9')
}

// isBareIdent reports whether s can be written as an unquoted identifier per
// the parser's lexer grammar, independent of whether it happens to also be a
// reserved word.
func isBareIdent(s string) bool {
	if s == "" {
		return false
	}

	for i, r := range s {
		if i == 0 {
			if !isIdentStartRune(r) {
				return false
			}

			continue
		}

		if !isIdentPartRune(r) {
			return false
		}
	}

	return true
}

// quoteIdent renders s as a bare identifier when possible, or backtick-quotes
// it (doubling any embedded backticks) when it isn't a valid bare identifier
// shape. When checkReserved is true, a reserved keyword is quoted even if its
// shape would otherwise be a valid bare identifier (e.g. `select`).
//
// checkReserved must be false for a FuncExpr's function name: this grammar's
// only reserved word ever used as a name there is the deprecated VALUES(col)
// pseudo-function, which MySQL requires to stay unquoted.
func quoteIdent(s string, checkReserved bool) string {
	if isBareIdent(s) && (!checkReserved || !isReservedWord(s)) {
		return s
	}

	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}
