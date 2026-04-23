package telegramlink

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

const DefaultLinkTokenTTL = 15 * time.Minute

var (
	ErrMissingBotUsername = errors.New("telegram bot username is required")
	ErrInvalidUserID      = errors.New("user id is required")
	ErrInvalidChatID      = errors.New("telegram chat id is required")
	ErrInvalidLinkToken   = errors.New("telegram link token is invalid or expired")
)

type Config struct {
	BotUsername string
	TokenTTL    time.Duration
}

type LinkToken struct {
	LinkURL   string
	ExpiresAt time.Time
}

type Repository interface {
	CreateLinkToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error
	ConsumeLinkToken(ctx context.Context, tokenHash string, chatID string, now time.Time) (string, error)
}

type Service struct {
	repo          Repository
	cfg           Config
	now           func() time.Time
	generateToken func() (string, error)
}

func LoadConfigFromEnv() Config {
	return Config{
		BotUsername: os.Getenv("TELEGRAM_BOT_USERNAME"),
		TokenTTL:    DefaultLinkTokenTTL,
	}
}

func NewService(repo Repository, cfg Config) *Service {
	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = DefaultLinkTokenTTL
	}

	return &Service{
		repo:          repo,
		cfg:           cfg,
		now:           func() time.Time { return time.Now().UTC() },
		generateToken: generateSecureToken,
	}
}

func (s *Service) CreateLinkToken(ctx context.Context, userID string) (LinkToken, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return LinkToken{}, ErrInvalidUserID
	}

	botUsername := normalizeBotUsername(s.cfg.BotUsername)
	if botUsername == "" {
		return LinkToken{}, ErrMissingBotUsername
	}

	rawToken, err := s.generateToken()
	if err != nil {
		return LinkToken{}, fmt.Errorf("generate telegram link token: %w", err)
	}

	expiresAt := s.now().UTC().Add(s.cfg.TokenTTL)
	if err := s.repo.CreateLinkToken(ctx, userID, HashToken(rawToken), expiresAt); err != nil {
		return LinkToken{}, fmt.Errorf("create telegram link token: %w", err)
	}

	return LinkToken{
		LinkURL:   fmt.Sprintf("https://t.me/%s?start=%s", botUsername, rawToken),
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) LinkChat(ctx context.Context, rawToken string, chatID string) error {
	rawToken = strings.TrimSpace(rawToken)
	chatID = strings.TrimSpace(chatID)
	if rawToken == "" {
		return ErrInvalidLinkToken
	}
	if chatID == "" {
		return ErrInvalidChatID
	}

	if _, err := s.repo.ConsumeLinkToken(ctx, HashToken(rawToken), chatID, s.now().UTC()); err != nil {
		if errors.Is(err, ErrInvalidLinkToken) {
			return ErrInvalidLinkToken
		}
		return fmt.Errorf("consume telegram link token: %w", err)
	}

	return nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateSecureToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
}

func normalizeBotUsername(username string) string {
	return strings.TrimPrefix(strings.TrimSpace(username), "@")
}
