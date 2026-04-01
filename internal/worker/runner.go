package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"pingme-golang/internal/models"
)

type Runner struct {
	config   Config
	repo     *Repository
	checker  Checker
	notifier Notifier
}

func NewRunner(config Config, repo *Repository, checker Checker, notifier Notifier) *Runner {
	return &Runner{
		config:   config,
		repo:     repo,
		checker:  checker,
		notifier: notifier,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	jobs := make(chan models.Monitor, r.config.QueueSize)
	errCh := make(chan error, 1)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(jobs)

		if err := r.runScheduler(ctx, jobs); err != nil && !errors.Is(err, context.Canceled) {
			select {
			case errCh <- err:
			default:
			}
		}
	}()

	for i := 0; i < r.config.WorkerCount; i++ {
		workerID := i + 1
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runWorker(ctx, jobs, workerID)
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		return err
	case <-done:
		return nil
	case <-ctx.Done():
		<-done
		return nil
	}
}

func (r *Runner) runScheduler(ctx context.Context, jobs chan<- models.Monitor) error {
	if err := r.enqueueDueMonitors(ctx, jobs, time.Now().UTC()); err != nil {
		return err
	}

	ticker := time.NewTicker(r.config.Tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case tickAt := <-ticker.C:
			if err := r.enqueueDueMonitors(ctx, jobs, tickAt.UTC()); err != nil {
				return err
			}
		}
	}
}

func (r *Runner) enqueueDueMonitors(ctx context.Context, jobs chan<- models.Monitor, now time.Time) error {
	monitors, err := r.repo.ClaimDueMonitors(ctx, now, r.config.BatchSize)
	if err != nil {
		return fmt.Errorf("claim due monitors: %w", err)
	}

	for _, monitor := range monitors {
		select {
		case <-ctx.Done():
			return nil
		case jobs <- monitor:
		}
	}

	return nil
}

func (r *Runner) runWorker(ctx context.Context, jobs <-chan models.Monitor, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case monitor, ok := <-jobs:
			if !ok {
				return
			}

			if err := r.processMonitor(ctx, monitor); err != nil {
				log.Printf("worker %d: process monitor %q failed: %v", workerID, monitor.ID, err)
			}
		}
	}
}

func (r *Runner) processMonitor(ctx context.Context, monitor models.Monitor) error {
	result := r.checker.Check(ctx, monitor)

	event, err := r.repo.ApplyCheckResult(ctx, monitor.ID, result)
	if err != nil {
		return fmt.Errorf("apply check result for monitor %q: %w", monitor.ID, err)
	}

	if event.Type == EventTypeNone {
		return nil
	}

	channels, err := r.repo.ListEnabledAlertChannels(ctx, event.Monitor.UserID)
	if err != nil {
		return fmt.Errorf("load alert channels for user %q: %w", event.Monitor.UserID, err)
	}

	if len(channels) == 0 {
		return nil
	}

	if err := r.notifier.Notify(ctx, event, channels); err != nil {
		return fmt.Errorf("notify %s for monitor %q: %w", event.Type, event.Monitor.ID, err)
	}

	return nil
}
