package leading

// Formatted with comma_style: leading, so it only passes when the plugin
// picks up that setting from the referenced config file.
const ListUsers = `
SELECT
  id
, name
FROM
  users
WHERE
  active = 1
`
