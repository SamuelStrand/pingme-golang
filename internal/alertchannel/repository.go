package alertchannel

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"pingme-golang/internal/models"
)

var ErrNotFound = errors.New("alert channel not found")
var ErrDuplicate = errors.New("alert channel already exists")

type Repository struct {
	DB *sqlx.DB
}

func (r *Repository) ListByUserID(ctx context.Context, userID string) ([]models.AlertChannel, error) {
	channels := []models.AlertChannel{}
	err := r.DB.SelectContext(ctx, &channels, `
		select id, user_id, type, address, enabled, created_at
		from alert_channels
		where user_id = $1
		order by created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (r *Repository) Create(
	ctx context.Context,
	userID string,
	channelType string,
	address string,
	enabled bool,
) (models.AlertChannel, error) {
	var channel models.AlertChannel
	err := r.DB.QueryRowxContext(ctx, `
		insert into alert_channels (user_id, type, address, enabled)
		values ($1, $2, $3, $4)
		returning id, user_id, type, address, enabled, created_at
	`, userID, channelType, address, enabled).Scan(
		&channel.ID,
		&channel.UserID,
		&channel.Type,
		&channel.Address,
		&channel.Enabled,
		&channel.CreatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == "23505" {
			return models.AlertChannel{}, ErrDuplicate
		}
		return models.AlertChannel{}, err
	}

	return channel, nil
}

func (r *Repository) GetByIDAndUserID(ctx context.Context, id string, userID string) (models.AlertChannel, error) {
	var channel models.AlertChannel
	err := r.DB.QueryRowxContext(ctx, `
		select id, user_id, type, address, enabled, created_at
		from alert_channels
		where id = $1 and user_id = $2
	`, id, userID).Scan(
		&channel.ID,
		&channel.UserID,
		&channel.Type,
		&channel.Address,
		&channel.Enabled,
		&channel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.AlertChannel{}, ErrNotFound
		}
		return models.AlertChannel{}, err
	}

	return channel, nil
}

func (r *Repository) Update(
	ctx context.Context,
	id string,
	userID string,
	channelType string,
	address string,
	enabled bool,
) (models.AlertChannel, error) {
	var channel models.AlertChannel
	err := r.DB.QueryRowxContext(ctx, `
		update alert_channels
		set type = $1, address = $2, enabled = $3
		where id = $4 and user_id = $5
		returning id, user_id, type, address, enabled, created_at
	`, channelType, address, enabled, id, userID).Scan(
		&channel.ID,
		&channel.UserID,
		&channel.Type,
		&channel.Address,
		&channel.Enabled,
		&channel.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.AlertChannel{}, ErrNotFound
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == "23505" {
			return models.AlertChannel{}, ErrDuplicate
		}
		return models.AlertChannel{}, err
	}

	return channel, nil
}

func (r *Repository) Delete(ctx context.Context, id string, userID string) error {
	res, err := r.DB.ExecContext(ctx, `
		delete from alert_channels
		where id = $1 and user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
