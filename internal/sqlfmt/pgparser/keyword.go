package pgparser

import "slices"

// keywordCategory mirrors PostgreSQL's keyword classes
// (src/include/parser/kwlist.h), which decide where a keyword may double as a
// name. Matching them, rather than treating every keyword as reserved, lets
// common column names such as key, first, or current parse unquoted.
type keywordCategory int8

const (
	// unreservedKeyword may be used as any name.
	unreservedKeyword keywordCategory = iota
	// colNameKeyword may name a column or table, but not a function or type.
	colNameKeyword
	// typeFuncNameKeyword may name a function or type, but not a column or
	// table.
	typeFuncNameKeyword
	// reservedKeyword is a name only where any keyword is (ColLabel).
	reservedKeyword
)

// keywordClass is a keyword's category, plus whether it may be used as a
// select-list alias without AS (PostgreSQL's BARE_LABEL).
type keywordClass struct {
	category  keywordCategory
	bareLabel bool
}

// keywordClasses holds every keyword's class, copied from PostgreSQL 18's
// kwlist.h. buildKeywordTable panics on a keyword missing from it, so a new
// keyword cannot skip classification.
var keywordClasses = map[TokenType]keywordClass{
	SELECT:       {reservedKeyword, true},
	FROM:         {reservedKeyword, false},
	WHERE:        {reservedKeyword, false},
	AS:           {reservedKeyword, false},
	AND:          {reservedKeyword, true},
	OR:           {reservedKeyword, true},
	NOT:          {reservedKeyword, true},
	IN:           {reservedKeyword, true},
	BETWEEN:      {colNameKeyword, true},
	LIKE:         {typeFuncNameKeyword, true},
	ILIKE:        {typeFuncNameKeyword, true},
	ESCAPE:       {unreservedKeyword, true},
	IS:           {typeFuncNameKeyword, true},
	NULL:         {reservedKeyword, true},
	TRUE:         {reservedKeyword, true},
	FALSE:        {reservedKeyword, true},
	EXISTS:       {colNameKeyword, true},
	CASE:         {reservedKeyword, true},
	WHEN:         {reservedKeyword, true},
	THEN:         {reservedKeyword, true},
	ELSE:         {reservedKeyword, true},
	END:          {reservedKeyword, true},
	DISTINCT:     {reservedKeyword, true},
	ON:           {reservedKeyword, false},
	ALL:          {reservedKeyword, true},
	CAST:         {reservedKeyword, true},
	ARRAY:        {reservedKeyword, false},
	FILTER:       {unreservedKeyword, false},
	ISNULL:       {typeFuncNameKeyword, false},
	NOTNULL:      {typeFuncNameKeyword, false},
	ORDER:        {reservedKeyword, false},
	BY:           {unreservedKeyword, true},
	GROUP:        {reservedKeyword, false},
	HAVING:       {reservedKeyword, false},
	WINDOW:       {reservedKeyword, false},
	ASC:          {reservedKeyword, true},
	DESC:         {reservedKeyword, true},
	NULLS:        {unreservedKeyword, true},
	FIRST:        {unreservedKeyword, true},
	LAST:         {unreservedKeyword, true},
	LIMIT:        {reservedKeyword, false},
	OFFSET:       {reservedKeyword, false},
	FETCH:        {reservedKeyword, false},
	NEXT:         {unreservedKeyword, true},
	ROW:          {colNameKeyword, true},
	ROWS:         {unreservedKeyword, true},
	GROUPS:       {unreservedKeyword, true},
	ONLY:         {reservedKeyword, true},
	WITH:         {reservedKeyword, false},
	TIES:         {unreservedKeyword, true},
	FOR:          {reservedKeyword, false},
	UPDATE:       {unreservedKeyword, true},
	NO:           {unreservedKeyword, true},
	KEY:          {unreservedKeyword, true},
	SHARE:        {unreservedKeyword, true},
	OF:           {unreservedKeyword, true},
	NOWAIT:       {unreservedKeyword, true},
	SKIP:         {unreservedKeyword, true},
	LOCKED:       {unreservedKeyword, true},
	JOIN:         {typeFuncNameKeyword, true},
	INNER:        {typeFuncNameKeyword, true},
	LEFT:         {typeFuncNameKeyword, true},
	RIGHT:        {typeFuncNameKeyword, true},
	FULL:         {typeFuncNameKeyword, true},
	OUTER:        {typeFuncNameKeyword, true},
	CROSS:        {typeFuncNameKeyword, true},
	NATURAL:      {typeFuncNameKeyword, true},
	LATERAL:      {reservedKeyword, true},
	USING:        {reservedKeyword, true},
	GROUPING:     {colNameKeyword, true},
	SETS:         {unreservedKeyword, true},
	CUBE:         {unreservedKeyword, true},
	ROLLUP:       {unreservedKeyword, true},
	OVER:         {unreservedKeyword, false},
	PARTITION:    {unreservedKeyword, true},
	RANGE:        {unreservedKeyword, true},
	UNBOUNDED:    {unreservedKeyword, true},
	PRECEDING:    {unreservedKeyword, true},
	FOLLOWING:    {unreservedKeyword, true},
	CURRENT:      {unreservedKeyword, true},
	UNION:        {reservedKeyword, false},
	INTERSECT:    {reservedKeyword, false},
	EXCEPT:       {reservedKeyword, false},
	INSERT:       {unreservedKeyword, true},
	INTO:         {reservedKeyword, false},
	VALUES:       {colNameKeyword, true},
	DEFAULT:      {reservedKeyword, true},
	CONFLICT:     {unreservedKeyword, true},
	DO:           {reservedKeyword, true},
	NOTHING:      {unreservedKeyword, true},
	SET:          {unreservedKeyword, true},
	DELETE:       {unreservedKeyword, true},
	RETURNING:    {reservedKeyword, false},
	RECURSIVE:    {unreservedKeyword, true},
	MATERIALIZED: {unreservedKeyword, true},
}

// isColID reports whether tok can be a ColId: a column, table, alias, CTE,
// or window name.
func isColID(tok Token) bool {
	return isNameToken(tok, unreservedKeyword, colNameKeyword)
}

// isTypeFuncName reports whether tok can be a type_function_name: an
// unqualified function or type name.
func isTypeFuncName(tok Token) bool {
	return isNameToken(tok, unreservedKeyword, typeFuncNameKeyword)
}

// isColLabel reports whether tok can be a ColLabel: a select-list alias after
// AS, or a name after a dot. Every keyword qualifies, since neither position
// can be confused with a following clause.
func isColLabel(tok Token) bool {
	return tok.Type == IDENT || tok.Type == QuotedIdent || tok.Type.IsKeyword()
}

// isBareColLabel reports whether tok can be a BareColLabel: a select-list
// alias written without AS.
func isBareColLabel(tok Token) bool {
	if tok.Type == IDENT || tok.Type == QuotedIdent {
		return true
	}

	class, ok := keywordClasses[tok.Type]

	return ok && class.bareLabel
}

func isNameToken(tok Token, categories ...keywordCategory) bool {
	if tok.Type == IDENT || tok.Type == QuotedIdent {
		return true
	}

	class, ok := keywordClasses[tok.Type]

	return ok && slices.Contains(categories, class.category)
}
