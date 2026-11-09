package sqlfmt

import "strings"

// renderer holds the resolved rendering options for a single FormatSQL call
// and the dialect-neutral layout helpers built on them. It knows nothing
// about any AST, so every dialect's statement renderer embeds it to share
// the same indentation, keyword-casing, and comma-style behavior.
type renderer struct {
	indent      int
	keywordCase string
	commaStyle  string
}

func newRenderer(opts Options) renderer {
	r := renderer{
		indent:      opts.Indent,
		keywordCase: opts.KeywordCase,
		commaStyle:  opts.CommaStyle,
	}

	if r.keywordCase == "" {
		r.keywordCase = KeywordCaseUpper
	}

	if r.commaStyle == "" {
		r.commaStyle = CommaStyleTrailing
	}

	return r
}

func (r *renderer) pad(depth int) string {
	return strings.Repeat(" ", depth*r.indent)
}

// keyword renders a hardcoded operator/predicate literal (e.g. "AS", "AND")
// according to the configured keyword case. The AST String() methods always
// render these uppercase on their own, so KeywordCaseUpper and
// KeywordCasePreserve both take the identity branch here; only
// KeywordCaseLower does any work.
func (r *renderer) keyword(word string) string {
	if r.keywordCase == KeywordCaseLower {
		return strings.ToLower(word)
	}

	return strings.ToUpper(word)
}

// itemPrefix returns the indentation/comma prefix to write before item i of
// an n-item list, per the configured comma style.
func (r *renderer) itemPrefix(pi string, i int) string {
	if r.commaStyle == CommaStyleLeading && i > 0 {
		return leadingCommaPrefix(pi)
	}

	return pi
}

// itemSuffix returns the trailing comma (if any) to write after item i of an
// n-item list, per the configured comma style.
func (r *renderer) itemSuffix(i, n int) string {
	if r.commaStyle == CommaStyleLeading {
		return ""
	}

	if i < n-1 {
		return ","
	}

	return ""
}

// writeList writes pre-rendered single-line items, applying the configured
// comma style.
func (r *renderer) writeList(b *strings.Builder, pi string, lines []string) {
	for i, line := range lines {
		b.WriteString(r.itemPrefix(pi, i) + line + r.itemSuffix(i, len(lines)) + "\n")
	}
}

func leadingCommaPrefix(pi string) string {
	n := max(len(pi)-2, 0)

	return strings.Repeat(" ", n) + ", "
}
