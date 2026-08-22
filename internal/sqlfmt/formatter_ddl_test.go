package sqlfmt_test

import (
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

func assertFormatSQL(t *testing.T, in, want string) {
	t.Helper()

	got, ok := sqlfmt.FormatSQL(in, 2)
	if !ok {
		t.Fatalf("FormatSQL(%q) expected ok", in)
	}

	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("FormatSQL(%q):\ngot:\n%s\n\nwant:\n%s", in, got, want)
	}
}

func TestFormatSQL_CreateTable(t *testing.T) {
	in := "create table if not exists users (" +
		"id INT not null auto_increment, " +
		"email VARCHAR(255) not null, " +
		"status ENUM('active', 'inactive') default 'active', " +
		"created_at TIMESTAMP default CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP, " +
		"primary key (id), " +
		"unique key uq_email (email), " +
		"index idx_status (status)" +
		") engine=InnoDB default charset=utf8mb4"

	want := join(
		"CREATE TABLE IF NOT EXISTS users (",
		"  id INT NOT NULL AUTO_INCREMENT,",
		"  email VARCHAR(255) NOT NULL,",
		"  status ENUM('active', 'inactive') DEFAULT 'active',",
		"  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,",
		"  PRIMARY KEY (id),",
		"  UNIQUE KEY uq_email (email),",
		"  INDEX idx_status (status)",
		") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
	)

	assertFormatSQL(t, in, want)
}

func TestFormatSQL_CreateTable_ForeignKey(t *testing.T) {
	in := "create table orders (" +
		"id INT, user_id INT, " +
		"constraint fk_user foreign key (user_id) references users (id) on delete cascade on update restrict" +
		")"

	want := join(
		"CREATE TABLE orders (",
		"  id INT,",
		"  user_id INT,",
		"  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE RESTRICT",
		")",
	)

	assertFormatSQL(t, in, want)
}

func TestFormatSQL_CreateTable_TypeNamePreservesSourceCase(t *testing.T) {
	// Unlike clause keywords, a data type name isn't canonicalized to a
	// fixed case — same treatment as an unrecognized function name (see
	// parseGenericFuncCall), since sqlast.DataType is deliberately generic
	// rather than an enum of known MySQL types.
	assertFormatSQL(t,
		"create table t (id int)",
		join(
			"CREATE TABLE t (",
			"  id int",
			")",
		),
	)
}

func TestFormatSQL_AlterTable(t *testing.T) {
	in := "alter table users add column age INT, drop column legacy_flag, rename to people"

	want := join(
		"ALTER TABLE users",
		"  ADD COLUMN age INT,",
		"  DROP COLUMN legacy_flag,",
		"  RENAME TO people",
	)

	assertFormatSQL(t, in, want)
}

func TestFormatSQL_AlterTable_SingleAction(t *testing.T) {
	in := "alter table users add index idx_name (name)"

	want := join(
		"ALTER TABLE users",
		"  ADD INDEX idx_name (name)",
	)

	assertFormatSQL(t, in, want)
}

func TestFormatSQL_CreateIndex(t *testing.T) {
	assertFormatSQL(t,
		"create unique index idx_email on users (email)",
		join(
			"CREATE UNIQUE INDEX idx_email ON users (",
			"  email",
			")",
		),
	)
}

func TestFormatSQL_DropIndex(t *testing.T) {
	assertFormatSQL(t, "drop index idx_email on users", "DROP INDEX idx_email ON users")
}

func TestFormatSQL_DropTable(t *testing.T) {
	assertFormatSQL(t, "drop table if exists users, orders", "DROP TABLE IF EXISTS users, orders")
}

func TestFormatSQL_TruncateTable(t *testing.T) {
	assertFormatSQL(t, "truncate table users", "TRUNCATE TABLE users")
	assertFormatSQL(t, "truncate users", "TRUNCATE TABLE users")
}

func TestFormatSQL_CreateTable_ParseFailureFallsThrough(t *testing.T) {
	in := "create table users (id int, primary key)"

	got, ok := sqlfmt.FormatSQL(in, 2)
	if ok {
		t.Fatalf("FormatSQL(%q) expected ok = false, got true (result %q)", in, got)
	}

	if got != in {
		t.Errorf("FormatSQL(%q) = %q, want original string unchanged", in, got)
	}
}

func TestFormatSQLWithOptions_CreateIndex_KeywordCaseLower(t *testing.T) {
	in := "create index idx on users (name desc)"

	got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, KeywordCase: sqlfmt.KeywordCaseLower})
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"CREATE INDEX idx ON users (",
		"  name desc",
		")",
	)

	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

// TestFormatSQLWithOptions_CreateTable_KeywordCaseLower_ClauseKeywordsStayUpper
// confirms that DDL clause keywords (NOT NULL, ON DELETE, ON UPDATE, ...)
// stay uppercase under keyword_case: lower, unlike the WHERE/SET-clause
// operator/predicate keywords keyword_case actually governs. Only DESC in an
// index column list (an operator/predicate keyword — see
// TestFormatSQLWithOptions_CreateIndex_KeywordCaseLower) is affected.
func TestFormatSQLWithOptions_CreateTable_KeywordCaseLower_ClauseKeywordsStayUpper(t *testing.T) {
	in := "create table orders (id int not null, user_id int, " +
		"foreign key (user_id) references users (id) on delete cascade on update restrict)"

	got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, KeywordCase: sqlfmt.KeywordCaseLower})
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"CREATE TABLE orders (",
		"  id int NOT NULL,",
		"  user_id int,",
		"  FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE RESTRICT",
		")",
	)

	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestFormatSQLWithOptions_AlterTable_CommaStyleLeading(t *testing.T) {
	in := "alter table users add column a INT, add column b INT"

	got, ok := sqlfmt.FormatSQLWithOptions(in, sqlfmt.Options{Indent: 2, CommaStyle: sqlfmt.CommaStyleLeading})
	if !ok {
		t.Fatal("expected ok")
	}

	want := join(
		"ALTER TABLE users",
		"  ADD COLUMN a INT",
		", ADD COLUMN b INT",
	)

	got = strings.TrimRight(got, "\n")
	want = strings.TrimRight(want, "\n")

	if got != want {
		t.Errorf("got:\n%s\n\nwant:\n%s", got, want)
	}
}
