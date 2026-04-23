package telegramlink

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type PostgresRepository struct {
	DB *sqlx.DB
}

func (r *PostgresRepository) CreateLinkToken(ctx context.Context, userID string, tokenHash string, expiresAt time.Time) error {
	_, err := r.DB.ExecContext(ctx, `
		insert into telegram_link_tokens (user_id, token_hash, expires_at)
		values ($1, $2, $3)
	`, userID, tokenHash, expiresAt.UTC())
	return err
}

func (r *PostgresRepository) ConsumeLinkToken(ctx context.Context, tokenHash string, chatID string, now time.Time) (string, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin telegram link tx: %w", err)
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRowxContext(ctx, `
		update telegram_link_tokens
		set used_at = $1
		where token_hash = $2
			and used_at is null
			and expires_at > $1
		returning user_id
	`, now.UTC(), tokenHash).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidLinkToken
		}
		return "", fmt.Errorf("consume telegram link token: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		insert into alert_channels (user_id, type, address, enabled)
		values ($1, 'telegram', $2, true)
		on conflict (user_id, type, address)
		do update set enabled = true
	`, userID, chatID)
	if err != nil {
		return "", fmt.Errorf("upsert telegram alert channel: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit telegram link tx: %w", err)
	}

	return userID, nil
}
