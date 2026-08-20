package sqlfmt_test

import "testing"

func TestFormatSQL_ShowTables(t *testing.T) {
	assertFormatSQL(t, "show tables", "SHOW TABLES")
	assertFormatSQL(t, "show tables from mydb", "SHOW TABLES FROM mydb")
	assertFormatSQL(t, "show tables like 'user%'", "SHOW TABLES LIKE 'user%'")
	assertFormatSQL(t, "show tables from mydb like 'user%'", "SHOW TABLES FROM mydb LIKE 'user%'")
}

func TestFormatSQL_ShowCreateTable(t *testing.T) {
	assertFormatSQL(t, "show create table users", "SHOW CREATE TABLE users")
}

func TestFormatSQL_ShowColumns(t *testing.T) {
	assertFormatSQL(t, "show columns from users", "SHOW COLUMNS FROM users")
}

func TestFormatSQL_ShowIndex(t *testing.T) {
	assertFormatSQL(t, "show index from users", "SHOW INDEX FROM users")
}

func TestFormatSQL_ShowDatabases(t *testing.T) {
	assertFormatSQL(t, "show databases", "SHOW DATABASES")
}

func TestFormatSQL_ShowVariables(t *testing.T) {
	assertFormatSQL(t, "show variables", "SHOW VARIABLES")
	assertFormatSQL(t, "show variables like 'max_%'", "SHOW VARIABLES LIKE 'max_%'")
}

func TestFormatSQL_ShowStatus(t *testing.T) {
	assertFormatSQL(t, "show status", "SHOW STATUS")
	assertFormatSQL(t, "show status like 'Threads%'", "SHOW STATUS LIKE 'Threads%'")
}

func TestFormatSQL_Describe(t *testing.T) {
	assertFormatSQL(t, "describe users", "DESCRIBE users")
	assertFormatSQL(t, "describe users id", "DESCRIBE users id")
}

func TestFormatSQL_Use(t *testing.T) {
	assertFormatSQL(t, "use mydb", "USE mydb")
}

func TestFormatSQL_Explain(t *testing.T) {
	assertFormatSQL(t, "explain select id from users",
		join(
			"EXPLAIN",
			"  SELECT",
			"    id",
			"  FROM",
			"    users",
		),
	)

	assertFormatSQL(t, "explain format = json select id from users",
		join(
			"EXPLAIN FORMAT = JSON",
			"  SELECT",
			"    id",
			"  FROM",
			"    users",
		),
	)
}
