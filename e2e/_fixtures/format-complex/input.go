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
	row := r.db.QueryRowContext(ctx, `select u.id, u.name, u.email from users u where u.id = ? and exists (select 1 from orders o where o.user_id = u.id)`, id)

	return row, nil
}

// Search returns active users matching status, newest first, paginated.
func (r *UserRepository) Search(ctx context.Context, status string, limit, offset int) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
		select u.id, u.name, count(o.id) as order_count
		from users u
		join orders o on o.user_id = u.id
		where u.status = ? and u.created_at between ? and ?
		group by u.id, u.name
		having count(o.id) > 0
		order by u.created_at desc
		limit ? offset ?
	`, status, "2024-01-01", "2024-12-31", limit, offset)
}

// RankByRegion ranks users within each region by total spend using a window
// function over a derived table.
func (r *UserRepository) RankByRegion(ctx context.Context) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
		select region, name, total_spent,
		       row_number() over (partition by region order by total_spent desc) as rnk
		from (select region, name, sum(amount) as total_spent from orders group by region, name) t
	`)
}

// ActivitySummary combines two differently-shaped selects into one report.
func (r *UserRepository) ActivitySummary(ctx context.Context) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, `
		select id, 'user' as kind from users where status = 'active'
		union all
		select id, 'order' as kind from orders where total > 100
	`)
}

// StatusLabel classifies a user's standing with a CASE expression.
func (r *UserRepository) StatusLabel(ctx context.Context, id int) (*sql.Row, error) {
	row := r.db.QueryRowContext(ctx, `
		select id,
		       case when status = 'active' then 'OK' when status = 'banned' then 'BLOCKED' else 'UNKNOWN' end as label
		from users
		where id = ?
	`, id)

	return row, nil
}

// Create inserts a new user.
func (r *UserRepository) Create(ctx context.Context, name, email string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `insert into users (name, email) values (?, ?)`, name, email)
}

// Upsert inserts a user or bumps its login count on conflict.
func (r *UserRepository) Upsert(ctx context.Context, id int, email string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `
		insert into users (id, email, login_count)
		values (?, ?, 1)
		on duplicate key update email = values(email), login_count = login_count + 1
	`, id, email)
}

// Deactivate soft-deletes the single oldest matching row.
func (r *UserRepository) Deactivate(ctx context.Context, status string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `update users set status = 'inactive' where status = ? order by created_at asc limit 1`, status)
}

// PurgeInactive hard-deletes long-inactive users.
func (r *UserRepository) PurgeInactive(ctx context.Context, cutoff string) (sql.Result, error) {
	return r.db.ExecContext(ctx, `delete from users where status = 'inactive' and last_login_at < ?`, cutoff)
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
