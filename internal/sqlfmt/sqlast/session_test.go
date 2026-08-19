package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestStartTransaction_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.StartTransaction
		want string
	}{
		{"no mode", &sqlast.StartTransaction{}, "START TRANSACTION"},
		{"read only", &sqlast.StartTransaction{Mode: sqlast.ReadOnlyMode}, "START TRANSACTION READ ONLY"},
		{"read write", &sqlast.StartTransaction{Mode: sqlast.ReadWriteMode}, "START TRANSACTION READ WRITE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("StartTransaction.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBegin_String(t *testing.T) {
	if got := (&sqlast.Begin{}).String(); got != "BEGIN" {
		t.Errorf("Begin.String() = %q, want %q", got, "BEGIN")
	}
}

func TestCommit_String(t *testing.T) {
	if got := (&sqlast.Commit{}).String(); got != "COMMIT" {
		t.Errorf("Commit.String() = %q, want %q", got, "COMMIT")
	}
}

func TestRollback_String(t *testing.T) {
	tests := []struct {
		name string
		r    *sqlast.Rollback
		want string
	}{
		{"plain", &sqlast.Rollback{}, "ROLLBACK"},
		{"to savepoint", &sqlast.Rollback{SavepointName: "sp1"}, "ROLLBACK TO SAVEPOINT sp1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("Rollback.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSavepoint_String(t *testing.T) {
	s := &sqlast.Savepoint{Name: "sp1"}

	want := "SAVEPOINT sp1"
	if got := s.String(); got != want {
		t.Errorf("Savepoint.String() = %q, want %q", got, want)
	}
}

func TestReleaseSavepoint_String(t *testing.T) {
	r := &sqlast.ReleaseSavepoint{Name: "sp1"}

	want := "RELEASE SAVEPOINT sp1"
	if got := r.String(); got != want {
		t.Errorf("ReleaseSavepoint.String() = %q, want %q", got, want)
	}
}

func TestSetVariable_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.SetVariable
		want string
	}{
		{"user variable", &sqlast.SetVariable{Name: "@rank", Value: lit("0")}, "SET @rank = 0"},
		{"no scope", &sqlast.SetVariable{Name: "sql_mode", Value: lit("STRICT_TRANS_TABLES")}, "SET sql_mode = STRICT_TRANS_TABLES"},
		{
			"session scope",
			&sqlast.SetVariable{Scope: sqlast.SessionScope, Name: "sql_mode", Value: lit("STRICT_TRANS_TABLES")},
			"SET SESSION sql_mode = STRICT_TRANS_TABLES",
		},
		{
			"global scope",
			&sqlast.SetVariable{Scope: sqlast.GlobalScope, Name: "max_connections", Value: lit("200")},
			"SET GLOBAL max_connections = 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("SetVariable.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetNames_String(t *testing.T) {
	tests := []struct {
		name string
		s    *sqlast.SetNames
		want string
	}{
		{"charset only", &sqlast.SetNames{Charset: lit("utf8mb4")}, "SET NAMES utf8mb4"},
		{
			"charset and collate",
			&sqlast.SetNames{Charset: lit("utf8mb4"), Collate: lit("utf8mb4_bin")},
			"SET NAMES utf8mb4 COLLATE utf8mb4_bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.String(); got != tt.want {
				t.Errorf("SetNames.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
