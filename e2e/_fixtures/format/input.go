package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select id, name from users where active = ?`, true)
}
