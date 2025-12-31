package helpers

import "strings"

// NormalizeIdentifier normalizes an identifier for case-insensitive comparison.
// Removes backticks and converts to lowercase.
func NormalizeIdentifier(s string) string {
	s = strings.Trim(s, "`")
	return strings.ToLower(s)
}
