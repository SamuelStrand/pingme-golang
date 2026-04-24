package monitor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"pingme-golang/internal/models"
)

const monitorColumns = `
	id,
	user_id,
	url,
	coalesce(name, '') as name,
	interval_seconds,
	timeout_seconds,
	enabled,
	last_status,
	consecutive_failures,
	next_check_at,
	last_checked_at,
	created_at
`

const checkLogColumns = `
	id,
	monitor_id,
	coalesce(status_code, 0) as status_code,
	coalesce(response_time_ms, 0) as response_time_ms,
	coalesce(success, false) as success,
	coalesce(error_message, '') as error_message,
	checked_at
`

type Repository struct {
	DB *sqlx.DB
}

type CreateParams struct {
	UserID   string
	URL      string
	Name     string
	Interval int
	Enabled  bool
}

type UpdateParams struct {
	ID       string
	UserID   string
	URL      string
	Name     string
	Interval int
	Enabled  bool
}

type ListLogsParams struct {
	MonitorID string
	Page      int
	PageSize  int
	From      *time.Time
	To        *time.Time
}

type MonitorStats struct {
	TotalChecks   int     `db:"total"`
	SuccessCount  int     `db:"success_count"`
	AvgResponseMs float64 `db:"avg_response"`
}

type TimelinePoint struct {
	Timestamp      time.Time `db:"checked_at" json:"timestamp"`
	Success        bool      `db:"success" json:"success"`
	ResponseTimeMs int       `db:"response_time_ms" json:"response_time_ms"`
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) Create(ctx context.Context, params CreateParams) (models.Monitor, error) {
	var item models.Monitor
	err := r.DB.GetContext(ctx, &item, fmt.Sprintf(`
		insert into monitors (user_id, url, name, interval_seconds, timeout_seconds, enabled, next_check_at)
		values ($1, $2, $3, $4, $5, $6, now())
		returning %s
	`, monitorColumns), params.UserID, params.URL, params.Name, params.Interval, DefaultTimeoutInSec, params.Enabled)
	if err != nil {
		return models.Monitor{}, fmt.Errorf("create monitor: %w", err)
	}

	return item, nil
}

func (r *Repository) ListByUserID(
	ctx context.Context,
	userID string,
	page int,
	pageSize int,
) ([]models.Monitor, int, error) {
	var total int
	if err := r.DB.GetContext(ctx, &total, `
		select count(*)
		from monitors
		where user_id = $1
	`, userID); err != nil {
		return nil, 0, fmt.Errorf("count monitors: %w", err)
	}

	items := []models.Monitor{}
	offset := (page - 1) * pageSize
	if err := r.DB.SelectContext(ctx, &items, fmt.Sprintf(`
		select %s
		from monitors
		where user_id = $1
		order by created_at desc, id desc
		limit $2 offset $3
	`, monitorColumns), userID, pageSize, offset); err != nil {
		return nil, 0, fmt.Errorf("list monitors: %w", err)
	}

	return items, total, nil
}

func (r *Repository) GetByIDAndUserID(ctx context.Context, id string, userID string) (models.Monitor, error) {
	var item models.Monitor
	err := r.DB.GetContext(ctx, &item, fmt.Sprintf(`
		select %s
		from monitors
		where id = $1 and user_id = $2
	`, monitorColumns), id, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Monitor{}, ErrNotFound
		}
		return models.Monitor{}, fmt.Errorf("get monitor: %w", err)
	}

	return item, nil
}

func (r *Repository) ExistsByIDAndUserID(ctx context.Context, id string, userID string) error {
	var exists bool
	err := r.DB.GetContext(ctx, &exists, `
		select exists(
			select 1
			from monitors
			where id = $1 and user_id = $2
		)
	`, id, userID)
	if err != nil {
		return fmt.Errorf("check monitor existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, params UpdateParams) (models.Monitor, error) {
	var item models.Monitor
	err := r.DB.GetContext(ctx, &item, fmt.Sprintf(`
		update monitors
		set url = $1,
			name = $2,
			interval_seconds = $3,
			enabled = $4
		where id = $5 and user_id = $6
		returning %s
	`, monitorColumns), params.URL, params.Name, params.Interval, params.Enabled, params.ID, params.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Monitor{}, ErrNotFound
		}
		return models.Monitor{}, fmt.Errorf("update monitor: %w", err)
	}

	return item, nil
}

func (r *Repository) Delete(ctx context.Context, id string, userID string) error {
	result, err := r.DB.ExecContext(ctx, `
		delete from monitors
		where id = $1 and user_id = $2
	`, id, userID)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete monitor rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repository) ListLogs(ctx context.Context, params ListLogsParams) ([]models.CheckLog, int, error) {
	whereParts := []string{"monitor_id = $1"}
	args := []any{params.MonitorID}
	placeholder := 2

	if params.From != nil {
		whereParts = append(whereParts, fmt.Sprintf("checked_at >= $%d", placeholder))
		args = append(args, params.From.UTC())
		placeholder++
	}
	if params.To != nil {
		whereParts = append(whereParts, fmt.Sprintf("checked_at <= $%d", placeholder))
		args = append(args, params.To.UTC())
		placeholder++
	}

	whereClause := strings.Join(whereParts, " and ")

	var total int
	if err := r.DB.GetContext(ctx, &total, fmt.Sprintf(`
		select count(*)
		from checklogs
		where %s
	`, whereClause), args...); err != nil {
		return nil, 0, fmt.Errorf("count monitor logs: %w", err)
	}

	items := []models.CheckLog{}
	offset := (params.Page - 1) * params.PageSize
	args = append(args, params.PageSize, offset)
	if err := r.DB.SelectContext(ctx, &items, fmt.Sprintf(`
		select %s
		from checklogs
		where %s
		order by checked_at desc, id desc
		limit $%d offset $%d
	`, checkLogColumns, whereClause, placeholder, placeholder+1), args...); err != nil {
		return nil, 0, fmt.Errorf("list monitor logs: %w", err)
	}

	return items, total, nil
}

func (r *Repository) GetMonitorStats(ctx context.Context, targetID, userID string, from, to time.Time) (MonitorStats, []TimelinePoint, error) {
	var stats MonitorStats

	err := r.DB.GetContext(ctx, &stats, `
		select 
			count(*) as total,
			coalesce(sum(case when success then 1 else 0 end), 0) as success_count,
			coalesce(avg(response_time_ms), 0) as avg_response
		from checklogs cl
		join monitors t on cl.monitor_id = t.id
		where cl.monitor_id = $1
		  and t.user_id = $2
		  and cl.checked_at between $3 and $4
	`, targetID, userID, from, to)
	if err != nil {
		return MonitorStats{}, nil, fmt.Errorf("get stats: %w", err)
	}

	var timeline []TimelinePoint
	err = r.DB.SelectContext(ctx, &timeline, `
		select cl.checked_at, cl.success, cl.response_time_ms
		from checklogs cl
		join monitors t on cl.monitor_id = t.id
		where cl.monitor_id = $1
		  and t.user_id = $2
		  and cl.checked_at between $3 and $4
		order by cl.checked_at asc
		limit 1000
	`, targetID, userID, from, to)
	if err != nil {
		return MonitorStats{}, nil, fmt.Errorf("get timeline: %w", err)
	}

	return stats, timeline, nil
}
