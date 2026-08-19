package sqlfmt_test

import "testing"

func TestFormatSQL_StartTransaction(t *testing.T) {
	assertFormatSQL(t, "start transaction", "START TRANSACTION")
	assertFormatSQL(t, "start transaction read only", "START TRANSACTION READ ONLY")
	assertFormatSQL(t, "start transaction read write", "START TRANSACTION READ WRITE")
}

func TestFormatSQL_Begin(t *testing.T) {
	assertFormatSQL(t, "begin", "BEGIN")
	assertFormatSQL(t, "begin work", "BEGIN")
}

func TestFormatSQL_Commit(t *testing.T) {
	assertFormatSQL(t, "commit", "COMMIT")
	assertFormatSQL(t, "commit work", "COMMIT")
}

func TestFormatSQL_Rollback(t *testing.T) {
	assertFormatSQL(t, "rollback", "ROLLBACK")
	assertFormatSQL(t, "rollback work", "ROLLBACK")
	assertFormatSQL(t, "rollback to savepoint sp1", "ROLLBACK TO SAVEPOINT sp1")
	assertFormatSQL(t, "rollback to sp1", "ROLLBACK TO SAVEPOINT sp1")
}

func TestFormatSQL_Savepoint(t *testing.T) {
	assertFormatSQL(t, "savepoint sp1", "SAVEPOINT sp1")
}

func TestFormatSQL_ReleaseSavepoint(t *testing.T) {
	assertFormatSQL(t, "release savepoint sp1", "RELEASE SAVEPOINT sp1")
}

func TestFormatSQL_SetVariable(t *testing.T) {
	assertFormatSQL(t, "set @rank = 0", "SET @rank = 0")
	assertFormatSQL(t, "set sql_mode = 'STRICT_TRANS_TABLES'", "SET sql_mode = 'STRICT_TRANS_TABLES'")
	assertFormatSQL(t, "set session sql_mode = 'STRICT_TRANS_TABLES'", "SET SESSION sql_mode = 'STRICT_TRANS_TABLES'")
	assertFormatSQL(t, "set global max_connections = 200", "SET GLOBAL max_connections = 200")
}

func TestFormatSQL_SetNames(t *testing.T) {
	assertFormatSQL(t, "set names utf8mb4", "SET NAMES utf8mb4")
	assertFormatSQL(t, "set names utf8mb4 collate utf8mb4_bin", "SET NAMES utf8mb4 COLLATE utf8mb4_bin")
	assertFormatSQL(t, "set names 'utf8mb4'", "SET NAMES 'utf8mb4'")
	assertFormatSQL(t, "set names 'utf8mb4' collate 'utf8mb4_bin'", "SET NAMES 'utf8mb4' COLLATE 'utf8mb4_bin'")
}
