package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Config struct {
	AccessSecret  []byte
	RefreshSecret []byte
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func LoadConfigFromEnv() (Config, error) {
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if accessSecret == "" || refreshSecret == "" {
		return Config{}, errors.New("JWT_ACCESS_SECRET and JWT_REFRESH_SECRET are required")
	}

	accessTTL := parseDurationOrDefault(os.Getenv("JWT_ACCESS_TTL"), 15*time.Minute)
	refreshTTL := parseDurationOrDefault(os.Getenv("JWT_REFRESH_TTL"), 30*24*time.Hour)

	return Config{
		AccessSecret:  []byte(accessSecret),
		RefreshSecret: []byte(refreshSecret),
		AccessTTL:     accessTTL,
		RefreshTTL:    refreshTTL,
	}, nil
}

func parseDurationOrDefault(v string, d time.Duration) time.Duration {
	if v == "" {
		return d
	}
	parsed, err := time.ParseDuration(v)
	if err != nil {
		return d
	}
	return parsed
}

func NewAccessToken(cfg Config, userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(cfg.AccessSecret)
}

func NewRefreshToken(cfg Config, userID, sessionID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ID:        sessionID, // jti
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(cfg.RefreshTTL)),
	}

	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(cfg.RefreshSecret)
}

func ParseAccessToken(cfg Config, tokenStr string) (*jwt.RegisteredClaims, error) {
	return parseRegisteredClaims(tokenStr, cfg.AccessSecret)
}

func ParseRefreshToken(cfg Config, tokenStr string) (*jwt.RegisteredClaims, error) {
	return parseRegisteredClaims(tokenStr, cfg.RefreshSecret)
}

func parseRegisteredClaims(tokenStr string, secret []byte) (*jwt.RegisteredClaims, error) {
	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, err
	}
	return claims, nil
}
