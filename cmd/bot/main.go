package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"pingme-golang/internal/config"
	"pingme-golang/internal/database"
	"pingme-golang/internal/telegramlink"
)

func main() {
	if err := config.LoadEnv(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

	token := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN"))
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is required")
	}

	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	repo := &telegramlink.PostgresRepository{DB: db}
	service := telegramlink.NewService(repo, telegramlink.Config{})
	bot := telegramlink.NewBot(token, service)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Print("telegram bot started")
	if err := bot.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
