package api

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"printcost/internal/models"
)

const filamentCols = `id, name, brand, material, diameter_mm, spool_price, spool_weight_kg, density_gcm3,
	drying_temp_c, drying_time_hours, requires_dry_cabinet, glue_stick_recommended, notes, colors_json, recommended_build_plate_ids_json, created_at, updated_at`

// filamentName is derived from brand + material (e.g. "Bambu Lab ABS") since
// those two already uniquely describe a filament record — no separate free
// text name needed.
func filamentName(brand, material string) string {
	return strings.TrimSpace(brand + " " + material)
}

func scanFilament(row interface{ Scan(...any) error }) (models.Filament, error) {
	var f models.Filament
	var requiresDryCabinet, glueStickRecommended int
	var colorsJSON, plateIDsJSON string
	err := row.Scan(&f.ID, &f.Name, &f.Brand, &f.Material, &f.DiameterMm, &f.SpoolPrice,
		&f.SpoolWeightKg, &f.DensityGcm3, &f.DryingTempC, &f.DryingTimeHours, &requiresDryCabinet, &glueStickRecommended,
		&f.Notes, &colorsJSON, &plateIDsJSON, &f.CreatedAt, &f.UpdatedAt)
	f.RequiresDryCabinet = requiresDryCabinet != 0
	f.GlueStickRecommended = glueStickRecommended != 0
	f.Colors = []models.FilamentColor{}
	json.Unmarshal([]byte(colorsJSON), &f.Colors)
	f.RecommendedBuildPlateIDs = []int64{}
	json.Unmarshal([]byte(plateIDsJSON), &f.RecommendedBuildPlateIDs)
	if f.SpoolWeightKg > 0 {
		f.PricePerKg = f.SpoolPrice / f.SpoolWeightKg
		f.PricePerG = f.SpoolPrice / (f.SpoolWeightKg * 1000)
	}
	radiusCm := (f.DiameterMm / 10) / 2
	if f.DensityGcm3 > 0 && radiusCm > 0 {
		volumeCm3PerM := math.Pi * radiusCm * radiusCm * 100
		f.LengthPerRollM = (f.SpoolWeightKg * 1000) / (f.DensityGcm3 * volumeCm3PerM)
	}
	return f, err
}

// resolveBuildPlateNames fills RecommendedBuildPlateNames on each filament,
// silently dropping any id that no longer matches a build plate (the id list
// is advisory, not a foreign key).
func (a *API) resolveBuildPlateNames(list []models.Filament) error {
	rows, err := a.DB.Query(`SELECT id, name FROM build_plates`)
	if err != nil {
		return err
	}
	defer rows.Close()
	names := map[int64]string{}
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		names[id] = name
	}
	for i := range list {
		list[i].RecommendedBuildPlateNames = []string{}
		for _, id := range list[i].RecommendedBuildPlateIDs {
			if name, ok := names[id]; ok {
				list[i].RecommendedBuildPlateNames = append(list[i].RecommendedBuildPlateNames, name)
			}
		}
	}
	return nil
}

func (a *API) filaments() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + filamentCols + ` FROM filaments ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.Filament{}
			for rows.Next() {
				f, err := scanFilament(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, f)
			}
			if err := a.resolveBuildPlateNames(list); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var f models.Filament
			if err := decodeJSON(r, &f); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			f.Name = filamentName(f.Brand, f.Material)
			if f.Name == "" || f.Material == "" || f.SpoolPrice <= 0 {
				writeError(w, http.StatusBadRequest, "material and spool_price are required")
				return
			}
			if f.DiameterMm == 0 {
				f.DiameterMm = 1.75
			}
			if f.SpoolWeightKg == 0 {
				f.SpoolWeightKg = 1.0
			}
			if f.DensityGcm3 == 0 {
				f.DensityGcm3 = 1.24
			}
			if f.Colors == nil {
				f.Colors = []models.FilamentColor{}
			}
			if f.RecommendedBuildPlateIDs == nil {
				f.RecommendedBuildPlateIDs = []int64{}
			}
			colorsJSON, _ := json.Marshal(f.Colors)
			plateIDsJSON, _ := json.Marshal(f.RecommendedBuildPlateIDs)
			res, err := a.DB.Exec(
				`INSERT INTO filaments (name, brand, material, diameter_mm, spool_price, spool_weight_kg, density_gcm3, drying_temp_c, drying_time_hours, requires_dry_cabinet, glue_stick_recommended, notes, colors_json, recommended_build_plate_ids_json)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				f.Name, f.Brand, f.Material, f.DiameterMm, f.SpoolPrice, f.SpoolWeightKg, f.DensityGcm3,
				f.DryingTempC, f.DryingTimeHours, f.RequiresDryCabinet, f.GlueStickRecommended, f.Notes, string(colorsJSON), string(plateIDsJSON),
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			f.ID = id
			writeJSON(w, http.StatusCreated, f)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+filamentCols+` FROM filaments WHERE id = ?`, id)
			f, err := scanFilament(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "filament not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			list := []models.Filament{f}
			if err := a.resolveBuildPlateNames(list); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, list[0])
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var f models.Filament
			if err := decodeJSON(r, &f); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			f.Name = filamentName(f.Brand, f.Material)
			if f.Colors == nil {
				f.Colors = []models.FilamentColor{}
			}
			if f.RecommendedBuildPlateIDs == nil {
				f.RecommendedBuildPlateIDs = []int64{}
			}
			colorsJSON, _ := json.Marshal(f.Colors)
			plateIDsJSON, _ := json.Marshal(f.RecommendedBuildPlateIDs)
			_, err = a.DB.Exec(
				`UPDATE filaments SET name=?, brand=?, material=?, diameter_mm=?, spool_price=?, spool_weight_kg=?, density_gcm3=?,
				 drying_temp_c=?, drying_time_hours=?, requires_dry_cabinet=?, glue_stick_recommended=?, notes=?, colors_json=?, recommended_build_plate_ids_json=?, updated_at=datetime('now') WHERE id=?`,
				f.Name, f.Brand, f.Material, f.DiameterMm, f.SpoolPrice, f.SpoolWeightKg, f.DensityGcm3,
				f.DryingTempC, f.DryingTimeHours, f.RequiresDryCabinet, f.GlueStickRecommended, f.Notes, string(colorsJSON), string(plateIDsJSON), id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			f.ID = id
			writeJSON(w, http.StatusOK, f)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM filaments WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this filament is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
