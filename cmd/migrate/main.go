package main

import (
	"fmt"
	"log"
	"os"

	"pingme-golang/internal/config"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	_ = config.LoadEnv()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatal(err)
	}

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("migrations applied")
	case "down":
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			log.Fatal(err)
		}
		fmt.Println("rolled back 1 migration")
	case "version":
		v, dirty, err := m.Version()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
	default:
		log.Fatalf("unknown command %q (use: up|down|version)", cmd)
	}
}

