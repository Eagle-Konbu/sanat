package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

func TestParseStartTransaction(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "START TRANSACTION", "START TRANSACTION"},
		{"lowercase", "start transaction", "START TRANSACTION"},
		{"read only", "START TRANSACTION READ ONLY", "START TRANSACTION READ ONLY"},
		{"read write", "START TRANSACTION READ WRITE", "START TRANSACTION READ WRITE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st, err := parser.ParseStartTransaction(tt.in)
			if err != nil {
				t.Fatalf("ParseStartTransaction(%q) error = %v", tt.in, err)
			}

			if got := st.String(); got != tt.want {
				t.Errorf("ParseStartTransaction(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseStartTransaction_errors(t *testing.T) {
	tests := []string{
		"START",
		"START TRANSACTION READ",
		"START TRANSACTION READ ONLY WRITE",
		"START TRANSACTION extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseStartTransaction(in); err == nil {
			t.Errorf("ParseStartTransaction(%q) expected error, got nil", in)
		}
	}
}

func TestParseBegin(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "BEGIN", "BEGIN"},
		{"work", "BEGIN WORK", "BEGIN"},
		{"lowercase", "begin work", "BEGIN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := parser.ParseBegin(tt.in)
			if err != nil {
				t.Fatalf("ParseBegin(%q) error = %v", tt.in, err)
			}

			if got := b.String(); got != tt.want {
				t.Errorf("ParseBegin(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseCommit(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "COMMIT", "COMMIT"},
		{"work", "COMMIT WORK", "COMMIT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := parser.ParseCommit(tt.in)
			if err != nil {
				t.Fatalf("ParseCommit(%q) error = %v", tt.in, err)
			}

			if got := c.String(); got != tt.want {
				t.Errorf("ParseCommit(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseRollback(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "ROLLBACK", "ROLLBACK"},
		{"work", "ROLLBACK WORK", "ROLLBACK"},
		{"to savepoint", "ROLLBACK TO SAVEPOINT sp1", "ROLLBACK TO SAVEPOINT sp1"},
		{"to without savepoint keyword", "ROLLBACK TO sp1", "ROLLBACK TO SAVEPOINT sp1"},
		{"work to savepoint", "ROLLBACK WORK TO SAVEPOINT sp1", "ROLLBACK TO SAVEPOINT sp1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := parser.ParseRollback(tt.in)
			if err != nil {
				t.Fatalf("ParseRollback(%q) error = %v", tt.in, err)
			}

			if got := r.String(); got != tt.want {
				t.Errorf("ParseRollback(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSavepoint(t *testing.T) {
	s, err := parser.ParseSavepoint("SAVEPOINT sp1")
	if err != nil {
		t.Fatalf("ParseSavepoint(...) error = %v", err)
	}

	want := "SAVEPOINT sp1"
	if got := s.String(); got != want {
		t.Errorf("ParseSavepoint(...).String() = %q, want %q", got, want)
	}
}

func TestParseReleaseSavepoint(t *testing.T) {
	r, err := parser.ParseReleaseSavepoint("RELEASE SAVEPOINT sp1")
	if err != nil {
		t.Fatalf("ParseReleaseSavepoint(...) error = %v", err)
	}

	want := "RELEASE SAVEPOINT sp1"
	if got := r.String(); got != want {
		t.Errorf("ParseReleaseSavepoint(...).String() = %q, want %q", got, want)
	}
}

func TestParseSetVariable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"user variable", "SET @rank = 0", "SET @rank = 0"},
		{"no scope", "SET sql_mode = 'STRICT_TRANS_TABLES'", "SET sql_mode = 'STRICT_TRANS_TABLES'"},
		{"session scope", "SET SESSION sql_mode = 'STRICT_TRANS_TABLES'", "SET SESSION sql_mode = 'STRICT_TRANS_TABLES'"},
		{"global scope", "SET GLOBAL max_connections = 200", "SET GLOBAL max_connections = 200"},
		{"expression value", "SET @total = 1 + 2", "SET @total = 1 + 2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sv, err := parser.ParseSetVariable(tt.in)
			if err != nil {
				t.Fatalf("ParseSetVariable(%q) error = %v", tt.in, err)
			}

			if got := sv.String(); got != tt.want {
				t.Errorf("ParseSetVariable(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSetNames(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"charset only", "SET NAMES utf8mb4", "SET NAMES utf8mb4"},
		{"quoted charset", "SET NAMES 'utf8mb4'", "SET NAMES 'utf8mb4'"},
		{"charset and collate", "SET NAMES utf8mb4 COLLATE utf8mb4_bin", "SET NAMES utf8mb4 COLLATE utf8mb4_bin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sn, err := parser.ParseSetNames(tt.in)
			if err != nil {
				t.Fatalf("ParseSetNames(%q) error = %v", tt.in, err)
			}

			if got := sn.String(); got != tt.want {
				t.Errorf("ParseSetNames(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseSetVariable_errors(t *testing.T) {
	tests := []string{
		"SET",
		"SET @rank",
		"SET sql_mode",
		"SET @rank = 0 extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseSetVariable(in); err == nil {
			t.Errorf("ParseSetVariable(%q) expected error, got nil", in)
		}
	}
}
