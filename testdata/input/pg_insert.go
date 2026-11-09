package pginsert

import "database/sql"

func queries(db *sql.DB) {
	// ON CONFLICT DO UPDATE with RETURNING: should be formatted
	db.QueryRow(`insert into users (email, name) values ($1, $2) on conflict (email) do update set name = excluded.name returning id`, "a@b.com", "a")

	// ON CONFLICT DO NOTHING: should be formatted
	db.Exec(`insert into tags (name) values ($1), ($2) on conflict do nothing`, "x", "y")

	// INSERT ... SELECT: should be formatted
	db.Exec(`insert into archive select * from users where deleted_at is not null`)

	// DEFAULT VALUES: should be formatted
	db.QueryRow(`insert into counters default values returning id`)
}
