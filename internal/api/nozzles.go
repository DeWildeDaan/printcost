package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const nozzleCols = `id, name, material, diameter_mm, purchase_price, lifespan_hours, notes, created_at, updated_at`

func scanNozzle(row interface{ Scan(...any) error }) (models.Nozzle, error) {
	var n models.Nozzle
	err := row.Scan(&n.ID, &n.Name, &n.Material, &n.DiameterMm, &n.PurchasePrice, &n.LifespanHours, &n.Notes, &n.CreatedAt, &n.UpdatedAt)
	if n.LifespanHours > 0 {
		n.HourlyCost = n.PurchasePrice / n.LifespanHours
	}
	return n, err
}

func (a *API) nozzles() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + nozzleCols + ` FROM nozzles ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.Nozzle{}
			for rows.Next() {
				n, err := scanNozzle(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, n)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var n models.Nozzle
			if err := decodeJSON(r, &n); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if n.Name == "" || n.LifespanHours <= 0 {
				writeError(w, http.StatusBadRequest, "name and lifespan_hours are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO nozzles (name, material, diameter_mm, purchase_price, lifespan_hours, notes) VALUES (?, ?, ?, ?, ?, ?)`,
				n.Name, n.Material, n.DiameterMm, n.PurchasePrice, n.LifespanHours, n.Notes,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			n.ID = id
			writeJSON(w, http.StatusCreated, n)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+nozzleCols+` FROM nozzles WHERE id = ?`, id)
			n, err := scanNozzle(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "nozzle not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, n)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var n models.Nozzle
			if err := decodeJSON(r, &n); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE nozzles SET name=?, material=?, diameter_mm=?, purchase_price=?, lifespan_hours=?, notes=?, updated_at=datetime('now') WHERE id=?`,
				n.Name, n.Material, n.DiameterMm, n.PurchasePrice, n.LifespanHours, n.Notes, id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			n.ID = id
			writeJSON(w, http.StatusOK, n)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM nozzles WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this nozzle is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
