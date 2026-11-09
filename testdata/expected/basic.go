package basic

import "database/sql"

func queries(db *sql.DB) {
	// Raw string SQL: should be formatted
	db.Exec(`
SELECT
  id,
  name
FROM
  users
WHERE
  id = ?
`, 1)
	db.Query(`
INSERT INTO
  users
(
  email,
  age
)
VALUES
  (?, ?)
`, "a@b.com", 20)
	db.Exec(`
UPDATE
  users
SET
  email = ?
WHERE
  id = ?
`, "new@b.com", 1)
	db.Exec(`
DELETE FROM
  users
WHERE
  id = ?
`, 1)

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
	db.Query(`
SELECT
  u.id
FROM
  users u
WHERE
  EXISTS (
    SELECT
      1
    FROM
      orders o
    WHERE
      o.user_id = u.id
  )
  AND u.active = ?
`, true)

	// Raw string SELECT with JOIN: should be formatted
	db.Query(`
SELECT
  u.id,
  o.total
FROM
  users u
  JOIN
  orders o
    ON u.id = o.user_id
WHERE
  o.total > ?
`, 100)

	// Raw string SELECT with CTE: should be formatted
	db.Query(`
WITH
  active AS (
    SELECT
      id
    FROM
      users
    WHERE
      active = ?
  )
SELECT
  id
FROM
  active
`, true)

	// Should NOT be changed: MERGE is detected but MySQL cannot parse it
	db.Exec(`merge into users u using staged s on u.id = s.id when matched then delete`)
}
