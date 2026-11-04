package options

var unformatted = `select id, name from users` // want "SQL literal is not formatted"

var formatted = `SELECT
  id
, name
FROM
  users`
