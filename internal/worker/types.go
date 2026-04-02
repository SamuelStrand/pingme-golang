package worker

import (
	"time"

	"pingme-golang/internal/models"
)

type CheckResult struct {
	StatusCode     int
	ResponseTimeMs int
	Success        bool
	ErrorMessage   string
	CheckedAt      time.Time
}

type EventType string

const (
	EventTypeNone      EventType = "none"
	EventTypeDown      EventType = "down"
	EventTypeRecovered EventType = "recovered"
)

type Event struct {
	Type       EventType
	Monitor    models.Monitor
	Check      CheckResult
	IncidentID string
}
