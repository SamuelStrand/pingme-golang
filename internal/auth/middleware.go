package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/httpx"
)

const userIDKey = "user_id"

func UserIDFromGin(c *gin.Context) (string, bool) {
	v, ok := c.Get(userIDKey)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok && s != ""
}

func AuthMiddleware(cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if authz == "" {
			c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "missing Authorization header"})
			c.Abort()
			return
		}
		parts := strings.SplitN(authz, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "invalid Authorization header"})
			c.Abort()
			return
		}

		claims, err := ParseAccessToken(cfg, parts[1])
		if err != nil || claims.Subject == "" {
			c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "unauthorized", Message: "invalid token"})
			c.Abort()
			return
		}

		c.Set(userIDKey, claims.Subject)
		c.Next()
	}
}
