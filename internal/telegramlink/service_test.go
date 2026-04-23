package telegramlink

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeRepository struct {
	createdUserID    string
	createdHash      string
	createdExpiresAt time.Time
	createErr        error

	consumeHash   string
	consumeChatID string
	consumeNow    time.Time
	consumeErr    error
}

func (r *fakeRepository) CreateLinkToken(_ context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	r.createdUserID = userID
	r.createdHash = tokenHash
	r.createdExpiresAt = expiresAt
	return r.createErr
}

func (r *fakeRepository) ConsumeLinkToken(_ context.Context, tokenHash string, chatID string, now time.Time) (string, error) {
	r.consumeHash = tokenHash
	r.consumeChatID = chatID
	r.consumeNow = now
	if r.consumeErr != nil {
		return "", r.consumeErr
	}
	return "user-1", nil
}

func TestServiceCreateLinkToken(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 23, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{}
	service := NewService(repo, Config{
		BotUsername: "@pingme_bot",
		TokenTTL:    DefaultLinkTokenTTL,
	})
	service.now = func() time.Time { return now }
	service.generateToken = func() (string, error) { return "raw-token", nil }

	got, err := service.CreateLinkToken(context.Background(), " user-1 ")
	if err != nil {
		t.Fatalf("CreateLinkToken() unexpected error: %v", err)
	}

	if got.LinkURL != "https://t.me/pingme_bot?start=raw-token" {
		t.Fatalf("link url = %q, want %q", got.LinkURL, "https://t.me/pingme_bot?start=raw-token")
	}
	if !got.ExpiresAt.Equal(now.Add(DefaultLinkTokenTTL)) {
		t.Fatalf("expiresAt = %s, want %s", got.ExpiresAt, now.Add(DefaultLinkTokenTTL))
	}
	if repo.createdUserID != "user-1" {
		t.Fatalf("created userID = %q, want %q", repo.createdUserID, "user-1")
	}
	if repo.createdHash == "raw-token" {
		t.Fatal("raw token was stored instead of a hash")
	}
	if repo.createdHash != HashToken("raw-token") {
		t.Fatalf("created hash = %q, want %q", repo.createdHash, HashToken("raw-token"))
	}
	if !repo.createdExpiresAt.Equal(now.Add(DefaultLinkTokenTTL)) {
		t.Fatalf("created expiresAt = %s, want %s", repo.createdExpiresAt, now.Add(DefaultLinkTokenTTL))
	}
}

func TestServiceCreateLinkTokenRequiresBotUsername(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{}, Config{})
	service.generateToken = func() (string, error) { return "raw-token", nil }

	_, err := service.CreateLinkToken(context.Background(), "user-1")
	if !errors.Is(err, ErrMissingBotUsername) {
		t.Fatalf("err = %v, want %v", err, ErrMissingBotUsername)
	}
}

func TestServiceLinkChatConsumesTokenHash(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 23, 12, 1, 0, 0, time.UTC)
	repo := &fakeRepository{}
	service := NewService(repo, Config{})
	service.now = func() time.Time { return now }

	err := service.LinkChat(context.Background(), " raw-token ", " 12345 ")
	if err != nil {
		t.Fatalf("LinkChat() unexpected error: %v", err)
	}

	if repo.consumeHash != HashToken("raw-token") {
		t.Fatalf("consume hash = %q, want %q", repo.consumeHash, HashToken("raw-token"))
	}
	if repo.consumeHash == "raw-token" {
		t.Fatal("raw token was consumed instead of a hash")
	}
	if repo.consumeChatID != "12345" {
		t.Fatalf("consume chatID = %q, want %q", repo.consumeChatID, "12345")
	}
	if !repo.consumeNow.Equal(now) {
		t.Fatalf("consume now = %s, want %s", repo.consumeNow, now)
	}
}

func TestServiceLinkChatRejectsExpiredOrUsedToken(t *testing.T) {
	t.Parallel()

	service := NewService(&fakeRepository{consumeErr: ErrInvalidLinkToken}, Config{})

	err := service.LinkChat(context.Background(), "raw-token", "12345")
	if !errors.Is(err, ErrInvalidLinkToken) {
		t.Fatalf("err = %v, want %v", err, ErrInvalidLinkToken)
	}
}
