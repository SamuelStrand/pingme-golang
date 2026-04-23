package telegramlink

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	telegramAPIBaseURL       = "https://api.telegram.org"
	telegramRequestTimeout   = 35 * time.Second
	telegramPollTimeout      = 30
	telegramRetryDelay       = 3 * time.Second
	telegramSuccessMessage   = "Telegram notifications are connected. You will receive alerts when your targets go down."
	telegramInvalidMessage   = "This Telegram link is invalid or expired. Create a new link in PingMe and open it again."
	telegramTemporaryMessage = "Telegram linking is temporarily unavailable. Try again in a few minutes."
)

type ChatLinker interface {
	LinkChat(ctx context.Context, rawToken string, chatID string) error
}

type Bot struct {
	token      string
	linker     ChatLinker
	client     *http.Client
	apiBaseURL string
}

type telegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	Text string       `json:"text"`
	Chat telegramChat `json:"chat"`
}

type telegramChat struct {
	ID int64 `json:"id"`
}

type telegramAPIResponse struct {
	OK          bool            `json:"ok"`
	Description string          `json:"description"`
	Result      json.RawMessage `json:"result"`
}

func NewBot(token string, linker ChatLinker) *Bot {
	return &Bot{
		token:  strings.TrimSpace(token),
		linker: linker,
		client: &http.Client{
			Timeout: telegramRequestTimeout,
		},
		apiBaseURL: telegramAPIBaseURL,
	}
}

func (b *Bot) Run(ctx context.Context) error {
	offset := 0

	for {
		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("telegram getUpdates failed: %v", err)
			if !sleepContext(ctx, telegramRetryDelay) {
				return nil
			}
			continue
		}

		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			b.handleUpdate(ctx, update)
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, update telegramUpdate) {
	if update.Message == nil {
		return
	}

	token, isStart := startToken(update.Message.Text)
	if !isStart {
		return
	}

	chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
	if token == "" {
		b.sendMessage(ctx, chatID, telegramInvalidMessage)
		return
	}

	if err := b.linker.LinkChat(ctx, token, chatID); err != nil {
		if errors.Is(err, ErrInvalidLinkToken) {
			b.sendMessage(ctx, chatID, telegramInvalidMessage)
			return
		}
		log.Printf("telegram link chat %s failed: %v", chatID, err)
		b.sendMessage(ctx, chatID, telegramTemporaryMessage)
		return
	}

	b.sendMessage(ctx, chatID, telegramSuccessMessage)
}

func (b *Bot) getUpdates(ctx context.Context, offset int) ([]telegramUpdate, error) {
	payload := map[string]any{
		"timeout":         telegramPollTimeout,
		"allowed_updates": []string{"message"},
	}
	if offset > 0 {
		payload["offset"] = offset
	}

	var updates []telegramUpdate
	if err := b.call(ctx, "getUpdates", payload, &updates); err != nil {
		return nil, err
	}
	return updates, nil
}

func (b *Bot) sendMessage(ctx context.Context, chatID string, text string) {
	payload := map[string]string{
		"chat_id": chatID,
		"text":    text,
	}
	if err := b.call(ctx, "sendMessage", payload, nil); err != nil && ctx.Err() == nil {
		log.Printf("telegram sendMessage to chat %s failed: %v", chatID, err)
	}
}

func (b *Bot) call(ctx context.Context, method string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal telegram %s payload: %w", method, err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/bot%s/%s", strings.TrimRight(b.apiBaseURL, "/"), b.token, method),
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("build telegram %s request: %w", method, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("perform telegram %s request: %w", method, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read telegram %s response: %w", method, err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram %s returned status %d: %s", method, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var envelope telegramAPIResponse
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode telegram %s response: %w", method, err)
	}
	if !envelope.OK {
		if envelope.Description == "" {
			envelope.Description = "unknown telegram api error"
		}
		return fmt.Errorf("telegram %s returned ok=false: %s", method, envelope.Description)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("decode telegram %s result: %w", method, err)
	}

	return nil
}

func startToken(text string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return "", false
	}

	command := fields[0]
	if command != "/start" && !strings.HasPrefix(command, "/start@") {
		return "", false
	}
	if len(fields) < 2 {
		return "", true
	}

	return fields[1], true
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
