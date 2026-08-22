package sqlfmt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

const (
	KeywordCaseUpper    = "upper"
	KeywordCaseLower    = "lower"
	KeywordCasePreserve = "preserve"

	CommaStyleTrailing = "trailing"
	CommaStyleLeading  = "leading"

	SQLModeDefault            = "default"
	SQLModeNoBackslashEscapes = "no_backslash_escapes"
)

var (
	sentinelRe    = regexp.MustCompile(`:_sqla_ph_(\d+)`)
	placeholderRe = regexp.MustCompile(`\?`)

	// keywordCaseRe matches whole-word operator/predicate keywords that the
	// sqlast package always renders uppercase. Word boundaries (rather than a
	// literal-substring replacer) keep adjacent keywords like "IS NULL" or
	// "NOT BETWEEN" independently addressable — a substring replacer would
	// consume the shared space between them and miss the second keyword.
	keywordCaseRe = regexp.MustCompile(
		`\b(AS|ASC|DESC|AND|OR|NOT|IN|IS|LIKE|BETWEEN|EXISTS|NULL|TRUE|FALSE|ON|USING)\b`,
	)
)

// Options controls how FormatSQLWithOptions renders a SQL statement.
type Options struct {
	Indent int

	// KeywordCase controls the casing of operator/predicate keywords
	// (AS, AND, OR, NOT, IN, IS, LIKE, ...). Defaults to KeywordCaseUpper.
	//
	// The in-house parser does not retain the source's original keyword
	// casing for these tokens either (they're canonicalized to uppercase
	// while parsing), so KeywordCasePreserve currently behaves the same as
	// KeywordCaseUpper (the parser's natural output).
	KeywordCase string

	// CommaStyle controls comma placement in rendered lists (SELECT
	// columns, INSERT columns/values, SET assignments, ...). Defaults to
	// CommaStyleTrailing.
	CommaStyle string

	// SQLMode selects MySQL sql_mode-dependent string-literal parsing
	// behavior. Defaults to SQLModeDefault, which matches MySQL without
	// NO_BACKSLASH_ESCAPES set: backslash is a string-literal escape
	// character. FormatSQL and FormatSQLWithOptions format one statement at
	// a time with no session or connection state, so callers whose
	// connection has NO_BACKSLASH_ESCAPES enabled must set SQLMode to
	// SQLModeNoBackslashEscapes explicitly — sanat cannot infer it from the
	// input SQL.
	SQLMode string
}

// formatter holds the resolved rendering options for a single FormatSQL call.
type formatter struct {
	indent      int
	keywordCase string
	commaStyle  string
}

func newFormatter(opts Options) *formatter {
	f := &formatter{
		indent:      opts.Indent,
		keywordCase: opts.KeywordCase,
		commaStyle:  opts.CommaStyle,
	}

	if f.keywordCase == "" {
		f.keywordCase = KeywordCaseUpper
	}

	if f.commaStyle == "" {
		f.commaStyle = CommaStyleTrailing
	}

	return f
}

// FormatSQL formats sql with the given indent width, using the default
// keyword casing and comma style. See FormatSQLWithOptions for full control.
func FormatSQL(sql string, indent int) (string, bool) {
	return FormatSQLWithOptions(sql, Options{Indent: indent})
}

func FormatSQLWithOptions(sql string, opts Options) (string, bool) {
	mode, ok := parserSQLMode(opts.SQLMode)
	if !ok {
		return sql, false
	}

	replaced, count := replacePlaceholders(sql)

	stmt, err := parser.ParseStatementWithMode(replaced, mode)
	if err != nil {
		return sql, false
	}

	result, ok := formatParsedStatement(opts, stmt)
	if !ok {
		return sql, false
	}

	return restorePlaceholders(result, count), true
}

// parserSQLMode translates an Options.SQLMode value into the parser
// package's SQLMode, reporting whether sqlMode was a recognized value.
// The empty string and SQLModeDefault both map to parser.ModeDefault;
// SQLModeNoBackslashEscapes maps to parser.ModeNoBackslashEscapes; any other
// value is rejected with ok == false.
func parserSQLMode(sqlMode string) (parser.SQLMode, bool) {
	switch sqlMode {
	case "", SQLModeDefault:
		return parser.ModeDefault, true
	case SQLModeNoBackslashEscapes:
		return parser.ModeNoBackslashEscapes, true
	default:
		return parser.ModeDefault, false
	}
}

// formatParsedStatement renders stmt, recovering a panic from an AST node
// type the formatter doesn't handle (see the "sqlfmt: unhandled ... type"
// panics below) into an (ok=false) failure instead of crashing the caller —
// the same fallback FormatSQLWithOptions already gives a parse failure.
//
//nolint:nonamedreturns // the named results are mutated by the deferred recover
func formatParsedStatement(opts Options, stmt sqlast.Statement) (result string, ok bool) {
	defer func() {
		if recover() != nil {
			result, ok = "", false
		}
	}()

	f := newFormatter(opts)

	var b strings.Builder

	f.formatStatement(&b, stmt, 0)

	return b.String(), true
}

