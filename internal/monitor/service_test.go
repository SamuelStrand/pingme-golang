package monitor

import (
	"context"
	"testing"
	"time"

	"pingme-golang/internal/models"
)

type stubRepository struct {
	createParams   CreateParams
	createResult   models.Monitor
	createErr      error
	listResult     []models.Monitor
	listTotal      int
	listErr        error
	getResult      models.Monitor
	getErr         error
	existsErr      error
	updateParams   UpdateParams
	updateResult   models.Monitor
	updateErr      error
	deleteErr      error
	listLogsParams ListLogsParams
	listLogsResult []models.CheckLog
	listLogsTotal  int
	listLogsErr    error
}

func (s *stubRepository) Create(_ context.Context, params CreateParams) (models.Monitor, error) {
	s.createParams = params
	return s.createResult, s.createErr
}

func (s *stubRepository) ListByUserID(_ context.Context, _ string, _ int, _ int) ([]models.Monitor, int, error) {
	return s.listResult, s.listTotal, s.listErr
}

func (s *stubRepository) GetByIDAndUserID(_ context.Context, _ string, _ string) (models.Monitor, error) {
	return s.getResult, s.getErr
}

func (s *stubRepository) ExistsByIDAndUserID(_ context.Context, _ string, _ string) error {
	return s.existsErr
}

func (s *stubRepository) Update(_ context.Context, params UpdateParams) (models.Monitor, error) {
	s.updateParams = params
	return s.updateResult, s.updateErr
}

func (s *stubRepository) Delete(_ context.Context, _ string, _ string) error {
	return s.deleteErr
}

func (s *stubRepository) ListLogs(_ context.Context, params ListLogsParams) ([]models.CheckLog, int, error) {
	s.listLogsParams = params
	return s.listLogsResult, s.listLogsTotal, s.listLogsErr
}

func TestServiceCreateNormalizesInput(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		createResult: models.Monitor{ID: "monitor-1"},
	}
	service := NewService(repo)

	_, err := service.Create(context.Background(), CreateInput{
		UserID:   "user-1",
		URL:      " https://example.com/health ",
		Name:     "  API health  ",
		Interval: 60,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repo.createParams.URL != "https://example.com/health" {
		t.Fatalf("URL = %q, want %q", repo.createParams.URL, "https://example.com/health")
	}
	if repo.createParams.Name != "API health" {
		t.Fatalf("Name = %q, want %q", repo.createParams.Name, "API health")
	}
	if repo.createParams.Interval != 60 {
		t.Fatalf("Interval = %d, want 60", repo.createParams.Interval)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		url      string
		interval int
		wantErr  error
	}{
		{
			name:     "rejects invalid url",
			url:      "ftp://example.com",
			interval: 60,
			wantErr:  ErrInvalidURL,
		},
		{
			name:     "rejects interval below minimum",
			url:      "https://example.com",
			interval: 29,
			wantErr:  ErrInvalidInterval,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := NewService(&stubRepository{})
			_, err := service.Create(context.Background(), CreateInput{
				UserID:   "user-1",
				URL:      testCase.url,
				Interval: testCase.interval,
				Enabled:  true,
			})
			if err != testCase.wantErr {
				t.Fatalf("Create() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestServiceUpdateMergesExistingMonitor(t *testing.T) {
	t.Parallel()

	repo := &stubRepository{
		getResult: models.Monitor{
			ID:       "monitor-1",
			UserID:   "user-1",
			URL:      "https://example.com/old",
			Name:     "Old name",
			Interval: 90,
			Enabled:  true,
		},
		updateResult: models.Monitor{ID: "monitor-1"},
	}
	service := NewService(repo)
	newName := "  New name  "
	enabled := false

	_, err := service.Update(context.Background(), UpdateInput{
		UserID:  "user-1",
		ID:      "monitor-1",
		Name:    &newName,
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if repo.updateParams.URL != "https://example.com/old" {
		t.Fatalf("URL = %q, want existing value", repo.updateParams.URL)
	}
	if repo.updateParams.Name != "New name" {
		t.Fatalf("Name = %q, want %q", repo.updateParams.Name, "New name")
	}
	if repo.updateParams.Interval != 90 {
		t.Fatalf("Interval = %d, want 90", repo.updateParams.Interval)
	}
	if repo.updateParams.Enabled {
		t.Fatal("Enabled = true, want false")
	}
}

func TestServiceUpdateRejectsEmptyInput(t *testing.T) {
	t.Parallel()

	service := NewService(&stubRepository{})
	_, err := service.Update(context.Background(), UpdateInput{UserID: "user-1", ID: "monitor-1"})
	if err != ErrEmptyUpdate {
		t.Fatalf("Update() error = %v, want %v", err, ErrEmptyUpdate)
	}
}

func TestServiceListLogsRejectsInvalidDateRange(t *testing.T) {
	t.Parallel()

	service := NewService(&stubRepository{})
	from := time.Date(2026, time.April, 2, 10, 0, 0, 0, time.UTC)
	to := from.Add(-time.Hour)

	_, err := service.ListLogs(context.Background(), ListLogsInput{
		UserID:    "user-1",
		MonitorID: "monitor-1",
		Page:      1,
		PageSize:  20,
		From:      &from,
		To:        &to,
	})
	if err != ErrInvalidDateRange {
		t.Fatalf("ListLogs() error = %v, want %v", err, ErrInvalidDateRange)
	}
}
