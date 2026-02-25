package main

import (
	"log"
	"net/http"

	"pingme-golang/internal/database"
	"pingme-golang/internal/handler"
)

func main() {
	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}

	healthHandler := &handler.HealthHandler{DB: db}

	http.HandleFunc("/health", healthHandler.Health)

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
