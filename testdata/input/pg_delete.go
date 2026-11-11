package pgdelete

import "database/sql"

func queries(db *sql.DB) {
	// DELETE ... USING with RETURNING: should be formatted
	db.Exec(`delete from sessions s using users u where s.user_id = u.id and u.disabled returning s.id`)

	// Data-modifying CTE: should be formatted
	db.Exec(`with gone as (delete from users where id = $1 returning id) delete from sessions where user_id in (select id from gone)`, 1)
}
