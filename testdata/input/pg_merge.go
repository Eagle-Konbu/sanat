package pgmerge

import "database/sql"

func queries(db *sql.DB) {
	// MERGE with every action kind: should be formatted
	db.Exec(`merge into inventory i using incoming n on i.sku = n.sku when matched and n.qty = 0 then delete when matched then update set qty = i.qty + n.qty when not matched then insert (sku, qty) values (n.sku, n.qty)`)
}
