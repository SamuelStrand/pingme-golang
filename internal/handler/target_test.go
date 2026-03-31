package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/httpx"
	"pingme-golang/internal/models"
	"pingme-golang/internal/monitor"
)

type stubTargetService struct {
	createInput    monitor.CreateInput
	createResult   models.Monitor
	createErr      error
	listInput      monitor.ListInput
	listResult     monitor.MonitorPage
	listErr        error
	updateInput    monitor.UpdateInput
	updateResult   models.Monitor
	updateErr      error
	deleteID       string
	deleteUserID   string
	deleteErr      error
	listLogsInput  monitor.ListLogsInput
	listLogsResult monitor.CheckLogPage
	listLogsErr    error
}

func (s *stubTargetService) Create(_ context.Context, input monitor.CreateInput) (models.Monitor, error) {
	s.createInput = input
	return s.createResult, s.createErr
}

func (s *stubTargetService) List(_ context.Context, input monitor.ListInput) (monitor.MonitorPage, error) {
	s.listInput = input
	return s.listResult, s.listErr
}

func (s *stubTargetService) Update(_ context.Context, input monitor.UpdateInput) (models.Monitor, error) {
	s.updateInput = input
	return s.updateResult, s.updateErr
}

func (s *stubTargetService) Delete(_ context.Context, id string, userID string) error {
	s.deleteID = id
	s.deleteUserID = userID
	return s.deleteErr
}

func (s *stubTargetService) ListLogs(_ context.Context, input monitor.ListLogsInput) (monitor.CheckLogPage, error) {
	s.listLogsInput = input
	return s.listLogsResult, s.listLogsErr
}

func TestTargetHandlerCreateDefaultsEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{
		createResult: models.Monitor{
			ID:         "monitor-1",
			URL:        "https://example.com",
			Name:       "API",
			Interval:   60,
			Enabled:    true,
			LastStatus: "unknown",
			CreatedAt:  time.Date(2026, time.April, 2, 12, 0, 0, 0, time.UTC),
		},
	}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/targets", bytes.NewBufferString(`{"url":"https://example.com","interval":60,"name":"API"}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if !service.createInput.Enabled {
		t.Fatal("Enabled = false, want true")
	}
}

func TestTargetHandlerListUsesDefaultPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{
		listResult: monitor.MonitorPage{
			Items:    []models.Monitor{{ID: "monitor-1", URL: "https://example.com", Interval: 60, Enabled: true, LastStatus: "up", CreatedAt: time.Now().UTC()}},
			Page:     1,
			PageSize: 20,
			Total:    1,
		},
	}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/targets", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if service.listInput.Page != 1 {
		t.Fatalf("Page = %d, want 1", service.listInput.Page)
	}
	if service.listInput.PageSize != 20 {
		t.Fatalf("PageSize = %d, want 20", service.listInput.PageSize)
	}
}

func TestTargetHandlerListRejectsInvalidPaginationQueries(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name      string
		rawQuery  string
		field     string
		wantValue string
	}{
		{
			name:      "page rejects non integer",
			rawQuery:  "?page=abc",
			field:     "page",
			wantValue: "must be a positive integer",
		},
		{
			name:      "page size rejects non integer",
			rawQuery:  "?page_size=abc",
			field:     "page_size",
			wantValue: "must be a positive integer",
		},
		{
			name:      "page size rejects out of range",
			rawQuery:  "?page_size=101",
			field:     "page_size",
			wantValue: "must be between 1 and 100",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			service := &stubTargetService{}
			router := newTargetTestRouter(service)

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/targets"+testCase.rawQuery, nil)

			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
			}

			var payload httpx.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if payload.Fields[testCase.field] != testCase.wantValue {
				t.Fatalf("%s = %q, want %q", testCase.field, payload.Fields[testCase.field], testCase.wantValue)
			}
		})
	}
}

func TestTargetHandlerUpdateReturnsValidationErrorOnEmptyPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{updateErr: monitor.ErrEmptyUpdate}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/targets/monitor-1", bytes.NewBufferString(`{}`))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestTargetHandlerDeleteReturnsNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/targets/monitor-1", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if service.deleteID != "monitor-1" {
		t.Fatalf("delete id = %q, want %q", service.deleteID, "monitor-1")
	}
	if service.deleteUserID != "user-1" {
		t.Fatalf("delete user = %q, want %q", service.deleteUserID, "user-1")
	}
}

func TestTargetHandlerLogsRejectsInvalidTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/targets/monitor-1/logs?from=bad-timestamp", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestTargetHandlerLogsReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubTargetService{listLogsErr: monitor.ErrNotFound}
	router := newTargetTestRouter(service)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/targets/monitor-1/logs", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestTargetHandlerLogsParsesFiltersAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	checkedAt := time.Date(2026, time.April, 2, 14, 0, 0, 0, time.UTC)
	service := &stubTargetService{
		listLogsResult: monitor.CheckLogPage{
			Items:    []models.CheckLog{{ID: "log-1", StatusCode: 200, ResponseTimeMs: 123, Success: true, CheckedAt: checkedAt}},
			Page:     2,
			PageSize: 5,
			Total:    1,
		},
	}
	router := newTargetTestRouter(service)
	fromRaw := "2026-04-01T00:00:00.123Z"
	toRaw := "2026-04-02T00:00:00.456Z"
	wantFrom, err := time.Parse(time.RFC3339Nano, fromRaw)
	if err != nil {
		t.Fatalf("Parse(from) error = %v", err)
	}
	wantTo, err := time.Parse(time.RFC3339Nano, toRaw)
	if err != nil {
		t.Fatalf("Parse(to) error = %v", err)
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/targets/monitor-1/logs?page=2&page_size=5&from="+fromRaw+"&to="+toRaw, nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if service.listLogsInput.Page != 2 {
		t.Fatalf("Page = %d, want 2", service.listLogsInput.Page)
	}
	if service.listLogsInput.PageSize != 5 {
		t.Fatalf("PageSize = %d, want 5", service.listLogsInput.PageSize)
	}
	if service.listLogsInput.From == nil || !service.listLogsInput.From.Equal(wantFrom) {
		t.Fatalf("From = %v, want %s", service.listLogsInput.From, wantFrom.Format(time.RFC3339Nano))
	}
	if service.listLogsInput.To == nil || !service.listLogsInput.To.Equal(wantTo) {
		t.Fatalf("To = %v, want %s", service.listLogsInput.To, wantTo.Format(time.RFC3339Nano))
	}

	var payload targetLogListResponse
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(payload.Items))
	}
}

func newTargetTestRouter(service targetService) *gin.Engine {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user-1")
		c.Next()
	})

	handler := &TargetHandler{Service: service}
	router.POST("/targets", handler.Create)
	router.GET("/targets", handler.List)
	router.PATCH("/targets/:id", handler.Update)
	router.DELETE("/targets/:id", handler.Delete)
	router.GET("/targets/:id/logs", handler.Logs)

	return router
}
