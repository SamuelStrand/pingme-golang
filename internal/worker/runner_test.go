package worker

import (
	"context"
	"testing"
	"time"

	"pingme-golang/internal/models"
)

type checkerMock struct{}

func (c *checkerMock) Check(ctx context.Context, m models.Monitor) CheckResult {
	return CheckResult{
		Success:   true,
		CheckedAt: time.Now(),
	}
}

type repoMockWorker struct {
	applied bool
}

func (r *repoMockWorker) ClaimDueMonitors(ctx context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	return nil, nil
}

func (r *repoMockWorker) ApplyCheckResult(ctx context.Context, monitorID string, result CheckResult) (Event, error) {
	r.applied = true

	return Event{
		Type: EventTypeNone,
		Monitor: models.Monitor{
			ID: monitorID,
		},
		Check: result,
	}, nil
}

func (r *repoMockWorker) ListEnabledAlertChannels(ctx context.Context, userID string) ([]models.AlertChannel, error) {
	return nil, nil
}

type notifierMock struct{}

func (n *notifierMock) Notify(ctx context.Context, event Event, channels []models.AlertChannel) error {
	return nil
}

func TestRunner_processMonitor(t *testing.T) {
	t.Parallel()

	repo := &repoMockWorker{}

	r := &Runner{
		repo:     repo,
		checker:  &checkerMock{},
		notifier: &notifierMock{},
	}

	monitor := models.Monitor{
		ID:     "test-id",
		UserID: "user-1",
	}

	err := r.processMonitor(context.Background(), monitor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.applied {
		t.Fatal("expected ApplyCheckResult to be called")
	}
}
