package pgselect

import "database/sql"

func queries(db *sql.DB) {
	// Positional parameters and the jsonb ? operator: should be formatted
	db.Query(`
SELECT
  id,
  data ->> 'name' AS name
FROM
  users
WHERE
  id = $1
  AND data ? 'email'
`, 1)

	// ? bind parameters, as some drivers and query builders write them: should be formatted
	db.Query(`
SELECT
  id
FROM
  users
WHERE
  status = ?
  AND data ?| ARRAY['a', 'b']
`, "active")

	// DISTINCT ON, window function, and FETCH: should be formatted
	db.Query(`
SELECT DISTINCT ON (user_id)
  user_id,
  amount,
  RANK() OVER (
    PARTITION BY user_id
    ORDER BY amount DESC
  )
FROM
  orders
ORDER BY
  user_id,
  amount DESC
FETCH FIRST 10 ROWS ONLY
`)

	// WITH RECURSIVE and a set operation: should be formatted
	db.Query(`
WITH RECURSIVE
  t (n) AS (
    SELECT
      1
    UNION ALL
    SELECT
      n + 1
    FROM
      t
    WHERE
      n < 10
  )
SELECT
  n
FROM
  t
`)

	// LATERAL join and a cast: should be formatted
	db.Query(`
SELECT
  u.id,
  o.total::numeric(10, 2)
FROM
  users u
  CROSS JOIN
  LATERAL (
  SELECT
    SUM(amount) AS total
  FROM
    orders
  WHERE
    orders.user_id = u.id
  ) o
`)

	// Literals and quoted identifiers keep their source spelling
	db.Query(`
SELECT
  "UserName",
  E'tab\there',
  $$it's$$
FROM
  "Users"
WHERE
  created_at > now() - interval '1 day'
`)

	// Should NOT be changed: contains a comment
	db.Query(`select id -- primary key
from users`)

	// Should NOT be changed: DDL is not formatted for PostgreSQL
	db.Exec(`create table users (id serial primary key)`)
}
