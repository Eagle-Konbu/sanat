package pgx

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func queries(ctx context.Context, pool *pgxpool.Pool) {
	// Raw string SQL with $-style placeholders: should be formatted
	pool.Exec(ctx, `
INSERT INTO
  users
(
  email,
  age
)
VALUES
  ($1, $2)
`, "a@b.com", 20)
	pool.QueryRow(ctx, `
SELECT
  id,
  email
FROM
  users
WHERE
  id = $1
`, 1)

	// Postgres-only upsert syntax the SQL formatter can't parse: should NOT be changed
	pool.Exec(ctx, `insert into users (email, age) values ($1, $2) on conflict (email) do update set age = excluded.age`, "a@b.com", 20)
}
