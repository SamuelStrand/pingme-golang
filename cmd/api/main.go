package main

import (
	_ "embed"
	"log"
	"os"
	"pingme-golang/internal/auth"
	"pingme-golang/internal/config"

	"pingme-golang/internal/database"
	"pingme-golang/internal/handler"

	"github.com/gin-gonic/gin"
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

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", healthHandler.Health)
	r.GET("/ready", healthHandler.Ready)
	r.GET("/swagger", handler.SwaggerUI)
	r.GET("/openapi.yaml", func(c *gin.Context) {
		c.Header("Content-Type", "application/yaml; charset=utf-8")
		c.String(200, string(openAPISpec))
	})

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)
		authGroup.POST("/refresh", authHandler.Refresh)
		authGroup.POST("/logout", authHandler.Logout)
	}

	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware(authCfg))
	{
		protected.GET("/me", userHandler.Me)
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("Server started on %s", addr)
	log.Fatal(r.Run(addr))
}
