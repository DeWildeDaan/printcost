package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const materialSettingsCols = `id, material, supports, fuzzy_skin, ironing, created_at, updated_at`

func scanMaterialSettings(row interface{ Scan(...any) error }) (models.MaterialSettings, error) {
	var m models.MaterialSettings
	err := row.Scan(&m.ID, &m.Material, &m.Supports, &m.FuzzySkin, &m.Ironing, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (a *API) materialSettings() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + materialSettingsCols + ` FROM material_settings ORDER BY material`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.MaterialSettings{}
			for rows.Next() {
				m, err := scanMaterialSettings(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, m)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var m models.MaterialSettings
			if err := decodeJSON(r, &m); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if m.Material == "" {
				writeError(w, http.StatusBadRequest, "material is required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO material_settings (material, supports, fuzzy_skin, ironing) VALUES (?, ?, ?, ?)`,
				m.Material, m.Supports, m.FuzzySkin, m.Ironing,
			)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this material already has settings")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			m.ID = id
			writeJSON(w, http.StatusCreated, m)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+materialSettingsCols+` FROM material_settings WHERE id = ?`, id)
			m, err := scanMaterialSettings(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "material settings not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, m)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var m models.MaterialSettings
			if err := decodeJSON(r, &m); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE material_settings SET material=?, supports=?, fuzzy_skin=?, ironing=?, updated_at=datetime('now') WHERE id=?`,
				m.Material, m.Supports, m.FuzzySkin, m.Ironing, id,
			)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this material already has settings")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			m.ID = id
			writeJSON(w, http.StatusOK, m)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM material_settings WHERE id = ?`, id)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
