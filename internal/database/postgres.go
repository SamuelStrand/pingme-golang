package database

import (
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func NewPostgres() (*sqlx.DB, error) {
	connStr := os.Getenv("DATABASE_URL")

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}
