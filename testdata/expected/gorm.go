package gorm

import "gorm.io/gorm"

type Order struct {
	ID    int
	Total int
}

func queries(db *gorm.DB) {
	var orders []Order

	// Raw string SQL via gorm.Raw: should be formatted
	db.Raw(`
SELECT
  id,
  total
FROM
  orders
WHERE
  total > ?
`, 100).Scan(&orders)

	// Raw string SQL via gorm.Exec: should be formatted
	db.Exec(`
UPDATE
  orders
SET
  total = ?
WHERE
  id = ?
`, 200, 1)

	// Double-quoted SQL: should NOT be changed
	db.Raw("select id from orders where id = ?", 1).Scan(&orders)
}
