package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// UserRepository exercises a wider surface of the scanner and formatter:
// multiple methods, context-aware driver calls, and a broader mix of SQL
// statement shapes than a single simple query.
type UserRepository struct {
	db *sql.DB
}

// FindByID looks up a single user, including a subquery to check for orders.
func (r *UserRepository) FindByID(ctx context.Context, id int) (*sql.Row, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT
  u.id,
  u.name,
  u.email
FROM
  users u
WHERE
  u.id = ?
  AND EXISTS (
    SELECT
      1
    FROM
      orders o
    WHERE
      o.user_id = u.id
  )
`, id)

	return row, nil
}

// Search returns active users matching status, newest first, paginated.
func (r *UserRepository) Search(ctx context.Context, status string, limit, offset int) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
SELECT
  u.id,
  u.name,
  COUNT(o.id) AS order_count
FROM
  users u
  JOIN
  orders o
    ON o.user_id = u.id
WHERE
  u.status = ?
  AND u.created_at BETWEEN ? AND ?
GROUP BY
  u.id,
  u.name
HAVING
  COUNT(o.id) > 0
ORDER BY
  u.created_at DESC
LIMIT
  ?
OFFSET
  ?
`, status, "2024-01-01", "2024-12-31", limit, offset)
}

// RankByRegion ranks users within each region by total spend using a window
// function over a derived table.
func (r *UserRepository) RankByRegion(ctx context.Context) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
SELECT
  region,
  name,
  total_spent,
  ROW_NUMBER() OVER (
    PARTITION BY region
    ORDER BY total_spent DESC
  ) AS rnk
FROM
  (
  SELECT
    region,
    name,
    SUM(amount) AS total_spent
  FROM
    orders
  GROUP BY
    region,
    name
  ) t
`)
}

// ActivitySummary combines two differently-shaped selects into one report.
func (r *UserRepository) ActivitySummary(ctx context.Context) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
SELECT
  id,
  'user' AS kind
FROM
  users
WHERE
  status = 'active'
UNION ALL
SELECT
  id,
  'order' AS kind
FROM
  orders
WHERE
  total > 100
`)
}

// StatusLabel classifies a user's standing with a CASE expression.
func (r *UserRepository) StatusLabel(ctx context.Context, id int) (*sql.Row, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT
  id,
  CASE
    WHEN status = 'active' THEN 'OK'
    WHEN status = 'banned' THEN 'BLOCKED'
    ELSE 'UNKNOWN'
  END AS label
FROM
  users
WHERE
  id = ?
`, id)

	return row, nil
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, name, email string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `
INSERT INTO
  users
(
  name,
  email
)
VALUES
  (?, ?)
`, name, email)
}

// Upsert inserts a user or bumps its login count on conflict.
func (r *UserRepository) Upsert(ctx context.Context, id int, email string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `
INSERT INTO
  users
(
  id,
  email,
  login_count
)
VALUES
  (?, ?, 1)
ON DUPLICATE KEY UPDATE
  email = values(email),
  login_count = login_count + 1
`, id, email)
}

// Deactivate soft-deletes the single oldest matching row.
func (r *UserRepository) Deactivate(ctx context.Context, status string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `
UPDATE
  users
SET
  status = 'inactive'
WHERE
  status = ?
ORDER BY
  created_at
LIMIT
  1
`, status)
}

// PurgeInactive hard-deletes long-inactive users.
func (r *UserRepository) PurgeInactive(ctx context.Context, cutoff string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `
DELETE FROM
  users
WHERE
  status = 'inactive'
  AND last_login_at < ?
`, cutoff)
}

// buildLegacyQuery shows a query assembled by concatenation: the scanner
// only inspects single raw-string literals, so this must NOT be touched.
func buildLegacyQuery(table string) string {
	return "SELECT * FROM " + table + " WHERE 1=1"
}

// summaryReport uses a WITH clause: MightBeSQL only recognizes a leading
// SELECT/INSERT/UPDATE/DELETE keyword, so this is never even attempted and
// must NOT be touched.
const summaryReport = `
with active_users as (select id from users where status = 'active')
select count(*) from active_users
`

// docExample is prose that happens to mention a SQL keyword; it must NOT be
// touched.
const docExample = `Run SELECT queries against the replica, not the primary.`

// logTemplate is a fmt template, not SQL, and must NOT be touched.
const logTemplate = "SELECT failed for user %d: %v"

func describeQuery(q string) string {
	// Double-quoted SQL is out of scanner scope and must NOT be touched.
	return fmt.Sprintf("query: %s", "select 1 from dual")
}
