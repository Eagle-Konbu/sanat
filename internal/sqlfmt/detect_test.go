package sqlfmt_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

func TestMightBeSQL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"empty", "", false},
		{"plain text", "hello world", false},
		{"url path", "https://example.com/select/users", false},
		{"select query", "SELECT id FROM users", true},
		{"select lower", "select id from users", true},
		{"select mixed case", "Select id From users", true},
		{"select leading space", "  SELECT id FROM users", true},
		{"insert", "INSERT INTO users (name) VALUES (?)", true},
		{"update", "UPDATE users SET name = ? WHERE id = ?", true},
		{"delete", "DELETE FROM users WHERE id = ?", true},
		{"fmt sprintf %s", "SELECT %s FROM %s", false},
		{"fmt sprintf %d", "SELECT * FROM users LIMIT %d", false},
		{"fmt sprintf %v", "SELECT %v FROM users", false},
		{"select in sentence", "SELECT is a SQL keyword", true},
		{"log message", "failed to execute query", false},
		{"contains select not prefix", "the SELECT statement", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqlfmt.MightBeSQL(tt.in)
			if got != tt.want {
				t.Errorf("MightBeSQL(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