func replacePlaceholders(sql string) (string, int) {
	count := 0
	result := placeholderRe.ReplaceAllStringFunc(sql, func(_ string) string {
		s := fmt.Sprintf(":_sqla_ph_%d", count)
		count++

		return s
	})

	return result, count
}

func restorePlaceholders(sql string, _ int) string {
	return sentinelRe.ReplaceAllString(sql, "?")
}

func (f *formatter) formatStatement(b *strings.Builder, stmt sqlast.Statement, depth int) {
	switch s := stmt.(type) {
	case *sqlast.Select:
		f.formatSelect(b, s, depth)
	case *sqlast.Insert:
		f.formatInsert(b, s, depth)
	case *sqlast.Update:
		f.formatUpdate(b, s, depth)
	case *sqlast.Delete:
		f.formatDelete(b, s, depth)
	case *sqlast.Union:
		f.formatUnion(b, s, depth)
	default:
		if f.formatDDLStatement(b, stmt, depth) {
			return
		}

		if f.formatSessionStatement(b, stmt, depth) {
			return
		}

		if !f.formatAdminStatement(b, stmt, depth) {
			panic(fmt.Sprintf("sqlfmt: unhandled statement type %T", stmt))
		}
	}
}

// formatDDLStatement handles the DDL statement types (CREATE/ALTER/DROP
// TABLE, CREATE/DROP INDEX, TRUNCATE TABLE), split out of formatStatement to
// keep that switch's cyclomatic complexity down. It reports whether stmt was
// a recognized DDL statement.
func (f *formatter) formatDDLStatement(b *strings.Builder, stmt sqlast.Statement, depth int) bool {
	switch s := stmt.(type) {
	case *sqlast.CreateTable:
		f.formatCreateTable(b, s, depth)
	case *sqlast.AlterTable:
		f.formatAlterTable(b, s, depth)
	case *sqlast.CreateIndex:
		f.formatCreateIndex(b, s, depth)
	case *sqlast.DropIndex:
		f.formatDropIndex(b, s, depth)
	case *sqlast.DropTable:
		f.formatDropTable(b, s, depth)
	case *sqlast.TruncateTable:
		f.formatTruncateTable(b, s, depth)
	default:
		return false
	}

	return true
}

// formatSessionStatement handles the transaction-control and SET statement
// types (START TRANSACTION, BEGIN, COMMIT, ROLLBACK, SAVEPOINT, RELEASE
// SAVEPOINT, SET), split out of formatStatement to keep that switch's
// cyclomatic complexity down. It reports whether stmt was a recognized
// transaction/session statement.
func (f *formatter) formatSessionStatement(b *strings.Builder, stmt sqlast.Statement, depth int) bool {
	switch s := stmt.(type) {
	case *sqlast.StartTransaction:
		f.formatStartTransaction(b, s, depth)
	case *sqlast.Begin:
		f.formatBegin(b, depth)
	case *sqlast.Commit:
		f.formatCommit(b, depth)
	case *sqlast.Rollback:
		f.formatRollback(b, s, depth)
	case *sqlast.Savepoint:
		f.formatSavepoint(b, s, depth)
	case *sqlast.ReleaseSavepoint:
		f.formatReleaseSavepoint(b, s, depth)
	case *sqlast.SetVariable:
		f.formatSetVariable(b, s, depth)
	case *sqlast.SetNames:
		f.formatSetNames(b, s, depth)
	default:
		return false
	}

	return true
}

// formatStartTransaction, formatRollback, formatSavepoint, and
// formatReleaseSavepoint all delegate to their sqlast node's String(): none
// of these statements have an Expr-valued field, so there's no keyword-case
// or indentation concern String() doesn't already handle correctly (unlike
// formatSetVariable/formatSetNames, whose values must go through
// formatExpr).
func (f *formatter) formatStartTransaction(b *strings.Builder, s *sqlast.StartTransaction, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(s.String())
	b.WriteString("\n")
}

func (f *formatter) formatBegin(b *strings.Builder, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("BEGIN\n")
}

func (f *formatter) formatCommit(b *strings.Builder, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("COMMIT\n")
}

func (f *formatter) formatRollback(b *strings.Builder, s *sqlast.Rollback, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(s.String())
	b.WriteString("\n")
}

func (f *formatter) formatSavepoint(b *strings.Builder, s *sqlast.Savepoint, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(s.String())
	b.WriteString("\n")
}

func (f *formatter) formatReleaseSavepoint(b *strings.Builder, s *sqlast.ReleaseSavepoint, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(s.String())
	b.WriteString("\n")
}

func (f *formatter) formatSetVariable(b *strings.Builder, s *sqlast.SetVariable, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("SET ")

	if scope := s.Scope.String(); scope != "" {
		b.WriteString(scope)
		b.WriteString(" ")
	}

	if s.IsUserVariable {
		b.WriteString("@")
	}

	b.WriteString(s.Name)
	b.WriteString(" = ")
	b.WriteString(f.formatExpr(s.Value, depth))
	b.WriteString("\n")
}

