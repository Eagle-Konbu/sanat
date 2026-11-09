package pgdelete

import "database/sql"

func queries(db *sql.DB) {
	// DELETE ... USING with RETURNING: should be formatted
	db.Exec(`
DELETE FROM
  sessions s
USING
  users u
WHERE
  s.user_id = u.id
  AND u.disabled
RETURNING
  s.id
`)

	// Data-modifying CTE: should be formatted
	db.Exec(`
WITH
  gone AS (
    DELETE FROM
      users
    WHERE
      id = $1
    RETURNING
      id
  )
DELETE FROM
  sessions
WHERE
  user_id IN (
    SELECT
      id
    FROM
      gone
  )
`, 1)
}
