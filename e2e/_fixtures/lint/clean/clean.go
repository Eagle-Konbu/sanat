package clean

import "fmt"

const activeUsers = `
SELECT
  id,
  name
FROM
  users
WHERE
  active = 1
  AND id > 2
`

const greeting = `Hello, world`

func describe(name string) string {
	return fmt.Sprintf(`user: %s`, name)
}
