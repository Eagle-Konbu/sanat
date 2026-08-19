package sqlx

import "github.com/jmoiron/sqlx"

// Struct tags are raw string literals too: should NOT be changed
type User struct {
	ID    int    `db:"id"`
	Email string `db:"email"`
}

func queries(db *sqlx.DB) {
	// Raw string SQL via sqlx.DB: should be formatted
	db.Exec(`
SELECT
  id,
  email
FROM
  users
WHERE
  id = ?
`, 1)
	db.MustExec(`
INSERT INTO
  users
(
  email,
  age
)
VALUES
  (?, ?)
`, "a@b.com", 20)

	var user User
	db.Get(&user, `
SELECT
  id,
  email
FROM
  users
WHERE
  id = ?
`, 1)

	var users []User
	db.Select(&users, `
SELECT
  id,
  email
FROM
  users
WHERE
  active = ?
`, true)

	// Named placeholders via sqlx.NamedExec: should be formatted
	db.NamedExec(`
INSERT INTO
  users
(
  email,
  age
)
VALUES
  (:email, :age)
`, user)
}
