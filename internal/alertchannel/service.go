package alertchannel

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"pingme-golang/internal/models"
)

const (
	TypeTelegram = models.AlertChannelTypeTelegram
	TypeWebhook  = models.AlertChannelTypeWebhook
)

var ErrInvalidType = errors.New("invalid alert channel type")
var ErrInvalidAddress = errors.New("invalid alert channel address")
var ErrEmptyUpdate = errors.New("empty alert channel update")

type repository interface {
	ListByUserID(ctx context.Context, userID string) ([]models.AlertChannel, error)
	Create(ctx context.Context, userID string, channelType string, address string, enabled bool) (models.AlertChannel, error)
	GetByIDAndUserID(ctx context.Context, id string, userID string) (models.AlertChannel, error)
	Update(ctx context.Context, id string, userID string, channelType string, address string, enabled bool) (models.AlertChannel, error)
	Delete(ctx context.Context, id string, userID string) error
}

type Service struct {
	repo repository
}

type CreateInput struct {
	UserID  string
	Type    string
	Address string
	Enabled bool
}

type UpdateInput struct {
	UserID  string
	ID      string
	Type    *string
	Address *string
	Enabled *bool
}

func NewService(repo repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string) ([]models.AlertChannel, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (models.AlertChannel, error) {
	channelType, address, err := normalizeAndValidate(input.Type, input.Address)
	if err != nil {
		return models.AlertChannel{}, err
	}

	return s.repo.Create(
		ctx,
		input.UserID,
		channelType,
		address,
		input.Enabled,
	)
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (models.AlertChannel, error) {
	if input.Type == nil && input.Address == nil && input.Enabled == nil {
		return models.AlertChannel{}, ErrEmptyUpdate
	}

	channel, err := s.repo.GetByIDAndUserID(ctx, input.ID, input.UserID)
	if err != nil {
		return models.AlertChannel{}, err
	}

	channelType := channel.Type
	address := channel.Address
	enabled := channel.Enabled

	if input.Type != nil {
		channelType = *input.Type
	}
	if input.Address != nil {
		address = *input.Address
	}
	if input.Enabled != nil {
		enabled = *input.Enabled
	}

	channelType, address, err = normalizeAndValidate(channelType, address)
	if err != nil {
		return models.AlertChannel{}, err
	}

	return s.repo.Update(
		ctx,
		input.ID,
		input.UserID,
		channelType,
		address,
		enabled,
	)
}

func (s *Service) Delete(ctx context.Context, id string, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}

func normalizeAndValidate(channelType string, address string) (string, string, error) {
	channelType = strings.ToLower(strings.TrimSpace(channelType))
	address = strings.TrimSpace(address)

	switch channelType {
	case TypeTelegram:
		if address == "" {
			return "", "", ErrInvalidAddress
		}
	case TypeWebhook:
		parsed, err := url.ParseRequestURI(address)
		if err != nil || parsed.Host == "" {
			return "", "", ErrInvalidAddress
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return "", "", ErrInvalidAddress
		}
	default:
		return "", "", ErrInvalidType
	}

	return channelType, address, nil
}
