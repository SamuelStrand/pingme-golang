package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/httpx"
	"pingme-golang/internal/monitor"
)

type statusPageService interface {
	GetPublicStatus(ctx context.Context, slug string, from, to time.Time) (monitor.PublicStatus, error)
}

type StatusPageHandler struct {
	Service statusPageService
}

func (h *StatusPageHandler) Get(c *gin.Context) {
	now := time.Now().UTC()
	to := now

	duration, err := parseStatusPageRange(c.DefaultQuery("range", "24h"))
	if err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid status page range",
			Fields:  map[string]string{"range": "must be a valid duration, for example 30m, 24h, 7d"},
		})
		return
	}

	from := now.Add(-duration)

	if rawFrom := c.Query("from"); rawFrom != "" {
		parsedFrom, err := time.Parse(time.RFC3339, rawFrom)
		if err != nil {
			c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
				Error:   "invalid_time_format",
				Message: "from/to must be RFC3339",
			})
			return
		}
		from = parsedFrom
	}

	if rawTo := c.Query("to"); rawTo != "" {
		parsedTo, err := time.Parse(time.RFC3339, rawTo)
		if err != nil {
			c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
				Error:   "invalid_time_format",
				Message: "from/to must be RFC3339",
			})
			return
		}
		to = parsedTo
	}

	if from.After(to) {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "invalid_range",
			Message: "from must be before to",
		})
		return
	}

	result, err := h.Service.GetPublicStatus(c.Request.Context(), c.Param("slug"), from, to)
	if err != nil {
		switch {
		case errors.Is(err, monitor.ErrNotFound):
			c.JSON(http.StatusNotFound, httpx.ErrorResponse{
				Error:   "not_found",
				Message: "status page not found",
			})
		case errors.Is(err, monitor.ErrInvalidSlug):
			c.JSON(http.StatusNotFound, httpx.ErrorResponse{
				Error:   "not_found",
				Message: "status page not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
				Error:   "internal_error",
				Message: "failed to load status page",
			})
		}
		return
	}

	uptime := 0.0
	if result.Stats.TotalChecks > 0 {
		uptime = float64(result.Stats.SuccessCount) / float64(result.Stats.TotalChecks) * 100
	}

	c.JSON(http.StatusOK, gin.H{
		"monitor_name":    result.Monitor.Name,
		"url":             result.Monitor.URL,
		"status":          result.Monitor.LastStatus,
		"uptime_percent":  uptime,
		"avg_response_ms": result.Stats.AvgResponseMs,
		"timeline":        result.Timeline,
	})
}

func parseStatusPageRange(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 24 * time.Hour, nil
	}

	if strings.HasSuffix(raw, "d") {
		daysRaw := strings.TrimSuffix(raw, "d")
		days, err := strconv.Atoi(daysRaw)
		if err != nil || days <= 0 {
			return 0, errors.New("invalid status page range")
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil || duration <= 0 {
		return 0, errors.New("invalid status page range")
	}

	return duration, nil
}
