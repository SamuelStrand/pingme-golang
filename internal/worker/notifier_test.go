package worker

import (
	"context"
	"encoding/json"
	"errors"
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

	notifier := NewAlertChannelNotifier("", SMTPConfig{})
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

	notifier := NewAlertChannelNotifier("", SMTPConfig{})
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

func TestAlertChannelNotifier_NotifySendsEmail(t *testing.T) {
	t.Parallel()

	type sentEmail struct {
		Config  SMTPConfig
		To      string
		Subject string
		Body    string
	}

	sentCh := make(chan sentEmail, 1)
	notifier := NewAlertChannelNotifier("", SMTPConfig{
		Host: "smtp.example.com",
		Port: "587",
		From: "PingMe <noreply@example.com>",
	})
	notifier.sendEmail = func(cfg SMTPConfig, to string, subject string, body string) error {
		sentCh <- sentEmail{
			Config:  cfg,
			To:      to,
			Subject: subject,
			Body:    body,
		}
		return nil
	}

	checkedAt := time.Date(2026, time.April, 2, 10, 11, 12, 0, time.UTC)
	event := Event{
		Type: EventTypeDown,
		Monitor: models.Monitor{
			ID:   "monitor-3",
			URL:  "https://example.com",
			Name: "Example",
		},
		Check: CheckResult{
			StatusCode:     http.StatusServiceUnavailable,
			ResponseTimeMs: 456,
			ErrorMessage:   "http status 503",
			CheckedAt:      checkedAt,
		},
		IncidentID: "incident-3",
	}

	err := notifier.Notify(context.Background(), event, []models.AlertChannel{
		{
			ID:      "channel-3",
			Type:    models.AlertChannelTypeEmail,
			Address: "team@example.com",
		},
	})
	if err != nil {
		t.Fatalf("Notify() unexpected error: %v", err)
	}

	select {
	case got := <-sentCh:
		if got.Config.Host != "smtp.example.com" {
			t.Fatalf("smtp host = %q, want smtp.example.com", got.Config.Host)
		}
		if got.To != "team@example.com" {
			t.Fatalf("to = %q, want team@example.com", got.To)
		}
		if !strings.Contains(got.Subject, "[PingMe] \U0001F534 Example is DOWN") {
			t.Fatalf("subject = %q, want down subject", got.Subject)
		}
		for _, want := range []string{
			"Monitor: Example",
			"URL: https://example.com",
			"Event: down",
			"Status: 503",
			"Error: http status 503",
			"Latency: 456 ms",
			"Checked at: 2026-04-02T10:11:12Z",
			"Incident: incident-3",
		} {
			if !strings.Contains(got.Body, want) {
				t.Fatalf("body = %q, want to contain %q", got.Body, want)
			}
		}
	default:
		t.Fatal("email was not sent")
	}
}

func TestAlertChannelNotifier_NotifyReturnsEmailErrors(t *testing.T) {
	t.Parallel()

	notifier := NewAlertChannelNotifier("", SMTPConfig{
		Host: "smtp.example.com",
		Port: "587",
		From: "noreply@example.com",
	})
	notifier.sendEmail = func(SMTPConfig, string, string, string) error {
		return errors.New("smtp refused")
	}

	err := notifier.Notify(context.Background(), Event{
		Type: EventTypeRecovered,
		Monitor: models.Monitor{
			ID:  "monitor-4",
			URL: "https://example.com",
		},
		Check: CheckResult{
			StatusCode:     http.StatusOK,
			ResponseTimeMs: 123,
			CheckedAt:      time.Date(2026, time.April, 2, 11, 12, 13, 0, time.UTC),
			Success:        true,
		},
	}, []models.AlertChannel{
		{
			ID:      "channel-4",
			Type:    models.AlertChannelTypeEmail,
			Address: "team@example.com",
		},
	})
	if err == nil {
		t.Fatal("Notify() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "send email notification") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "send email notification")
	}
	if !strings.Contains(err.Error(), "smtp refused") {
		t.Fatalf("error = %q, want to contain %q", err.Error(), "smtp refused")
	}
}

func TestEmailContentRecovered(t *testing.T) {
	t.Parallel()

	subject, body := emailContent(Event{
		Type: EventTypeRecovered,
		Monitor: models.Monitor{
			URL: "https://example.com",
		},
		Check: CheckResult{
			StatusCode:     http.StatusOK,
			ResponseTimeMs: 98,
			CheckedAt:      time.Date(2026, time.April, 2, 12, 13, 14, 0, time.UTC),
		},
	})

	if subject != "[PingMe] \U0001F7E2 https://example.com recovered" {
		t.Fatalf("subject = %q, want recovered subject", subject)
	}
	for _, want := range []string{
		"Monitor: https://example.com",
		"Event: recovered",
		"Status: 200",
		"Latency: 98 ms",
		"Checked at: 2026-04-02T12:13:14Z",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body = %q, want to contain %q", body, want)
		}
	}
}
