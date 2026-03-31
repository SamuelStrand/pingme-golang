package handler

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pingme-golang/internal/auth"
	"pingme-golang/internal/httpx"
)

type AuthHandler struct {
	Repo *auth.Repository
	Cfg  auth.Config
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if len(req.Email) < 3 || len(req.Password) < 8 {
		fields := map[string]string{}
		if len(req.Email) < 3 {
			fields["email"] = "must be a valid email"
		}
		if len(req.Password) < 8 {
			fields["password"] = "min length is 8"
		}
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "validation_error", Message: "invalid email or password", Fields: fields})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to hash password"})
		return
	}

	u, err := h.Repo.CreateUser(c.Request.Context(), req.Email, hash)
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			c.JSON(http.StatusConflict, httpx.ErrorResponse{Error: "email_taken", Message: "email already exists", Fields: map[string]string{"email": "already exists"}})
			return
		}
		log.Printf("register: create user failed: %v", err)
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to create user"})
		return
	}

	pair, err := h.issueTokenPair(c.Request.Context(), u.ID)
	if err != nil {
		log.Printf("register: issue tokens failed: %v", err)
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusCreated, pair)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_credentials", Message: "invalid credentials"})
		return
	}

	u, err := h.Repo.GetUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_credentials", Message: "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to load user"})
		return
	}

	if err := auth.ComparePassword(u.Password, req.Password); err != nil {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_credentials", Message: "invalid credentials"})
		return
	}

	pair, err := h.issueTokenPair(c.Request.Context(), u.ID)
	if err != nil {
		log.Printf("login: issue tokens failed: %v", err)
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusOK, pair)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "validation_error", Message: "missing refresh_token", Fields: map[string]string{"refresh_token": "required"}})
		return
	}

	claims, err := auth.ParseRefreshToken(h.Cfg, req.RefreshToken)
	if err != nil || claims.Subject == "" || claims.ID == "" {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_token", Message: "invalid token"})
		return
	}

	userID, expiresAt, revokedAt, err := h.Repo.GetSession(c.Request.Context(), claims.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_token", Message: "invalid token"})
			return
		}
		log.Printf("refresh: load session failed: %v", err)
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to load session"})
		return
	}
	if userID != claims.Subject || revokedAt != nil || time.Now().After(expiresAt) {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_token", Message: "invalid token"})
		return
	}

	_ = h.Repo.RevokeSession(c.Request.Context(), claims.ID)

	pair, err := h.issueTokenPair(c.Request.Context(), userID)
	if err != nil {
		log.Printf("refresh: issue tokens failed: %v", err)
		c.JSON(http.StatusInternalServerError, httpx.ErrorResponse{Error: "internal_error", Message: "failed to issue tokens"})
		return
	}

	c.JSON(http.StatusOK, pair)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "invalid_json", Message: "invalid json"})
		return
	}
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, httpx.ErrorResponse{Error: "validation_error", Message: "missing refresh_token", Fields: map[string]string{"refresh_token": "required"}})
		return
	}

	claims, err := auth.ParseRefreshToken(h.Cfg, req.RefreshToken)
	if err != nil || claims.ID == "" {
		c.JSON(http.StatusUnauthorized, httpx.ErrorResponse{Error: "invalid_token", Message: "invalid token"})
		return
	}

	// Best-effort revoke (do not leak existence details)
	_ = h.Repo.RevokeSession(c.Request.Context(), claims.ID)
	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) issueTokenPair(ctx context.Context, userID string) (auth.TokenPair, error) {
	accessToken, err := auth.NewAccessToken(h.Cfg, userID)
	if err != nil {
		return auth.TokenPair{}, err
	}

	sessionID, err := h.Repo.CreateSession(ctx, userID, time.Now().Add(h.Cfg.RefreshTTL))
	if err != nil {
		return auth.TokenPair{}, err
	}

	refreshToken, err := auth.NewRefreshToken(h.Cfg, userID, sessionID)
	if err != nil {
		return auth.TokenPair{}, err
	}

	return auth.TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}
