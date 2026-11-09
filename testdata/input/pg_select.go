package pgselect

import "database/sql"

func queries(db *sql.DB) {
	// Positional parameters and the jsonb ? operator: should be formatted
	db.Query(`select id, data->>'name' as name from users where id = $1 and data ? 'email'`, 1)

	// ? bind parameters, as some drivers and query builders write them: should be formatted
	db.Query(`select id from users where status = ? and data ?| array['a', 'b']`, "active")

	// DISTINCT ON, window function, and FETCH: should be formatted
	db.Query(`select distinct on (user_id) user_id, amount, rank() over (partition by user_id order by amount desc) from orders order by user_id, amount desc fetch first 10 rows only`)

	// WITH RECURSIVE and a set operation: should be formatted
	db.Query(`with recursive t(n) as (select 1 union all select n + 1 from t where n < 10) select n from t`)

	// LATERAL join and a cast: should be formatted
	db.Query(`select u.id, o.total::numeric(10, 2) from users u cross join lateral (select sum(amount) as total from orders where orders.user_id = u.id) o`)

	// Literals and quoted identifiers keep their source spelling
	db.Query(`select "UserName", E'tab\there', $$it's$$ from "Users" where created_at > now() - interval '1 day'`)

	// Should NOT be changed: contains a comment
	db.Query(`select id -- primary key
from users`)

	// Should NOT be changed: DDL is not formatted for PostgreSQL
	db.Exec(`create table users (id serial primary key)`)
}
