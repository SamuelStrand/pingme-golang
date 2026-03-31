package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
	"pingme-golang/internal/models"
	"pingme-golang/internal/monitor"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

var (
	errInvalidInteger = errors.New("invalid integer")
	errOutOfRange     = errors.New("integer out of range")
)

type targetService interface {
	Create(ctx context.Context, input monitor.CreateInput) (models.Monitor, error)
	List(ctx context.Context, input monitor.ListInput) (monitor.MonitorPage, error)
	Update(ctx context.Context, input monitor.UpdateInput) (models.Monitor, error)
	Delete(ctx context.Context, id string, userID string) error
	ListLogs(ctx context.Context, input monitor.ListLogsInput) (monitor.CheckLogPage, error)
}

type TargetHandler struct {
	Service targetService
}

type createTargetRequest struct {
	URL      string `json:"url"`
	Interval int    `json:"interval"`
	Enabled  *bool  `json:"enabled"`
	Name     string `json:"name"`
}

type updateTargetRequest struct {
	URL      *string `json:"url"`
	Interval *int    `json:"interval"`
	Enabled  *bool   `json:"enabled"`
	Name     *string `json:"name"`
}

type targetResponse struct {
	ID            string     `json:"id"`
	URL           string     `json:"url"`
	Name          string     `json:"name"`
	Interval      int        `json:"interval"`
	Enabled       bool       `json:"enabled"`
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type targetListResponse struct {
	Items    []targetResponse `json:"items"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
	Total    int              `json:"total"`
}

type targetLogResponse struct {
	ID             string    `json:"id"`
	StatusCode     int       `json:"status_code"`
	ResponseTimeMs int       `json:"response_time_ms"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message"`
	CheckedAt      time.Time `json:"checked_at"`
}

type targetLogListResponse struct {
	Items    []targetLogResponse `json:"items"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Total    int                 `json:"total"`
}

func (h *TargetHandler) Create(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	var req createTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	item, err := h.Service.Create(c.Request.Context(), monitor.CreateInput{
		UserID:   userID,
		URL:      req.URL,
		Name:     req.Name,
		Interval: req.Interval,
		Enabled:  enabled,
	})
	if err != nil {
		writeTargetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, newTargetResponse(item))
}

func (h *TargetHandler) List(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	page, pageSize, fields, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid pagination query",
			Fields:  fields,
		})
		return
	}

	result, err := h.Service.List(c.Request.Context(), monitor.ListInput{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to load targets",
		})
		return
	}

	items := make([]targetResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, newTargetResponse(item))
	}

	c.JSON(http.StatusOK, targetListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func (h *TargetHandler) Update(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	var req updateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}

	item, err := h.Service.Update(c.Request.Context(), monitor.UpdateInput{
		UserID:   userID,
		ID:       c.Param("id"),
		URL:      req.URL,
		Name:     req.Name,
		Interval: req.Interval,
		Enabled:  req.Enabled,
	})
	if err != nil {
		writeTargetError(c, err)
		return
	}

	c.JSON(http.StatusOK, newTargetResponse(item))
}

func (h *TargetHandler) Delete(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	if err := h.Service.Delete(c.Request.Context(), c.Param("id"), userID); err != nil {
		writeTargetError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *TargetHandler) Logs(c *gin.Context) {
	userID, ok := auth.UserIDFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "unauthorized"})
		return
	}

	page, pageSize, fields, err := parsePagination(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid pagination query",
			Fields:  fields,
		})
		return
	}

	from, fromErr := parseRFC3339Query(c.Query("from"))
	if fromErr != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid logs query",
			Fields:  map[string]string{"from": "must be a valid RFC3339 timestamp"},
		})
		return
	}

	to, toErr := parseRFC3339Query(c.Query("to"))
	if toErr != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid logs query",
			Fields:  map[string]string{"to": "must be a valid RFC3339 timestamp"},
		})
		return
	}

	result, err := h.Service.ListLogs(c.Request.Context(), monitor.ListLogsInput{
		UserID:    userID,
		MonitorID: c.Param("id"),
		Page:      page,
		PageSize:  pageSize,
		From:      from,
		To:        to,
	})
	if err != nil {
		writeTargetError(c, err)
		return
	}

	items := make([]targetLogResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, targetLogResponse{
			ID:             item.ID,
			StatusCode:     item.StatusCode,
			ResponseTimeMs: item.ResponseTimeMs,
			Success:        item.Success,
			ErrorMessage:   item.ErrorMessage,
			CheckedAt:      item.CheckedAt,
		})
	}

	c.JSON(http.StatusOK, targetLogListResponse{
		Items:    items,
		Page:     result.Page,
		PageSize: result.PageSize,
		Total:    result.Total,
	})
}

func newTargetResponse(item models.Monitor) targetResponse {
	return targetResponse{
		ID:            item.ID,
		URL:           item.URL,
		Name:          item.Name,
		Interval:      item.Interval,
		Enabled:       item.Enabled,
		Status:        item.LastStatus,
		LastCheckedAt: item.LastCheckedAt,
		CreatedAt:     item.CreatedAt,
	}
}

func writeTargetError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, monitor.ErrInvalidURL):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid target payload",
			Fields:  map[string]string{"url": "must be a valid http or https URL"},
		})
	case errors.Is(err, monitor.ErrInvalidInterval):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid target payload",
			Fields: map[string]string{
				"interval": fmt.Sprintf(
					"must be between %d and %d seconds",
					monitor.MinIntervalSeconds,
					monitor.MaxIntervalSeconds,
				),
			},
		})
	case errors.Is(err, monitor.ErrEmptyUpdate):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "empty update payload",
		})
	case errors.Is(err, monitor.ErrInvalidDateRange):
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{
			Error:   "validation_error",
			Message: "invalid date range",
			Fields: map[string]string{
				"from": "must be before or equal to to",
				"to":   "must be after or equal to from",
			},
		})
	case errors.Is(err, monitor.ErrNotFound):
		c.JSON(http.StatusNotFound, httpx.ErrorResponse{
			Error:   "not_found",
			Message: "target not found",
		})
	default:
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{
			Error:   "internal_error",
			Message: "failed to process target",
		})
	}
}

func parsePagination(c *gin.Context) (int, int, map[string]string, error) {
	page, pageErr := parsePositiveInt(c.Query("page"), defaultPage)
	pageSize, pageSizeErr := parsePositiveInt(c.Query("page_size"), defaultPageSize)

	fields := map[string]string{}
	if pageErr != nil {
		fields["page"] = "must be a positive integer"
	}
	switch {
	case errors.Is(pageSizeErr, errInvalidInteger):
		fields["page_size"] = "must be a positive integer"
	case errors.Is(pageSizeErr, errOutOfRange):
		fields["page_size"] = fmt.Sprintf("must be between 1 and %d", maxPageSize)
	}
	if len(fields) > 0 {
		return 0, 0, fields, errors.New("invalid pagination query")
	}
	if pageSize > maxPageSize {
		return 0, 0, map[string]string{
			"page_size": fmt.Sprintf("must be between 1 and %d", maxPageSize),
		}, errors.New("invalid pagination query")
	}

	return page, pageSize, nil, nil
}

func parsePositiveInt(raw string, defaultValue int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errInvalidInteger
	}
	if value < 1 {
		return 0, errOutOfRange
	}

	return value, nil
}

func parseRFC3339Query(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	layouts := []string{time.RFC3339, time.RFC3339Nano}
	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return &parsed, nil
		}
		lastErr = err
	}

	return nil, lastErr
}
