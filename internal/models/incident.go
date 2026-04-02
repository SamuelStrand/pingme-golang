package models

import "time"

type Incident struct {
	ID         string     `db:"id" json:"id"`
	MonitorID  string     `db:"monitor_id" json:"monitor_id"`
	Status     string     `db:"status" json:"status"`
	Reason     *string    `db:"reason" json:"reason"`
	StartedAt  time.Time  `db:"started_at" json:"started_at"`
	ResolvedAt *time.Time `db:"resolved_at" json:"resolved_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}
