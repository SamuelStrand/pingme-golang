package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/models"
	"pingme-golang/internal/monitor"
)

type stubStatusPageService struct {
	slug   string
	from   time.Time
	to     time.Time
	result monitor.PublicStatus
	err    error
}

func (s *stubStatusPageService) GetPublicStatus(_ context.Context, slug string, from, to time.Time) (monitor.PublicStatus, error) {
	s.slug = slug
	s.from = from
	s.to = to
	return s.result, s.err
}

func TestStatusPageHandlerGetReturnsPublicStatusWithoutAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	checkedAt := time.Date(2026, time.May, 19, 10, 0, 0, 0, time.UTC)
	service := &stubStatusPageService{
		result: monitor.PublicStatus{
			Monitor: models.Monitor{
				Name:       "My API",
				URL:        "https://example.com",
				LastStatus: "up",
			},
			Stats: monitor.MonitorStats{
				TotalChecks:   4,
				SuccessCount:  3,
				AvgResponseMs: 120,
			},
			Timeline: []monitor.TimelinePoint{
				{
					Timestamp:      checkedAt,
					Success:        true,
					ResponseTimeMs: 100,
				},
			},
		},
	}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/status/my-api", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if service.slug != "my-api" {
		t.Fatalf("slug = %q, want %q", service.slug, "my-api")
	}

	var payload map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if payload["monitor_name"] != "My API" {
		t.Fatalf("monitor_name = %v, want My API", payload["monitor_name"])
	}
	if payload["url"] != "https://example.com" {
		t.Fatalf("url = %v, want https://example.com", payload["url"])
	}
	if payload["status"] != "up" {
		t.Fatalf("status = %v, want up", payload["status"])
	}
	if payload["uptime_percent"] != float64(75) {
		t.Fatalf("uptime_percent = %v, want 75", payload["uptime_percent"])
	}
	if _, ok := payload["user_id"]; ok {
		t.Fatal("response leaked user_id")
	}
	if _, ok := payload["monitor_id"]; ok {
		t.Fatal("response leaked monitor_id")
	}
}

func TestStatusPageHandlerGetReturnsNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubStatusPageService{err: monitor.ErrNotFound}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/status/missing-api", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestStatusPageHandlerGetRejectsInvalidRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubStatusPageService{}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/status/my-api?from=2026-05-20T00:00:00Z&to=2026-05-19T00:00:00Z",
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestStatusPageHandlerGetParsesCustomRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubStatusPageService{
		result: monitor.PublicStatus{
			Monitor: models.Monitor{
				Name:       "My API",
				URL:        "https://example.com",
				LastStatus: "up",
			},
		},
	}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/status/my-api?range=6h", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	gotDuration := service.to.Sub(service.from)
	wantDuration := 6 * time.Hour

	if gotDuration < wantDuration-time.Second || gotDuration > wantDuration+time.Second {
		t.Fatalf("range duration = %s, want about %s", gotDuration, wantDuration)
	}
}

func TestStatusPageHandlerGetRejectsInvalidCustomRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubStatusPageService{}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/status/my-api?range=bad-range", nil)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestStatusPageHandlerGetFromToOverrideRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &stubStatusPageService{
		result: monitor.PublicStatus{
			Monitor: models.Monitor{
				Name:       "My API",
				URL:        "https://example.com",
				LastStatus: "up",
			},
		},
	}

	router := gin.New()
	handler := &StatusPageHandler{Service: service}
	router.GET("/status/:slug", handler.Get)

	fromRaw := "2026-05-20T00:00:00Z"
	toRaw := "2026-05-20T03:00:00Z"

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/status/my-api?range=24h&from="+fromRaw+"&to="+toRaw,
		nil,
	)

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	wantFrom, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		t.Fatalf("Parse(from) error = %v", err)
	}
	wantTo, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		t.Fatalf("Parse(to) error = %v", err)
	}

	if !service.from.Equal(wantFrom) {
		t.Fatalf("from = %s, want %s", service.from, wantFrom)
	}
	if !service.to.Equal(wantTo) {
		t.Fatalf("to = %s, want %s", service.to, wantTo)
	}
}
