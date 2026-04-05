package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pingme-golang/internal/models"
)

func TestAlertChannelNotifier_NotifySendsWebhook(t *testing.T) {
	t.Parallel()

	type receivedRequest struct {
		Method      string
		ContentType string
		Payload     webhookPayload
	}

	receivedCh := make(chan receivedRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var payload webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		receivedCh <- receivedRequest{
			Method:      r.Method,
			ContentType: r.Header.Get("Content-Type"),
			Payload:     payload,
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	checkedAt := time.Date(2026, time.April, 1, 10, 11, 12, 0, time.UTC)
	event := Event{
		Type: EventTypeDown,
		Monitor: models.Monitor{
			ID:         "monitor-1",
			URL:        "https://example.com",
			Name:       "Example",
			LastStatus: "down",
		},
		Check: CheckResult{
			StatusCode:     http.StatusServiceUnavailable,
			ResponseTimeMs: 321,
			Success:        false,
			ErrorMessage:   "http status 503",
			CheckedAt:      checkedAt,
		},
		IncidentID: "incident-1",
	}

	notifier := NewAlertChannelNotifier("")
	notifier.client = server.Client()

	err := notifier.Notify(context.Background(), event, []models.AlertChannel{
		{
			ID:      "channel-1",
			Type:    models.AlertChannelTypeWebhook,
			Address: server.URL,
		},
	})
	if err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	select {
	case got := <-receivedCh:
		if got.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", got.Method, http.MethodPost)
		}
		if got.ContentType != "application/json" {
			t.Fatalf("contentType = %q, want %q", got.ContentType, "application/json")
		}
		if got.Payload.MonitorID != event.Monitor.ID {
			t.Fatalf("monitorID = %q, want %q", got.Payload.MonitorID, event.Monitor.ID)
		}
		if got.Payload.URL != event.Monitor.URL {
			t.Fatalf("url = %q, want %q", got.Payload.URL, event.Monitor.URL)
		}
		if got.Payload.EventType != event.Type {
			t.Fatalf("eventType = %q, want %q", got.Payload.EventType, event.Type)
		}
		if got.Payload.Details.MonitorName != event.Monitor.Name {
			t.Fatalf("monitorName = %q, want %q", got.Payload.Details.MonitorName, event.Monitor.Name)
		}
		if got.Payload.Details.MonitorStatus != event.Monitor.LastStatus {
			t.Fatalf("monitorStatus = %q, want %q", got.Payload.Details.MonitorStatus, event.Monitor.LastStatus)
		}
		if got.Payload.Details.StatusCode != event.Check.StatusCode {
			t.Fatalf("statusCode = %d, want %d", got.Payload.Details.StatusCode, event.Check.StatusCode)
		}
		if got.Payload.Details.ResponseTimeMs != event.Check.ResponseTimeMs {
			t.Fatalf("responseTimeMs = %d, want %d", got.Payload.Details.ResponseTimeMs, event.Check.ResponseTimeMs)
		}
		if got.Payload.Details.ErrorMessage != event.Check.ErrorMessage {
			t.Fatalf("errorMessage = %q, want %q", got.Payload.Details.ErrorMessage, event.Check.ErrorMessage)
		}
		if !got.Payload.Details.CheckedAt.Equal(checkedAt) {
			t.Fatalf("checkedAt = %s, want %s", got.Payload.Details.CheckedAt, checkedAt)
		}
		if got.Payload.Details.IncidentID != event.IncidentID {
			t.Fatalf("incidentID = %q, want %q", got.Payload.Details.IncidentID, event.IncidentID)
		}
	default:
		t.Fatal("webhook request was not received")
	}
}

func TestAlertChannelNotifier_NotifyReturnsWebhookErrors(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream failure", http.StatusBadGateway)
	}))
	t.Cleanup(server.Close)

	notifier := NewAlertChannelNotifier("")
	notifier.client = server.Client()

	err := notifier.Notify(context.Background(), Event{
		Type: EventTypeRecovered,
		Monitor: models.Monitor{
			ID:  "monitor-2",
			URL: "https://example.com",
		},
		Check: CheckResult{
			StatusCode: http.StatusOK,
			CheckedAt:  time.Now().UTC(),
			Success:    true,
		},
	}, []models.AlertChannel{
		{
			ID:      "channel-2",
			Type:    models.AlertChannelTypeWebhook,
			Address: server.URL,
		},
	})
	if err == nil {
		t.Fatal("Notify() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "send webhook notification") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "send webhook notification")
	}
	if !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "status 502")
	}
}