func (f *formatter) formatSetNames(b *strings.Builder, s *sqlast.SetNames, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("SET NAMES ")
	b.WriteString(f.formatExpr(s.Charset, depth))

	if s.Collate != nil {
		b.WriteString(" COLLATE ")
		b.WriteString(f.formatExpr(s.Collate, depth))
	}

	b.WriteString("\n")
}

// formatAdminStatement handles the admin/utility statement types (SHOW
// TABLES/CREATE TABLE/COLUMNS/INDEX/DATABASES/VARIABLES/STATUS, DESCRIBE,
// EXPLAIN, USE), split out of formatStatement to keep that switch's
// cyclomatic complexity down. It reports whether stmt was a recognized
// admin/utility statement.
func (f *formatter) formatAdminStatement(b *strings.Builder, stmt sqlast.Statement, depth int) bool {
	switch s := stmt.(type) {
	case *sqlast.ShowTables, *sqlast.ShowCreateTable, *sqlast.ShowColumns, *sqlast.ShowIndex,
		*sqlast.ShowDatabases, *sqlast.ShowVariables, *sqlast.ShowStatus, *sqlast.Describe, *sqlast.Use:
		f.formatSingleLineStatement(b, s, depth)
	case *sqlast.Explain:
		f.formatExplain(b, s, depth)
	default:
		return false
	}

	return true
}

// formatSingleLineStatement writes a statement whose sqlast node's String()
// is already the exact rendered output: none of the SHOW/DESCRIBE/USE
// statement kinds have an Expr-valued field that needs keyword-case or
// indentation handling beyond what String() already does (a SHOW ...
// LIKE pattern is always a quoted string literal, never a bareword
// expression keyword-case could affect).
func (f *formatter) formatSingleLineStatement(b *strings.Builder, s sqlast.Statement, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString(s.String())
	b.WriteString("\n")
}

// formatExplain writes "EXPLAIN [FORMAT = fmt]" on its own line, then
// formats the wrapped statement one level deeper, the same nesting
// convention formatWith uses for a CTE's subquery body.
func (f *formatter) formatExplain(b *strings.Builder, s *sqlast.Explain, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("EXPLAIN")

	if format := s.Format.String(); format != "" {
		b.WriteString(" FORMAT = ")
		b.WriteString(format)
	}

	b.WriteString("\n")
	f.formatStatement(b, s.Statement, depth+1)
}

func (f *formatter) pad(depth int) string {
	return strings.Repeat(" ", depth*f.indent)
}

// keyword renders a hardcoded operator/predicate literal (e.g. "AS", "AND")
// according to the configured keyword case. sqlast always renders these
// uppercase on its own, so KeywordCaseUpper and KeywordCasePreserve both take
// the identity branch here; only KeywordCaseLower does any work.
func (f *formatter) keyword(word string) string {
	if f.keywordCase == KeywordCaseLower {
		return strings.ToLower(word)
	}

	return strings.ToUpper(word)
}

// applyKeywordCase lowercases operator/predicate keywords embedded in text
// produced by an sqlast node's String() method (which always renders them
// uppercase), skipping quoted regions so literal contents are left intact.
// It is a no-op unless keywordCase is KeywordCaseLower.
func (f *formatter) applyKeywordCase(s string) string {
	if f.keywordCase != KeywordCaseLower {
		return s
	}

	var b strings.Builder

	b.Grow(len(s))

	segStart := 0

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\'' && c != '"' && c != '`' {
			continue
		}

		b.WriteString(keywordCaseRe.ReplaceAllStringFunc(s[segStart:i], strings.ToLower))

		end := quotedRegionEnd(s, i, c)
		b.WriteString(s[i:end])
		segStart = end
		i = end - 1
	}

	b.WriteString(keywordCaseRe.ReplaceAllStringFunc(s[segStart:], strings.ToLower))

	return b.String()
}

// quotedRegionEnd returns the index just past the closing quote of the
// quoted region starting at s[start] (a c-quote character), treating
// backslash as an escape for the following character.
func quotedRegionEnd(s string, start int, c byte) int {
	i := start + 1

	for i < len(s) {
		switch {
		case s[i] == '\\' && i+1 < len(s):
			i += 2
		case s[i] == c:
			return i + 1
		default:
			i++
		}
	}

	return i
}

// itemPrefix returns the indentation/comma prefix to write before item i of
// an n-item list, per the configured comma style.
func (f *formatter) itemPrefix(pi string, i int) string {
	if f.commaStyle == CommaStyleLeading && i > 0 {
		return leadingCommaPrefix(pi)
	}

	return pi
}

// itemSuffix returns the trailing comma (if any) to write after item i of an
// n-item list, per the configured comma style.
func (f *formatter) itemSuffix(i, n int) string {
	if f.commaStyle == CommaStyleLeading {
		return ""
	}

	if i < n-1 {
		return ","
	}

	return ""
}

// writeList writes pre-rendered single-line items, applying the configured
// comma style.
func (f *formatter) writeList(b *strings.Builder, pi string, lines []string) {
	for i, line := range lines {
		b.WriteString(f.itemPrefix(pi, i) + line + f.itemSuffix(i, len(lines)) + "\n")
	}
}

func leadingCommaPrefix(pi string) string {
	n := max(len(pi)-2, 0)

	return strings.Repeat(" ", n) + ", "
}

