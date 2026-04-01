package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/alertchannel"
	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
)

type AlertChannelHandler struct {
	Service *alertchannel.Service
}

type createAlertChannelRequest struct {
	Type    string `json:"type"`
	Address string `json:"address"`
	Enabled *bool  `json:"enabled"`
}

type updateAlertChannelRequest struct {
	Type    *string `json:"type"`
	Address *string `json:"address"`
	Enabled *bool   `json:"enabled"`
}

func (h *AlertChannelHandler) List(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	channels, err := h.Service.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to load alert channels"})
		return
	}

	c.JSON(http.StatusOK, channels)
}

func (h *AlertChannelHandler) Create(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	var req createAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	channel, err := h.Service.Create(c.Request.Context(), alertchannel.CreateInput{
		UserID:  userID,
		Type:    req.Type,
		Address: req.Address,
		Enabled: enabled,
	})
	if err != nil {
		writeAlertChannelError(c, err)
		return
	}

	c.JSON(http.StatusCreated, channel)
}

func (h *AlertChannelHandler) Update(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	var req updateAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}

	if req.Type != nil {
		normalizedType := strings.TrimSpace(*req.Type)
		req.Type = &normalizedType
	}
	if req.Address != nil {
		normalizedAddress := strings.TrimSpace(*req.Address)
		req.Address = &normalizedAddress
	}

	channel, err := h.Service.Update(c.Request.Context(), alertchannel.UpdateInput{
		UserID:  userID,
		ID:      c.Param("id"),
		Type:    req.Type,
		Address: req.Address,
		Enabled: req.Enabled,
	})
	if err != nil {
		writeAlertChannelError(c, err)
		return
	}

	c.JSON(http.StatusOK, channel)
}

func (h *AlertChannelHandler) Delete(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	err := h.Service.Delete(c.Request.Context(), c.Param("id"), userID)
	if err != nil {
		writeAlertChannelError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func writeAlertChannelError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, alertchannel.ErrInvalidType):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid alert channel type",
			Fields:  map[string]string{"type": "must be one of: telegram, webhook"},
		})
	case errors.Is(err, alertchannel.ErrInvalidAddress):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid alert channel address",
			Fields:  map[string]string{"address": "invalid address for the selected type"},
		})
	case errors.Is(err, alertchannel.ErrEmptyUpdate):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "empty update payload",
		})
	case errors.Is(err, alertchannel.ErrDuplicate):
		c.JSON(http.StatusConflict, httpx.ErrorResponse{
			Error:   "conflict",
			Message: "alert channel already exists",
		})
	case errors.Is(err, alertchannel.ErrNotFound):
		c.JSON(http.StatusNotFound, httpx.ErrorResponse{
			Error:   "not_found",
			Message: "alert channel not found",
		})
	default:
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to process alert channel",
		})
	}
}
