package monitor

import (
	"context"
	"testing"
	"time"

	"pingme-golang/internal/models"
)

type stubRepository struct {
	createParams         CreateParams
	createResult         models.Monitor
	createErr            error
	listResult           []models.Monitor
	listTotal            int
	listErr              error
	getResult            models.Monitor
	getErr               error
	existsErr            error
	updateParams         UpdateParams
	updateResult         models.Monitor
	updateErr            error
	deleteErr            error
	listLogsParams       ListLogsParams
	listLogsResult       []models.CheckLog
	listLogsTotal        int
	listLogsErr          error
	getBySlugResult      models.Monitor
	getBySlugErr         error
	publicStatsResult    MonitorStats
	publicTimelineResult []TimelinePoint
	publicStatsErr       error
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

func (r *stubRepository) GetMonitorStats(ctx context.Context, targetID, userID string, from, to time.Time) (MonitorStats, []TimelinePoint, error) {
	return MonitorStats{}, []TimelinePoint{}, nil
}

func (s *stubRepository) GetBySlug(_ context.Context, _ string) (models.Monitor, error) {
	return s.getBySlugResult, s.getBySlugErr
}

func (s *stubRepository) GetPublicStats(_ context.Context, _ string, _, _ time.Time) (MonitorStats, []TimelinePoint, error) {
	return s.publicStatsResult, s.publicTimelineResult, s.publicStatsErr
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

func TestServiceCreateValidatesSlug(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		slug    *string
		wantErr error
	}{
		{
			name: "accepts lowercase slug",
			slug: stringPtr("my-api"),
		},
		{
			name: "accepts digits",
			slug: stringPtr("api123"),
		},
		{
			name:    "rejects uppercase",
			slug:    stringPtr("MyApi"),
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "rejects underscore",
			slug:    stringPtr("api_1"),
			wantErr: ErrInvalidSlug,
		},
		{
			name:    "rejects too short slug",
			slug:    stringPtr("ab"),
			wantErr: ErrInvalidSlug,
		},
		{
			name: "allows empty slug",
			slug: stringPtr(""),
		},
		{
			name: "allows missing slug",
			slug: nil,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := &stubRepository{
				createResult: models.Monitor{ID: "monitor-1"},
			}
			service := NewService(repo)

			_, err := service.Create(context.Background(), CreateInput{
				UserID:   "user-1",
				URL:      "https://example.com",
				Name:     "API",
				Interval: 60,
				Enabled:  true,
				Slug:     testCase.slug,
			})

			if err != testCase.wantErr {
				t.Fatalf("Create() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestServiceCreatePassesStatusPageFields(t *testing.T) {
	t.Parallel()

	slug := "my-api"
	repo := &stubRepository{
		createResult: models.Monitor{ID: "monitor-1"},
	}
	service := NewService(repo)

	_, err := service.Create(context.Background(), CreateInput{
		UserID:            "user-1",
		URL:               "https://example.com",
		Name:              "API",
		Interval:          60,
		Enabled:           true,
		Slug:              &slug,
		StatusPageEnabled: true,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if repo.createParams.Slug == nil || *repo.createParams.Slug != "my-api" {
		t.Fatalf("Slug = %v, want my-api", repo.createParams.Slug)
	}
	if !repo.createParams.StatusPageEnabled {
		t.Fatal("StatusPageEnabled = false, want true")
	}
}

func TestServiceUpdatePassesStatusPageFields(t *testing.T) {
	t.Parallel()

	oldSlug := "old-api"
	newSlug := "new-api"
	statusPageEnabled := true

	repo := &stubRepository{
		getResult: models.Monitor{
			ID:                "monitor-1",
			UserID:            "user-1",
			URL:               "https://example.com",
			Name:              "API",
			Interval:          60,
			Enabled:           true,
			Slug:              &oldSlug,
			StatusPageEnabled: false,
		},
		updateResult: models.Monitor{ID: "monitor-1"},
	}
	service := NewService(repo)

	_, err := service.Update(context.Background(), UpdateInput{
		UserID:            "user-1",
		ID:                "monitor-1",
		Slug:              &newSlug,
		StatusPageEnabled: &statusPageEnabled,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if repo.updateParams.Slug == nil || *repo.updateParams.Slug != "new-api" {
		t.Fatalf("Slug = %v, want new-api", repo.updateParams.Slug)
	}
	if !repo.updateParams.StatusPageEnabled {
		t.Fatal("StatusPageEnabled = false, want true")
	}
}

func TestServiceGetPublicStatusReturnsNotFoundWhenDisabled(t *testing.T) {
	t.Parallel()

	slug := "my-api"
	repo := &stubRepository{
		getBySlugResult: models.Monitor{
			ID:                "monitor-1",
			Slug:              &slug,
			StatusPageEnabled: false,
		},
	}
	service := NewService(repo)

	_, err := service.GetPublicStatus(
		context.Background(),
		"my-api",
		time.Now().Add(-24*time.Hour),
		time.Now(),
	)
	if err != ErrNotFound {
		t.Fatalf("GetPublicStatus() error = %v, want %v", err, ErrNotFound)
	}
}

func TestServiceGetPublicStatusReturnsStats(t *testing.T) {
	t.Parallel()

	slug := "my-api"
	from := time.Date(2026, time.May, 18, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, time.May, 19, 0, 0, 0, 0, time.UTC)

	repo := &stubRepository{
		getBySlugResult: models.Monitor{
			ID:                "monitor-1",
			Name:              "API",
			Slug:              &slug,
			StatusPageEnabled: true,
		},
		publicStatsResult: MonitorStats{
			TotalChecks:   10,
			SuccessCount:  9,
			AvgResponseMs: 123,
		},
		publicTimelineResult: []TimelinePoint{
			{
				Timestamp:      from.Add(time.Hour),
				Success:        true,
				ResponseTimeMs: 120,
			},
		},
	}
	service := NewService(repo)

	result, err := service.GetPublicStatus(context.Background(), "my-api", from, to)
	if err != nil {
		t.Fatalf("GetPublicStatus() error = %v", err)
	}

	if result.Monitor.ID != "monitor-1" {
		t.Fatalf("Monitor.ID = %q, want monitor-1", result.Monitor.ID)
	}
	if result.Stats.TotalChecks != 10 {
		t.Fatalf("TotalChecks = %d, want 10", result.Stats.TotalChecks)
	}
	if len(result.Timeline) != 1 {
		t.Fatalf("len(Timeline) = %d, want 1", len(result.Timeline))
	}
}

func stringPtr(value string) *string {
	return &value
}
