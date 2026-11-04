package defaults

var unformatted = `select id from users` // want "SQL literal is not formatted"

var formatted = `
SELECT
  id
FROM
  users
`
