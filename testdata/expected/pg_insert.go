package pginsert

import "database/sql"

func queries(db *sql.DB) {
	// ON CONFLICT DO UPDATE with RETURNING: should be formatted
	db.QueryRow(`
INSERT INTO
  users
(
  email,
  name
)
VALUES
  ($1, $2)
ON CONFLICT (email) DO UPDATE SET
  name = excluded.name
RETURNING
  id
`, "a@b.com", "a")

	// ON CONFLICT DO NOTHING: should be formatted
	db.Exec(`
INSERT INTO
  tags
(
  name
)
VALUES
  ($1),
  ($2)
ON CONFLICT DO NOTHING
`, "x", "y")

	// INSERT ... SELECT: should be formatted
	db.Exec(`
INSERT INTO
  archive
SELECT
  *
FROM
  users
WHERE
  deleted_at IS NOT NULL
`)

	// DEFAULT VALUES: should be formatted
	db.QueryRow(`
INSERT INTO
  counters
DEFAULT VALUES
RETURNING
  id
`)
}
