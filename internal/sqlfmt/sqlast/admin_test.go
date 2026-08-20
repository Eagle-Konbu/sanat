package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestShowTables_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.ShowTables
		want string
	}{
		{"plain", &sqlast.ShowTables{}, "SHOW TABLES"},
		{"from database", &sqlast.ShowTables{Database: "mydb"}, "SHOW TABLES FROM mydb"},
		{"like pattern", &sqlast.ShowTables{Like: lit("'user%'")}, "SHOW TABLES LIKE 'user%'"},
		{
			"from database and like pattern",
			&sqlast.ShowTables{Database: "mydb", Like: lit("'user%'")},
			"SHOW TABLES FROM mydb LIKE 'user%'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("ShowTables.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShowCreateTable_String(t *testing.T) {
	s := &sqlast.ShowCreateTable{Table: sqlast.TableName{Name: "users"}}

	want := "SHOW CREATE TABLE users"
	if got := s.String(); got != want {
		t.Errorf("ShowCreateTable.String() = %q, want %q", got, want)
	}
}

func TestShowColumns_String(t *testing.T) {
	s := &sqlast.ShowColumns{Table: sqlast.TableName{Name: "users"}}

	want := "SHOW COLUMNS FROM users"
	if got := s.String(); got != want {
		t.Errorf("ShowColumns.String() = %q, want %q", got, want)
	}
}

func TestShowIndex_String(t *testing.T) {
	s := &sqlast.ShowIndex{Table: sqlast.TableName{Name: "users"}}

	want := "SHOW INDEX FROM users"
	if got := s.String(); got != want {
		t.Errorf("ShowIndex.String() = %q, want %q", got, want)
	}
}

func TestShowDatabases_String(t *testing.T) {
	if got := (&sqlast.ShowDatabases{}).String(); got != "SHOW DATABASES" {
		t.Errorf("ShowDatabases.String() = %q, want %q", got, "SHOW DATABASES")
	}
}

func TestShowVariables_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.ShowVariables
		want string
	}{
		{"plain", &sqlast.ShowVariables{}, "SHOW VARIABLES"},
		{"like pattern", &sqlast.ShowVariables{Like: lit("'max_%'")}, "SHOW VARIABLES LIKE 'max_%'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("ShowVariables.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestShowStatus_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.ShowStatus
		want string
	}{
		{"plain", &sqlast.ShowStatus{}, "SHOW STATUS"},
		{"like pattern", &sqlast.ShowStatus{Like: lit("'Threads%'")}, "SHOW STATUS LIKE 'Threads%'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("ShowStatus.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDescribe_String(t *testing.T) {
	tests := []struct {
		name string
		d    *sqlast.Describe
		want string
	}{
		{"table only", &sqlast.Describe{Table: sqlast.TableName{Name: "users"}}, "DESCRIBE users"},
		{
			"with column",
			&sqlast.Describe{Table: sqlast.TableName{Name: "users"}, Column: "id"},
			"DESCRIBE users id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("Describe.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExplain_String(t *testing.T) {
	sel := &sqlast.Select{SelectExprs: []sqlast.SelectExpr{&sqlast.StarExpr{}}, From: []sqlast.TableExpr{
		&sqlast.AliasedTableExpr{Expr: sqlast.TableName{Name: "users"}},
	}}

	tests := []struct {
		name string
		e    *sqlast.Explain
		want string
	}{
		{"no format", &sqlast.Explain{Statement: sel}, "EXPLAIN SELECT * FROM users"},
		{
			"traditional format",
			&sqlast.Explain{Format: sqlast.TraditionalFormat, Statement: sel},
			"EXPLAIN FORMAT = TRADITIONAL SELECT * FROM users",
		},
		{
			"json format",
			&sqlast.Explain{Format: sqlast.JSONFormat, Statement: sel},
			"EXPLAIN FORMAT = JSON SELECT * FROM users",
		},
		{
			"tree format",
			&sqlast.Explain{Format: sqlast.TreeFormat, Statement: sel},
			"EXPLAIN FORMAT = TREE SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.e.String(); got != tt.want {
				t.Errorf("Explain.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUse_String(t *testing.T) {
	u := &sqlast.Use{Database: "mydb"}

	want := "USE mydb"
	if got := u.String(); got != want {
		t.Errorf("Use.String() = %q, want %q", got, want)
	}
}
