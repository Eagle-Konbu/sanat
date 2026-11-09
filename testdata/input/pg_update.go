package pgupdate

import "database/sql"

func queries(db *sql.DB) {
	// UPDATE ... FROM with RETURNING: should be formatted
	db.Exec(`update orders o set status = 'paid', paid_at = now() from payments p where p.order_id = o.id and p.id = $1 returning o.id`, 1)

	// Multi-column assignment: should be formatted
	db.Exec(`update users set (name, email) = ($1, $2) where id = $3`, "a", "a@b.com", 1)
}
