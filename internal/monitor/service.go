// Package monitor implements the existing monitor domain layer.
// The public HTTP API exposes these resources as targets, but storage and business logic keep monitor naming.
package monitor

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"pingme-golang/internal/models"
)

const (
	MinIntervalSeconds  = 30
	MaxIntervalSeconds  = 3600
	DefaultTimeoutInSec = 5
)

var (
	ErrInvalidURL       = errors.New("invalid monitor url")
	ErrInvalidInterval  = errors.New("invalid monitor interval")
	ErrEmptyUpdate      = errors.New("empty monitor update")
	ErrNotFound         = errors.New("monitor not found")
	ErrInvalidDateRange = errors.New("invalid log date range")
)

type repository interface {
	Create(ctx context.Context, params CreateParams) (models.Monitor, error)
	ListByUserID(ctx context.Context, userID string, page int, pageSize int) ([]models.Monitor, int, error)
	GetByIDAndUserID(ctx context.Context, id string, userID string) (models.Monitor, error)
	ExistsByIDAndUserID(ctx context.Context, id string, userID string) error
	Update(ctx context.Context, params UpdateParams) (models.Monitor, error)
	Delete(ctx context.Context, id string, userID string) error
	ListLogs(ctx context.Context, params ListLogsParams) ([]models.CheckLog, int, error)
}

type Service struct {
	repo repository
}

type CreateInput struct {
	UserID   string
	URL      string
	Name     string
	Interval int
	Enabled  bool
}

type UpdateInput struct {
	UserID   string
	ID       string
	URL      *string
	Name     *string
	Interval *int
	Enabled  *bool
}

type ListInput struct {
	UserID   string
	Page     int
	PageSize int
}

type MonitorPage struct {
	Items    []models.Monitor
	Page     int
	PageSize int
	Total    int
}

type ListLogsInput struct {
	UserID    string
	MonitorID string
	Page      int
	PageSize  int
	From      *time.Time
	To        *time.Time
}

type CheckLogPage struct {
	Items    []models.CheckLog
	Page     int
	PageSize int
	Total    int
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (models.Monitor, error) {
	normalizedURL, err := normalizeURL(input.URL)
	if err != nil {
		return models.Monitor{}, err
	}

	interval, err := normalizeInterval(input.Interval)
	if err != nil {
		return models.Monitor{}, err
	}

	return s.repo.Create(ctx, CreateParams{
		UserID:   input.UserID,
		URL:      normalizedURL,
		Name:     normalizeName(input.Name),
		Interval: interval,
		Enabled:  input.Enabled,
	})
}

func (s *Service) List(ctx context.Context, input ListInput) (MonitorPage, error) {
	items, total, err := s.repo.ListByUserID(ctx, input.UserID, input.Page, input.PageSize)
	if err != nil {
		return MonitorPage{}, err
	}

	return MonitorPage{
		Items:    items,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (models.Monitor, error) {
	if input.URL == nil && input.Name == nil && input.Interval == nil && input.Enabled == nil {
		return models.Monitor{}, ErrEmptyUpdate
	}

	current, err := s.repo.GetByIDAndUserID(ctx, input.ID, input.UserID)
	if err != nil {
		return models.Monitor{}, err
	}

	if input.URL != nil {
		current.URL = *input.URL
	}
	if input.Name != nil {
		current.Name = *input.Name
	}
	if input.Interval != nil {
		current.Interval = *input.Interval
	}
	if input.Enabled != nil {
		current.Enabled = *input.Enabled
	}

	normalizedURL, err := normalizeURL(current.URL)
	if err != nil {
		return models.Monitor{}, err
	}

	interval, err := normalizeInterval(current.Interval)
	if err != nil {
		return models.Monitor{}, err
	}

	return s.repo.Update(ctx, UpdateParams{
		ID:       input.ID,
		UserID:   input.UserID,
		URL:      normalizedURL,
		Name:     normalizeName(current.Name),
		Interval: interval,
		Enabled:  current.Enabled,
	})
}

func (s *Service) Delete(ctx context.Context, id string, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func (s *Service) ListLogs(ctx context.Context, input ListLogsInput) (CheckLogPage, error) {
	if input.From != nil && input.To != nil && input.From.After(*input.To) {
		return CheckLogPage{}, ErrInvalidDateRange
	}

	if err := s.repo.ExistsByIDAndUserID(ctx, input.MonitorID, input.UserID); err != nil {
		return CheckLogPage{}, err
	}

	items, total, err := s.repo.ListLogs(ctx, ListLogsParams{
		MonitorID: input.MonitorID,
		Page:      input.Page,
		PageSize:  input.PageSize,
		From:      input.From,
		To:        input.To,
	})
	if err != nil {
		return CheckLogPage{}, err
	}

	return CheckLogPage{
		Items:    items,
		Page:     input.Page,
		PageSize: input.PageSize,
		Total:    total,
	}, nil
}

func normalizeURL(rawURL string) (string, error) {
	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return "", ErrInvalidURL
	}

	parsedURL, err := url.ParseRequestURI(trimmedURL)
	if err != nil || parsedURL.Host == "" {
		return "", ErrInvalidURL
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", ErrInvalidURL
	}

	return trimmedURL, nil
}

func normalizeInterval(interval int) (int, error) {
	if interval < MinIntervalSeconds || interval > MaxIntervalSeconds {
		return 0, ErrInvalidInterval
	}

	return interval, nil
}

func normalizeName(name string) string {
	return strings.TrimSpace(name)
}
