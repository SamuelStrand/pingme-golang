package handler

import (
	"database/sql"
	"net/http"

	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
)

type UserHandler struct {
	Repo *auth.Repository
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "unauthorized", nil)
		return
	}

	u, err := h.Repo.GetUserByID(r.Context(), userID)
	if err != nil {
		if err == sql.ErrNoRows {
			httpx.Error(w, http.StatusNotFound, "not_found", "not found", nil)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load user", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, u)
}
