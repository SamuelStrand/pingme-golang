package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	WriteJSON(w, status, ErrorResponse{
		Error:   code,
		Message: message,
		Fields:  fields,
	})
}
