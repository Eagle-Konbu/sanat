package sqlast

// This file gathers the marker methods that seal the Expr, Statement,
// TableExpr, SelectExpr, SimpleTableExpr, and InsertRows interfaces to
// types defined in this package. Each method body is intentionally empty,
// so grouping them here keeps the exclusion in codecov.yml scoped to this
// file instead of hiding real gaps in the String() implementations they
// sit next to in expr.go, statement.go, table.go, node.go, and clause.go.

// --- Expr ---

func (*ComparisonExpr) iExpr()         {}
func (*RangeCond) iExpr()              {}
func (*IsExpr) iExpr()                 {}
func (ValTuple) iExpr()                {}
func (*ArithmeticExpr) iExpr()         {}
func (*UnaryExpr) iExpr()              {}
func (*AndExpr) iExpr()                {}
func (*OrExpr) iExpr()                 {}
func (*NotExpr) iExpr()                {}
func (*CaseExpr) iExpr()               {}
func (*ExistsExpr) iExpr()             {}
func (*Subquery) iExpr()               {}
func (*ColName) iExpr()                {}
func (*Literal) iExpr()                {}
func (*FuncExpr) iExpr()               {}
func (*ParenExpr) iExpr()              {}
func (*Count) iExpr()                  {}
func (*CountStar) iExpr()              {}
func (*Sum) iExpr()                    {}
func (*Avg) iExpr()                    {}
func (*Min) iExpr()                    {}
func (*Max) iExpr()                    {}
func (*BitAnd) iExpr()                 {}
func (*BitOr) iExpr()                  {}
func (*BitXor) iExpr()                 {}
func (*Std) iExpr()                    {}
func (*StdDev) iExpr()                 {}
func (*StdPop) iExpr()                 {}
func (*StdSamp) iExpr()                {}
func (*Variance) iExpr()               {}
func (*VarPop) iExpr()                 {}
func (*VarSamp) iExpr()                {}
func (*ArgumentLessWindowExpr) iExpr() {}
func (*FirstOrLastValueExpr) iExpr()   {}
func (*NtileExpr) iExpr()              {}
func (*NTHValueExpr) iExpr()           {}
func (*LagLeadExpr) iExpr()            {}
func (*JSONArrayAgg) iExpr()           {}
func (*JSONObjectAgg) iExpr()          {}

// --- Statement ---

func (*Select) iStatement() {}
func (*Insert) iStatement() {}
func (*Update) iStatement() {}
func (*Delete) iStatement() {}
func (*Union) iStatement()  {}

// --- InsertRows ---

func (*Select) iInsertRows()  {}
func (*Union) iInsertRows()   {}
func (Values) iInsertRows()   {}
func (SetExprs) iInsertRows() {}

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
