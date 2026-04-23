package main

import (
	_ "embed"
	"errors"
	"log"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/alertchannel"
	"pingme-golang/internal/auth"
	"pingme-golang/internal/config"
	"pingme-golang/internal/database"
	"pingme-golang/internal/handler"
	"pingme-golang/internal/monitor"
	"pingme-golang/internal/telegramlink"
)

//go:embed openapi.yaml
var openAPISpec []byte

func main() {
	if err := config.LoadEnv(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}

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
	alertChannelRepo := &alertchannel.Repository{DB: db}
	alertChannelService := alertchannel.NewService(alertChannelRepo)
	alertChannelHandler := &handler.AlertChannelHandler{Service: alertChannelService}
	telegramLinkRepo := &telegramlink.PostgresRepository{DB: db}
	telegramLinkService := telegramlink.NewService(telegramLinkRepo, telegramlink.LoadConfigFromEnv())
	telegramLinkHandler := &handler.TelegramLinkHandler{Service: telegramLinkService}
	monitorRepo := monitor.NewRepository(db)
	monitorService := monitor.NewService(monitorRepo)
	targetHandler := &handler.TargetHandler{Service: monitorService}

	r := gin.New()
	if err := r.SetTrustedProxies(loadTrustedProxies()); err != nil {
		log.Fatal(err)
	}
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
		protected.GET("/alert-channels", alertChannelHandler.List)
		protected.POST("/alert-channels", alertChannelHandler.Create)
		protected.PATCH("/alert-channels/:id", alertChannelHandler.Update)
		protected.DELETE("/alert-channels/:id", alertChannelHandler.Delete)
		protected.POST("/telegram/link-token", telegramLinkHandler.CreateLinkToken)
		protected.POST("/targets", targetHandler.Create)
		protected.GET("/targets", targetHandler.List)
		protected.PATCH("/targets/:id", targetHandler.Update)
		protected.DELETE("/targets/:id", targetHandler.Delete)
		protected.GET("/targets/:id/logs", targetHandler.Logs)
		protected.GET("/targets/:id/stats", targetHandler.GetMonitorStats)
	}

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Server started on %s", addr)
	log.Fatal(r.Run(addr))
}

func loadTrustedProxies() []string {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	proxies := make([]string, 0, len(parts))
	for _, part := range parts {
		if proxy := strings.TrimSpace(part); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	if len(proxies) == 0 {
		return nil
	}

	return proxies
}
