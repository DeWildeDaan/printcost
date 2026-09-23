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
	if !hasColumn(sqlDB, "filaments", "colors_json") {
		if _, err := sqlDB.Exec(`ALTER TABLE filaments ADD COLUMN colors_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return err
		}
	}
	if !hasColumn(sqlDB, "filaments", "recommended_build_plate_ids_json") {
		if _, err := sqlDB.Exec(`ALTER TABLE filaments ADD COLUMN recommended_build_plate_ids_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return err
		}
	}
	if !hasColumn(sqlDB, "filaments", "glue_stick_recommended") {
		if _, err := sqlDB.Exec(`ALTER TABLE filaments ADD COLUMN glue_stick_recommended INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	// glue_stick_recommended moved from material_settings (shared per material)
	// to filaments (per spool) — carry over any values set under the old shape
	// before dropping the column.
	if !hasColumn(sqlDB, "consumables", "default_on_quote") {
		if _, err := sqlDB.Exec(`ALTER TABLE consumables ADD COLUMN default_on_quote INTEGER NOT NULL DEFAULT 0`); err != nil {
			return err
		}
	}
	if hasTable(sqlDB, "material_settings") && hasColumn(sqlDB, "material_settings", "glue_stick_recommended") {
		if _, err := sqlDB.Exec(`UPDATE filaments SET glue_stick_recommended = 1
			WHERE material IN (SELECT material FROM material_settings WHERE glue_stick_recommended = 1)`); err != nil {
			return err
		}
		if _, err := sqlDB.Exec(`ALTER TABLE material_settings DROP COLUMN glue_stick_recommended`); err != nil {
			return err
		}
	}
	return nil
}

func hasTable(sqlDB *sql.DB, table string) bool {
	var name string
	err := sqlDB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name)
	return err == nil
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
