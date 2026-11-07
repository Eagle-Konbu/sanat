package pgparser_test

import "testing"

func TestParseInsert_onConflictTargets(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"on constraint do update",
			"INSERT INTO t VALUES (1) ON CONFLICT ON CONSTRAINT t_pkey DO UPDATE SET a = 1",
			"INSERT INTO t VALUES (1) ON CONFLICT ON CONSTRAINT t_pkey DO UPDATE SET a = 1",
		},
		{
			"on constraint do nothing",
			"INSERT INTO t VALUES (1) ON CONFLICT ON CONSTRAINT \"T_pkey\" DO NOTHING",
			"INSERT INTO t VALUES (1) ON CONFLICT ON CONSTRAINT \"T_pkey\" DO NOTHING",
		},
		{
			"expression target",
			"INSERT INTO t VALUES (1) ON CONFLICT (lower(email)) DO NOTHING",
			"INSERT INTO t VALUES (1) ON CONFLICT (lower(email)) DO NOTHING",
		},
		{
			"parenthesized expression target",
			"INSERT INTO t VALUES (1) ON CONFLICT ((a + b), c) DO NOTHING",
			"INSERT INTO t VALUES (1) ON CONFLICT ((a + b), c) DO NOTHING",
		},
		{
			"collation and operator class",
			"INSERT INTO t VALUES (1) ON CONFLICT (name COLLATE \"C\" text_pattern_ops) DO NOTHING",
			"INSERT INTO t VALUES (1) ON CONFLICT (name COLLATE \"C\" text_pattern_ops) DO NOTHING",
		},
		{
			"index predicate",
			"INSERT INTO t VALUES (1) ON CONFLICT (a) WHERE deleted_at IS NULL DO UPDATE SET b = 1 WHERE t.b < 1",
			"INSERT INTO t VALUES (1) ON CONFLICT (a) WHERE deleted_at IS NULL DO UPDATE SET b = 1 WHERE t.b < 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertInsertRoundTrip(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{
		"INSERT INTO t VALUES (1) ON CONFLICT ON CONSTRAINT DO NOTHING",
		"INSERT INTO t VALUES (1) ON CONFLICT () DO NOTHING",
		"INSERT INTO t VALUES (1) ON CONFLICT (a COLLATE) DO NOTHING",
		"INSERT INTO t VALUES (1) ON CONFLICT (a) WHERE DO NOTHING",
	} {
		t.Run("error "+in, func(t *testing.T) {
			assertInsertError(t, in)
		})
	}
}
