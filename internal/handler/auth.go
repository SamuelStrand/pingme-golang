package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

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

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "invalid json", nil)
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
		httpx.Error(w, http.StatusBadRequest, "validation_error", "invalid email or password", fields)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to hash password", nil)
		return
	}

	u, err := h.Repo.CreateUser(r.Context(), req.Email, hash)
	if err != nil {
		if errors.Is(err, auth.ErrEmailTaken) {
			httpx.Error(w, http.StatusConflict, "email_taken", "email already exists", map[string]string{"email": "already exists"})
			return
		}
		log.Printf("register: create user failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to create user", nil)
		return
	}

	pair, err := h.issueTokenPair(r.Context(), u.ID)
	if err != nil {
		log.Printf("register: issue tokens failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, pair)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "invalid json", nil)
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || req.Password == "" {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
		return
	}

	u, err := h.Repo.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
			return
		}
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load user", nil)
		return
	}

	if err := auth.ComparePassword(u.Password, req.Password); err != nil {
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "invalid credentials", nil)
		return
	}

	pair, err := h.issueTokenPair(r.Context(), u.ID)
	if err != nil {
		log.Printf("login: issue tokens failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "invalid json", nil)
		return
	}
	if req.RefreshToken == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "missing refresh_token", map[string]string{"refresh_token": "required"})
		return
	}

	claims, err := auth.ParseRefreshToken(h.Cfg, req.RefreshToken)
	if err != nil || claims.Subject == "" || claims.ID == "" {
		httpx.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
		return
	}

	userID, expiresAt, revokedAt, err := h.Repo.GetSession(r.Context(), claims.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
			return
		}
		log.Printf("refresh: load session failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to load session", nil)
		return
	}
	if userID != claims.Subject || revokedAt.Valid || time.Now().After(expiresAt) {
		httpx.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
		return
	}

	_ = h.Repo.RevokeSession(r.Context(), claims.ID)

	pair, err := h.issueTokenPair(r.Context(), userID)
	if err != nil {
		log.Printf("refresh: issue tokens failed: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "failed to issue tokens", nil)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, pair)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_json", "invalid json", nil)
		return
	}
	if req.RefreshToken == "" {
		httpx.Error(w, http.StatusBadRequest, "validation_error", "missing refresh_token", map[string]string{"refresh_token": "required"})
		return
	}

	claims, err := auth.ParseRefreshToken(h.Cfg, req.RefreshToken)
	if err != nil || claims.ID == "" {
		httpx.Error(w, http.StatusUnauthorized, "invalid_token", "invalid token", nil)
		return
	}

	// Best-effort revoke (do not leak existence details)
	_ = h.Repo.RevokeSession(r.Context(), claims.ID)
	w.WriteHeader(http.StatusNoContent)
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
