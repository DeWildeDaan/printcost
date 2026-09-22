package db

import "database/sql"

var defaultSettings = map[string]string{
	"energy_cost":  "0.28",
	"labor_rate":   "20.00",
	"failure_rate": "5.0",
	"markup":       "20.0",
	"currency":     "€",
	"company_name": "",
}

// Seed inserts default settings (if missing) and demo equipment/filament/
// consumable data (only on a completely empty database).
func Seed(sqlDB *sql.DB) error {
	for key, value := range defaultSettings {
		if _, err := sqlDB.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO NOTHING`,
			key, value,
		); err != nil {
			return err
		}
	}

	var machineCount int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM machines`).Scan(&machineCount); err != nil {
		return err
	}
	if machineCount > 0 {
		return nil
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO machines (name, type_tag, purchase_price, lifespan_hours, service_cost, power_kw)
		 VALUES ('Bambu Lab H2S', 'FDM', 1499, 5000, 150, 0.35)`,
	); err != nil {
		return err
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO ams_units (name, purchase_price, lifespan_hours, service_cost, power_kw)
		 VALUES ('Bambu AMS 2 Pro', 299, 5000, 0, 0.03)`,
	); err != nil {
		return err
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO build_plates (name, purchase_price, lifespan_hours)
		 VALUES ('Bambu Smooth PEI', 25, 300)`,
	); err != nil {
		return err
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO nozzles (name, material, diameter_mm, purchase_price, lifespan_hours)
		 VALUES ('Bambu 0.4mm Hardened', 'Hardened steel', 0.4, 12, 300)`,
	); err != nil {
		return err
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO filaments (name, brand, material, diameter_mm, spool_price, spool_weight_kg, density_gcm3)
		 VALUES ('Bambu Lab ABS', 'Bambu Lab', 'ABS', 1.75, 28, 1.0, 1.04)`,
	); err != nil {
		return err
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO filaments (name, brand, material, diameter_mm, spool_price, spool_weight_kg, density_gcm3)
		 VALUES ('Bambu Lab PETG HF', 'Bambu Lab', 'PETG', 1.75, 24, 1.0, 1.27)`,
	); err != nil {
		return err
	}

	if _, err := sqlDB.Exec(
		`INSERT INTO consumables (name, unit_label, unit_cost) VALUES ('IPA wipe', 'piece', 0.05)`,
	); err != nil {
		return err
	}
	if _, err := sqlDB.Exec(
		`INSERT INTO consumables (name, unit_label, unit_cost) VALUES ('Glue stick application', 'piece', 0.15)`,
	); err != nil {
		return err
	}

	return nil
}
