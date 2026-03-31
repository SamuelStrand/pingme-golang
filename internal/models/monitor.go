package models

import "time"

type Monitor struct {
	ID         int       `db:"id" json:"id"`
	UserID     int       `db:"user_id" json:"user_id"`
	URL        string    `db:"url" json:"url"`
	Name       string    `db:"name" json:"name"`
	Interval   int       `db:"interval_seconds" json:"interval"`
	Timeout    int       `db:"timeout_seconds" json:"timeout"`
	Enabled    bool      `db:"enabled" json:"enabled"`
	LastStatus string    `db:"last_status" json:"last_status"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}
