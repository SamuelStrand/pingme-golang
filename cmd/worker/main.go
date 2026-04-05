package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"pingme-golang/internal/config"
	"pingme-golang/internal/database"
	"pingme-golang/internal/worker"
)

func main() {
	if err := config.LoadEnv(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	workerConfig := worker.LoadConfigFromEnv()
	repo := worker.NewRepository(db)
	checker := worker.NewHTTPChecker()
	notifier := loadNotifier()
	runner := worker.NewRunner(workerConfig, repo, checker, notifier)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf(
		"worker started workers=%d batch_size=%d queue_size=%d tick=%s",
		workerConfig.WorkerCount,
		workerConfig.BatchSize,
		workerConfig.QueueSize,
		workerConfig.Tick,
	)

	if err := runner.Run(ctx); err != nil {
		log.Fatal(err)
	}
}

func loadNotifier() worker.Notifier {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Print("TELEGRAM_BOT_TOKEN is not set, telegram notifications are disabled")
	}

	return worker.NewAlertChannelNotifier(token)
}
