package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`
SELECT
  id,
  name
FROM
  users
WHERE
  active = ?
`, true)
}
