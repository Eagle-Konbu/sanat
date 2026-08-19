package sqlx

import "github.com/jmoiron/sqlx"

// Struct tags are raw string literals too: should NOT be changed
type User struct {
	ID    int    `db:"id"`
	Email string `db:"email"`
}

func queries(db *sqlx.DB) {
	// Raw string SQL via sqlx.DB: should be formatted
	db.Exec(`select id, email from users where id = ?`, 1)
	db.MustExec(`insert into users (email, age) values (?, ?)`, "a@b.com", 20)

	var user User
	db.Get(&user, `select id, email from users where id = ?`, 1)

	var users []User
	db.Select(&users, `select id, email from users where active = ?`, true)

	// Named placeholders via sqlx.NamedExec: should be formatted
	db.NamedExec(`insert into users (email, age) values (:email, :age)`, user)
}
