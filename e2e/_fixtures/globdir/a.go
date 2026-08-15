package globdir

import "database/sql"

func queryA(db *sql.DB) {
	db.Exec(`select id from a where id = ?`, 1)
}
