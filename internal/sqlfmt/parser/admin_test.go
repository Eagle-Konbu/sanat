package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

func TestParseShowTables(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "SHOW TABLES", "SHOW TABLES"},
		{"lowercase", "show tables", "SHOW TABLES"},
		{"from database", "SHOW TABLES FROM mydb", "SHOW TABLES FROM mydb"},
		{"like pattern", "SHOW TABLES LIKE 'user%'", "SHOW TABLES LIKE 'user%'"},
		{
			"from database and like pattern",
			"SHOW TABLES FROM mydb LIKE 'user%'",
			"SHOW TABLES FROM mydb LIKE 'user%'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := parser.ParseShowTables(tt.in)
			if err != nil {
				t.Fatalf("ParseShowTables(%q) error = %v", tt.in, err)
			}

			if got := st.String(); got != tt.want {
				t.Errorf("ParseShowTables(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseShowTables_errors(t *testing.T) {
	tests := []string{
		"SHOW",
		"SHOW TABLES FROM",
		"SHOW TABLES LIKE",
		"SHOW TABLES LIKE mydb",
		"SHOW TABLES extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowTables(in); err == nil {
			t.Errorf("ParseShowTables(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowCreateTable(t *testing.T) {
	s, err := parser.ParseShowCreateTable("SHOW CREATE TABLE users")
	if err != nil {
		t.Fatalf("ParseShowCreateTable(...) error = %v", err)
	}

	want := "SHOW CREATE TABLE users"
	if got := s.String(); got != want {
		t.Errorf("ParseShowCreateTable(...).String() = %q, want %q", got, want)
	}
}

func TestParseShowCreateTable_errors(t *testing.T) {
	tests := []string{
		"SHOW CREATE",
		"SHOW CREATE TABLE",
		"SHOW CREATE TABLE users extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowCreateTable(in); err == nil {
			t.Errorf("ParseShowCreateTable(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowColumns(t *testing.T) {
	s, err := parser.ParseShowColumns("SHOW COLUMNS FROM users")
	if err != nil {
		t.Fatalf("ParseShowColumns(...) error = %v", err)
	}

	want := "SHOW COLUMNS FROM users"
	if got := s.String(); got != want {
		t.Errorf("ParseShowColumns(...).String() = %q, want %q", got, want)
	}
}

func TestParseShowColumns_errors(t *testing.T) {
	tests := []string{
		"SHOW COLUMNS",
		"SHOW COLUMNS users",
		"SHOW COLUMNS FROM users extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowColumns(in); err == nil {
			t.Errorf("ParseShowColumns(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowIndex(t *testing.T) {
	s, err := parser.ParseShowIndex("SHOW INDEX FROM users")
	if err != nil {
		t.Fatalf("ParseShowIndex(...) error = %v", err)
	}

	want := "SHOW INDEX FROM users"
	if got := s.String(); got != want {
		t.Errorf("ParseShowIndex(...).String() = %q, want %q", got, want)
	}
}

func TestParseShowIndex_errors(t *testing.T) {
	tests := []string{
		"SHOW INDEX",
		"SHOW INDEX users",
		"SHOW INDEX FROM users extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowIndex(in); err == nil {
			t.Errorf("ParseShowIndex(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowDatabases(t *testing.T) {
	s, err := parser.ParseShowDatabases("SHOW DATABASES")
	if err != nil {
		t.Fatalf("ParseShowDatabases(...) error = %v", err)
	}

	want := "SHOW DATABASES"
	if got := s.String(); got != want {
		t.Errorf("ParseShowDatabases(...).String() = %q, want %q", got, want)
	}
}

func TestParseShowDatabases_errors(t *testing.T) {
	tests := []string{
		"SHOW DATABASES extra",
		"SHOW DATABASES LIKE 'a%'",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowDatabases(in); err == nil {
			t.Errorf("ParseShowDatabases(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowVariables(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "SHOW VARIABLES", "SHOW VARIABLES"},
		{"like pattern", "SHOW VARIABLES LIKE 'max_%'", "SHOW VARIABLES LIKE 'max_%'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := parser.ParseShowVariables(tt.in)
			if err != nil {
				t.Fatalf("ParseShowVariables(%q) error = %v", tt.in, err)
			}

			if got := s.String(); got != tt.want {
				t.Errorf("ParseShowVariables(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseShowVariables_errors(t *testing.T) {
	tests := []string{
		"SHOW VARIABLES extra",
		"SHOW VARIABLES LIKE",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowVariables(in); err == nil {
			t.Errorf("ParseShowVariables(%q) expected error, got nil", in)
		}
	}
}

func TestParseShowStatus(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "SHOW STATUS", "SHOW STATUS"},
		{"like pattern", "SHOW STATUS LIKE 'Threads%'", "SHOW STATUS LIKE 'Threads%'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := parser.ParseShowStatus(tt.in)
			if err != nil {
				t.Fatalf("ParseShowStatus(%q) error = %v", tt.in, err)
			}

			if got := s.String(); got != tt.want {
				t.Errorf("ParseShowStatus(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseShowStatus_errors(t *testing.T) {
	tests := []string{
		"SHOW STATUS extra",
		"SHOW STATUS LIKE",
	}

	for _, in := range tests {
		if _, err := parser.ParseShowStatus(in); err == nil {
			t.Errorf("ParseShowStatus(%q) expected error, got nil", in)
		}
	}
}

func TestParseShow_unrecognizedVariant(t *testing.T) {
	if _, err := parser.ParseShowTables("SHOW ENGINES"); err == nil {
		t.Error("ParseShowTables(\"SHOW ENGINES\") expected error, got nil")
	}
}

func TestParseDescribe(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"table only", "DESCRIBE users", "DESCRIBE users"},
		{"lowercase", "describe users", "DESCRIBE users"},
		{"with column", "DESCRIBE users id", "DESCRIBE users id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parser.ParseDescribe(tt.in)
			if err != nil {
				t.Fatalf("ParseDescribe(%q) error = %v", tt.in, err)
			}

			if got := d.String(); got != tt.want {
				t.Errorf("ParseDescribe(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseDescribe_errors(t *testing.T) {
	tests := []string{
		"DESCRIBE",
		"DESCRIBE users id extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseDescribe(in); err == nil {
			t.Errorf("ParseDescribe(%q) expected error, got nil", in)
		}
	}
}

func TestParseExplain(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain select", "EXPLAIN SELECT * FROM users", "EXPLAIN SELECT * FROM users"},
		{"lowercase", "explain select * from users", "EXPLAIN SELECT * FROM users"},
		{
			"traditional format",
			"EXPLAIN FORMAT = TRADITIONAL SELECT * FROM users",
			"EXPLAIN FORMAT = TRADITIONAL SELECT * FROM users",
		},
		{
			"json format lowercase",
			"explain format = json select * from users",
			"EXPLAIN FORMAT = JSON SELECT * FROM users",
		},
		{
			"tree format",
			"EXPLAIN FORMAT = TREE SELECT * FROM users",
			"EXPLAIN FORMAT = TREE SELECT * FROM users",
		},
		{
			"union",
			"EXPLAIN SELECT 1 UNION SELECT 2",
			"EXPLAIN SELECT 1 UNION SELECT 2",
		},
		{
			"with clause",
			"EXPLAIN WITH c AS (SELECT 1) SELECT * FROM c",
			"EXPLAIN WITH c AS (SELECT 1) SELECT * FROM c",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := parser.ParseExplain(tt.in)
			if err != nil {
				t.Fatalf("ParseExplain(%q) error = %v", tt.in, err)
			}

			if got := e.String(); got != tt.want {
				t.Errorf("ParseExplain(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseExplain_errors(t *testing.T) {
	tests := []string{
		"EXPLAIN",
		"EXPLAIN FORMAT",
		"EXPLAIN FORMAT = XML SELECT 1",
		"EXPLAIN INSERT INTO t (a) VALUES (1)",
		"EXPLAIN SELECT 1 foo bar",
	}

	for _, in := range tests {
		if _, err := parser.ParseExplain(in); err == nil {
			t.Errorf("ParseExplain(%q) expected error, got nil", in)
		}
	}
}

func TestParseUse(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "USE mydb", "USE mydb"},
		{"lowercase", "use mydb", "USE mydb"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := parser.ParseUse(tt.in)
			if err != nil {
				t.Fatalf("ParseUse(%q) error = %v", tt.in, err)
			}

			if got := u.String(); got != tt.want {
				t.Errorf("ParseUse(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseUse_errors(t *testing.T) {
	tests := []string{
		"USE",
		"USE mydb extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseUse(in); err == nil {
			t.Errorf("ParseUse(%q) expected error, got nil", in)
		}
	}
}
