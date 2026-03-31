package handler

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
)

type UserHandler struct {
	Repo *auth.Repository
}

func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	u, err := h.Repo.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, httpx.ErrorResponse{Error: "not_found", Message: "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to load user"})
		return
	}

	c.JSON(http.StatusOK, u)
}
