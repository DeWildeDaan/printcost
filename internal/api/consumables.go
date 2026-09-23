package api

import (
	"database/sql"
	"net/http"

	"printcost/internal/models"
)

const consumableCols = `id, name, unit_label, unit_cost, default_on_quote, notes, created_at, updated_at`

func scanConsumable(row interface{ Scan(...any) error }) (models.Consumable, error) {
	var c models.Consumable
	var defaultOnQuote int
	err := row.Scan(&c.ID, &c.Name, &c.UnitLabel, &c.UnitCost, &defaultOnQuote, &c.Notes, &c.CreatedAt, &c.UpdatedAt)
	c.DefaultOnQuote = defaultOnQuote != 0
	return c, err
}

func (a *API) consumables() resource {
	return resource{
		List: func(w http.ResponseWriter, r *http.Request) {
			rows, err := a.DB.Query(`SELECT ` + consumableCols + ` FROM consumables ORDER BY name`)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			defer rows.Close()
			list := []models.Consumable{}
			for rows.Next() {
				c, err := scanConsumable(rows)
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				list = append(list, c)
			}
			writeList(w, list, len(list))
		},
		Create: func(w http.ResponseWriter, r *http.Request) {
			var c models.Consumable
			if err := decodeJSON(r, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			if c.Name == "" || c.UnitLabel == "" {
				writeError(w, http.StatusBadRequest, "name and unit_label are required")
				return
			}
			res, err := a.DB.Exec(
				`INSERT INTO consumables (name, unit_label, unit_cost, default_on_quote, notes) VALUES (?, ?, ?, ?, ?)`,
				c.Name, c.UnitLabel, c.UnitCost, c.DefaultOnQuote, c.Notes,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			id, _ := res.LastInsertId()
			c.ID = id
			writeJSON(w, http.StatusCreated, c)
		},
		Get: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			row := a.DB.QueryRow(`SELECT `+consumableCols+` FROM consumables WHERE id = ?`, id)
			c, err := scanConsumable(row)
			if err == sql.ErrNoRows {
				writeError(w, http.StatusNotFound, "consumable not found")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, c)
		},
		Update: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			var c models.Consumable
			if err := decodeJSON(r, &c); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
			_, err = a.DB.Exec(
				`UPDATE consumables SET name=?, unit_label=?, unit_cost=?, default_on_quote=?, notes=?, updated_at=datetime('now') WHERE id=?`,
				c.Name, c.UnitLabel, c.UnitCost, c.DefaultOnQuote, c.Notes, id,
			)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			c.ID = id
			writeJSON(w, http.StatusOK, c)
		},
		Delete: func(w http.ResponseWriter, r *http.Request) {
			id, err := idParam(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "invalid id")
				return
			}
			_, err = a.DB.Exec(`DELETE FROM consumables WHERE id = ?`, id)
			if isForeignKeyError(err) {
				writeError(w, http.StatusConflict, "this consumable is used by one or more quotes and cannot be deleted")
				return
			} else if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			w.WriteHeader(http.StatusNoContent)
		},
	}
}
