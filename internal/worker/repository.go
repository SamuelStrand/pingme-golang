package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"pingme-golang/internal/models"
)

var ErrMonitorNotFound = errors.New("monitor not found")

const monitorColumns = `
	id,
	user_id,
	url,
	name,
	interval_seconds,
	timeout_seconds,
	enabled,
	last_status,
	consecutive_failures,
	next_check_at,
	last_checked_at,
	created_at
`

type Repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) ClaimDueMonitors(ctx context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim monitors tx: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryxContext(ctx, fmt.Sprintf(`
		select %s
		from monitors
		where enabled = true and next_check_at <= $1
		order by next_check_at asc
		limit $2
		for update skip locked
	`, monitorColumns), now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("select due monitors: %w", err)
	}
	defer rows.Close()

	monitors := []models.Monitor{}
	for rows.Next() {
		var monitor models.Monitor
		if err := rows.StructScan(&monitor); err != nil {
			return nil, fmt.Errorf("scan due monitor: %w", err)
		}
		monitors = append(monitors, monitor)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate due monitors: %w", err)
	}

	for i := range monitors {
		nextCheckAt := now.UTC().Add(time.Duration(monitors[i].Interval) * time.Second)
		_, err := tx.ExecContext(ctx, `
			update monitors
			set next_check_at = $1
			where id = $2
		`, nextCheckAt, monitors[i].ID)
		if err != nil {
			return nil, fmt.Errorf("update next_check_at for monitor %q: %w", monitors[i].ID, err)
		}
		monitors[i].NextCheckAt = nextCheckAt
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim monitors tx: %w", err)
	}

	return monitors, nil
}

func (r *Repository) ApplyCheckResult(ctx context.Context, monitorID string, result CheckResult) (Event, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return Event{}, fmt.Errorf("begin apply check tx: %w", err)
	}
	defer tx.Rollback()

	var monitor models.Monitor
	err = tx.QueryRowxContext(ctx, fmt.Sprintf(`
		select %s
		from monitors
		where id = $1
		for update
	`, monitorColumns), monitorID).StructScan(&monitor)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Event{}, ErrMonitorNotFound
		}
		return Event{}, fmt.Errorf("load monitor %q for update: %w", monitorID, err)
	}

	_, err = tx.ExecContext(ctx, `
		insert into checklogs (monitor_id, status_code, response_time_ms, success, error_message, checked_at)
		values ($1, $2, $3, $4, $5, $6)
	`, monitorID, result.StatusCode, result.ResponseTimeMs, result.Success, result.ErrorMessage, result.CheckedAt.UTC())
	if err != nil {
		return Event{}, fmt.Errorf("insert checklog for monitor %q: %w", monitorID, err)
	}

	event := Event{
		Type:    EventTypeNone,
		Monitor: monitor,
		Check:   result,
	}

	transition := evaluateStateTransition(monitor, result)
	nextStatus := transition.nextStatus
	nextFailures := transition.nextFailures
	event.Type = transition.eventType

	_, err = tx.ExecContext(ctx, `
		update monitors
		set last_status = $1,
			consecutive_failures = $2,
			last_checked_at = $3
		where id = $4
	`, nextStatus, nextFailures, result.CheckedAt.UTC(), monitorID)
	if err != nil {
		return Event{}, fmt.Errorf("update monitor %q runtime state: %w", monitorID, err)
	}

	monitor.LastStatus = nextStatus
	monitor.ConsecutiveFailures = nextFailures
	checkedAt := result.CheckedAt.UTC()
	monitor.LastCheckedAt = &checkedAt

	switch event.Type {
	case EventTypeDown:
		reason := incidentReason(result)
		err = tx.QueryRowxContext(ctx, `
			insert into incidents (monitor_id, status, reason, started_at)
			values ($1, 'open', $2, $3)
			returning id
		`, monitorID, reason, checkedAt).Scan(&event.IncidentID)
		if err != nil {
			return Event{}, fmt.Errorf("create incident for monitor %q: %w", monitorID, err)
		}
	case EventTypeRecovered:
		err = tx.QueryRowxContext(ctx, `
			update incidents
			set status = 'resolved',
				resolved_at = $1
			where monitor_id = $2 and status = 'open'
			returning id
		`, checkedAt, monitorID).Scan(&event.IncidentID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return Event{}, fmt.Errorf("resolve incident for monitor %q: %w", monitorID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return Event{}, fmt.Errorf("commit apply check tx: %w", err)
	}

	event.Monitor = monitor
	return event, nil
}

func (r *Repository) ListEnabledAlertChannels(ctx context.Context, userID string) ([]models.AlertChannel, error) {
	channels := []models.AlertChannel{}
	err := r.DB.SelectContext(ctx, &channels, `
		select id, user_id, type, address, enabled, created_at
		from alert_channels
		where user_id = $1 and enabled = true
		order by created_at asc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list enabled alert channels for user %q: %w", userID, err)
	}

	return channels, nil
}

func incidentReason(result CheckResult) string {
	switch {
	case result.ErrorMessage != "":
		return result.ErrorMessage
	case result.StatusCode != 0:
		return fmt.Sprintf("http status %d", result.StatusCode)
	default:
		return "check failed"
	}
}