func (f *formatter) formatWith(b *strings.Builder, with *sqlast.With, depth int) {
	if with == nil {
		return
	}

	p := f.pad(depth)
	pi := f.pad(depth + 1)

	keyword := "WITH"
	if with.Recursive {
		keyword = "WITH RECURSIVE"
	}

	b.WriteString(p)
	b.WriteString(keyword)
	b.WriteString("\n")

	n := len(with.CTEs)

	for i, cte := range with.CTEs {
		name := cte.ID.String()

		if len(cte.Columns) > 0 {
			name += " (" + cte.Columns.String() + ")"
		}

		b.WriteString(f.itemPrefix(pi, i))
		b.WriteString(name)
		b.WriteString(" ")
		b.WriteString(f.keyword("AS"))
		b.WriteString(" (\n")
		f.formatStatement(b, cte.Subquery, depth+2)
		b.WriteString(pi)
		b.WriteString(")")
		b.WriteString(f.itemSuffix(i, n))
		b.WriteString("\n")
	}
}

func (f *formatter) formatSelect(b *strings.Builder, s *sqlast.Select, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	b.WriteString(p)
	b.WriteString("SELECT")

	if s.Distinct {
		b.WriteString(" DISTINCT")
	}

	b.WriteString("\n")
	f.formatSelectExprs(b, s.SelectExprs, pi, depth)

	if len(s.From) > 0 {
		b.WriteString(p)
		b.WriteString("FROM\n")
		f.formatTableExprs(b, s.From, pi, depth)
	}

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatGroupBy(b, s.GroupBy, p, pi, depth)

	if s.Having != nil {
		b.WriteString(p)
		b.WriteString("HAVING\n")
		f.formatWhere(b, s.Having.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}

	formatLock(b, s.Lock, s.LockWait, p)
}

func (f *formatter) formatSelectExprs(b *strings.Builder, exprs []sqlast.SelectExpr, pi string, depth int) {
	lines := make([]string, len(exprs))
	for i, expr := range exprs {
		lines[i] = f.formatSelectExpr(expr, depth)
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatSelectExpr(expr sqlast.SelectExpr, depth int) string {
	switch e := expr.(type) {
	case *sqlast.AliasedExpr:
		s := f.formatExpr(e.Expr, depth+1)
		if !e.As.IsEmpty() {
			s += " " + f.keyword("AS") + " " + e.As.String()
		}

		return s
	case *sqlast.StarExpr:
		return e.String()
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled select expr type %T", expr))
	}
}

// formatTableExprs renders a comma-separated table reference list (a FROM
// clause, an UPDATE/DELETE target list, or a parenthesized table list). Each
// element is rendered into its own builder first so the configured comma
// style's prefix/suffix can be spliced onto its first/last line, without
// threading "is this the first/last element" through formatTableExpr's
// JOIN/paren recursion.
func (f *formatter) formatTableExprs(b *strings.Builder, exprs []sqlast.TableExpr, pi string, depth int) {
	n := len(exprs)

	for i, expr := range exprs {
		var item strings.Builder

		f.formatTableExpr(&item, expr, pi, depth)

		lines := strings.Split(strings.TrimSuffix(item.String(), "\n"), "\n")
		lines[0] = f.itemPrefix(pi, i) + strings.TrimPrefix(lines[0], pi)
		lines[len(lines)-1] += f.itemSuffix(i, n)

		for _, line := range lines {
			b.WriteString(line + "\n")
		}
	}
}

func (f *formatter) formatTableExpr(b *strings.Builder, expr sqlast.TableExpr, pi string, depth int) {
	switch e := expr.(type) {
	case *sqlast.AliasedTableExpr:
		f.formatAliasedTableExpr(b, e, pi, depth)
	case *sqlast.JoinTableExpr:
		f.formatJoinTableExpr(b, e, pi, depth)
	case *sqlast.ParenTableExpr:
		b.WriteString(pi)
		b.WriteString("(\n")
		f.formatTableExprs(b, e.Exprs, f.pad(depth+2), depth+1)
		b.WriteString(pi)
		b.WriteString(")\n")
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled table expr type %T", expr))
	}
}

func (f *formatter) formatAliasedTableExpr(b *strings.Builder, e *sqlast.AliasedTableExpr, pi string, depth int) {
	if sub, ok := e.Expr.(*sqlast.DerivedTable); ok {
		b.WriteString(pi)
		b.WriteString("(\n")
		f.formatStatement(b, sub.Select, depth+1)
		b.WriteString(pi)
		b.WriteString(")")
	} else {
		b.WriteString(pi)
		b.WriteString(e.Expr.String())
	}

	if !e.As.IsEmpty() {
		b.WriteString(" ")
		b.WriteString(e.As.String())
	}

	b.WriteString(formatIndexHints(e.Hints))
	b.WriteString("\n")
}

func formatIndexHints(hints sqlast.IndexHints) string {
	if len(hints) == 0 {
		return ""
	}

	parts := make([]string, len(hints))
	for i, hint := range hints {
		parts[i] = hint.String()
	}

	return " " + strings.Join(parts, " ")
}

func (f *formatter) formatJoinTableExpr(b *strings.Builder, e *sqlast.JoinTableExpr, pi string, depth int) {
	f.formatTableExpr(b, e.LeftExpr, pi, depth)
	joinStr := strings.ToUpper(e.Join.ToString())

	b.WriteString(pi)
	b.WriteString(joinStr)
	b.WriteString("\n")
	f.formatTableExpr(b, e.RightExpr, pi, depth)

	if e.Condition != nil && e.Condition.On != nil {
		b.WriteString(f.pad(depth + 2))
		b.WriteString(f.keyword("ON"))
		b.WriteString(" ")
		b.WriteString(f.formatExpr(e.Condition.On, depth))
		b.WriteString("\n")
	}
}

func (f *formatter) formatWhere(b *strings.Builder, expr sqlast.Expr, pi string, depth int) {
	f.formatWhereExpr(b, expr, pi, depth, true)
}

func (f *formatter) formatWhereExpr(b *strings.Builder, expr sqlast.Expr, pi string, depth int, first bool) {
	exprDepth := depth + 1

	switch e := expr.(type) {
	case *sqlast.AndExpr:
		f.formatWhereExpr(b, e.Left, pi, depth, first)
		f.formatWhereExpr(b, e.Right, pi, depth, false)
	case *sqlast.OrExpr:
		f.formatWhereExpr(b, e.Left, pi, depth, first)
		b.WriteString(pi)
		b.WriteString(f.keyword("OR"))
		b.WriteString(" ")
		b.WriteString(f.formatExpr(e.Right, exprDepth))
		b.WriteString("\n")
	default:
		if first {
			b.WriteString(pi)
			b.WriteString(f.formatExpr(expr, exprDepth))
			b.WriteString("\n")
		} else {
			b.WriteString(pi)
			b.WriteString(f.keyword("AND"))
			b.WriteString(" ")
			b.WriteString(f.formatExpr(expr, exprDepth))
			b.WriteString("\n")
		}
	}
}

func (f *formatter) formatExpr(expr sqlast.Expr, depth int) string {
	switch e := expr.(type) {
	case *sqlast.ExistsExpr:
		var b strings.Builder

		b.WriteString(f.keyword("EXISTS"))
		b.WriteString(" (\n")
		f.formatStatement(&b, e.Subquery.Select, depth+1)
		b.WriteString(f.pad(depth))
		b.WriteString(")")

		return b.String()
	case *sqlast.Subquery:
		var b strings.Builder

		b.WriteString("(\n")
		f.formatStatement(&b, e.Select, depth+1)
		b.WriteString(f.pad(depth))
		b.WriteString(")")

		return b.String()
	case *sqlast.ComparisonExpr:
		right := f.formatExpr(e.Right, depth)

		return f.formatExpr(e.Left, depth) + " " + f.keyword(e.Operator.ToString()) + " " + right
	case *sqlast.NotExpr:
		return f.keyword("NOT") + " " + f.formatExpr(e.Expr, depth)
	case *sqlast.CaseExpr:
		return f.formatCaseExpr(e, depth)
	default:
		if accessor := getOverAccessor(expr); accessor != nil {
			if oc := accessor.getOverClause(); oc != nil {
				return f.formatExprWithOver(expr, accessor, depth)
			}
		}

		return f.applyKeywordCase(expr.String())
	}
}

func (f *formatter) formatCaseExpr(e *sqlast.CaseExpr, depth int) string {
	var b strings.Builder

	pi := f.pad(depth + 1)
	p := f.pad(depth)

	b.WriteString("CASE")

	if e.Expr != nil {
		b.WriteString(" ")
		b.WriteString(f.formatExpr(e.Expr, depth))
	}

	b.WriteString("\n")

	for _, when := range e.Whens {
		cond := f.formatExpr(when.Cond, depth+1)
		val := f.formatExpr(when.Val, depth+1)

		b.WriteString(pi)
		b.WriteString("WHEN ")
		b.WriteString(cond)
		b.WriteString(" THEN ")
		b.WriteString(val)
		b.WriteString("\n")
	}

	if e.Else != nil {
		b.WriteString(pi)
		b.WriteString("ELSE ")
		b.WriteString(f.formatExpr(e.Else, depth+1))
		b.WriteString("\n")
	}

	b.WriteString(p)
	b.WriteString("END")

	return b.String()
}

type overClauseAccessor interface {
	getOverClause() *sqlast.OverClause
	setOverClause(oc *sqlast.OverClause)
}

type overClauseField struct {
	field **sqlast.OverClause
}

func (o overClauseField) getOverClause() *sqlast.OverClause   { return *o.field }
func (o overClauseField) setOverClause(oc *sqlast.OverClause) { *o.field = oc }

// Each case is trivially the same — complexity comes from the number of AST types, not logic.
func getOverAccessor(expr sqlast.Expr) overClauseAccessor { //nolint:cyclop,funlen,ireturn
	switch e := expr.(type) {
	case *sqlast.Count:
		return overClauseField{&e.OverClause}
	case *sqlast.CountStar:
		return overClauseField{&e.OverClause}
	case *sqlast.Sum:
		return overClauseField{&e.OverClause}
	case *sqlast.Avg:
		return overClauseField{&e.OverClause}
	case *sqlast.Min:
		return overClauseField{&e.OverClause}
	case *sqlast.Max:
		return overClauseField{&e.OverClause}
	case *sqlast.BitAnd:
		return overClauseField{&e.OverClause}
	case *sqlast.BitOr:
		return overClauseField{&e.OverClause}
	case *sqlast.BitXor:
		return overClauseField{&e.OverClause}
	case *sqlast.Std:
		return overClauseField{&e.OverClause}
	case *sqlast.StdDev:
		return overClauseField{&e.OverClause}
	case *sqlast.StdPop:
		return overClauseField{&e.OverClause}
	case *sqlast.StdSamp:
		return overClauseField{&e.OverClause}
	case *sqlast.VarPop:
		return overClauseField{&e.OverClause}
	case *sqlast.VarSamp:
		return overClauseField{&e.OverClause}
	case *sqlast.Variance:
		return overClauseField{&e.OverClause}
	case *sqlast.ArgumentLessWindowExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.FirstOrLastValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.NtileExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.NTHValueExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.LagLeadExpr:
		return overClauseField{&e.OverClause}
	case *sqlast.JSONArrayAgg:
		return overClauseField{&e.OverClause}
	case *sqlast.JSONObjectAgg:
		return overClauseField{&e.OverClause}
	default:
		return nil
	}
}

func (f *formatter) formatExprWithOver(expr sqlast.Expr, accessor overClauseAccessor, depth int) string {
	oc := accessor.getOverClause()

	// Temporarily remove the OverClause to get the base function string
	accessor.setOverClause(nil)

	base := f.applyKeywordCase(expr.String())

	accessor.setOverClause(oc)

	return base + " " + f.formatOverClause(oc, depth)
}

func (f *formatter) formatOverClause(oc *sqlast.OverClause, depth int) string {
	if !oc.WindowName.IsEmpty() {
		return "OVER " + oc.WindowName.String()
	}

	parts := f.formatWindowSpecParts(oc.WindowSpec, depth)
	if len(parts) == 0 {
		return "OVER ()"
	}

	pi := f.pad(depth + 1)
	p := f.pad(depth)

	var b strings.Builder

	b.WriteString("OVER (\n")

	for _, part := range parts {
		b.WriteString(pi)
		b.WriteString(part)
		b.WriteString("\n")
	}

	b.WriteString(p)
	b.WriteString(")")

	return b.String()
}

func (f *formatter) formatWindowSpecParts(spec *sqlast.WindowSpecification, depth int) []string {
	var parts []string

	if len(spec.PartitionClause) > 0 {
		exprs := make([]string, len(spec.PartitionClause))
		for i, e := range spec.PartitionClause {
			exprs[i] = f.applyKeywordCase(e.String())
		}

		parts = append(parts, "PARTITION BY "+strings.Join(exprs, ", "))
	}

	if len(spec.OrderClause) > 0 {
		parts = append(parts, "ORDER BY "+f.formatOrderExprs(spec.OrderClause, depth))
	}

	if spec.FrameClause != nil {
		parts = append(parts, spec.FrameClause.String())
	}

	return parts
}

// formatOrderExprs renders ORDER BY items as a single comma-joined line, used
// for a window spec's ORDER BY (unlike the top-level ORDER BY clause, which
// formatOrderBy renders one item per line via writeList).
func (f *formatter) formatOrderExprs(orders sqlast.OrderBy, depth int) string {
	strs := make([]string, len(orders))

	for i, o := range orders {
		dir := ""
		if o.Direction == sqlast.DescOrder {
			dir = f.descSuffix()
		}

		strs[i] = f.formatExpr(o.Expr, depth) + dir
	}

	return strings.Join(strs, ", ")
}

func (f *formatter) descSuffix() string {
	return " " + f.keyword("DESC")
}

func (f *formatter) formatInsert(b *strings.Builder, s *sqlast.Insert, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	action := "INSERT"
	if s.Action == sqlast.ReplaceAct {
		action = "REPLACE"
	}

	if s.Ignore {
		action += " IGNORE"
	}

	b.WriteString(p)
	b.WriteString(action)
	b.WriteString(" INTO\n")
	b.WriteString(pi)
	b.WriteString(s.Table.String())
	b.WriteString("\n")

	f.formatInsertColumns(b, s.Columns, p, pi)
	f.formatInsertRows(b, s.Rows, p, pi, depth)
	f.formatOnDupUpdate(b, s.OnDup, p, pi)
}

func (f *formatter) formatInsertColumns(b *strings.Builder, cols sqlast.Columns, p, pi string) {
	if len(cols) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("(\n")

	lines := make([]string, len(cols))
	for i, col := range cols {
		lines[i] = col.String()
	}

	f.writeList(b, pi, lines)

	b.WriteString(p)
	b.WriteString(")\n")
}

func (f *formatter) formatInsertRows(b *strings.Builder, rows sqlast.InsertRows, p, pi string, depth int) {
	switch r := rows.(type) {
	case sqlast.Values:
		f.formatValuesRows(b, r, p, pi, depth)
	case *sqlast.Select:
		f.formatSelect(b, r, depth)
	case *sqlast.Union:
		f.formatUnion(b, r, depth)
	case sqlast.SetExprs:
		f.formatSetExprs(b, r, p, pi)
	default:
		panic(fmt.Sprintf("sqlfmt: unhandled insert rows type %T", rows))
	}
}

func (f *formatter) formatValuesRows(b *strings.Builder, rows sqlast.Values, p, pi string, depth int) {
	b.WriteString(p)
	b.WriteString("VALUES\n")

	lines := make([]string, len(rows))

	for i, row := range rows {
		vals := make([]string, len(row))
		for j, v := range row {
			vals[j] = f.formatExpr(v, depth)
		}

		lines[i] = "(" + strings.Join(vals, ", ") + ")"
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatSetExprs(b *strings.Builder, exprs sqlast.SetExprs, p, pi string) {
	b.WriteString(p)
	b.WriteString("SET\n")

	lines := make([]string, len(exprs))
	for i, expr := range exprs {
		lines[i] = f.applyKeywordCase(expr.String())
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatOnDupUpdate(b *strings.Builder, onDup sqlast.OnDup, p, pi string) {
	if len(onDup) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("ON DUPLICATE KEY UPDATE\n")

	lines := make([]string, len(onDup))
	for i, expr := range onDup {
		lines[i] = f.applyKeywordCase(expr.String())
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatUpdate(b *strings.Builder, s *sqlast.Update, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	action := "UPDATE"
	if s.Ignore {
		action = "UPDATE IGNORE"
	}

	b.WriteString(p)
	b.WriteString(action)
	b.WriteString("\n")
	f.formatTableExprs(b, s.TableExprs, pi, depth)

	b.WriteString(p)
	b.WriteString("SET\n")

	lines := make([]string, len(s.Exprs))
	for i, expr := range s.Exprs {
		lines[i] = f.applyKeywordCase(expr.String())
	}

	f.writeList(b, pi, lines)

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}
}

func (f *formatter) formatDelete(b *strings.Builder, s *sqlast.Delete, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	f.formatWith(b, s.With, depth)

	action := "DELETE"
	if s.Ignore {
		action = "DELETE IGNORE"
	}

	if len(s.Targets) > 0 {
		b.WriteString(p)
		b.WriteString(action)
		b.WriteString("\n")

		lines := make([]string, len(s.Targets))
		for i, target := range s.Targets {
			lines[i] = target.String()
		}

		f.writeList(b, pi, lines)

		b.WriteString(p)
		b.WriteString("FROM\n")
	} else {
		b.WriteString(p)
		b.WriteString(action)
		b.WriteString(" FROM\n")
	}

	f.formatTableExprs(b, s.TableExprs, pi, depth)

	if s.Where != nil {
		b.WriteString(p)
		b.WriteString("WHERE\n")
		f.formatWhere(b, s.Where.Expr, pi, depth)
	}

	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}
}

func (f *formatter) formatUnion(b *strings.Builder, s *sqlast.Union, depth int) {
	p := f.pad(depth)

	f.formatWith(b, s.With, depth)

	f.formatStatement(b, s.Left, depth)

	op := "UNION"
	if !s.Distinct {
		op = "UNION ALL"
	}

	b.WriteString(p)
	b.WriteString(op)
	b.WriteString("\n")
	f.formatStatement(b, s.Right, depth)

	pi := f.pad(depth + 1)
	f.formatOrderBy(b, s.OrderBy, p, pi, depth)

	if s.Limit != nil {
		f.formatLimit(b, s.Limit, p, depth)
	}

	formatLock(b, s.Lock, s.LockWait, p)
}

func formatLock(b *strings.Builder, lock sqlast.Lock, wait sqlast.LockWaitType, p string) {
	if lock == sqlast.NoLock {
		return
	}

	lockStr := lock.String()
	if w := wait.String(); w != "" {
		lockStr += " " + w
	}

	b.WriteString(p)
	b.WriteString(lockStr)
	b.WriteString("\n")
}

func (f *formatter) formatGroupBy(b *strings.Builder, groupBy *sqlast.GroupBy, p, pi string, depth int) {
	if groupBy == nil || len(groupBy.Exprs) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("GROUP BY\n")

	lines := make([]string, len(groupBy.Exprs))
	for i, expr := range groupBy.Exprs {
		lines[i] = f.formatExpr(expr, depth)
	}

	f.writeList(b, pi, lines)

	if groupBy.WithRollup {
		b.WriteString(p)
		b.WriteString("WITH ROLLUP\n")
	}
}

func (f *formatter) formatOrderBy(b *strings.Builder, orders sqlast.OrderBy, p, pi string, depth int) {
	if len(orders) == 0 {
		return
	}

	b.WriteString(p)
	b.WriteString("ORDER BY\n")

	lines := make([]string, len(orders))

	for i, order := range orders {
		dir := ""
		if order.Direction == sqlast.DescOrder {
			dir = f.descSuffix()
		}

		lines[i] = f.formatExpr(order.Expr, depth) + dir
	}

	f.writeList(b, pi, lines)
}

func (f *formatter) formatLimit(b *strings.Builder, limit *sqlast.Limit, p string, depth int) {
	pi := f.pad(depth + 1)

	if limit.Offset != nil {
		b.WriteString(p)
		b.WriteString("LIMIT\n")
		b.WriteString(pi)
		b.WriteString(f.formatExpr(limit.Rowcount, depth))
		b.WriteString("\n")
		b.WriteString(p)
		b.WriteString("OFFSET\n")
		b.WriteString(pi)
		b.WriteString(f.formatExpr(limit.Offset, depth))
		b.WriteString("\n")
	} else {
		b.WriteString(p)
		b.WriteString("LIMIT\n")
		b.WriteString(pi)
		b.WriteString(f.formatExpr(limit.Rowcount, depth))
		b.WriteString("\n")
	}
}

// formatCreateTable renders a CREATE TABLE statement, exploding its column
// list one element per line like every other bracketed list in this
// formatter (see formatInsertColumns), with trailing options on the closing
// paren's line.
func (f *formatter) formatCreateTable(b *strings.Builder, s *sqlast.CreateTable, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	b.WriteString(p)
	b.WriteString("CREATE TABLE ")

	if s.IfNotExists {
		b.WriteString("IF NOT EXISTS ")
	}

	b.WriteString(s.Table.String())
	b.WriteString(" (\n")

	// Elements aren't passed through applyKeywordCase: a column/constraint's
	// rendered text is dominated by clause keywords (NOT NULL, ON UPDATE,
	// PRIMARY KEY, ...) that must stay uppercase regardless of keyword_case,
	// and the regex-based lowering can't tell those apart from a genuine
	// operator/predicate keyword occurring at the same position.
	lines := make([]string, len(s.Elements))
	for i, e := range s.Elements {
		lines[i] = e.String()
	}

	f.writeList(b, pi, lines)

	b.WriteString(p)
	b.WriteString(")")

	for _, opt := range s.Options {
		b.WriteString(" ")
		b.WriteString(opt.String())
	}

	b.WriteString("\n")
}

// formatAlterTable renders an ALTER TABLE statement, exploding its action
// list one action per line even when there's only one, matching how this
// formatter always breaks out UPDATE's SET list (see formatSetExprs).
func (f *formatter) formatAlterTable(b *strings.Builder, s *sqlast.AlterTable, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	b.WriteString(p)
	b.WriteString("ALTER TABLE ")
	b.WriteString(s.Table.String())
	b.WriteString("\n")

	// See formatCreateTable: actions aren't passed through applyKeywordCase
	// for the same reason (ADD COLUMN, ON DELETE/ON UPDATE, ... must stay
	// uppercase regardless of keyword_case).
	lines := make([]string, len(s.Actions))
	for i, a := range s.Actions {
		lines[i] = a.String()
	}

	f.writeList(b, pi, lines)
}

// formatCreateIndex renders a CREATE [UNIQUE] INDEX statement, exploding its
// column list like formatCreateTable does.
func (f *formatter) formatCreateIndex(b *strings.Builder, s *sqlast.CreateIndex, depth int) {
	p := f.pad(depth)
	pi := f.pad(depth + 1)

	b.WriteString(p)
	b.WriteString("CREATE ")

	if s.Unique {
		b.WriteString("UNIQUE ")
	}

	b.WriteString("INDEX ")
	b.WriteString(s.Name.String())
	b.WriteString(" ON ")
	b.WriteString(s.Table.String())
	b.WriteString(" (\n")

	// Unlike formatCreateTable/formatAlterTable, applyKeywordCase is safe
	// here: an IndexColumn only ever renders a column name, an optional
	// prefix length, and an optional DESC — DESC is a genuine
	// operator/predicate keyword (matching ORDER BY's DESC), and there's no
	// NOT/ON text in this position to false-positive on.
	lines := make([]string, len(s.Columns))
	for i, c := range s.Columns {
		lines[i] = f.applyKeywordCase(c.String())
	}

	f.writeList(b, pi, lines)

	b.WriteString(p)
	b.WriteString(")\n")
}

func (f *formatter) formatDropIndex(b *strings.Builder, s *sqlast.DropIndex, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("DROP INDEX ")
	b.WriteString(s.Name.String())
	b.WriteString(" ON ")
	b.WriteString(s.Table.String())
	b.WriteString("\n")
}

func (f *formatter) formatDropTable(b *strings.Builder, s *sqlast.DropTable, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("DROP TABLE ")

	if s.IfExists {
		b.WriteString("IF EXISTS ")
	}

	names := make([]string, len(s.Tables))
	for i, t := range s.Tables {
		names[i] = t.String()
	}

	b.WriteString(strings.Join(names, ", "))
	b.WriteString("\n")
}

func (f *formatter) formatTruncateTable(b *strings.Builder, s *sqlast.TruncateTable, depth int) {
	b.WriteString(f.pad(depth))
	b.WriteString("TRUNCATE TABLE ")
	b.WriteString(s.Table.String())
	b.WriteString("\n")
}
