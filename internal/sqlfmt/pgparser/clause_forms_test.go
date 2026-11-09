package pgparser_test

import "testing"

func TestParseSelect_frameExclusionAndSortOperator(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"exclude current row",
			"SELECT sum(a) OVER (ORDER BY b ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING EXCLUDE CURRENT ROW) FROM t",
			"SELECT sum(a) OVER (ORDER BY b ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING EXCLUDE CURRENT ROW) FROM t",
		},
		{"exclude group", "SELECT sum(a) OVER (GROUPS UNBOUNDED PRECEDING exclude group) FROM t", "SELECT sum(a) OVER (GROUPS UNBOUNDED PRECEDING EXCLUDE GROUP) FROM t"},
		{"exclude ties", "SELECT sum(a) OVER w FROM t WINDOW w AS (RANGE CURRENT ROW EXCLUDE TIES)", "SELECT sum(a) OVER w FROM t WINDOW w AS (RANGE CURRENT ROW EXCLUDE TIES)"},
		{"exclude no others", "SELECT sum(a) OVER (ROWS CURRENT ROW EXCLUDE NO OTHERS) FROM t", "SELECT sum(a) OVER (ROWS CURRENT ROW EXCLUDE NO OTHERS) FROM t"},
		{"order by using", "SELECT a FROM t ORDER BY a USING >", "SELECT a FROM t ORDER BY a USING >"},
		{"order by using with nulls", "SELECT a FROM t ORDER BY a USING ~<~ NULLS FIRST, b DESC", "SELECT a FROM t ORDER BY a USING ~<~ NULLS FIRST, b DESC"},
		{"order by using explicit operator", "SELECT a FROM t ORDER BY a USING OPERATOR(pg_catalog.<)", "SELECT a FROM t ORDER BY a USING OPERATOR(pg_catalog.<)"},
		{"aggregate order by using", "SELECT array_agg(a ORDER BY a USING <) FROM t", "SELECT array_agg(a ORDER BY a USING <) FROM t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSelectRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{
		"SELECT sum(a) OVER (ROWS CURRENT ROW EXCLUDE) FROM t",
		"SELECT sum(a) OVER (ROWS CURRENT ROW EXCLUDE NO) FROM t",
		"SELECT a FROM t ORDER BY a USING",
		"SELECT a FROM t ORDER BY a USING x",
		"SELECT a FROM t ORDER BY a USING < DESC",
	} {
		t.Run("error "+in, func(t *testing.T) {
			assertSelectError(t, in)
		})
	}
}
