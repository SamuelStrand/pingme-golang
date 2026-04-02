package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type HealthHandler struct {
	DB *sqlx.DB
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	err := h.DB.PingContext(ctx)
	status := "connected"

	if err != nil {
		status = "not connected"
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "up",
			"db":     status,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "up",
		"db":     status,
	})
}

func (h *HealthHandler) Ready(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}
