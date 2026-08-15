package nested

import "database/sql"

func queryB(db *sql.DB) {
	db.Exec(`select id from b where id = ?`, 2)
}
