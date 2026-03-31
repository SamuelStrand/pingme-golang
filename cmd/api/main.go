package main

import (
	"log"
	"net/http"
	"pingme-golang/internal/middleware"

	"pingme-golang/internal/database"
	"pingme-golang/internal/handler"
)

func main() {
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}

	healthHandler := &handler.HealthHandler{DB: db}

	http.Handle("/health", middleware.Logging(http.HandlerFunc(healthHandler.Health)))
	http.Handle("/ready", middleware.Logging(http.HandlerFunc(healthHandler.Ready)))

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
