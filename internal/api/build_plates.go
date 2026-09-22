package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const buildPlateCols = `id, name, purchase_price, lifespan_hours, notes, created_at, updated_at`

func scanBuildPlate(row interface{ Scan(...any) error }) (models.BuildPlate, error) {
	var p models.BuildPlate
	err := row.Scan(&p.ID, &p.Name, &p.PurchasePrice, &p.LifespanHours, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if p.LifespanHours > 0 {
		p.HourlyCost = p.PurchasePrice / p.LifespanHours
	}
	return p, err
}

func (a *API) buildPlates() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + buildPlateCols + ` FROM build_plates ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.BuildPlate{}
			for rows.Next() {
				p, err := scanBuildPlate(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, p)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var p models.BuildPlate
			if err := decodeJSON(r, &p); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if p.Name == "" || p.LifespanHours <= 0 {
				writeError(w, http.StatusBadRequest, "name and lifespan_hours are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO build_plates (name, purchase_price, lifespan_hours, notes) VALUES (?, ?, ?, ?)`,
				p.Name, p.PurchasePrice, p.LifespanHours, p.Notes,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			p.ID = id
			writeJSON(w, http.StatusCreated, p)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+buildPlateCols+` FROM build_plates WHERE id = ?`, id)
			p, err := scanBuildPlate(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "build plate not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, p)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var p models.BuildPlate
			if err := decodeJSON(r, &p); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE build_plates SET name=?, purchase_price=?, lifespan_hours=?, notes=?, updated_at=datetime('now') WHERE id=?`,
				p.Name, p.PurchasePrice, p.LifespanHours, p.Notes, id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			p.ID = id
			writeJSON(w, http.StatusOK, p)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM build_plates WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this build plate is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
