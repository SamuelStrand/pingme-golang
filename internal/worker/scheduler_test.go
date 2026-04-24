package worker

import (
	"context"
	"testing"
	"time"

	"pingme-golang/internal/models"
)

type repoMockScheduler struct {
	called bool
}

func (m *repoMockScheduler) ClaimDueMonitors(ctx context.Context, now time.Time, limit int) ([]models.Monitor, error) {
	m.called = true
	return []models.Monitor{
		{ID: "1"},
		{ID: "2"},
	}, nil
}

func (m *repoMockScheduler) ApplyCheckResult(ctx context.Context, monitorID string, result CheckResult) (Event, error) {
	return Event{}, nil
}

func (m *repoMockScheduler) ListEnabledAlertChannels(ctx context.Context, userID string) ([]models.AlertChannel, error) {
	return nil, nil
}

func Test_enqueueDueMonitors(t *testing.T) {
	repo := &repoMockScheduler{}

	r := &Runner{
		repo: repo,
		config: Config{
			BatchSize: 10,
		},
	}

	jobs := make(chan models.Monitor, 2)

	err := r.enqueueDueMonitors(context.Background(), jobs, time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	close(jobs)

	count := 0
	for range jobs {
		count++
	}

	if !repo.called {
		t.Fatal("ClaimDueMonitors was not called")
	}

	if count != 2 {
		t.Fatalf("expected 2 jobs, got %d", count)
	}
}
