package pgupdate

import "database/sql"

func queries(db *sql.DB) {
	// UPDATE ... FROM with RETURNING: should be formatted
	db.Exec(`
UPDATE
  orders o
SET
  status = 'paid',
  paid_at = now()
FROM
  payments p
WHERE
  p.order_id = o.id
  AND p.id = $1
RETURNING
  o.id
`, 1)

	// Multi-column assignment: should be formatted
	db.Exec(`
UPDATE
  users
SET
  (name, email) = ($1, $2)
WHERE
  id = $3
`, "a", "a@b.com", 1)
}
