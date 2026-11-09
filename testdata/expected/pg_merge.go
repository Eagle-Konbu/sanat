package pgmerge

import "database/sql"

func queries(db *sql.DB) {
	// MERGE with every action kind: should be formatted
	db.Exec(`
MERGE INTO
  inventory i
USING
  incoming n
ON
  i.sku = n.sku
WHEN MATCHED AND n.qty = 0 THEN
  DELETE
WHEN MATCHED THEN
  UPDATE SET
    qty = i.qty + n.qty
WHEN NOT MATCHED THEN
  INSERT (sku, qty) VALUES (n.sku, n.qty)
`)
}
