package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeList(w http.ResponseWriter, data any, total int) {
	writeJSON(w, http.StatusOK, map[string]any{"data": data, "total": total})
}

// isForeignKeyError reports whether err is a SQLite foreign-key-constraint
// violation, so handlers can turn it into a friendly 409 instead of a 500.
func isForeignKeyError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "constraint failed")
}
