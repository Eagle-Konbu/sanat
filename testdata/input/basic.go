package basic

import "database/sql"

func queries(db *sql.DB) {
	// Raw string SQL: should be formatted
	db.Exec(`select id, name from users where id = ?`, 1)
	db.Query(`insert into users (email, age) values (?, ?)`, "a@b.com", 20)
	db.Exec(`update users set email = ? where id = ?`, "new@b.com", 1)
	db.Exec(`delete from users where id = ?`, 1)

	// Double-quoted SQL: should NOT be changed
	db.Exec("select id from users where id = ?", 1)

	// Should NOT be changed: plain strings
	msg := "hello world"
	_ = msg

	// Should NOT be changed: SQL-like but unparseable
	note := `SELECT is a SQL keyword`
	_ = note

	// Should NOT be changed: fmt template
	tpl := "SELECT %s FROM %s"
	_ = tpl

	// Should NOT be changed: URL
	url := "https://example.com/select/users"
	_ = url

	// Should NOT be changed: string concatenation
	q := "SELECT " + "* FROM users"
	_ = q

	// Raw string SELECT with subquery: should be formatted
	db.Query(`select u.id from users u where exists (select 1 from orders o where o.user_id = u.id) and u.active = ?`, true)

	// Raw string SELECT with JOIN: should be formatted
	db.Query(`select u.id, o.total from users u join orders o on u.id = o.user_id where o.total > ?`, 100)

	// Raw string SELECT with CTE: should be formatted
	db.Query(`with active as (select id from users where active = ?) select id from active`, true)

	// Should NOT be changed: MERGE is detected but MySQL cannot parse it
	db.Exec(`merge into users u using staged s on u.id = s.id when matched then delete`)
}
