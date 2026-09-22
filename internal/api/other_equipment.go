package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const otherEquipmentCols = `id, name, purchase_price, lifespan_hours, service_cost, power_kw, notes, created_at, updated_at`

func scanOtherEquipment(row interface{ Scan(...any) error }) (models.OtherEquipment, error) {
	var e models.OtherEquipment
	err := row.Scan(&e.ID, &e.Name, &e.PurchasePrice, &e.LifespanHours, &e.ServiceCost, &e.PowerKw, &e.Notes, &e.CreatedAt, &e.UpdatedAt)
	if e.LifespanHours > 0 {
		e.HourlyCost = (e.PurchasePrice + e.ServiceCost) / e.LifespanHours
	}
	return e, err
}

func (a *API) otherEquipment() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + otherEquipmentCols + ` FROM other_equipment ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.OtherEquipment{}
			for rows.Next() {
				e, err := scanOtherEquipment(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, e)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var e models.OtherEquipment
			if err := decodeJSON(r, &e); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if e.Name == "" || e.LifespanHours <= 0 {
				writeError(w, http.StatusBadRequest, "name and lifespan_hours are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO other_equipment (name, purchase_price, lifespan_hours, service_cost, power_kw, notes) VALUES (?, ?, ?, ?, ?, ?)`,
				e.Name, e.PurchasePrice, e.LifespanHours, e.ServiceCost, e.PowerKw, e.Notes,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			e.ID = id
			writeJSON(w, http.StatusCreated, e)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+otherEquipmentCols+` FROM other_equipment WHERE id = ?`, id)
			e, err := scanOtherEquipment(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "equipment not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, e)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var e models.OtherEquipment
			if err := decodeJSON(r, &e); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE other_equipment SET name=?, purchase_price=?, lifespan_hours=?, service_cost=?, power_kw=?, notes=?, updated_at=datetime('now') WHERE id=?`,
				e.Name, e.PurchasePrice, e.LifespanHours, e.ServiceCost, e.PowerKw, e.Notes, id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			e.ID = id
			writeJSON(w, http.StatusOK, e)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM other_equipment WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this equipment is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
