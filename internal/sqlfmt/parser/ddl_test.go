package parser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/parser"
)

func assertCreateTableRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	ct, err := parser.ParseCreateTable(input)
	if err != nil {
		t.Fatalf("ParseCreateTable(%q) error = %v", input, err)
	}

	if got := ct.String(); got != want {
		t.Errorf("ParseCreateTable(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseCreateTable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{
			"single column",
			"CREATE TABLE t (id INT)",
			"CREATE TABLE t (id INT)",
		},
		{
			"if not exists",
			"CREATE TABLE IF NOT EXISTS t (id INT)",
			"CREATE TABLE IF NOT EXISTS t (id INT)",
		},
		{
			"multiple columns with constraints",
			"CREATE TABLE t (id INT NOT NULL AUTO_INCREMENT, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE, PRIMARY KEY (id))",
			"CREATE TABLE t (id INT NOT NULL AUTO_INCREMENT, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE, PRIMARY KEY (id))",
		},
		{
			"default and comment",
			"CREATE TABLE t (active TINYINT DEFAULT 1 COMMENT 'is active')",
			"CREATE TABLE t (active TINYINT DEFAULT 1 COMMENT 'is active')",
		},
		{
			"timestamp on update",
			"CREATE TABLE t (updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)",
			"CREATE TABLE t (updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP)",
		},
		{
			"explicit null and inline primary key",
			"CREATE TABLE t (id INT PRIMARY KEY, note VARCHAR(20) NULL)",
			"CREATE TABLE t (id INT PRIMARY KEY, note VARCHAR(20) NULL)",
		},
		{
			"unsigned zerofill charset collate",
			"CREATE TABLE t (n INT UNSIGNED ZEROFILL, s VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin)",
			"CREATE TABLE t (n INT UNSIGNED ZEROFILL, s VARCHAR(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin)",
		},
		{
			"charset shorthand",
			"CREATE TABLE t (s VARCHAR(20) CHARSET utf8mb4)",
			"CREATE TABLE t (s VARCHAR(20) CHARACTER SET utf8mb4)",
		},
		{
			"enum and set types",
			"CREATE TABLE t (status ENUM('a', 'b'), flags SET('x', 'y'))",
			"CREATE TABLE t (status ENUM('a', 'b'), flags SET('x', 'y'))",
		},
		{
			"named unique key with index name",
			"CREATE TABLE t (id INT, email VARCHAR(255), UNIQUE KEY uq_email (email))",
			"CREATE TABLE t (id INT, email VARCHAR(255), UNIQUE KEY uq_email (email))",
		},
		{
			"plain index with key synonym",
			"CREATE TABLE t (id INT, name VARCHAR(255), KEY idx_name (name))",
			"CREATE TABLE t (id INT, name VARCHAR(255), INDEX idx_name (name))",
		},
		{
			"index with prefix length and desc",
			"CREATE TABLE t (id INT, name VARCHAR(255), INDEX idx_name (name(20) DESC))",
			"CREATE TABLE t (id INT, name VARCHAR(255), INDEX idx_name (name(20) DESC))",
		},
		{
			"foreign key with actions",
			"CREATE TABLE t (id INT, user_id INT, FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE RESTRICT)",
			"CREATE TABLE t (id INT, user_id INT, FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE RESTRICT)",
		},
		{
			"foreign key with set null, set default, and no action",
			"CREATE TABLE t (id INT, a INT, b INT, " +
				"FOREIGN KEY (a) REFERENCES x (id) ON DELETE SET NULL ON UPDATE SET DEFAULT, " +
				"FOREIGN KEY (b) REFERENCES y (id) ON DELETE NO ACTION)",
			"CREATE TABLE t (id INT, a INT, b INT, " +
				"FOREIGN KEY (a) REFERENCES x (id) ON DELETE SET NULL ON UPDATE SET DEFAULT, " +
				"FOREIGN KEY (b) REFERENCES y (id) ON DELETE NO ACTION)",
		},
		{
			"named constraint foreign key",
			"CREATE TABLE t (id INT, user_id INT, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id))",
			"CREATE TABLE t (id INT, user_id INT, CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id))",
		},
		{
			"table options",
			"CREATE TABLE t (id INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='a table'",
			"CREATE TABLE t (id INT) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin COMMENT='a table'",
		},
		{
			"table options without equals",
			"CREATE TABLE t (id INT) ENGINE InnoDB",
			"CREATE TABLE t (id INT) ENGINE=InnoDB",
		},
		{
			"every recognized table option name form",
			"CREATE TABLE t (id INT) DEFAULT CHARACTER SET=utf8mb4, DEFAULT COLLATE=utf8mb4_bin, " +
				"CHARACTER SET=utf8mb4, CHARSET=utf8mb4, COLLATE=utf8mb4_bin, AUTO_INCREMENT=100",
			"CREATE TABLE t (id INT) DEFAULT CHARACTER SET=utf8mb4 DEFAULT COLLATE=utf8mb4_bin " +
				"CHARACTER SET=utf8mb4 CHARSET=utf8mb4 COLLATE=utf8mb4_bin AUTO_INCREMENT=100",
		},
		{
			"unknown table option falls back to bare identifier",
			"CREATE TABLE t (id INT) ROW_FORMAT=DYNAMIC",
			"CREATE TABLE t (id INT) ROW_FORMAT=DYNAMIC",
		},
		{
			"qualified table name",
			"CREATE TABLE mydb.t (id INT)",
			"CREATE TABLE mydb.t (id INT)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCreateTableRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseCreateTable_errors(t *testing.T) {
	tests := []string{
		"",
		"CREATE",
		"CREATE TABLE",
		"CREATE TABLE t",
		"CREATE TABLE t (",
		"CREATE TABLE t ()",
		"CREATE TABLE t (id",
		"CREATE TABLE t (id INT",
		"CREATE TABLE t (id INT)(",
		"CREATE TABLE t (PRIMARY KEY)",
		"CREATE TABLE t (id INT, FOREIGN KEY (id) REFERENCES)",
		"CREATE TABLE t (id INT) 1",
		"CREATE TABLE t (id INT) DEFAULT",
		"CREATE TABLE t (id INT) DEFAULT INVALID",
		"CREATE TABLE t (id VARCHAR(x))",
		"CREATE TABLE t (id INT, CONSTRAINT c INDEX (id))",
		"CREATE TABLE t (id INT, FOREIGN KEY (id) REFERENCES t2 (id) ON DELETE MAYBE)",
		"CREATE TABLE t (id INT, FOREIGN KEY (id) REFERENCES t2 (id) ON DELETE SET)",
		"CREATE TABLE t (id INT, FOREIGN KEY (id) REFERENCES t2 (id) ON MAYBE)",
	}

	for _, in := range tests {
		if _, err := parser.ParseCreateTable(in); err == nil {
			t.Errorf("ParseCreateTable(%q) expected error, got nil", in)
		}
	}
}

func assertAlterTableRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	at, err := parser.ParseAlterTable(input)
	if err != nil {
		t.Fatalf("ParseAlterTable(%q) error = %v", input, err)
	}

	if got := at.String(); got != want {
		t.Errorf("ParseAlterTable(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseAlterTable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{
			"add column",
			"ALTER TABLE t ADD COLUMN age INT",
			"ALTER TABLE t ADD COLUMN age INT",
		},
		{
			"add column without column keyword",
			"ALTER TABLE t ADD age INT",
			"ALTER TABLE t ADD COLUMN age INT",
		},
		{
			"drop column",
			"ALTER TABLE t DROP COLUMN age",
			"ALTER TABLE t DROP COLUMN age",
		},
		{
			"drop column without column keyword",
			"ALTER TABLE t DROP age",
			"ALTER TABLE t DROP COLUMN age",
		},
		{
			"modify column",
			"ALTER TABLE t MODIFY COLUMN age BIGINT NOT NULL",
			"ALTER TABLE t MODIFY COLUMN age BIGINT NOT NULL",
		},
		{
			"add index",
			"ALTER TABLE t ADD INDEX idx_name (name)",
			"ALTER TABLE t ADD INDEX idx_name (name)",
		},
		{
			"drop index",
			"ALTER TABLE t DROP INDEX idx_name",
			"ALTER TABLE t DROP INDEX idx_name",
		},
		{
			"drop key synonym",
			"ALTER TABLE t DROP KEY idx_name",
			"ALTER TABLE t DROP INDEX idx_name",
		},
		{
			"add constraint foreign key",
			"ALTER TABLE t ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id)",
			"ALTER TABLE t ADD CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id)",
		},
		{
			"add primary key",
			"ALTER TABLE t ADD PRIMARY KEY (id)",
			"ALTER TABLE t ADD PRIMARY KEY (id)",
		},
		{
			"add unique",
			"ALTER TABLE t ADD UNIQUE (email)",
			"ALTER TABLE t ADD UNIQUE KEY (email)",
		},
		{
			"rename to",
			"ALTER TABLE t RENAME TO t2",
			"ALTER TABLE t RENAME TO t2",
		},
		{
			"multiple actions",
			"ALTER TABLE t ADD COLUMN a INT, DROP COLUMN b, RENAME TO t2",
			"ALTER TABLE t ADD COLUMN a INT, DROP COLUMN b, RENAME TO t2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertAlterTableRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseAlterTable_errors(t *testing.T) {
	tests := []string{
		"",
		"ALTER",
		"ALTER TABLE",
		"ALTER TABLE t",
		"ALTER TABLE t ADD",
		"ALTER TABLE t DROP",
		"ALTER TABLE t MODIFY",
		"ALTER TABLE t RENAME",
		"ALTER TABLE t RENAME t2",
		"ALTER TABLE t UNKNOWN",
		"ALTER TABLE t ADD COLUMN a INT,",
		"ALTER TABLE t DROP PRIMARY KEY",
		"ALTER TABLE t CHANGE COLUMN a b INT",
		"ALTER TABLE t ADD COLUMN a INT extra",
	}

	for _, in := range tests {
		if _, err := parser.ParseAlterTable(in); err == nil {
			t.Errorf("ParseAlterTable(%q) expected error, got nil", in)
		}
	}
}

func assertCreateIndexRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	ci, err := parser.ParseCreateIndex(input)
	if err != nil {
		t.Fatalf("ParseCreateIndex(%q) error = %v", input, err)
	}

	if got := ci.String(); got != want {
		t.Errorf("ParseCreateIndex(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseCreateIndex(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{
			"plain",
			"CREATE INDEX idx_name ON t (name)",
			"CREATE INDEX idx_name ON t (name)",
		},
		{
			"unique",
			"CREATE UNIQUE INDEX idx_email ON t (email)",
			"CREATE UNIQUE INDEX idx_email ON t (email)",
		},
		{
			"multi column with prefix and direction",
			"CREATE INDEX idx ON t (a(10) ASC, b DESC)",
			"CREATE INDEX idx ON t (a(10), b DESC)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertCreateIndexRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseCreateTable_anonymousConstraint(t *testing.T) {
	assertCreateTableRoundTrip(t,
		"CREATE TABLE t (id INT, CONSTRAINT PRIMARY KEY (id))",
		"CREATE TABLE t (id INT, PRIMARY KEY (id))",
	)
}

func TestParseCreateStatement_unknownKeyword(t *testing.T) {
	if _, err := parser.ParseStatement("CREATE VIEW v AS SELECT 1"); err == nil {
		t.Error(`ParseStatement("CREATE VIEW v AS SELECT 1") expected error, got nil`)
	}

	if _, err := parser.ParseStatement("DROP VIEW v"); err == nil {
		t.Error(`ParseStatement("DROP VIEW v") expected error, got nil`)
	}
}

func TestParseCreateIndex_errors(t *testing.T) {
	tests := []string{
		"",
		"CREATE INDEX",
		"CREATE INDEX idx",
		"CREATE INDEX idx ON",
		"CREATE INDEX idx ON t",
		"CREATE INDEX idx ON t (",
		"CREATE INDEX idx ON t ()",
		"CREATE INDEX idx ON t (a) extra",
		"CREATE INDEX idx ON t (a(x))",
		"CREATE INDEX idx ON t (a(99999999999999999999))",
	}

	for _, in := range tests {
		if _, err := parser.ParseCreateIndex(in); err == nil {
			t.Errorf("ParseCreateIndex(%q) expected error, got nil", in)
		}
	}
}

func TestParseDropIndex(t *testing.T) {
	di, err := parser.ParseDropIndex("DROP INDEX idx_name ON t")
	if err != nil {
		t.Fatalf("ParseDropIndex error = %v", err)
	}

	want := "DROP INDEX idx_name ON t"
	if got := di.String(); got != want {
		t.Errorf("ParseDropIndex().String() = %q, want %q", got, want)
	}
}

func TestParseDropIndex_errors(t *testing.T) {
	tests := []string{"", "DROP INDEX", "DROP INDEX idx", "DROP INDEX idx ON", "DROP INDEX idx ON t extra"}

	for _, in := range tests {
		if _, err := parser.ParseDropIndex(in); err == nil {
			t.Errorf("ParseDropIndex(%q) expected error, got nil", in)
		}
	}
}

func assertDropTableRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	dt, err := parser.ParseDropTable(input)
	if err != nil {
		t.Fatalf("ParseDropTable(%q) error = %v", input, err)
	}

	if got := dt.String(); got != want {
		t.Errorf("ParseDropTable(%q).String() = %q, want %q", input, got, want)
	}
}

func TestParseDropTable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"single", "DROP TABLE t", "DROP TABLE t"},
		{"if exists", "DROP TABLE IF EXISTS t", "DROP TABLE IF EXISTS t"},
		{"multiple", "DROP TABLE t1, t2, t3", "DROP TABLE t1, t2, t3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertDropTableRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseDropTable_errors(t *testing.T) {
	tests := []string{"", "DROP", "DROP TABLE", "DROP TABLE t,", "DROP TABLE IF", "DROP TABLE t extra"}

	for _, in := range tests {
		if _, err := parser.ParseDropTable(in); err == nil {
			t.Errorf("ParseDropTable(%q) expected error, got nil", in)
		}
	}
}

func TestParseTruncateTable(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"with table keyword", "TRUNCATE TABLE t", "TRUNCATE TABLE t"},
		{"without table keyword", "TRUNCATE t", "TRUNCATE TABLE t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, err := parser.ParseTruncateTable(tt.in)
			if err != nil {
				t.Fatalf("ParseTruncateTable(%q) error = %v", tt.in, err)
			}

			if got := tr.String(); got != tt.want {
				t.Errorf("ParseTruncateTable(%q).String() = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestParseTruncateTable_errors(t *testing.T) {
	tests := []string{"", "TRUNCATE", "TRUNCATE TABLE t extra"}

	for _, in := range tests {
		if _, err := parser.ParseTruncateTable(in); err == nil {
			t.Errorf("ParseTruncateTable(%q) expected error, got nil", in)
		}
	}
}
