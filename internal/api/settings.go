package api

import (
	"net/http"
	"strconv"

	"printcost/internal/models"
)

func (a *API) loadSettings() (models.Settings, error) {
	rows, err := a.DB.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return models.Settings{}, err
	}
	defer rows.Close()

	var s models.Settings
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return s, err
		}
		switch key {
		case "energy_cost":
			s.EnergyCost, _ = strconv.ParseFloat(value, 64)
		case "labor_rate":
			s.LaborRate, _ = strconv.ParseFloat(value, 64)
		case "failure_rate":
			s.FailureRate, _ = strconv.ParseFloat(value, 64)
		case "markup":
			s.Markup, _ = strconv.ParseFloat(value, 64)
		case "currency":
			s.Currency = value
		case "company_name":
			s.CompanyName = value
		}
	}
	return s, nil
}

func (a *API) getSettings(w http.ResponseWriter, r *http.Request) {
	s, err := a.loadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (a *API) putSettings(w http.ResponseWriter, r *http.Request) {
	var s models.Settings
	if err := decodeJSON(r, &s); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	values := map[string]string{
		"energy_cost":  strconv.FormatFloat(s.EnergyCost, 'f', -1, 64),
		"labor_rate":   strconv.FormatFloat(s.LaborRate, 'f', -1, 64),
		"failure_rate": strconv.FormatFloat(s.FailureRate, 'f', -1, 64),
		"markup":       strconv.FormatFloat(s.Markup, 'f', -1, 64),
		"currency":     s.Currency,
		"company_name": s.CompanyName,
	}
	for key, value := range values {
		if _, err := a.DB.Exec(
			`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
			key, value,
		); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, s)
}
