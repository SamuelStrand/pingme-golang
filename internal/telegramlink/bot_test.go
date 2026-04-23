package telegramlink

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeLinker struct {
	token  string
	chatID string
	calls  int
	err    error
}

func (l *fakeLinker) LinkChat(_ context.Context, rawToken string, chatID string) error {
	l.calls++
	l.token = rawToken
	l.chatID = chatID
	return l.err
}

type sentTelegramMessage struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func TestBotHandleStartLinksChat(t *testing.T) {
	t.Parallel()

	sentMessages := make(chan sentTelegramMessage, 1)
	server := telegramSendMessageServer(t, sentMessages)
	t.Cleanup(server.Close)

	linker := &fakeLinker{}
	bot := NewBot("secret-token", linker)
	bot.apiBaseURL = server.URL
	bot.client = server.Client()

	bot.handleUpdate(context.Background(), telegramUpdate{
		Message: &telegramMessage{
			Text: "/start raw-token",
			Chat: telegramChat{ID: 12345},
		},
	})

	if linker.calls != 1 {
		t.Fatalf("LinkChat calls = %d, want 1", linker.calls)
	}
	if linker.token != "raw-token" {
		t.Fatalf("linked token = %q, want %q", linker.token, "raw-token")
	}
	if linker.chatID != "12345" {
		t.Fatalf("linked chatID = %q, want %q", linker.chatID, "12345")
	}

	message := receiveTelegramMessage(t, sentMessages)
	if message.ChatID != "12345" {
		t.Fatalf("message chatID = %q, want %q", message.ChatID, "12345")
	}
	if !strings.Contains(message.Text, "connected") {
		t.Fatalf("message text = %q, want confirmation", message.Text)
	}
}

func TestBotHandleStartWithoutTokenReturnsInvalidMessage(t *testing.T) {
	t.Parallel()

	sentMessages := make(chan sentTelegramMessage, 1)
	server := telegramSendMessageServer(t, sentMessages)
	t.Cleanup(server.Close)

	linker := &fakeLinker{}
	bot := NewBot("secret-token", linker)
	bot.apiBaseURL = server.URL
	bot.client = server.Client()

	bot.handleUpdate(context.Background(), telegramUpdate{
		Message: &telegramMessage{
			Text: "/start",
			Chat: telegramChat{ID: 12345},
		},
	})

	if linker.calls != 0 {
		t.Fatalf("LinkChat calls = %d, want 0", linker.calls)
	}

	message := receiveTelegramMessage(t, sentMessages)
	if !strings.Contains(message.Text, "invalid or expired") {
		t.Fatalf("message text = %q, want invalid token message", message.Text)
	}
}

func TestBotHandleStartInvalidTokenReturnsInvalidMessage(t *testing.T) {
	t.Parallel()

	sentMessages := make(chan sentTelegramMessage, 1)
	server := telegramSendMessageServer(t, sentMessages)
	t.Cleanup(server.Close)

	linker := &fakeLinker{err: ErrInvalidLinkToken}
	bot := NewBot("secret-token", linker)
	bot.apiBaseURL = server.URL
	bot.client = server.Client()

	bot.handleUpdate(context.Background(), telegramUpdate{
		Message: &telegramMessage{
			Text: "/start expired-token",
			Chat: telegramChat{ID: 12345},
		},
	})

	if linker.calls != 1 {
		t.Fatalf("LinkChat calls = %d, want 1", linker.calls)
	}

	message := receiveTelegramMessage(t, sentMessages)
	if !strings.Contains(message.Text, "invalid or expired") {
		t.Fatalf("message text = %q, want invalid token message", message.Text)
	}
}

func TestBotHandleStartTemporaryFailure(t *testing.T) {
	t.Parallel()

	sentMessages := make(chan sentTelegramMessage, 1)
	server := telegramSendMessageServer(t, sentMessages)
	t.Cleanup(server.Close)

	linker := &fakeLinker{err: errors.New("database unavailable")}
	bot := NewBot("secret-token", linker)
	bot.apiBaseURL = server.URL
	bot.client = server.Client()

	bot.handleUpdate(context.Background(), telegramUpdate{
		Message: &telegramMessage{
			Text: "/start raw-token",
			Chat: telegramChat{ID: 12345},
		},
	})

	message := receiveTelegramMessage(t, sentMessages)
	if !strings.Contains(message.Text, "temporarily unavailable") {
		t.Fatalf("message text = %q, want temporary failure message", message.Text)
	}
}

func telegramSendMessageServer(t *testing.T, sentMessages chan<- sentTelegramMessage) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if !strings.HasSuffix(r.URL.Path, "/sendMessage") {
			t.Fatalf("path = %s, want suffix /sendMessage", r.URL.Path)
		}

		var message sentTelegramMessage
		if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
			t.Fatalf("decode sendMessage payload: %v", err)
		}

		sentMessages <- message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
}

func receiveTelegramMessage(t *testing.T, sentMessages <-chan sentTelegramMessage) sentTelegramMessage {
	t.Helper()

	select {
	case message := <-sentMessages:
		return message
	case <-time.After(time.Second):
		t.Fatal("telegram sendMessage request was not received")
		return sentTelegramMessage{}
	}
}
