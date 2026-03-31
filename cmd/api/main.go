package main

import (
	_ "embed"
	"log"
	"net/http"
	"os"
	"pingme-golang/internal/auth"
	"pingme-golang/internal/config"
	"pingme-golang/internal/middleware"

	"pingme-golang/internal/database"
	"pingme-golang/internal/handler"

	"github.com/go-chi/chi/v5"
)

//go:embed openapi.yaml
var openAPISpec []byte

func main() {
	_ = config.LoadEnv()

	db, err := database.NewPostgres()
	if err != nil {
		log.Fatal(err)
	}

	healthHandler := &handler.HealthHandler{DB: db}
	authCfg, err := auth.LoadConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	authRepo := &auth.Repository{DB: db}
	authHandler := &handler.AuthHandler{Repo: authRepo, Cfg: authCfg}
	userHandler := &handler.UserHandler{Repo: authRepo}

	r := chi.NewRouter()
	r.Use(middleware.Logging)

	r.Get("/health", healthHandler.Health)
	r.Get("/ready", healthHandler.Ready)
	r.Get("/swagger", handler.SwaggerUI)
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		_, _ = w.Write(openAPISpec)
	})

	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/logout", authHandler.Logout)
	})

	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware(authCfg))
		r.Get("/me", userHandler.Me)
	})

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Server started on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
