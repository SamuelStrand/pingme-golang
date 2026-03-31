package auth

import (
	"context"
	"net/http"
	"strings"

	"pingme-golang/internal/httpx"
)

type contextKey string

const userIDKey contextKey = "user_id"

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok && v != ""
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func AuthMiddleware(cfg Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if authz == "" {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "missing Authorization header", nil)
				return
			}
			parts := strings.SplitN(authz, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid Authorization header", nil)
				return
			}

			claims, err := ParseAccessToken(cfg, parts[1])
			if err != nil || claims.Subject == "" {
				httpx.Error(w, http.StatusUnauthorized, "unauthorized", "invalid token", nil)
				return
			}

			ctx := WithUserID(r.Context(), claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
