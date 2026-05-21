package config

import (
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Optional overrides for host-side `go run` while Postgres runs in Docker.
	_ = godotenv.Load(".env.local")

	return nil
}
