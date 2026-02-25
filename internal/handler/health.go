package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

type HealthHandler struct {
	DB *sql.DB
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	err := h.DB.Ping()
	status := "connected"

	if err != nil {
		status = "not connected"
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "up",
		"db":     status})
}
