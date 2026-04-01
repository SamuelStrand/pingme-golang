package models

import "time"

const (
	AlertChannelTypeTelegram = "telegram"
	AlertChannelTypeWebhook  = "webhook"
)

type AlertChannel struct {
	ID        string    `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Type      string    `db:"type" json:"type"`
	Address   string    `db:"address" json:"address"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
