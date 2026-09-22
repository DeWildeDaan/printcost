package db

import "database/sql"

// migrate patches existing databases created before a couple of schema
// changes: build_plates moved from a print-count lifespan to an hourly one,
// and consumables dropped stock tracking. New databases already get the
// current shape from schema.sql, so these are no-ops there.
func migrate(sqlDB *sql.DB) error {
	if hasColumn(sqlDB, "build_plates", "lifespan_prints") && !hasColumn(sqlDB, "build_plates", "lifespan_hours") {
		if _, err := sqlDB.Exec(`ALTER TABLE build_plates RENAME COLUMN lifespan_prints TO lifespan_hours`); err != nil {
			return err
		}
	}
	if hasColumn(sqlDB, "consumables", "stock_qty") {
		if _, err := sqlDB.Exec(`ALTER TABLE consumables DROP COLUMN stock_qty`); err != nil {
			return err
		}
	}
	if hasColumn(sqlDB, "consumables", "low_stock_at") {
		if _, err := sqlDB.Exec(`ALTER TABLE consumables DROP COLUMN low_stock_at`); err != nil {
			return err
		}
	}
	if hasColumn(sqlDB, "filaments", "colour") {
		if _, err := sqlDB.Exec(`ALTER TABLE filaments DROP COLUMN colour`); err != nil {
			return err
		}
	}
	return nil
}

func hasColumn(sqlDB *sql.DB, table, column string) bool {
	rows, err := sqlDB.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if rows.Scan(&name) == nil && name == column {
			return true
		}
	}
	return false
}
