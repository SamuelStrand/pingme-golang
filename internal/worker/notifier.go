package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"pingme-golang/internal/models"
)

const notificationRequestTimeout = 5 * time.Second

type Notifier interface {
	Notify(ctx context.Context, event Event, channels []models.AlertChannel) error
}

type LoggingNotifier struct{}

func (n *LoggingNotifier) Notify(_ context.Context, event Event, channels []models.AlertChannel) error {
	log.Printf(
		"notification event=%s monitor_id=%s monitor_url=%s channels=%d status=%s",
		event.Type,
		event.Monitor.ID,
		event.Monitor.URL,
		len(channels),
		event.Monitor.LastStatus,
	)
	return nil
}

type AlertChannelNotifier struct {
	telegramToken string
	client        *http.Client
}

func NewAlertChannelNotifier(telegramToken string) *AlertChannelNotifier {
	return &AlertChannelNotifier{
		telegramToken: strings.TrimSpace(telegramToken),
		client: &http.Client{
			Timeout: notificationRequestTimeout,
		},
	}
}

func NewTelegramNotifier(token string) *AlertChannelNotifier {
	return NewAlertChannelNotifier(token)
}

func (n *AlertChannelNotifier) Notify(ctx context.Context, event Event, channels []models.AlertChannel) error {
	message := telegramMessage(event)
	var notifyErrs []error

	for _, channel := range channels {
		switch channel.Type {
		case models.AlertChannelTypeTelegram:
			if n.telegramToken == "" {
				log.Printf(
					"skip alert channel id=%s type=%s: TELEGRAM_BOT_TOKEN is not configured",
					channel.ID,
					channel.Type,
				)
				continue
			}

			if err := n.sendTelegramMessage(ctx, channel.Address, message); err != nil {
				notifyErrs = append(
					notifyErrs,
					fmt.Errorf("send telegram message to %q: %w", channel.Address, err),
				)
			}
		case models.AlertChannelTypeWebhook:
			if err := n.sendWebhook(ctx, channel.Address, event); err != nil {
				notifyErrs = append(
					notifyErrs,
					fmt.Errorf("send webhook notification to %q: %w", channel.Address, err),
				)
			}
		default:
			log.Printf(
				"skip unsupported alert channel id=%s type=%q address=%q",
				channel.ID,
				channel.Type,
				channel.Address,
			)
		}
	}

	return errors.Join(notifyErrs...)
}

func (n *AlertChannelNotifier) sendTelegramMessage(ctx context.Context, chatID string, message string) error {
	payload := map[string]string{
		"chat_id": chatID,
		"text":    message,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", n.telegramToken),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("perform telegram request: %w", err)
	}
	defer resp.Body.Close()

	if isHTTPSuccess(resp.StatusCode) {
		return nil
	}

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("telegram api returned status %d and body read failed: %w", resp.StatusCode, readErr)
	}

	return fmt.Errorf("telegram api returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
}

func (n *AlertChannelNotifier) sendWebhook(ctx context.Context, address string, event Event) error {
	body, err := json.Marshal(newWebhookPayload(event))
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		address,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("perform webhook request: %w", err)
	}
	defer resp.Body.Close()

	if isHTTPSuccess(resp.StatusCode) {
		return nil
	}

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if readErr != nil {
		return fmt.Errorf("webhook returned status %d and body read failed: %w", resp.StatusCode, readErr)
	}

	return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
}

func isHTTPSuccess(statusCode int) bool {
	return statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices
}

type webhookPayload struct {
	MonitorID string                `json:"monitor_id"`
	URL       string                `json:"url"`
	EventType EventType             `json:"event_type"`
	Details   webhookPayloadDetails `json:"details"`
}

type webhookPayloadDetails struct {
	MonitorName    string    `json:"monitor_name,omitempty"`
	MonitorStatus  string    `json:"monitor_status,omitempty"`
	StatusCode     int       `json:"status_code,omitempty"`
	ResponseTimeMs int       `json:"response_time_ms"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	CheckedAt      time.Time `json:"checked_at"`
	IncidentID     string    `json:"incident_id,omitempty"`
}

func newWebhookPayload(event Event) webhookPayload {
	return webhookPayload{
		MonitorID: event.Monitor.ID,
		URL:       event.Monitor.URL,
		EventType: event.Type,
		Details: webhookPayloadDetails{
			MonitorName:    event.Monitor.Name,
			MonitorStatus:  event.Monitor.LastStatus,
			StatusCode:     event.Check.StatusCode,
			ResponseTimeMs: event.Check.ResponseTimeMs,
			ErrorMessage:   event.Check.ErrorMessage,
			CheckedAt:      event.Check.CheckedAt.UTC(),
			IncidentID:     event.IncidentID,
		},
	}
}

func telegramMessage(event Event) string {
	monitorName := event.Monitor.Name
	if strings.TrimSpace(monitorName) == "" {
		monitorName = event.Monitor.URL
	}

	switch event.Type {
	case EventTypeDown:
		return fmt.Sprintf(
			"DOWN\nMonitor: %s\nURL: %s\nDetails: %s\nLatency: %d ms",
			monitorName,
			event.Monitor.URL,
			event.Check.ErrorMessage,
			event.Check.ResponseTimeMs,
		)
	case EventTypeRecovered:
		return fmt.Sprintf(
			"RECOVERED\nMonitor: %s\nURL: %s\nStatus: %d\nLatency: %d ms",
			monitorName,
			event.Monitor.URL,
			event.Check.StatusCode,
			event.Check.ResponseTimeMs,
		)
	default:
		return fmt.Sprintf("Monitor event %s for %s", event.Type, monitorName)
	}
}
