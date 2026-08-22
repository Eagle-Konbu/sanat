package sqlfmt

import (
	"regexp"
	"strings"
)

var (
	sqlPrefixRe = regexp.MustCompile(`(?i)^\s*(SELECT|INSERT|UPDATE|DELETE|` +
		`CREATE|ALTER|DROP|TRUNCATE|` +
		`START|BEGIN|COMMIT|ROLLBACK|SAVEPOINT|RELEASE|SET|` +
		`SHOW|DESCRIBE|EXPLAIN|USE)\b`)
	fmtVerbRe = regexp.MustCompile(`%[+\-# 0]*[*]?[0-9]*[.*]?[0-9]*[vTtbcdoOqxXUeEfFgGsp]`)
)

func MightBeSQL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	if fmtVerbRe.MatchString(s) {
		return false
	}

	return sqlPrefixRe.MatchString(s)
}
