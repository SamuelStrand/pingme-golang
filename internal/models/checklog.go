package models

import "time"

type CheckLog struct {
	ID             string    `db:"id" json:"id"`
	TargetID       string    `db:"target_id" json:"target_id"`
	StatusCode     int       `db:"status_code" json:"status_code"`
	ResponseTimeMs int       `db:"response_time_ms" json:"response_time_ms"`
	Success        bool      `db:"success" json:"success"`
	ErrorMessage   string    `db:"error_message" json:"error_message"`
	CheckedAt      time.Time `db:"checked_at" json:"checked_at"`
}
