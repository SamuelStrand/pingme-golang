package models

import "time"

type CheckLog struct {
	ID             int       `db:"id"`
	MonitorID      int       `db:"monitor_id"`
	StatusCode     int       `db:"status_code"`
	ResponseTimeMs int       `db:"response_time_ms"`
	CheckedAt      time.Time `db:"checked_at"`
}
