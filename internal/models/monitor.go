package models

import "time"

type Monitor struct {
	ID                  string     `db:"id" json:"id"`
	UserID              string     `db:"user_id" json:"user_id"`
	URL                 string     `db:"url" json:"url"`
	Name                string     `db:"name" json:"name"`
	Interval            int        `db:"interval_seconds" json:"interval_seconds"`
	Timeout             int        `db:"timeout_seconds" json:"timeout_seconds"`
	Enabled             bool       `db:"enabled" json:"enabled"`
	LastStatus          string     `db:"last_status" json:"last_status"`
	ConsecutiveFailures int        `db:"consecutive_failures" json:"consecutive_failures"`
	NextCheckAt         time.Time  `db:"next_check_at" json:"next_check_at"`
	LastCheckedAt       *time.Time `db:"last_checked_at" json:"last_checked_at"`
	CreatedAt           time.Time  `db:"created_at" json:"created_at"`
}
