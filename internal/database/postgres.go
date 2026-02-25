package database

import (
	"database/sql"
	"fmt"
	"os"
)

func NewPostgres() (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password =%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"))
	
	return sql.Open("postgres", connStr)
}
