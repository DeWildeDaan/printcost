package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const amsCols = `id, name, purchase_price, lifespan_hours, service_cost, power_kw, notes, created_at, updated_at`

func scanAMS(row interface{ Scan(...any) error }) (models.AMSUnit, error) {
	var u models.AMSUnit
	err := row.Scan(&u.ID, &u.Name, &u.PurchasePrice, &u.LifespanHours, &u.ServiceCost, &u.PowerKw, &u.Notes, &u.CreatedAt, &u.UpdatedAt)
	if u.LifespanHours > 0 {
		u.HourlyCost = (u.PurchasePrice + u.ServiceCost) / u.LifespanHours
	}
	return u, err
}

func (a *API) amsUnits() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + amsCols + ` FROM ams_units ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.AMSUnit{}
			for rows.Next() {
				u, err := scanAMS(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, u)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var u models.AMSUnit
			if err := decodeJSON(r, &u); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if u.Name == "" || u.LifespanHours <= 0 {
				writeError(w, http.StatusBadRequest, "name and lifespan_hours are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO ams_units (name, purchase_price, lifespan_hours, service_cost, power_kw, notes) VALUES (?, ?, ?, ?, ?, ?)`,
				u.Name, u.PurchasePrice, u.LifespanHours, u.ServiceCost, u.PowerKw, u.Notes,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			u.ID = id
			writeJSON(w, http.StatusCreated, u)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+amsCols+` FROM ams_units WHERE id = ?`, id)
			u, err := scanAMS(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "AMS unit not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, u)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var u models.AMSUnit
			if err := decodeJSON(r, &u); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE ams_units SET name=?, purchase_price=?, lifespan_hours=?, service_cost=?, power_kw=?, notes=?, updated_at=datetime('now') WHERE id=?`,
				u.Name, u.PurchasePrice, u.LifespanHours, u.ServiceCost, u.PowerKw, u.Notes, id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			u.ID = id
			writeJSON(w, http.StatusOK, u)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM ams_units WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this AMS unit is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
