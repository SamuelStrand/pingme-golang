package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
	"pingme-golang/internal/telegramlink"
)

type TelegramLinkHandler struct {
	Service telegramLinkService
}

type telegramLinkService interface {
	CreateLinkToken(ctx context.Context, userID string) (telegramlink.LinkToken, error)
}

type telegramLinkTokenResponse struct {
	LinkURL   string    `json:"link_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h *TelegramLinkHandler) CreateLinkToken(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	token, err := h.Service.CreateLinkToken(c.Request.Context(), userID)
	if err != nil {
		writeTelegramLinkError(c, err)
		return
	}

	c.JSON(http.StatusCreated, telegramLinkTokenResponse{
		LinkURL:   token.LinkURL,
		ExpiresAt: token.ExpiresAt,
	})
}

func writeTelegramLinkError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, telegramlink.ErrMissingBotUsername):
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
			Error:   "telegram_not_configured",
			Message: "telegram bot username is not configured",
		})
	default:
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to create telegram link token",
		})
	}
}
