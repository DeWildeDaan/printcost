package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

func scanMachine(row interface{ Scan(...any) error }) (models.Machine, error) {
	var m models.Machine
	err := row.Scan(&m.ID, &m.Name, &m.TypeTag, &m.PurchasePrice, &m.LifespanHours,
		&m.ServiceCost, &m.PowerKw, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if m.LifespanHours > 0 {
		m.HourlyDepreciation = (m.PurchasePrice + m.ServiceCost) / m.LifespanHours
	}
	return m, err
}

const machineCols = `id, name, type_tag, purchase_price, lifespan_hours, service_cost, power_kw, notes, created_at, updated_at`

func (a *API) machines() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + machineCols + ` FROM machines ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.Machine{}
			for rows.Next() {
				m, err := scanMachine(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, m)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var m models.Machine
			if err := decodeJSON(r, &m); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if m.Name == "" || m.LifespanHours <= 0 {
				writeError(w, http.StatusBadRequest, "name and lifespan_hours are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO machines (name, type_tag, purchase_price, lifespan_hours, service_cost, power_kw, notes)
				 VALUES (?, ?, ?, ?, ?, ?, ?)`,
				m.Name, m.TypeTag, m.PurchasePrice, m.LifespanHours, m.ServiceCost, m.PowerKw, m.Notes,
			)
			if err != nil {
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
			row := a.DB.QueryRow(`SELECT `+machineCols+` FROM machines WHERE id = ?`, id)
			m, err := scanMachine(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "machine not found")
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
			var m models.Machine
			if err := decodeJSON(r, &m); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE machines SET name=?, type_tag=?, purchase_price=?, lifespan_hours=?, service_cost=?, power_kw=?, notes=?, updated_at=datetime('now')
				 WHERE id=?`,
				m.Name, m.TypeTag, m.PurchasePrice, m.LifespanHours, m.ServiceCost, m.PowerKw, m.Notes, id,
			)
			if err != nil {
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
			_, err = a.DB.Exec(`DELETE FROM machines WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this printer is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
