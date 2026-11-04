package pgast

// This file gathers the marker methods that seal the Statement, Expr,
// TableExpr, SelectExpr, SimpleTableExpr, and InsertRows interfaces to types
// defined in this package. Each method body is intentionally empty, so
// grouping them here keeps the exclusion in codecov.yml scoped to this file
// instead of hiding real gaps in the String() implementations they sit next
// to in expr.go, statement.go, table.go, and clause.go.

// --- Statement ---

func (*Select) iStatement() {}
func (*Insert) iStatement() {}
func (*Update) iStatement() {}
func (*Delete) iStatement() {}
func (*SetOp) iStatement()  {}
func (*Merge) iStatement()  {}

// --- InsertRows ---

func (*Select) iInsertRows()       {}
func (*SetOp) iInsertRows()        {}
func (Values) iInsertRows()        {}
func (DefaultValues) iInsertRows() {}

// --- Expr ---

func (*ComparisonExpr) iExpr()     {}
func (*ILikeExpr) iExpr()          {}
func (*RangeCond) iExpr()          {}
func (*IsExpr) iExpr()             {}
func (*IsDistinctFromExpr) iExpr() {}
func (ValTuple) iExpr()            {}
func (*ArithmeticExpr) iExpr()     {}
func (*UnaryExpr) iExpr()          {}
func (*AndExpr) iExpr()            {}
func (*OrExpr) iExpr()             {}
func (*NotExpr) iExpr()            {}
func (*CaseExpr) iExpr()           {}
func (*ExistsExpr) iExpr()         {}
func (*Subquery) iExpr()           {}
func (*ColName) iExpr()            {}
func (*Literal) iExpr()            {}
func (*FuncExpr) iExpr()           {}
func (*ParenExpr) iExpr()          {}
func (*CastExpr) iExpr()           {}
func (*ArrayExpr) iExpr()          {}
func (*ArraySubscriptExpr) iExpr() {}
func (*JSONOpExpr) iExpr()         {}
func (*GroupingSets) iExpr()       {}
func (*Cube) iExpr()               {}
func (*Rollup) iExpr()             {}

// --- TableExpr ---

func (*AliasedTableExpr) iTableExpr() {}
func (*JoinTableExpr) iTableExpr()    {}
func (*ParenTableExpr) iTableExpr()   {}

// --- SimpleTableExpr ---

func (*DerivedTable) iSimpleTableExpr() {}
func (TableName) iSimpleTableExpr()     {}

// --- SelectExpr ---

func (*AliasedExpr) iSelectExpr() {}
func (*StarExpr) iSelectExpr()    {}
