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
		{"create table", "CREATE TABLE users (id INT)", true},
		{"alter table", "ALTER TABLE users ADD COLUMN name VARCHAR(255)", true},
		{"drop table", "DROP TABLE users", true},
		{"truncate table", "TRUNCATE TABLE users", true},
		{"start transaction", "START TRANSACTION", true},
		{"begin", "BEGIN", true},
		{"commit", "COMMIT", true},
		{"rollback", "ROLLBACK", true},
		{"savepoint", "SAVEPOINT sp1", true},
		{"release savepoint", "RELEASE SAVEPOINT sp1", true},
		{"set", "SET @x = 1", true},
		{"show tables", "SHOW TABLES", true},
		{"describe", "DESCRIBE users", true},
		{"explain", "EXPLAIN SELECT * FROM users", true},
		{"use", "USE mydb", true},
		{"call not detected", "CALL my_proc()", false},
		{"prepare not detected", "PREPARE stmt FROM 'SELECT 1'", false},
		{"execute not detected", "EXECUTE stmt", false},
		{"deallocate not detected", "DEALLOCATE PREPARE stmt", false},
		{"desc not detected", "DESC users", false},
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
